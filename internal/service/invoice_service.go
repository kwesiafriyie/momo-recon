package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/kwesiafriyie/momo-recon/internal/model"
	"github.com/kwesiafriyie/momo-recon/internal/repository"
)

type InvoiceService struct {
	repo *repository.InvoiceRepository
}

func NewInvoiceService(repo *repository.InvoiceRepository) *InvoiceService {
	return &InvoiceService{repo: repo}
}

type CreateInvoiceRequest struct {
	Amount      float64    `json:"amount"`
	CustomerRef string     `json:"customer_ref"`
	DueDate     *time.Time `json:"due_date"`
}

func (s *InvoiceService) Create(ctx context.Context, req CreateInvoiceRequest) (*model.Invoice, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	inv := &model.Invoice{
		ID:             uuid.New(),
		ReferenceCode:  generateReferenceCode(),
		ExpectedAmount: req.Amount,
		Status:         model.InvoiceStatusPending,
		CustomerRef:    req.CustomerRef,
		DueDate:        req.DueDate,
		CreatedAt:      time.Now(),
	}

	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}
	return inv, nil
}

func (s *InvoiceService) GetByReferenceCode(ctx context.Context, code string) (*model.Invoice, error) {
	return s.repo.GetByReferenceCode(ctx, code)
}

func (s *InvoiceService) List(ctx context.Context) ([]*model.Invoice, error) {
	return s.repo.List(ctx)
}

func generateReferenceCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return "INV-" + string(b)
}
