package model

import (
	"time"

	"github.com/google/uuid"
)

type InvoiceStatus string

const (
	InvoiceStatusPending  InvoiceStatus = "PENDING"
	InvoiceStatusPartial  InvoiceStatus = "PARTIAL"
	InvoiceStatusPaid     InvoiceStatus = "PAID"
	InvoiceStatusOverpaid InvoiceStatus = "OVERPAID"
)

type Invoice struct {
	ID             uuid.UUID     `json:"id"`
	ReferenceCode  string        `json:"reference_code"`
	ExpectedAmount float64       `json:"expected_amount"`
	Status         InvoiceStatus `json:"status"`
	CustomerRef    string        `json:"customer_ref,omitempty"`
	DueDate        *time.Time    `json:"due_date,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
}
