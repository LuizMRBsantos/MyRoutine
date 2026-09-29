package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/myroutine/backend/internal/config"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotFound           = errors.New("not found")
)

// dummyPasswordHash is a real bcrypt hash (same cost as user passwords)
// compared against when the email has no account, so a failed login takes
// the same time whether or not the account exists. A malformed constant here
// would make bcrypt fail instantly and leak which emails are registered.
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer-not-a-password"), 12)

// AuthService handles authentication business logic.
type AuthService struct {
	cfg    *config.Config
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewAuthService(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *AuthService {
	return &AuthService{cfg: cfg, db: db, logger: logger}
}

type AuthResult struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserDTO   `json:"user"`
}

type UserDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

// Register creates a new user with a hashed password.
//
// Registration is invite-only: inviteToken must be a usable invite issued
// for this email. Emails in ADMIN_EMAILS may register without one (that is
// how the first account on an empty database gets created) and become admin.
func (s *AuthService) Register(ctx context.Context, name, email, password, timezone, inviteToken string) (*AuthResult, error) {
	email = normalizeEmail(email)
	// The browser sends its detected zone; anything unusable falls back to
	// the product default instead of failing the sign-up.
	if !validTimezone(timezone) {
		timezone = "America/Sao_Paulo"
	}
	isAdmin := s.cfg.IsAdminEmail(email)
	inviteToken = strings.TrimSpace(inviteToken)
	if !isAdmin && inviteToken == "" {
		return nil, ErrInviteRequired
	}

	// Fast path: reject known emails before paying for bcrypt. Compared with
	// lower() so accounts created before normalization still count.
	exists, err := s.emailExists(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	// Hash password with bcrypt cost 12
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	// Claiming the invite and creating the user happen in one transaction:
	// either both land or neither does.
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning registration: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var inviteID string
	if !isAdmin {
		inviteID, err = claimInvite(ctx, tx, inviteToken, email)
		if errors.Is(err, ErrInviteInvalid) {
			// A concurrent registration with this same link may have just
			// used it for this very email: report that as the conflict it is.
			if exists, checkErr := s.emailExists(ctx, email); checkErr == nil && exists {
				return nil, ErrEmailAlreadyExists
			}
		}
		if err != nil {
			return nil, err
		}
	}

	// Create user
	var user struct {
		ID        uuid.UUID
		Name      string
		Email     string
		CreatedAt time.Time
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, timezone, is_admin)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, email, created_at`,
		email, string(hash), name, timezone, isAdmin,
	).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		// A concurrent registration can win the race between the check above
		// and this INSERT; the UNIQUE constraint turns that into a 23505.
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("creating user: %w", err)
	}

	if inviteID != "" {
		if _, err := tx.Exec(ctx,
			"UPDATE invites SET used_at = NOW(), used_by = $1 WHERE id = $2",
			user.ID, inviteID,
		); err != nil {
			return nil, fmt.Errorf("marking invite used: %w", err)
		}
	}

	via := "invite"
	if isAdmin {
		via = "admin_email"
	}
	if err := recordAudit(ctx, tx, auditEvent{
		UserID: user.ID.String(), Action: AuditRegister, Metadata: map[string]any{"via": via},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing registration: %w", err)
	}

	return s.generateTokens(ctx, user.ID.String(), user.Name, user.Email, isAdmin, user.CreatedAt)
}

// emailExists reports whether an account already uses email (case-insensitive).
func (s *AuthService) emailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	if err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE lower(email) = $1)", email,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("checking email: %w", err)
	}
	return exists, nil
}

// Login validates credentials and returns tokens.
func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	email = normalizeEmail(email)

	var user struct {
		ID           uuid.UUID
		Name         string
		Email        string
		PasswordHash string
		IsAdmin      bool
		CreatedAt    time.Time
	}

	err := s.db.QueryRow(ctx,
		`SELECT id, name, email, password_hash, is_admin, created_at
		 FROM users WHERE lower(email) = $1 AND is_active = true
		 ORDER BY created_at
		 LIMIT 1`,
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		// Same bcrypt cost as a real check, so response time does not reveal
		// whether the email has an account.
		bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password)) //nolint:errcheck
		s.auditBestEffort(ctx, auditEvent{Action: AuditLoginFailed, Metadata: map[string]any{"email": email}})
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.auditBestEffort(ctx, auditEvent{UserID: user.ID.String(), Action: AuditLoginFailed})
		return nil, ErrInvalidCredentials
	}

	s.auditBestEffort(ctx, auditEvent{UserID: user.ID.String(), Action: AuditLogin})
	return s.generateTokens(ctx, user.ID.String(), user.Name, user.Email, user.IsAdmin, user.CreatedAt)
}

// Refresh validates a refresh token and issues new access + refresh tokens.
//
// Presenting a token that exists but was already revoked is treated as token
// theft (reuse detection): every active refresh token of that user is revoked
// so both the attacker and the victim must log in again.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	tokenHash := hashToken(refreshToken)

	var stored struct {
		ID        uuid.UUID
		UserID    uuid.UUID
		ExpiresAt time.Time
		RevokedAt *time.Time
	}
	// No revoked_at filter: we need to tell "revoked" apart from "unknown".
	err := s.db.QueryRow(ctx,
		`SELECT rt.id, rt.user_id, rt.expires_at, rt.revoked_at
		 FROM refresh_tokens rt
		 WHERE rt.token_hash = $1`,
		tokenHash,
	).Scan(&stored.ID, &stored.UserID, &stored.ExpiresAt, &stored.RevokedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("getting refresh token: %w", err)
	}

	if stored.RevokedAt != nil {
		return nil, s.handleRefreshReuse(ctx, stored.UserID)
	}
	if !stored.ExpiresAt.After(time.Now()) {
		return nil, ErrInvalidCredentials
	}

	// Revoke old refresh token (rotation). The revoked_at guard makes this
	// atomic: if a concurrent request already rotated it, this one is a reuse.
	tag, err := s.db.Exec(ctx,
		"UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL",
		stored.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("revoking token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, s.handleRefreshReuse(ctx, stored.UserID)
	}

	// Get user
	var user struct {
		Name      string
		Email     string
		IsAdmin   bool
		CreatedAt time.Time
	}
	err = s.db.QueryRow(ctx,
		"SELECT name, email, is_admin, created_at FROM users WHERE id = $1 AND is_active = true",
		stored.UserID,
	).Scan(&user.Name, &user.Email, &user.IsAdmin, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return s.generateTokens(ctx, stored.UserID.String(), user.Name, user.Email, user.IsAdmin, user.CreatedAt)
}

// handleRefreshReuse revokes every active refresh token of the user and
// returns ErrInvalidCredentials (or a wrapped DB error if revocation failed).
func (s *AuthService) handleRefreshReuse(ctx context.Context, userID uuid.UUID) error {
	s.logger.Warn("refresh token reuse detected; revoking all sessions",
		zap.String("user_id", userID.String()))
	s.auditBestEffort(ctx, auditEvent{UserID: userID.String(), Action: AuditSessionReuseDetected})
	if err := revokeAllRefreshTokens(ctx, s.db, userID.String()); err != nil {
		return fmt.Errorf("revoking tokens after reuse: %w", err)
	}
	return ErrInvalidCredentials
}

// Logout revokes a refresh token.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	_, err := s.db.Exec(ctx,
		"UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1",
		tokenHash,
	)
	return err
}

// generateTokens creates JWT access token and opaque refresh token.
func (s *AuthService) generateTokens(ctx context.Context, userID, name, email string, isAdmin bool, createdAt time.Time) (*AuthResult, error) {
	expiryHours, _ := strconv.Atoi(s.cfg.JWTExpiryHours)
	expiresAt := time.Now().Add(time.Duration(expiryHours) * time.Hour)

	// Access token (JWT)
	claims := jwt.MapClaims{
		"sub":   userID,
		"name":  name,
		"email": email,
		"iat":   time.Now().Unix(),
		"exp":   expiresAt.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	// Refresh token (opaque random bytes, stored as hash)
	rawRefresh := make([]byte, 32)
	if _, err := rand.Read(rawRefresh); err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}
	refreshToken := hex.EncodeToString(rawRefresh)
	tokenHash := hashToken(refreshToken)

	refreshExpiryDays, _ := strconv.Atoi(s.cfg.JWTRefreshExpiryDays)
	refreshExpiresAt := time.Now().AddDate(0, 0, refreshExpiryDays)

	// Store refresh token hash (never the raw token)
	_, err = s.db.Exec(ctx,
		"INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		userID, tokenHash, refreshExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User: UserDTO{
			ID:        userID,
			Name:      name,
			Email:     email,
			IsAdmin:   isAdmin,
			CreatedAt: createdAt,
		},
	}, nil
}

// dbExecer is satisfied by *pgxpool.Pool and pgx.Tx.
type dbExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// revokeAllRefreshTokens revokes every still-active refresh token of a user.
func revokeAllRefreshTokens(ctx context.Context, db dbExecer, userID string) error {
	_, err := db.Exec(ctx,
		"UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL",
		userID,
	)
	return err
}

// normalizeEmail canonicalizes an email for storage and lookup.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// isUniqueViolation reports whether err is a Postgres unique_violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// hashToken creates a SHA-256 hash of a token for safe storage.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// auditBestEffort records an audit row but never fails the caller: a broken
// audit log must not lock everyone out of logging in. Failures go to the
// server log instead.
func (s *AuthService) auditBestEffort(ctx context.Context, ev auditEvent) {
	if err := recordAudit(ctx, s.db, ev); err != nil {
		s.logger.Error("audit log write failed", zap.String("action", ev.Action), zap.Error(err))
	}
}
