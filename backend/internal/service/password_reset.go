package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrResetInvalid covers unknown, used, superseded and expired reset links.
	ErrResetInvalid = errors.New("reset link is invalid or expired")
	// ErrPasswordTooShort: the new password is under the minimum length.
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
)

// resetTTL is short on purpose: a reset link grants access to an account.
const resetTTL = time.Hour

// minPasswordLength matches the registration rule.
const minPasswordLength = 8

// PasswordResetService issues admin-generated reset links and applies them.
// The app sends no email: the admin generates the link and sends it.
type PasswordResetService struct {
	db *pgxpool.Pool
}

func NewPasswordResetService(db *pgxpool.Pool) *PasswordResetService {
	return &PasswordResetService{db: db}
}

// CreatedReset is returned once, at creation — the only time the raw token
// exists outside the link.
type CreatedReset struct {
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Create issues a reset link for the active account with this email and
// invalidates any earlier unused link for it, so at most one works at a time.
// An unknown email is ErrNotFound (the caller is the admin, not an outsider).
func (s *PasswordResetService) Create(ctx context.Context, adminID, email string) (*CreatedReset, error) {
	email = normalizeEmail(email)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning reset: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var userID, storedEmail string
	err = tx.QueryRow(ctx,
		`SELECT id::text, email FROM users
		 WHERE lower(email) = $1 AND is_active = true
		 ORDER BY created_at LIMIT 1`,
		email,
	).Scan(&userID, &storedEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}

	// Supersede earlier unused links: only the newest one may work.
	if _, err := tx.Exec(ctx,
		`UPDATE password_resets SET expires_at = NOW()
		 WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()`,
		userID,
	); err != nil {
		return nil, fmt.Errorf("superseding old resets: %w", err)
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generating reset token: %w", err)
	}
	token := hex.EncodeToString(raw)

	// Recorded on the affected person's own history (it comes with their LGPD
	// export): the admin can open any account this way, so it must be visible.
	if err := recordAudit(ctx, tx, auditEvent{
		UserID: userID, Action: AuditResetLinkCreated, Metadata: map[string]any{"by_admin": adminID},
	}); err != nil {
		return nil, err
	}

	reset := CreatedReset{Email: storedEmail, Token: token}
	if err := tx.QueryRow(ctx,
		`INSERT INTO password_resets (user_id, token_hash, created_by, expires_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING expires_at`,
		userID, hashToken(token), adminID, time.Now().Add(resetTTL),
	).Scan(&reset.ExpiresAt); err != nil {
		return nil, fmt.Errorf("creating reset: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing reset: %w", err)
	}
	return &reset, nil
}

// Lookup returns the email of the account a usable reset link belongs to, so
// the reset page can show whose password is being changed.
func (s *PasswordResetService) Lookup(ctx context.Context, token string) (string, error) {
	var email string
	err := s.db.QueryRow(ctx,
		`SELECT u.email FROM password_resets pr
		 JOIN users u ON u.id = pr.user_id
		 WHERE pr.token_hash = $1 AND pr.used_at IS NULL AND pr.expires_at > NOW()
		   AND u.is_active = true`,
		hashToken(token),
	).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrResetInvalid
	}
	if err != nil {
		return "", fmt.Errorf("looking up reset: %w", err)
	}
	return email, nil
}

// Reset sets a new password through a usable link, marks the link used and
// logs the account out everywhere — all in one transaction. The row lock
// makes a link work exactly once even if submitted twice at the same time.
func (s *PasswordResetService) Reset(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < minPasswordLength {
		return ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning reset: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var resetID, userID string
	err = tx.QueryRow(ctx,
		`SELECT pr.id::text, pr.user_id::text FROM password_resets pr
		 JOIN users u ON u.id = pr.user_id
		 WHERE pr.token_hash = $1 AND pr.used_at IS NULL AND pr.expires_at > NOW()
		   AND u.is_active = true
		 FOR UPDATE OF pr`,
		hashToken(token),
	).Scan(&resetID, &userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrResetInvalid
	}
	if err != nil {
		return fmt.Errorf("claiming reset: %w", err)
	}

	if _, err := tx.Exec(ctx,
		"UPDATE users SET password_hash = $1 WHERE id = $2", string(hash), userID,
	); err != nil {
		return fmt.Errorf("updating password: %w", err)
	}
	if _, err := tx.Exec(ctx,
		"UPDATE password_resets SET used_at = NOW() WHERE id = $1", resetID,
	); err != nil {
		return fmt.Errorf("marking reset used: %w", err)
	}
	// Whoever held the old password (or a stolen session) is logged out.
	if err := revokeAllRefreshTokens(ctx, tx, userID); err != nil {
		return fmt.Errorf("revoking sessions: %w", err)
	}
	if err := recordAudit(ctx, tx, auditEvent{
		UserID: userID, Action: AuditPasswordReset, Metadata: map[string]any{"via": "admin_link"},
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing reset: %w", err)
	}
	return nil
}
