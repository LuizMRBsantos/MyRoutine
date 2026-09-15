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
	// Cartão de crédito e parcelamento (nulos em pix/débito/dinheiro)
	CreditCardID       *string `json:"credit_card_id,omitempty"`
	CreditCardName     *string `json:"credit_card_name,omitempty"`
	PurchasedOn        *string `json:"purchased_on,omitempty"`
	InstallmentGroupID *string `json:"installment_group_id,omitempty"`
	InstallmentNumber  *int    `json:"installment_number,omitempty"`
	InstallmentTotal   *int    `json:"installment_total,omitempty"`
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
	// Compra no cartão: amount_cents é o valor TOTAL da compra e
	// occurred_on é ignorado — as datas de cobrança saem da regra do cartão.
	CreditCardID *string `json:"credit_card_id"`
	Installments int     `json:"installments"`
}

type CreditCardDTO struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	ClosingDay int     `json:"closing_day"`
	DueDay     int     `json:"due_day"`
	Color      *string `json:"color"`
	IsActive   bool    `json:"is_active"`
}

type CreateCreditCardInput struct {
	Name       string  `json:"name"`
	ClosingDay int     `json:"closing_day"`
	DueDay     int     `json:"due_day"`
	Color      *string `json:"color"`
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

// Colunas com prefixo t. para permitir o LEFT JOIN com credit_cards.
const transactionColumns = `t.id::text, t.amount_cents, t.kind, t.category, t.description, t.method,
	t.occurred_on::text, t.source_type, t.source_id::text, t.created_at,
	t.credit_card_id::text, c.name, t.purchased_on::text,
	t.installment_group_id::text, t.installment_number, t.installment_total`

const transactionFrom = `FROM transactions t LEFT JOIN credit_cards c ON c.id = t.credit_card_id`

// returningJoined wraps a writing statement in a CTE so the row it returns can
// still be joined with credit_cards — RETURNING alone cannot join.
func returningJoined(write string) string {
	return `WITH written AS (` + write + ` RETURNING *)
		SELECT ` + transactionColumns + ` FROM written t
		LEFT JOIN credit_cards c ON c.id = t.credit_card_id`
}

func scanTransaction(scan func(dest ...any) error) (TransactionDTO, error) {
	var t TransactionDTO
	err := scan(&t.ID, &t.AmountCents, &t.Kind, &t.Category, &t.Description, &t.Method,
		&t.OccurredOn, &t.SourceType, &t.SourceID, &t.CreatedAt,
		&t.CreditCardID, &t.CreditCardName, &t.PurchasedOn,
		&t.InstallmentGroupID, &t.InstallmentNumber, &t.InstallmentTotal)
	return t, err
}

func (s *FinanceService) ListTransactions(ctx context.Context, userID, from, to, category string) ([]TransactionDTO, error) {
	query := `SELECT ` + transactionColumns + ` ` + transactionFrom + `
		WHERE t.user_id = $1 AND t.occurred_on BETWEEN $2::date AND $3::date`
	args := []any{userID, from, to}
	if category != "" {
		args = append(args, category)
		query += fmt.Sprintf(" AND t.category = $%d", len(args))
	}
	query += " ORDER BY t.occurred_on DESC, t.installment_number ASC NULLS FIRST, t.created_at DESC"

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

	write := `INSERT INTO transactions
		(user_id, amount_cents, kind, category, description, method, occurred_on, source_type, source_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9)`
	if input.SourceID != nil {
		write += `
		ON CONFLICT (source_type, source_id) WHERE source_id IS NOT NULL
		DO UPDATE SET amount_cents = EXCLUDED.amount_cents, kind = EXCLUDED.kind,
		  category = EXCLUDED.category, description = EXCLUDED.description,
		  method = EXCLUDED.method, occurred_on = EXCLUDED.occurred_on, updated_at = NOW()`
	}
	query := returningJoined(write)

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

// CreateCardPurchase records a credit card purchase, expanding it into one
// transaction per installment.
//
// Each installment is a real cash event on its own due date, so the monthly
// summary and budgets keep working untouched and future commitments become
// visible. amount_cents is the purchase total; the split never loses a cent.
func (s *FinanceService) CreateCardPurchase(ctx context.Context, userID string, input CreateTransactionInput) ([]TransactionDTO, error) {
	if input.CreditCardID == nil {
		return nil, fmt.Errorf("credit_card_id is required")
	}
	if input.Installments < 1 {
		input.Installments = 1
	}
	if input.Category == "" {
		input.Category = "other"
	}

	card, err := s.GetCard(ctx, *input.CreditCardID, userID)
	if err != nil {
		return nil, err
	}

	purchasedOn := time.Now()
	if input.OccurredOn != "" {
		purchasedOn, err = time.Parse("2006-01-02", input.OccurredOn)
		if err != nil {
			return nil, fmt.Errorf("invalid purchase date: %w", err)
		}
	}

	amounts := splitInstallments(input.AmountCents, input.Installments)
	dates := installmentDates(purchasedOn, card.ClosingDay, card.DueDay, input.Installments)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var groupID *string
	if input.Installments > 1 {
		var id string
		if err := tx.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&id); err != nil {
			return nil, fmt.Errorf("generating installment group: %w", err)
		}
		groupID = &id
	}

	created := make([]TransactionDTO, 0, input.Installments)
	for i := 0; i < input.Installments; i++ {
		var number, total *int
		if input.Installments > 1 {
			n, t := i+1, input.Installments
			number, total = &n, &t
		}

		row := tx.QueryRow(ctx, returningJoined(`INSERT INTO transactions
			(user_id, amount_cents, kind, category, description, method, occurred_on,
			 source_type, credit_card_id, purchased_on, installment_group_id,
			 installment_number, installment_total)
			VALUES ($1, $2, 'expense', $3, $4, 'credit', $5::date, $6, $7, $8::date, $9, $10, $11)`),
			userID, amounts[i], input.Category, input.Description,
			dates[i].Format("2006-01-02"), input.SourceType,
			*input.CreditCardID, purchasedOn.Format("2006-01-02"),
			groupID, number, total,
		)

		t, err := scanTransaction(row.Scan)
		if err != nil {
			return nil, fmt.Errorf("creating installment %d: %w", i+1, err)
		}
		created = append(created, t)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing purchase: %w", err)
	}
	return created, nil
}

// DeleteInstallmentGroup removes every installment of a purchase at once.
func (s *FinanceService) DeleteInstallmentGroup(ctx context.Context, groupID, userID string) error {
	tag, err := s.db.Exec(ctx,
		"DELETE FROM transactions WHERE installment_group_id = $1 AND user_id = $2",
		groupID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Credit cards ─────────────────────────────────────────────────────────────

const creditCardColumns = `id::text, name, closing_day, due_day, color, is_active`

func scanCreditCard(scan func(dest ...any) error) (CreditCardDTO, error) {
	var c CreditCardDTO
	err := scan(&c.ID, &c.Name, &c.ClosingDay, &c.DueDay, &c.Color, &c.IsActive)
	return c, err
}

func (s *FinanceService) ListCards(ctx context.Context, userID string) ([]CreditCardDTO, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+creditCardColumns+`
		 FROM credit_cards WHERE user_id = $1 AND is_active = true ORDER BY name ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing cards: %w", err)
	}
	defer rows.Close()

	cards := []CreditCardDTO{}
	for rows.Next() {
		c, err := scanCreditCard(rows.Scan)
		if err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

func (s *FinanceService) GetCard(ctx context.Context, cardID, userID string) (*CreditCardDTO, error) {
	row := s.db.QueryRow(ctx,
		`SELECT `+creditCardColumns+`
		 FROM credit_cards WHERE id = $1 AND user_id = $2 AND is_active = true`,
		cardID, userID,
	)
	c, err := scanCreditCard(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting card: %w", err)
	}
	return &c, nil
}

func (s *FinanceService) CreateCard(ctx context.Context, userID string, input CreateCreditCardInput) (*CreditCardDTO, error) {
	row := s.db.QueryRow(ctx,
		`INSERT INTO credit_cards (user_id, name, closing_day, due_day, color)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+creditCardColumns,
		userID, input.Name, input.ClosingDay, input.DueDay, input.Color,
	)
	c, err := scanCreditCard(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("creating card: %w", err)
	}
	return &c, nil
}

func (s *FinanceService) UpdateCard(ctx context.Context, cardID, userID string, input CreateCreditCardInput) (*CreditCardDTO, error) {
	row := s.db.QueryRow(ctx,
		`UPDATE credit_cards
		 SET name = $3, closing_day = $4, due_day = $5, color = $6
		 WHERE id = $1 AND user_id = $2 AND is_active = true
		 RETURNING `+creditCardColumns,
		cardID, userID, input.Name, input.ClosingDay, input.DueDay, input.Color,
	)
	c, err := scanCreditCard(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("updating card: %w", err)
	}
	return &c, nil
}

// DeleteCard soft-deletes the card. Transactions keep their credit_card_id, so
// past purchases stay attributed to the card that made them.
func (s *FinanceService) DeleteCard(ctx context.Context, cardID, userID string) error {
	tag, err := s.db.Exec(ctx,
		"UPDATE credit_cards SET is_active = false WHERE id = $1 AND user_id = $2 AND is_active = true",
		cardID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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
		returningJoined(`UPDATE transactions SET `+strings.Join(setClauses, ", ")+`
		 WHERE id = $1 AND user_id = $2`),
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
