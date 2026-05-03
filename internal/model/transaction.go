package model

import (
	"time"

	"github.com/google/uuid"
)

type TransactionStatus string

const (
	TransactionStatusPending    TransactionStatus = "PENDING"
	TransactionStatusSuccessful TransactionStatus = "SUCCESSFUL"
	TransactionStatusFailed     TransactionStatus = "FAILED"
	TransactionStatusExpired    TransactionStatus = "EXPIRED"
	TransactionStatusUnmatched  TransactionStatus = "UNMATCHED"
)

type Transaction struct {
	ID                 uuid.UUID         `json:"id"`
	ReferenceID        uuid.UUID         `json:"reference_id"`         // internal tracking ID sent to MoMo
	ExternalID         string            `json:"external_id"`           // = invoice.reference_code
	MoMoTransactionID  string            `json:"momo_transaction_id"`   // financialTransactionId from MoMo
	Amount             float64           `json:"amount"`
	PhoneNumber        string            `json:"phone_number"`
	Status             TransactionStatus `json:"status"`
	LastPolledAt       *time.Time        `json:"last_polled_at,omitempty"`
	PollCount          int               `json:"poll_count"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}
