package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/yourorg/momo-recon/internal/service"
)

type MoMoHandler struct {
	svc *service.MoMoService
}

func NewMoMoHandler(svc *service.MoMoService) *MoMoHandler {
	return &MoMoHandler{svc: svc}
}

// POST /api/pay  — initiates a payment request
func (h *MoMoHandler) Pay(w http.ResponseWriter, r *http.Request) {
	var req service.PayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ReferenceCode == "" || req.PhoneNumber == "" {
		writeError(w, http.StatusBadRequest, "reference_code and phone_number are required")
		return
	}

	tx, err := h.svc.InitiatePayment(r.Context(), req)
	if err != nil {
		log.Printf("pay: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":       "PENDING",
		"transaction":  tx,
	})
}

// POST /api/momo/callback  — MoMo webhook receiver
// Must respond quickly — store only, no processing here.
func (h *MoMoHandler) Callback(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		log.Printf("callback: read body: %v", err)
		w.WriteHeader(http.StatusOK) // always 200 to MoMo
		return
	}

	if err := h.svc.StoreCallback(r.Context(), body); err != nil {
		log.Printf("callback: store event: %v", err)
		// Still return 200 — we don't want MoMo to retry based on our DB issues
	}

	w.WriteHeader(http.StatusOK)
}

// GET /api/transactions
func (h *MoMoHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	txs, err := h.svc.ListTransactions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch transactions")
		return
	}
	writeJSON(w, http.StatusOK, txs)
}
