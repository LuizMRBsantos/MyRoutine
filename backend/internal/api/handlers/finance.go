package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/service"
)

type FinanceHandler struct {
	cfg        *config.Config
	logger     *zap.Logger
	financeSvc *service.FinanceService
}

func NewFinanceHandler(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *FinanceHandler {
	return &FinanceHandler{
		cfg:        cfg,
		logger:     logger,
		financeSvc: service.NewFinanceService(db, logger),
	}
}

var validTransactionKinds = map[string]bool{"expense": true, "income": true}

// GET /finance/transactions?from&to&category
func (h *FinanceHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}
	if from == "" {
		from = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}

	txs, err := h.financeSvc.ListTransactions(r.Context(), userID, from, to, r.URL.Query().Get("category"))
	if err != nil {
		h.logger.Error("list transactions", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list transactions")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"transactions": txs})
}

// POST /finance/transactions
func (h *FinanceHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input service.CreateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.AmountCents <= 0 {
		respondError(w, http.StatusBadRequest, "amount_cents must be positive")
		return
	}
	if input.Description == "" {
		respondError(w, http.StatusBadRequest, "description is required")
		return
	}
	if input.Kind != "" && !validTransactionKinds[input.Kind] {
		respondError(w, http.StatusBadRequest, "kind must be expense or income")
		return
	}

	tx, err := h.financeSvc.CreateTransaction(r.Context(), userID, input)
	if err != nil {
		h.logger.Error("create transaction", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create transaction")
		return
	}

	respondJSON(w, http.StatusCreated, tx)
}

// PATCH /finance/transactions/{id}
func (h *FinanceHandler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	txID := chi.URLParam(r, "id")

	var fields map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(fields) == 0 {
		respondError(w, http.StatusBadRequest, "no fields to update")
		return
	}
	if raw, ok := fields["kind"]; ok {
		var kind string
		if err := json.Unmarshal(raw, &kind); err != nil || !validTransactionKinds[kind] {
			respondError(w, http.StatusBadRequest, "kind must be expense or income")
			return
		}
	}

	tx, err := h.financeSvc.UpdateTransaction(r.Context(), txID, userID, fields)
	if err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "transaction not found")
			return
		}
		h.logger.Error("update transaction", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to update transaction")
		return
	}

	respondJSON(w, http.StatusOK, tx)
}

// DELETE /finance/transactions/{id}
func (h *FinanceHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	txID := chi.URLParam(r, "id")

	if err := h.financeSvc.DeleteTransaction(r.Context(), txID, userID); err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "transaction not found")
			return
		}
		h.logger.Error("delete transaction", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to delete transaction")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /finance/summary?month=YYYY-MM-DD
func (h *FinanceHandler) Summary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01") + "-01"
	}

	summary, err := h.financeSvc.GetSummary(r.Context(), userID, month)
	if err != nil {
		h.logger.Error("finance summary", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to build summary")
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

// GET /finance/budgets?month=YYYY-MM-DD
func (h *FinanceHandler) ListBudgets(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01") + "-01"
	}

	budgets, err := h.financeSvc.ListBudgets(r.Context(), userID, month)
	if err != nil {
		h.logger.Error("list budgets", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list budgets")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"budgets": budgets})
}

// PUT /finance/budgets — upsert by (category, month)
func (h *FinanceHandler) UpsertBudget(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var body struct {
		Category    string `json:"category"`
		Month       string `json:"month"`
		AmountCents int64  `json:"amount_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Category == "" || body.Month == "" {
		respondError(w, http.StatusBadRequest, "category and month are required")
		return
	}
	if body.AmountCents <= 0 {
		respondError(w, http.StatusBadRequest, "amount_cents must be positive")
		return
	}

	budget, err := h.financeSvc.UpsertBudget(r.Context(), userID, body.Category, body.Month, body.AmountCents)
	if err != nil {
		h.logger.Error("upsert budget", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to save budget")
		return
	}

	respondJSON(w, http.StatusOK, budget)
}

// DELETE /finance/budgets/{id}
func (h *FinanceHandler) DeleteBudget(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	budgetID := chi.URLParam(r, "id")

	if err := h.financeSvc.DeleteBudget(r.Context(), budgetID, userID); err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "budget not found")
			return
		}
		h.logger.Error("delete budget", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to delete budget")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
