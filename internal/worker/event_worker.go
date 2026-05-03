package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/momo-recon/internal/model"
	"github.com/yourorg/momo-recon/internal/repository"
	"github.com/yourorg/momo-recon/internal/service"
)

// callbackPayload is the shape MoMo sends to our webhook.
type callbackPayload struct {
	FinancialTransactionId string `json:"financialTransactionId"`
	ExternalId             string `json:"externalId"`
	Amount                 string `json:"amount"`
	Currency               string `json:"currency"`
	Payer                  struct {
		PartyIdType string `json:"partyIdType"`
		PartyId     string `json:"partyId"`
	} `json:"payer"`
	Status string `json:"status"` // SUCCESSFUL | FAILED
}

type EventWorker struct {
	eventRepo       *repository.EventRepository
	transactionRepo *repository.TransactionRepository
	reconciler      *service.ReconciliationService
	interval        time.Duration
}

func NewEventWorker(
	eventRepo *repository.EventRepository,
	transactionRepo *repository.TransactionRepository,
	reconciler *service.ReconciliationService,
) *EventWorker {
	return &EventWorker{
		eventRepo:       eventRepo,
		transactionRepo: transactionRepo,
		reconciler:      reconciler,
		interval:        5 * time.Second,
	}
}

func (w *EventWorker) Run(ctx context.Context) {
	log.Println("event worker: started")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("event worker: stopped")
			return
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				log.Printf("event worker: batch error: %v", err)
			}
		}
	}
}

func (w *EventWorker) processBatch(ctx context.Context) error {
	// ClaimUnprocessed uses FOR UPDATE SKIP LOCKED — safe for concurrent workers
	events, err := w.eventRepo.ClaimUnprocessed(ctx, 10)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := w.processEvent(ctx, event); err != nil {
			log.Printf("event worker: process event %s: %v", event.ID, err)
			// Continue processing other events — don't let one failure block the batch
			continue
		}
		if err := w.eventRepo.MarkProcessed(ctx, event.ID); err != nil {
			log.Printf("event worker: mark processed %s: %v", event.ID, err)
		}
	}
	return nil
}

func (w *EventWorker) processEvent(ctx context.Context, event *model.MoMoEvent) error {
	var p callbackPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return err
	}

	// Map MoMo status string to our enum
	var txStatus model.TransactionStatus
	switch p.Status {
	case "SUCCESSFUL":
		txStatus = model.TransactionStatusSuccessful
	case "FAILED":
		txStatus = model.TransactionStatusFailed
	default:
		txStatus = model.TransactionStatusPending
	}

	// externalId = referenceCode = the X-Reference-Id we sent — use it to find our transaction
	refID, err := uuid.Parse(p.ExternalId)
	if err != nil {
		// externalId might be the invoice reference_code, not a UUID.
		// In that case we look up by external_id. For now log and skip.
		log.Printf("event worker: non-UUID externalId %q in event %s", p.ExternalId, event.ID)
		return nil
	}

	if err := w.transactionRepo.UpdateFromCallback(ctx, refID, p.FinancialTransactionId, txStatus); err != nil {
		return err
	}

	// Fetch the updated transaction for reconciliation
	tx, err := w.transactionRepo.GetByReferenceID(ctx, refID)
	if err != nil {
		return err
	}

	return w.reconciler.MatchTransaction(ctx, tx)
}
