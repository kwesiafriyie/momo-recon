package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kwesiafriyie/momo-recon/internal/model"
)

type EventRepository struct {
	db *pgxpool.Pool
}

func NewEventRepository(db *pgxpool.Pool) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, event *model.MoMoEvent) error {
	q := `INSERT INTO momo_events (id, payload, processed, created_at)
		  VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(ctx, q, event.ID, event.Payload, event.Processed, event.CreatedAt)
	return err
}

// ClaimUnprocessed atomically fetches and locks up to `limit` unprocessed events.
// Uses FOR UPDATE SKIP LOCKED so concurrent workers don't double-process.
func (r *EventRepository) ClaimUnprocessed(ctx context.Context, limit int) ([]*model.MoMoEvent, error) {
	q := `
		SELECT id, payload, processed, created_at
		FROM momo_events
		WHERE processed = FALSE
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED`
	rows, err := r.db.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*model.MoMoEvent
	for rows.Next() {
		var e model.MoMoEvent
		if err := rows.Scan(&e.ID, &e.Payload, &e.Processed, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

func (r *EventRepository) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	q := `UPDATE momo_events SET processed = TRUE WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}
