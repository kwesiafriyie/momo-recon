package handler

import (
	"encoding/json"
	"net/http"

	"github.com/kwesiafriyie/momo-recon/internal/service"
)

type InvoiceHandler struct {
	svc *service.InvoiceService
}

func NewInvoiceHandler(svc *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{svc: svc}
}

// POST /api/invoices
func (h *InvoiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req service.CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	inv, err := h.svc.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, inv)
}

// GET /api/invoices
func (h *InvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	invoices, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch invoices")
		return
	}
	writeJSON(w, http.StatusOK, invoices)
}

// GET /api/invoices/{code}
func (h *InvoiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	inv, err := h.svc.GetByReferenceCode(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}
	writeJSON(w, http.StatusOK, inv)
}
