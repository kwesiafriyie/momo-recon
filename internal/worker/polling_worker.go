package worker

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/kwesiafriyie/momo-recon/internal/model"
	"github.com/kwesiafriyie/momo-recon/internal/repository"
	"github.com/kwesiafriyie/momo-recon/internal/service"
	"github.com/kwesiafriyie/momo-recon/pkg/momo"
)

type PollingWorker struct {
	momoClient      *momo.Client
	transactionRepo *repository.TransactionRepository
	reconciler      *service.ReconciliationService
	interval        time.Duration
}

func NewPollingWorker(
	momoClient *momo.Client,
	transactionRepo *repository.TransactionRepository,
	reconciler *service.ReconciliationService,
) *PollingWorker {
	return &PollingWorker{
		momoClient:      momoClient,
		transactionRepo: transactionRepo,
		reconciler:      reconciler,
		interval:        3 * time.Minute,
	}
}

func (w *PollingWorker) Run(ctx context.Context) {
	log.Println("polling worker: started")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("polling worker: stopped")
			return
		case <-ticker.C:
			if err := w.poll(ctx); err != nil {
				log.Printf("polling worker: error: %v", err)
			}
		}
	}
}

func (w *PollingWorker) poll(ctx context.Context) error {
	// First promote any exhausted transactions to EXPIRED
	if err := w.transactionRepo.MarkExpired(ctx); err != nil {
		log.Printf("polling worker: mark expired: %v", err)
	}

	pending, err := w.transactionRepo.ListPendingForPolling(ctx)
	if err != nil {
		return err
	}

	log.Printf("polling worker: checking %d pending transactions", len(pending))

	for _, tx := range pending {
		if err := w.checkTransaction(ctx, tx); err != nil {
			log.Printf("polling worker: check tx %s: %v", tx.ID, err)
		}
	}
	return nil
}

func (w *PollingWorker) checkTransaction(ctx context.Context, tx *model.Transaction) error {
	status, err := w.momoClient.GetTransactionStatus(ctx, tx.ReferenceID.String())
	if err != nil {
		// Network / MoMo error — still increment poll count to avoid infinite retries
		_ = w.transactionRepo.UpdateFromPoll(ctx, tx.ReferenceID, "", tx.Status)
		return err
	}

	var newStatus model.TransactionStatus
	switch status.Status {
	case "SUCCESSFUL":
		newStatus = model.TransactionStatusSuccessful
	case "FAILED":
		newStatus = model.TransactionStatusFailed
	default:
		newStatus = model.TransactionStatusPending
	}

	// Parse amount returned by MoMo for reconciliation
	amount, _ := strconv.ParseFloat(status.Amount, 64)
	tx.Amount = amount
	tx.MoMoTransactionID = status.FinancialTransactionId
	tx.Status = newStatus

	if err := w.transactionRepo.UpdateFromPoll(ctx, tx.ReferenceID, status.FinancialTransactionId, newStatus); err != nil {
		return err
	}

	return w.reconciler.MatchTransaction(ctx, tx)
}
