package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// FinanceService handles transactions and monthly budgets.
// All money values are integer cents.
type FinanceService struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewFinanceService(db *pgxpool.Pool, logger *zap.Logger) *FinanceService {
	return &FinanceService{db: db, logger: logger}
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type TransactionDTO struct {
	ID          string    `json:"id"`
	AmountCents int64     `json:"amount_cents"`
	Kind        string    `json:"kind"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Method      *string   `json:"method"`
	OccurredOn  string    `json:"occurred_on"`
	SourceType  string    `json:"source_type"`
	SourceID    *string   `json:"source_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateTransactionInput struct {
	AmountCents int64   `json:"amount_cents"`
	Kind        string  `json:"kind"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Method      *string `json:"method"`
	OccurredOn  string  `json:"occurred_on"`
	SourceType  string  `json:"source_type"`
	SourceID    *string `json:"source_id"`
}

type BudgetDTO struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Month       string `json:"month"`
	AmountCents int64  `json:"amount_cents"`
}

type CategorySummary struct {
	Category    string `json:"category"`
	SpentCents  int64  `json:"spent_cents"`
	BudgetCents *int64 `json:"budget_cents,omitempty"`
}

type FinanceSummaryDTO struct {
	Month        string            `json:"month"`
	IncomeCents  int64             `json:"income_cents"`
	ExpenseCents int64             `json:"expense_cents"`
	ByCategory   []CategorySummary `json:"by_category"`
}

// ─── Transactions ─────────────────────────────────────────────────────────────

const transactionColumns = `id::text, amount_cents, kind, category, description, method,
	occurred_on::text, source_type, source_id::text, created_at`

func scanTransaction(scan func(dest ...any) error) (TransactionDTO, error) {
	var t TransactionDTO
	err := scan(&t.ID, &t.AmountCents, &t.Kind, &t.Category, &t.Description, &t.Method,
		&t.OccurredOn, &t.SourceType, &t.SourceID, &t.CreatedAt)
	return t, err
}

func (s *FinanceService) ListTransactions(ctx context.Context, userID, from, to, category string) ([]TransactionDTO, error) {
	query := `SELECT ` + transactionColumns + `
		FROM transactions
		WHERE user_id = $1 AND occurred_on BETWEEN $2::date AND $3::date`
	args := []any{userID, from, to}
	if category != "" {
		args = append(args, category)
		query += fmt.Sprintf(" AND category = $%d", len(args))
	}
	query += " ORDER BY occurred_on DESC, created_at DESC"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing transactions: %w", err)
	}
	defer rows.Close()

	var txs []TransactionDTO
	for rows.Next() {
		t, err := scanTransaction(rows.Scan)
		if err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	if txs == nil {
		txs = []TransactionDTO{}
	}
	return txs, rows.Err()
}

// CreateTransaction inserts a transaction. When source_id is present, the
// insert is idempotent per (source_type, source_id): re-pushing the same
// event updates the row instead of duplicating it.
func (s *FinanceService) CreateTransaction(ctx context.Context, userID string, input CreateTransactionInput) (*TransactionDTO, error) {
	if input.Kind == "" {
		input.Kind = "expense"
	}
	if input.Category == "" {
		input.Category = "other"
	}
	if input.SourceType == "" {
		input.SourceType = "manual"
	}
	if input.OccurredOn == "" {
		input.OccurredOn = time.Now().Format("2006-01-02")
	}

	query := `INSERT INTO transactions
		(user_id, amount_cents, kind, category, description, method, occurred_on, source_type, source_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9)`
	if input.SourceID != nil {
		query += `
		ON CONFLICT (source_type, source_id) WHERE source_id IS NOT NULL
		DO UPDATE SET amount_cents = EXCLUDED.amount_cents, kind = EXCLUDED.kind,
		  category = EXCLUDED.category, description = EXCLUDED.description,
		  method = EXCLUDED.method, occurred_on = EXCLUDED.occurred_on, updated_at = NOW()`
	}
	query += `
		RETURNING ` + transactionColumns

	row := s.db.QueryRow(ctx, query,
		userID, input.AmountCents, input.Kind, input.Category, input.Description,
		input.Method, input.OccurredOn, input.SourceType, input.SourceID,
	)
	t, err := scanTransaction(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("creating transaction: %w", err)
	}
	return &t, nil
}

// UpdateTransaction applies a presence-based partial update (same contract
// as task PATCH: absent keeps, null clears nullable columns).
func (s *FinanceService) UpdateTransaction(ctx context.Context, txID, userID string, fields map[string]json.RawMessage) (*TransactionDTO, error) {
	allowed := map[string]string{
		"amount_cents": "amount_cents",
		"kind":         "kind",
		"category":     "category",
		"description":  "description",
		"method":       "method",
		"occurred_on":  "occurred_on",
	}
	casts := map[string]string{"occurred_on": "::date"}

	setClauses := []string{"updated_at = now()"}
	args := []any{txID, userID}

	for field, raw := range fields {
		col, ok := allowed[field]
		if !ok {
			continue
		}
		var value any
		if string(raw) == "null" {
			value = nil
		} else if field == "amount_cents" {
			var v int64
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, fmt.Errorf("invalid %s: %w", field, err)
			}
			value = v
		} else {
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, fmt.Errorf("invalid %s: %w", field, err)
			}
			value = v
		}
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d%s", col, len(args), casts[field]))
	}

	row := s.db.QueryRow(ctx,
		`UPDATE transactions SET `+strings.Join(setClauses, ", ")+`
		 WHERE id = $1 AND user_id = $2
		 RETURNING `+transactionColumns,
		args...,
	)
	t, err := scanTransaction(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("updating transaction: %w", err)
	}
	return &t, nil
}

func (s *FinanceService) DeleteTransaction(ctx context.Context, txID, userID string) error {
	tag, err := s.db.Exec(ctx,
		"DELETE FROM transactions WHERE id = $1 AND user_id = $2",
		txID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Summary ──────────────────────────────────────────────────────────────────

// GetSummary aggregates a month: totals, per-category spend and the budget
// set for each category (categories with a budget but no spend included).
func (s *FinanceService) GetSummary(ctx context.Context, userID, month string) (*FinanceSummaryDTO, error) {
	summary := &FinanceSummaryDTO{Month: month, ByCategory: []CategorySummary{}}

	err := s.db.QueryRow(ctx, `
		SELECT
		  COALESCE(SUM(amount_cents) FILTER (WHERE kind = 'income'), 0),
		  COALESCE(SUM(amount_cents) FILTER (WHERE kind = 'expense'), 0)
		FROM transactions
		WHERE user_id = $1 AND date_trunc('month', occurred_on) = date_trunc('month', $2::date)`,
		userID, month,
	).Scan(&summary.IncomeCents, &summary.ExpenseCents)
	if err != nil {
		return nil, fmt.Errorf("summarizing totals: %w", err)
	}

	rows, err := s.db.Query(ctx, `
		WITH spend AS (
		  SELECT category, SUM(amount_cents) AS spent
		  FROM transactions
		  WHERE user_id = $1 AND kind = 'expense'
		    AND date_trunc('month', occurred_on) = date_trunc('month', $2::date)
		  GROUP BY category
		),
		budget AS (
		  SELECT category, amount_cents
		  FROM budgets
		  WHERE user_id = $1 AND month = date_trunc('month', $2::date)
		)
		SELECT COALESCE(s.category, b.category), COALESCE(s.spent, 0), b.amount_cents
		FROM spend s
		FULL OUTER JOIN budget b ON b.category = s.category
		ORDER BY COALESCE(s.spent, 0) DESC`,
		userID, month,
	)
	if err != nil {
		return nil, fmt.Errorf("summarizing categories: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var c CategorySummary
		if err := rows.Scan(&c.Category, &c.SpentCents, &c.BudgetCents); err != nil {
			return nil, err
		}
		summary.ByCategory = append(summary.ByCategory, c)
	}
	return summary, rows.Err()
}

// ─── Budgets ──────────────────────────────────────────────────────────────────

func (s *FinanceService) ListBudgets(ctx context.Context, userID, month string) ([]BudgetDTO, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, category, month::text, amount_cents
		FROM budgets
		WHERE user_id = $1 AND month = date_trunc('month', $2::date)
		ORDER BY category ASC`,
		userID, month,
	)
	if err != nil {
		return nil, fmt.Errorf("listing budgets: %w", err)
	}
	defer rows.Close()

	var budgets []BudgetDTO
	for rows.Next() {
		var b BudgetDTO
		if err := rows.Scan(&b.ID, &b.Category, &b.Month, &b.AmountCents); err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}
	if budgets == nil {
		budgets = []BudgetDTO{}
	}
	return budgets, rows.Err()
}

// UpsertBudget sets the budget for (category, month) — one row per pair.
func (s *FinanceService) UpsertBudget(ctx context.Context, userID, category, month string, amountCents int64) (*BudgetDTO, error) {
	var b BudgetDTO
	err := s.db.QueryRow(ctx, `
		INSERT INTO budgets (user_id, category, month, amount_cents)
		VALUES ($1, $2, date_trunc('month', $3::date), $4)
		ON CONFLICT (user_id, category, month)
		DO UPDATE SET amount_cents = EXCLUDED.amount_cents
		RETURNING id::text, category, month::text, amount_cents`,
		userID, category, month, amountCents,
	).Scan(&b.ID, &b.Category, &b.Month, &b.AmountCents)
	if err != nil {
		return nil, fmt.Errorf("upserting budget: %w", err)
	}
	return &b, nil
}

func (s *FinanceService) DeleteBudget(ctx context.Context, budgetID, userID string) error {
	tag, err := s.db.Exec(ctx,
		"DELETE FROM budgets WHERE id = $1 AND user_id = $2",
		budgetID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
