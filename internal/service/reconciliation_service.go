package service

import (
	"context"
	"fmt"
	"log"

	"github.com/yourorg/momo-recon/internal/model"
	"github.com/yourorg/momo-recon/internal/repository"
)

type ReconciliationService struct {
	invoiceRepo     *repository.InvoiceRepository
	transactionRepo *repository.TransactionRepository
}

func NewReconciliationService(
	invoiceRepo *repository.InvoiceRepository,
	transactionRepo *repository.TransactionRepository,
) *ReconciliationService {
	return &ReconciliationService{
		invoiceRepo:     invoiceRepo,
		transactionRepo: transactionRepo,
	}
}

// MatchTransaction reconciles a completed transaction against its invoice.
// Only SUCCESSFUL transactions trigger an invoice status change.
func (s *ReconciliationService) MatchTransaction(ctx context.Context, tx *model.Transaction) error {
	if tx.Status != model.TransactionStatusSuccessful {
		// Nothing to reconcile — FAILED / EXPIRED don't change invoice status
		return nil
	}

	invoice, err := s.invoiceRepo.GetByReferenceCode(ctx, tx.ExternalID)
	if err != nil {
		// No matching invoice — mark transaction UNMATCHED and continue
		log.Printf("reconcile: no invoice for external_id=%s tx=%s", tx.ExternalID, tx.ID)
		return s.transactionRepo.UpdateStatus(ctx, tx.ID, model.TransactionStatusUnmatched)
	}

	var newStatus model.InvoiceStatus
	switch {
	case tx.Amount == invoice.ExpectedAmount:
		newStatus = model.InvoiceStatusPaid
	case tx.Amount < invoice.ExpectedAmount:
		newStatus = model.InvoiceStatusPartial
	case tx.Amount > invoice.ExpectedAmount:
		newStatus = model.InvoiceStatusOverpaid
	default:
		return fmt.Errorf("reconcile: unexpected amount comparison")
	}

	if err := s.invoiceRepo.UpdateStatus(ctx, invoice.ID, newStatus); err != nil {
		return fmt.Errorf("reconcile: update invoice status: %w", err)
	}

	log.Printf("reconcile: invoice=%s tx=%s paid=%.2f expected=%.2f -> %s",
		invoice.ReferenceCode, tx.ID, tx.Amount, invoice.ExpectedAmount, newStatus)
	return nil
}
