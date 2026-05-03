package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kwesiafriyie/momo-recon/internal/model"
	"github.com/kwesiafriyie/momo-recon/internal/repository"
	"github.com/kwesiafriyie/momo-recon/pkg/momo"
)

const momoCurrency = "GHS" // Ghana Cedis

type MoMoService struct {
	momoClient      *momo.Client
	invoiceRepo     *repository.InvoiceRepository
	transactionRepo *repository.TransactionRepository
	eventRepo       *repository.EventRepository
}

func NewMoMoService(
	client *momo.Client,
	invoiceRepo *repository.InvoiceRepository,
	transactionRepo *repository.TransactionRepository,
	eventRepo *repository.EventRepository,
) *MoMoService {
	return &MoMoService{
		momoClient:      client,
		invoiceRepo:     invoiceRepo,
		transactionRepo: transactionRepo,
		eventRepo:       eventRepo,
	}
}

type PayRequest struct {
	ReferenceCode string `json:"reference_code"`
	PhoneNumber   string `json:"phone_number"`
}

// InitiatePayment looks up the invoice, calls MoMo requesttopay, and stores a PENDING transaction.
func (s *MoMoService) InitiatePayment(ctx context.Context, req PayRequest) (*model.Transaction, error) {
	invoice, err := s.invoiceRepo.GetByReferenceCode(ctx, req.ReferenceCode)
	if err != nil {
		return nil, fmt.Errorf("invoice not found: %w", err)
	}

	// referenceId is our internal UUID sent to MoMo as X-Reference-Id.
	// MoMo echoes it back and we use it to poll.
	referenceID := uuid.New()

	if err := s.momoClient.RequestToPay(
		ctx,
		invoice.ExpectedAmount,
		momoCurrency,
		req.PhoneNumber,
		invoice.ReferenceCode, // externalId — ties callback back to our invoice
		referenceID.String(),
	); err != nil {
		return nil, fmt.Errorf("momo requesttopay: %w", err)
	}

	tx := &model.Transaction{
		ID:          uuid.New(),
		ReferenceID: referenceID,
		ExternalID:  invoice.ReferenceCode,
		Amount:      invoice.ExpectedAmount,
		PhoneNumber: req.PhoneNumber,
		Status:      model.TransactionStatusPending,
		PollCount:   0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("store transaction: %w", err)
	}
	return tx, nil
}

// StoreCallback persists the raw callback payload immediately and returns.
// Processing happens asynchronously in the event worker.
func (s *MoMoService) StoreCallback(ctx context.Context, payload []byte) error {
	event := &model.MoMoEvent{
		ID:        uuid.New(),
		Payload:   payload,
		Processed: false,
		CreatedAt: time.Now(),
	}
	return s.eventRepo.Create(ctx, event)
}

func (s *MoMoService) ListTransactions(ctx context.Context) ([]*model.Transaction, error) {
	return s.transactionRepo.List(ctx)
}
