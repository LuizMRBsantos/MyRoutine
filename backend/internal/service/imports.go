package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ImportService stages statement lines and turns the approved ones into
// transactions. Nothing reaches `transactions` before the user decides: an
// unreviewed line is a proposal, not a fact about the money.
type ImportService struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewImportService(db *pgxpool.Pool, logger *zap.Logger) *ImportService {
	return &ImportService{db: db, logger: logger}
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type ImportEntryDTO struct {
	ID                string  `json:"id"`
	OccurredOn        string  `json:"occurred_on"`
	AmountCents       int64   `json:"amount_cents"`
	Kind              string  `json:"kind"`
	RawDescription    string  `json:"raw_description"`
	SuggestedCategory *string `json:"suggested_category"`
	Status            string  `json:"status"`
	// Preenchido quando a linha parece duplicata de algo que já existe
	MatchedTransactionID          *string `json:"matched_transaction_id,omitempty"`
	MatchedTransactionDescription *string `json:"matched_transaction_description,omitempty"`
	MatchedTransactionDate        *string `json:"matched_transaction_date,omitempty"`
}

type ImportBatchDTO struct {
	ID           string           `json:"id"`
	Filename     string           `json:"filename"`
	CreditCardID *string          `json:"credit_card_id"`
	AccountLabel *string          `json:"account_label"`
	CreatedAt    time.Time        `json:"created_at"`
	Entries      []ImportEntryDTO `json:"entries,omitempty"`
	// Resumo do que aconteceu na leitura do arquivo
	NewCount       int      `json:"new_count"`
	DuplicateCount int      `json:"duplicate_count"`
	MatchedCount   int      `json:"matched_count"`
	SkippedLines   []string `json:"skipped_lines"`
}

type CreateImportInput struct {
	Filename     string     `json:"filename"`
	Content      string     `json:"content"`
	Mapping      CSVMapping `json:"mapping"`
	CreditCardID *string    `json:"credit_card_id"`
	AccountLabel *string    `json:"account_label"`
}

// ─── Importar ─────────────────────────────────────────────────────────────────

// CreateBatch parses the file, drops lines already imported before, suggests a
// category from the learned rules and flags likely duplicates of existing
// transactions. It writes only to the staging tables.
func (s *ImportService) CreateBatch(ctx context.Context, userID string, input CreateImportInput) (*ImportBatchDTO, error) {
	parsed, skipped, err := ParseCSVStatement(input.Content, input.Mapping)
	if err != nil {
		return nil, err
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("nenhum lançamento reconhecido no arquivo")
	}

	rules, err := s.ListRules(ctx, userID)
	if err != nil {
		return nil, err
	}

	candidates, err := s.matchCandidates(ctx, userID, parsed)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	batch := ImportBatchDTO{SkippedLines: skipped}
	err = tx.QueryRow(ctx, `
		INSERT INTO import_batches (user_id, filename, credit_card_id, account_label)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, filename, credit_card_id::text, account_label, created_at`,
		userID, input.Filename, input.CreditCardID, input.AccountLabel,
	).Scan(&batch.ID, &batch.Filename, &batch.CreditCardID, &batch.AccountLabel, &batch.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating batch: %w", err)
	}

	for _, p := range parsed {
		fingerprint := entryFingerprint(p.OccurredOn, p.AmountCents, p.RawDescription)

		var suggested *string
		if category := suggestCategory(p.RawDescription, rules); category != "" {
			suggested = &category
		}

		var matchedID *string
		status := "pending"
		if match := findMatch(p.OccurredOn, p.AmountCents, p.RawDescription, candidates); match != nil {
			matchedID = &match.ID
		}

		var entryID string
		err := tx.QueryRow(ctx, `
			INSERT INTO import_entries
			  (batch_id, user_id, occurred_on, amount_cents, kind, raw_description,
			   suggested_category, matched_transaction_id, status, fingerprint)
			VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (user_id, fingerprint) DO NOTHING
			RETURNING id::text`,
			batch.ID, userID, p.OccurredOn, p.AmountCents, p.Kind, p.RawDescription,
			suggested, matchedID, status, fingerprint,
		).Scan(&entryID)

		if errors.Is(err, pgx.ErrNoRows) {
			// Já importada numa remessa anterior — silenciosamente ignorada.
			batch.DuplicateCount++
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("creating entry: %w", err)
		}

		batch.NewCount++
		if matchedID != nil {
			batch.MatchedCount++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing batch: %w", err)
	}

	entries, err := s.ListEntries(ctx, batch.ID, userID)
	if err != nil {
		return nil, err
	}
	batch.Entries = entries
	return &batch, nil
}

// matchCandidates loads existing transactions in the date range the file
// covers, so duplicate detection compares against the right slice only.
func (s *ImportService) matchCandidates(ctx context.Context, userID string, parsed []ParsedEntry) ([]MatchCandidate, error) {
	from, to := parsed[0].OccurredOn, parsed[0].OccurredOn
	for _, p := range parsed {
		if p.OccurredOn < from {
			from = p.OccurredOn
		}
		if p.OccurredOn > to {
			to = p.OccurredOn
		}
	}

	// Alarga a janela em Go: aritmética de data no SQL exigia casts extras
	// só para o Postgres inferir o tipo do parâmetro.
	widen := func(day string, days int) string {
		d, err := time.Parse("2006-01-02", day)
		if err != nil {
			return day
		}
		return d.AddDate(0, 0, days).Format("2006-01-02")
	}

	rows, err := s.db.Query(ctx, `
		SELECT id::text, amount_cents, occurred_on::text, description
		FROM transactions
		WHERE user_id = $1 AND occurred_on BETWEEN $2::date AND $3::date`,
		userID, widen(from, -matchWindowDays), widen(to, matchWindowDays),
	)
	if err != nil {
		return nil, fmt.Errorf("loading match candidates: %w", err)
	}
	defer rows.Close()

	candidates := []MatchCandidate{}
	for rows.Next() {
		var c MatchCandidate
		if err := rows.Scan(&c.ID, &c.AmountCents, &c.OccurredOn, &c.Description); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}

// ─── Consultar ────────────────────────────────────────────────────────────────

const importEntryColumns = `e.id::text, e.occurred_on::text, e.amount_cents, e.kind,
	e.raw_description, e.suggested_category, e.status,
	e.matched_transaction_id::text, t.description, t.occurred_on::text`

func scanImportEntry(scan func(dest ...any) error) (ImportEntryDTO, error) {
	var e ImportEntryDTO
	err := scan(&e.ID, &e.OccurredOn, &e.AmountCents, &e.Kind,
		&e.RawDescription, &e.SuggestedCategory, &e.Status,
		&e.MatchedTransactionID, &e.MatchedTransactionDescription, &e.MatchedTransactionDate)
	return e, err
}

func (s *ImportService) ListEntries(ctx context.Context, batchID, userID string) ([]ImportEntryDTO, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+importEntryColumns+`
		FROM import_entries e
		LEFT JOIN transactions t ON t.id = e.matched_transaction_id
		WHERE e.batch_id = $1 AND e.user_id = $2
		ORDER BY e.occurred_on ASC, e.created_at ASC`,
		batchID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing entries: %w", err)
	}
	defer rows.Close()

	entries := []ImportEntryDTO{}
	for rows.Next() {
		e, err := scanImportEntry(rows.Scan)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ListPendingEntries returns everything still awaiting a decision, across
// batches — the reconciliation inbox.
func (s *ImportService) ListPendingEntries(ctx context.Context, userID string) ([]ImportEntryDTO, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+importEntryColumns+`
		FROM import_entries e
		LEFT JOIN transactions t ON t.id = e.matched_transaction_id
		WHERE e.user_id = $1 AND e.status = 'pending'
		ORDER BY e.occurred_on ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing pending entries: %w", err)
	}
	defer rows.Close()

	entries := []ImportEntryDTO{}
	for rows.Next() {
		e, err := scanImportEntry(rows.Scan)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *ImportService) ListBatches(ctx context.Context, userID string) ([]ImportBatchDTO, error) {
	rows, err := s.db.Query(ctx, `
		SELECT b.id::text, b.filename, b.credit_card_id::text, b.account_label, b.created_at,
		       COUNT(e.id) FILTER (WHERE e.status = 'pending')
		FROM import_batches b
		LEFT JOIN import_entries e ON e.batch_id = b.id
		WHERE b.user_id = $1
		GROUP BY b.id
		ORDER BY b.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing batches: %w", err)
	}
	defer rows.Close()

	batches := []ImportBatchDTO{}
	for rows.Next() {
		var b ImportBatchDTO
		if err := rows.Scan(&b.ID, &b.Filename, &b.CreditCardID, &b.AccountLabel,
			&b.CreatedAt, &b.NewCount); err != nil {
			return nil, err
		}
		batches = append(batches, b)
	}
	return batches, rows.Err()
}

// ─── Decidir ──────────────────────────────────────────────────────────────────

// ApproveEntry turns a staged line into a real transaction and remembers the
// category choice as a rule, so the next statement comes pre-categorized.
func (s *ImportService) ApproveEntry(ctx context.Context, entryID, userID, category string) (*TransactionDTO, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var e ImportEntryDTO
	var batchCardID *string
	err = tx.QueryRow(ctx, `
		SELECT e.id::text, e.occurred_on::text, e.amount_cents, e.kind,
		       e.raw_description, e.status, b.credit_card_id::text
		FROM import_entries e
		JOIN import_batches b ON b.id = e.batch_id
		WHERE e.id = $1 AND e.user_id = $2`,
		entryID, userID,
	).Scan(&e.ID, &e.OccurredOn, &e.AmountCents, &e.Kind, &e.RawDescription, &e.Status, &batchCardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("loading entry: %w", err)
	}
	if e.Status != "pending" {
		return nil, fmt.Errorf("lançamento já foi decidido (%s)", e.Status)
	}

	if category == "" {
		category = "other"
	}

	// source_type='import' + source_id garantem que reprocessar não duplica.
	var created TransactionDTO
	err = tx.QueryRow(ctx, returningJoined(`INSERT INTO transactions
		(user_id, amount_cents, kind, category, description, occurred_on,
		 source_type, source_id, credit_card_id)
		VALUES ($1, $2, $3, $4, $5, $6::date, 'import', $7, $8)`),
		userID, e.AmountCents, e.Kind, category, e.RawDescription, e.OccurredOn,
		e.ID, batchCardID,
	).Scan(&created.ID, &created.AmountCents, &created.Kind, &created.Category,
		&created.Description, &created.Method, &created.OccurredOn, &created.SourceType,
		&created.SourceID, &created.CreatedAt, &created.CreditCardID, &created.CreditCardName,
		&created.PurchasedOn, &created.InstallmentGroupID, &created.InstallmentNumber,
		&created.InstallmentTotal)
	if err != nil {
		return nil, fmt.Errorf("creating transaction from entry: %w", err)
	}

	if _, err := tx.Exec(ctx,
		"UPDATE import_entries SET status = 'imported' WHERE id = $1", entryID,
	); err != nil {
		return nil, fmt.Errorf("marking entry imported: %w", err)
	}

	// Aprende com a escolha: o mesmo estabelecimento já vem categorizado depois.
	if pattern := learnPattern(e.RawDescription); pattern != "" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO category_rules (user_id, pattern, category)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, pattern)
			DO UPDATE SET category = EXCLUDED.category, hit_count = category_rules.hit_count + 1`,
			userID, pattern, category,
		); err != nil {
			return nil, fmt.Errorf("learning rule: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing approval: %w", err)
	}
	return &created, nil
}

// DecideEntry records a decision that creates nothing: 'ignored' for a line
// the user does not want, 'matched' for one that duplicates an existing
// transaction.
func (s *ImportService) DecideEntry(ctx context.Context, entryID, userID, status string) error {
	if status != "ignored" && status != "matched" {
		return fmt.Errorf("status inválido: %s", status)
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE import_entries SET status = $3
		WHERE id = $1 AND user_id = $2 AND status = 'pending'`,
		entryID, userID, status,
	)
	if err != nil {
		return fmt.Errorf("deciding entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Regras de categoria ──────────────────────────────────────────────────────

func (s *ImportService) ListRules(ctx context.Context, userID string) ([]CategoryRule, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, pattern, category, hit_count
		FROM category_rules WHERE user_id = $1
		ORDER BY hit_count DESC, pattern ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing rules: %w", err)
	}
	defer rows.Close()

	rules := []CategoryRule{}
	for rows.Next() {
		var r CategoryRule
		if err := rows.Scan(&r.ID, &r.Pattern, &r.Category, &r.HitCount); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (s *ImportService) DeleteRule(ctx context.Context, ruleID, userID string) error {
	tag, err := s.db.Exec(ctx,
		"DELETE FROM category_rules WHERE id = $1 AND user_id = $2", ruleID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
