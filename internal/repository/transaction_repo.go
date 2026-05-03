package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kwesiafriyie/momo-recon/internal/model"
)

type TransactionRepository struct {
	db *pgxpool.Pool
}

func (r *TransactionRepository) GetByID(context context.Context, id uuid.UUID) (any, error) {
	panic("unimplemented")
}

func NewTransactionRepository(db *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, tx *model.Transaction) error {
	q := `
		INSERT INTO transactions
			(id, reference_id, external_id, amount, phone_number, status, poll_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.Exec(ctx, q,
		tx.ID, tx.ReferenceID, tx.ExternalID, tx.Amount, tx.PhoneNumber,
		tx.Status, tx.PollCount, tx.CreatedAt, tx.UpdatedAt,
	)
	return err
}

func (r *TransactionRepository) GetByReferenceID(ctx context.Context, refID uuid.UUID) (*model.Transaction, error) {
	q := `SELECT id, reference_id, external_id, momo_transaction_id, amount, phone_number,
			status, last_polled_at, poll_count, created_at, updated_at
		  FROM transactions WHERE reference_id = $1`
	row := r.db.QueryRow(ctx, q, refID)
	return scanTransaction(row)
}

// UpdateFromCallback sets status and momo_transaction_id idempotently.
// Uses momo_transaction_id UNIQUE constraint — duplicate callbacks are silently ignored.
func (r *TransactionRepository) UpdateFromCallback(
	ctx context.Context,
	referenceID uuid.UUID,
	momoTxID string,
	status model.TransactionStatus,
) error {
	q := `
		UPDATE transactions
		SET status = $1, momo_transaction_id = $2, updated_at = NOW()
		WHERE reference_id = $3
		  AND status = 'PENDING'` // only advance from PENDING — idempotency guard
	_, err := r.db.Exec(ctx, q, status, momoTxID, referenceID)
	return err
}

// UpdateFromPoll updates status and increments poll_count. If poll_count reaches 10, marks EXPIRED.
func (r *TransactionRepository) UpdateFromPoll(
	ctx context.Context,
	referenceID uuid.UUID,
	momoTxID string,
	status model.TransactionStatus,
) error {
	now := time.Now()
	q := `
		UPDATE transactions
		SET status          = $1,
		    momo_transaction_id = COALESCE(NULLIF($2, ''), momo_transaction_id),
		    last_polled_at  = $3,
		    poll_count      = poll_count + 1,
		    updated_at      = $3
		WHERE reference_id = $4
		  AND status = 'PENDING'`
	_, err := r.db.Exec(ctx, q, status, momoTxID, now, referenceID)
	return err
}

// MarkExpired promotes any PENDING transactions that have exhausted their poll budget.
func (r *TransactionRepository) MarkExpired(ctx context.Context) error {
	q := `
		UPDATE transactions
		SET status = 'EXPIRED', updated_at = NOW()
		WHERE status = 'PENDING' AND poll_count >= 10`
	_, err := r.db.Exec(ctx, q)
	return err
}

// ListPendingForPolling returns PENDING transactions that still have poll budget.
func (r *TransactionRepository) ListPendingForPolling(ctx context.Context) ([]*model.Transaction, error) {
	q := `
		SELECT id, reference_id, external_id, momo_transaction_id, amount, phone_number,
		       status, last_polled_at, poll_count, created_at, updated_at
		FROM transactions
		WHERE status = 'PENDING' AND poll_count < 10
		ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*model.Transaction
	for rows.Next() {
		tx, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

func (r *TransactionRepository) List(ctx context.Context) ([]*model.Transaction, error) {
	q := `SELECT id, reference_id, external_id, momo_transaction_id, amount, phone_number,
			status, last_polled_at, poll_count, created_at, updated_at
		  FROM transactions ORDER BY created_at DESC LIMIT 100`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*model.Transaction
	for rows.Next() {
		tx, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

func (r *TransactionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.TransactionStatus) error {
	q := `UPDATE transactions SET status = $1, updated_at = NOW() WHERE id = $2`
	tag, err := r.db.Exec(ctx, q, status, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("transaction %s not found", id)
	}
	return nil
}

func scanTransaction(s scanner) (*model.Transaction, error) {
	var tx model.Transaction
	err := s.Scan(
		&tx.ID, &tx.ReferenceID, &tx.ExternalID, &tx.MoMoTransactionID,
		&tx.Amount, &tx.PhoneNumber, &tx.Status,
		&tx.LastPolledAt, &tx.PollCount, &tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}
