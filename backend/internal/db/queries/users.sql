-- name: CreateUser :one
INSERT INTO users (email, password_hash, name, timezone)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND is_active = true
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND is_active = true
LIMIT 1;

-- name: UpdateUser :one
UPDATE users
SET name = $2, avatar_url = $3, timezone = $4
WHERE id = $1
RETURNING *;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1
  AND expires_at > NOW()
  AND revoked_at IS NULL
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE token_hash = $1;

-- name: RevokeAllUserTokens :exec
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: CreateAuditLog :exec
INSERT INTO audit_logs (user_id, action, entity_type, entity_id, ip_address, user_agent, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7);
