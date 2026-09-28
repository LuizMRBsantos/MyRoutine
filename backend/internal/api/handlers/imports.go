package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/service"
)

type ImportHandler struct {
	cfg       *config.Config
	logger    *zap.Logger
	importSvc *service.ImportService
}

func NewImportHandler(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *ImportHandler {
	return &ImportHandler{
		cfg:       cfg,
		logger:    logger,
		importSvc: service.NewImportService(db, logger),
	}
}

// maxImportBodyBytes caps the JSON body of the import routes. The CSV travels
// inside it as a string, so this bounds how much a single request can make the
// server buffer and parse.
const maxImportBodyBytes = 2 << 20 // 2 MiB

// decodeImportBody decodes a size-capped JSON body. It answers the request
// itself on failure (413 over the cap, 400 otherwise) and reports whether the
// handler may go on.
func decodeImportBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondError(w, http.StatusRequestEntityTooLarge, "file too large (max 2 MiB)")
			return false
		}
		respondError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// POST /finance/imports/preview
// Reads the header and guesses the column layout, so the user usually only
// confirms instead of configuring.
func (h *ImportHandler) Preview(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if !decodeImportBody(w, r, &body) {
		return
	}
	if body.Content == "" {
		respondError(w, http.StatusBadRequest, "content is required")
		return
	}

	mapping, complete := service.SniffMapping(body.Content)

	// Amostra do arquivo para a tela mostrar as primeiras linhas.
	sample, _, err := service.ParseCSVStatement(body.Content, mapping)
	if err != nil || !complete {
		respondJSON(w, http.StatusOK, map[string]any{
			"mapping": mapping, "complete": complete, "sample": []any{},
		})
		return
	}
	if len(sample) > 5 {
		sample = sample[:5]
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"mapping": mapping, "complete": complete, "sample": sample,
	})
}

// POST /finance/imports
func (h *ImportHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input service.CreateImportInput
	if !decodeImportBody(w, r, &input) {
		return
	}
	if input.Content == "" {
		respondError(w, http.StatusBadRequest, "content is required")
		return
	}
	if input.Filename == "" {
		input.Filename = "extrato.csv"
	}

	batch, err := h.importSvc.CreateBatch(r.Context(), userID, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidReference) {
			respondError(w, http.StatusBadRequest, "credit card not found")
			return
		}
		h.logger.Warn("create import batch", zap.Error(err))
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, batch)
}

// GET /finance/imports
func (h *ImportHandler) ListBatches(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	batches, err := h.importSvc.ListBatches(r.Context(), userID)
	if err != nil {
		h.logger.Error("list import batches", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list imports")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"batches": batches})
}

// GET /finance/imports/pending — a caixa de entrada da conciliação
func (h *ImportHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	entries, err := h.importSvc.ListPendingEntries(r.Context(), userID)
	if err != nil {
		h.logger.Error("list pending entries", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list pending entries")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

// GET /finance/imports/{id}
func (h *ImportHandler) GetBatch(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	batchID := chi.URLParam(r, "id")

	entries, err := h.importSvc.ListEntries(r.Context(), batchID, userID)
	if err != nil {
		h.logger.Error("list batch entries", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to load import")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

// POST /finance/imports/entries/{id}/approve
func (h *ImportHandler) ApproveEntry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entryID := chi.URLParam(r, "id")

	var body struct {
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tx, err := h.importSvc.ApproveEntry(r.Context(), entryID, userID, body.Category)
	if err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "entry not found")
			return
		}
		if errors.Is(err, service.ErrInvalidReference) {
			respondError(w, http.StatusBadRequest, "credit card not found")
			return
		}
		h.logger.Warn("approve entry", zap.Error(err))
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, tx)
}

// POST /finance/imports/entries/{id}/decide — ignorar ou marcar como duplicata
func (h *ImportHandler) DecideEntry(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entryID := chi.URLParam(r, "id")

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.importSvc.DecideEntry(r.Context(), entryID, userID, body.Status); err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "entry not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /finance/rules
func (h *ImportHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	rules, err := h.importSvc.ListRules(r.Context(), userID)
	if err != nil {
		h.logger.Error("list rules", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list rules")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"rules": rules})
}

// DELETE /finance/rules/{id}
func (h *ImportHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	ruleID := chi.URLParam(r, "id")

	if err := h.importSvc.DeleteRule(r.Context(), ruleID, userID); err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "rule not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete rule")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
