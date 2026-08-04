# Running MyRoutine locally

## Prerequisites

- Docker + Docker Compose
- Go 1.26+
- Node 20+
- A `.env` file at the repo root (copy `.env.example` and fill in `JWT_SECRET` ≥ 32 chars). Note: the frontend Vite proxy targets port **8082**, so keep `APP_PORT=8082`.

## Start

```bash
# 1. Infrastructure (Postgres + Redis)
docker compose up -d postgres redis

# 2. Backend API — migrations run automatically on startup (golang-migrate,
#    embedded in the binary, tracked in the schema_migrations table)
cd backend && go run ./cmd/api/

# 3. Frontend (http://localhost:5173, proxies /api → localhost:8082)
cd frontend && npm install && npm run dev
```

## Smoke test (API)

```bash
BASE=http://localhost:8082/api/v1
curl -X POST $BASE/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"me@example.com","password":"testpassword123","name":"Me"}'
# → returns access_token; use it as: -H "Authorization: Bearer <token>"
curl $BASE/habits -H "Authorization: Bearer <token>"
```

Health check: `curl http://localhost:8082/health`.

## Migrations

- Files live in `backend/internal/db/migrations/` using the golang-migrate
  naming scheme `NNNNNN_description.up.sql`.
- They are embedded into the binary and applied automatically at startup
  (`internal/db/migrate.go`). No manual `psql` step is needed.
- Policy (see `.agents/skills/go-migration-safety`): incremental and
  non-destructive; new columns NULLable or with DEFAULT; never DROP in the
  same migration that adds a replacement.

## Known conscious debt

- Redis runs in compose but no application code uses it yet (future: caching,
  rate limiting).
- `audit_logs` table exists but nothing writes to it yet.
- `internal/ai/` is empty — Claude API integration is a future module.
