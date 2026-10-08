package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// LGPD: every user can take their data with them (Export) and have it
// erased for real (DeleteAccount).

// exportTables lists every table holding a user's personal data. Each has a
// user_id column. TestExportCoversEveryUserTable fails when a new table with
// user_id appears and is neither here nor in exportExcludedTables — so a new
// module cannot be silently left out of the export.
var exportTables = []string{
	"habits",
	"habit_logs",
	"habit_day_reviews",
	"tasks",
	"monthly_goals",
	"transactions",
	"budgets",
	"credit_cards",
	"import_batches",
	"import_entries",
	"category_rules",
	"body_metrics",
	"study_sessions",
	"journal_entries",
	"notification_settings",
	"notification_deliveries",
	"audit_logs",
}

// exportExcludedTables have a user_id but hold only security secrets (token
// hashes, a device's push delivery keys) that are meaningless outside the
// server.
var exportExcludedTables = []string{
	"refresh_tokens",
	"password_resets",
	"push_subscriptions",
}

// exportFormatVersion changes if the export layout ever changes shape.
const exportFormatVersion = 1

// Export returns all of the user's data as a JSON-ready map: the profile
// (never the password hash) plus one array per table, rows as stored.
func (s *UserService) Export(ctx context.Context, userID string) (map[string]any, error) {
	profile, err := s.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := map[string]any{
		"format_version": exportFormatVersion,
		"exported_at":    time.Now().UTC(),
		"profile":        profile,
	}

	for _, table := range exportTables {
		var rows json.RawMessage
		// Table names come from the fixed list above, never from input.
		err := s.db.QueryRow(ctx,
			"SELECT COALESCE(json_agg(t), '[]'::json) FROM "+table+" t WHERE user_id = $1",
			userID,
		).Scan(&rows)
		if err != nil {
			return nil, fmt.Errorf("exporting %s: %w", table, err)
		}
		out[table] = rows
	}

	// Recorded after reading, so the export does not contain its own row.
	if err := recordAudit(ctx, s.db, auditEvent{UserID: userID, Action: AuditAccountExported}); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteAccount erases the account after confirming the password. Every
// user-owned row goes with it through ON DELETE CASCADE. Audit rows are kept
// for security history but lose their owner (SET NULL) — and, here, the IP
// and user agent too, which would still identify the person.
func (s *UserService) DeleteAccount(ctx context.Context, userID, password string) error {
	var hash string
	err := s.db.QueryRow(ctx,
		"SELECT password_hash FROM users WHERE id = $1 AND is_active = true",
		userID,
	).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("loading password hash: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning account deletion: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		"UPDATE audit_logs SET ip_address = NULL, user_agent = NULL WHERE user_id = $1",
		userID,
	); err != nil {
		return fmt.Errorf("anonymizing audit logs: %w", err)
	}
	// "An account was deleted at time X" survives, with no owner (the DELETE
	// below sets user_id NULL), no IP and no user agent.
	if err := recordAudit(ctx, tx, auditEvent{UserID: userID, Action: AuditAccountDeleted, Anonymous: true}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "DELETE FROM users WHERE id = $1", userID); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing account deletion: %w", err)
	}
	return nil
}
