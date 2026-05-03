package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kwesiafriyie/momo-recon/internal/model"
)

type InvoiceRepository struct {
	db *pgxpool.Pool
}

func NewInvoiceRepository(db *pgxpool.Pool) *InvoiceRepository {
	return &InvoiceRepository{db: db}
}

func (r *InvoiceRepository) Create(ctx context.Context, inv *model.Invoice) error {
	q := `
		INSERT INTO invoices (id, reference_code, expected_amount, status, customer_ref, due_date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, q,
		inv.ID, inv.ReferenceCode, inv.ExpectedAmount, inv.Status,
		inv.CustomerRef, inv.DueDate, inv.CreatedAt,
	)
	return err
}

func (r *InvoiceRepository) GetByReferenceCode(ctx context.Context, code string) (*model.Invoice, error) {
	q := `SELECT id, reference_code, expected_amount, status, customer_ref, due_date, created_at
		  FROM invoices WHERE reference_code = $1`
	row := r.db.QueryRow(ctx, q, code)
	return scanInvoice(row)
}

func (r *InvoiceRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Invoice, error) {
	q := `SELECT id, reference_code, expected_amount, status, customer_ref, due_date, created_at
		  FROM invoices WHERE id = $1`
	row := r.db.QueryRow(ctx, q, id)
	return scanInvoice(row)
}

func (r *InvoiceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.InvoiceStatus) error {
	q := `UPDATE invoices SET status = $1 WHERE id = $2`
	tag, err := r.db.Exec(ctx, q, status, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("invoice %s not found", id)
	}
	return nil
}

func (r *InvoiceRepository) List(ctx context.Context) ([]*model.Invoice, error) {
	q := `SELECT id, reference_code, expected_amount, status, customer_ref, due_date, created_at
		  FROM invoices ORDER BY created_at DESC LIMIT 100`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*model.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanInvoice(s scanner) (*model.Invoice, error) {
	var inv model.Invoice
	err := s.Scan(
		&inv.ID, &inv.ReferenceCode, &inv.ExpectedAmount, &inv.Status,
		&inv.CustomerRef, &inv.DueDate, &inv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}
