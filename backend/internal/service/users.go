package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles the authenticated user's own profile.
type UserService struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewUserService(db *pgxpool.Pool, logger *zap.Logger) *UserService {
	return &UserService{db: db, logger: logger}
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type ProfileDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL *string   `json:"avatar_url"`
	Timezone  string    `json:"timezone"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateProfileInput struct {
	Name     *string `json:"name"`
	Timezone *string `json:"timezone"`
}

// ─── Methods ─────────────────────────────────────────────────────────────────

const profileColumns = `id::text, name, email, avatar_url, timezone, created_at`

func (s *UserService) GetProfile(ctx context.Context, userID string) (*ProfileDTO, error) {
	var p ProfileDTO
	err := s.db.QueryRow(ctx,
		`SELECT `+profileColumns+` FROM users WHERE id = $1 AND is_active = true`,
		userID,
	).Scan(&p.ID, &p.Name, &p.Email, &p.AvatarURL, &p.Timezone, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting profile: %w", err)
	}
	return &p, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*ProfileDTO, error) {
	var p ProfileDTO
	err := s.db.QueryRow(ctx,
		`UPDATE users SET
		   name = COALESCE($2, name),
		   timezone = COALESCE($3, timezone)
		 WHERE id = $1 AND is_active = true
		 RETURNING `+profileColumns,
		userID, input.Name, input.Timezone,
	).Scan(&p.ID, &p.Name, &p.Email, &p.AvatarURL, &p.Timezone, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("updating profile: %w", err)
	}
	return &p, nil
}

// ChangePassword verifies the current password before setting a new one.
func (s *UserService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	var hash string
	err := s.db.QueryRow(ctx,
		"SELECT password_hash FROM users WHERE id = $1 AND is_active = true",
		userID,
	).Scan(&hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("loading password hash: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("hashing new password: %w", err)
	}

	// Update the hash and kill every existing session atomically, so a stolen
	// refresh token stops working as soon as the password changes.
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning password change: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		"UPDATE users SET password_hash = $2 WHERE id = $1",
		userID, string(newHash),
	); err != nil {
		return fmt.Errorf("saving new password: %w", err)
	}
	if err := revokeAllRefreshTokens(ctx, tx, userID); err != nil {
		return fmt.Errorf("revoking refresh tokens: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing password change: %w", err)
	}
	return nil
}
