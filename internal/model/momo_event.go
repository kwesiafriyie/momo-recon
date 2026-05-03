package model

import (
	"time"

	"github.com/google/uuid"
)

type MoMoEvent struct {
	ID        uuid.UUID `json:"id"`
	Payload   []byte    `json:"payload"` // raw JSON from MoMo callback
	Processed bool      `json:"processed"`
	CreatedAt time.Time `json:"created_at"`
}
