package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrInviteRequired: a non-admin tried to register without an invite.
	ErrInviteRequired = errors.New("an invite is required to register")
	// ErrInviteInvalid covers unknown, used, revoked and expired invites, and
	// an email that does not match the invite — one error for all, so the
	// response never tells an outsider which case it hit.
	ErrInviteInvalid = errors.New("invite is invalid or expired")
	ErrInvalidEmail  = errors.New("invalid email")
)

// inviteTTL is how long an invite link stays usable.
const inviteTTL = 7 * 24 * time.Hour

// InviteService manages invite-only registration. Only admins reach it
// (routes are behind RequireAdmin), except Lookup, which the public
// registration page uses to show which email the invite is for.
type InviteService struct {
	db *pgxpool.Pool
}

func NewInviteService(db *pgxpool.Pool) *InviteService {
	return &InviteService{db: db}
}

// CreatedInvite is returned once, at creation: it is the only time the raw
// token exists outside the link the admin sends.
type CreatedInvite struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type InviteDTO struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Status    string     `json:"status"` // pending | used | revoked | expired
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at"`
}

// Create issues an invite for email. It refuses emails that already have an
// account — there is nothing to invite them to.
func (s *InviteService) Create(ctx context.Context, adminID, email string) (*CreatedInvite, error) {
	email = normalizeEmail(email)
	if !strings.Contains(email, "@") || len(email) > 255 {
		return nil, ErrInvalidEmail
	}

	var exists bool
	if err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE lower(email) = $1)", email,
	).Scan(&exists); err != nil {
		return nil, fmt.Errorf("checking email: %w", err)
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generating invite token: %w", err)
	}
	token := hex.EncodeToString(raw)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning invite: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	inv := CreatedInvite{Email: email, Token: token}
	err = tx.QueryRow(ctx,
		`INSERT INTO invites (token_hash, email, invited_by, expires_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id::text, expires_at`,
		hashToken(token), email, adminID, time.Now().Add(inviteTTL),
	).Scan(&inv.ID, &inv.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("creating invite: %w", err)
	}
	if err := recordAudit(ctx, tx, auditEvent{
		UserID: adminID, Action: AuditInviteCreated, EntityType: "invite", EntityID: inv.ID,
		Metadata: map[string]any{"email": email},
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing invite: %w", err)
	}
	return &inv, nil
}

// List returns every invite, newest first, with its current status.
func (s *InviteService) List(ctx context.Context) ([]InviteDTO, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id::text, email,
		        CASE WHEN used_at IS NOT NULL THEN 'used'
		             WHEN revoked_at IS NOT NULL THEN 'revoked'
		             WHEN expires_at <= NOW() THEN 'expired'
		             ELSE 'pending' END,
		        expires_at, created_at, used_at
		 FROM invites
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing invites: %w", err)
	}
	defer rows.Close()

	invites := []InviteDTO{}
	for rows.Next() {
		var inv InviteDTO
		if err := rows.Scan(&inv.ID, &inv.Email, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt, &inv.UsedAt); err != nil {
			return nil, fmt.Errorf("scanning invite: %w", err)
		}
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

// Revoke cancels a pending invite so its link stops working. Used, revoked or
// unknown invites return ErrNotFound. adminID is who cancelled it (audit).
func (s *InviteService) Revoke(ctx context.Context, adminID, id string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning revoke: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var email string
	err = tx.QueryRow(ctx,
		`UPDATE invites SET revoked_at = NOW()
		 WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL
		 RETURNING email`,
		id,
	).Scan(&email)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == "22P02") { // malformed uuid
			return ErrNotFound
		}
		return fmt.Errorf("revoking invite: %w", err)
	}
	if err := recordAudit(ctx, tx, auditEvent{
		UserID: adminID, Action: AuditInviteRevoked, EntityType: "invite", EntityID: id,
		Metadata: map[string]any{"email": email},
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing revoke: %w", err)
	}
	return nil
}

// Lookup returns the email a usable invite was issued for, so the
// registration page can show it. Anything unusable is ErrInviteInvalid.
func (s *InviteService) Lookup(ctx context.Context, token string) (string, error) {
	var email string
	err := s.db.QueryRow(ctx,
		`SELECT email FROM invites
		 WHERE token_hash = $1 AND used_at IS NULL AND revoked_at IS NULL AND expires_at > NOW()`,
		hashToken(token),
	).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInviteInvalid
	}
	if err != nil {
		return "", fmt.Errorf("looking up invite: %w", err)
	}
	return email, nil
}

// claimInvite locks a usable invite inside tx and checks it was issued for
// email. The row lock serializes concurrent registrations with the same link:
// the second one waits, then sees used_at set and gets ErrInviteInvalid.
func claimInvite(ctx context.Context, tx pgx.Tx, token, email string) (string, error) {
	var id, inviteEmail string
	err := tx.QueryRow(ctx,
		`SELECT id::text, email FROM invites
		 WHERE token_hash = $1 AND used_at IS NULL AND revoked_at IS NULL AND expires_at > NOW()
		 FOR UPDATE`,
		hashToken(token),
	).Scan(&id, &inviteEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInviteInvalid
	}
	if err != nil {
		return "", fmt.Errorf("claiming invite: %w", err)
	}
	if inviteEmail != email {
		return "", ErrInviteInvalid
	}
	return id, nil
}

// PromoteAdmins marks the users whose email is in emails as admins. Run at
// startup so an account created before ADMIN_EMAILS was set (or before this
// migration) gains access without touching the database by hand. It never
// demotes: removing an email from the list does not revoke admin.
func PromoteAdmins(ctx context.Context, db *pgxpool.Pool, emails []string) error {
	if len(emails) == 0 {
		return nil
	}
	_, err := db.Exec(ctx,
		"UPDATE users SET is_admin = true WHERE lower(email) = ANY($1) AND is_admin = false",
		emails,
	)
	if err != nil {
		return fmt.Errorf("promoting admins: %w", err)
	}
	return nil
}
