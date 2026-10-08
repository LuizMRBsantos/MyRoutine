# Running MyRoutine locally

## Prerequisites

- Docker + Docker Compose
- Go 1.26+
- Node 20+
- A `.env` file at the repo root (copy `.env.example` and fill in `JWT_SECRET` ≥ 32 chars). Note: the frontend Vite proxy targets port **8082**, so keep `APP_PORT=8082`.

## Start — development (hot reload)

```bash
# 1. Infrastructure (Postgres + Redis)
docker compose up -d postgres redis

# 2. Backend API — migrations run automatically on startup (golang-migrate,
#    embedded in the binary, tracked in the schema_migrations table)
cd backend && go run ./cmd/api/

# 3. Frontend (http://localhost:5173, proxies /api → localhost:8082)
cd frontend && npm install && npm run dev
```

## Start — full stack in containers

```bash
docker compose up -d --build
```

Everything is then reachable through nginx on port 80: the SPA at
<http://localhost/> and the API under <http://localhost/api/v1/>. Grafana is on
:3001 and Prometheus on :9090 (scraping the API's `/metrics`).

## Tests

```bash
cd backend  && go test ./...          # unit + integration (testcontainers)
cd frontend && npm test               # Vitest
cd mobile   && npm test               # bullet-journal parser
```

Backend integration tests start a throwaway Postgres through testcontainers and
apply the real migrations. They skip automatically when Docker is unavailable —
set `SKIP_INTEGRATION=1` to skip them explicitly and run only unit tests.

## Access: invite-only registration

Registration requires an invite. The app sends no email: an admin generates
links in the **Acessos** screen (`/acessos`) and sends them personally.

- Set `ADMIN_EMAILS` in `.env` (comma-separated). Those emails can register
  without an invite, become admins, and are promoted at startup if the account
  already exists. On an empty database this is how the first account is made.
- Invite link: `/register?convite=<code>`, valid for 7 days, single use, bound
  to the invited email.
- Password reset link (admin-generated): `/redefinir-senha?codigo=<code>`,
  valid for 1 hour, single use. A new link cancels the previous one, and a
  successful reset logs the account out everywhere.
- Only SHA-256 hashes of these codes are stored.

## Smoke test (API)

```bash
BASE=http://localhost:8082/api/v1
# Register the admin (email must be in ADMIN_EMAILS — no invite needed)
curl -X POST $BASE/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"<admin email>","password":"testpassword123","name":"Me"}'
# → returns access_token; use it as: -H "Authorization: Bearer <token>"

# Invite someone; the response carries the one-time token for the link
curl -X POST $BASE/admin/invites -H "Authorization: Bearer <token>" \
  -H 'Content-Type: application/json' -d '{"email":"friend@example.com"}'

curl $BASE/habits -H "Authorization: Bearer <token>"
```

Health check: `curl http://localhost:8082/health`.

## Mobile (Expo SDK 56)

```bash
cd mobile && npm install

# WatermelonDB uses JSI (native code) — Expo Go does NOT work.
# A development build is required:
npx expo prebuild
npx expo run:ios      # or: npx expo run:android
```

The app resolves the API base URL from the Metro host (`http://<host>:8082`);
override with `EXPO_PUBLIC_API_URL` when pointing at another server. Log in with
the same account used on the web — the journal is stored locally and pushed to
the server with the "Sincronizar" button (one-way push, idempotent by
`source_type='track_day'` + `source_id`).

## Migrations

- Files live in `backend/internal/db/migrations/` using the golang-migrate
  naming scheme `NNNNNN_description.up.sql`.
- They are embedded into the binary and applied automatically at startup
  (`internal/db/migrate.go`). No manual `psql` step is needed.
- Policy (see `.claude/skills/go-migration-safety`): incremental and
  non-destructive; new columns NULLable or with DEFAULT; never DROP in the
  same migration that adds a replacement.

## Observability

The API exports Prometheus metrics at `/metrics`:

- `myroutine_http_requests_total{method,route,status}`
- `myroutine_http_request_duration_seconds{method,route}`
- `myroutine_http_requests_in_flight`

Labels use the chi **route pattern** (`/api/v1/habits/{habitID}`), never the raw
path — otherwise every id would create its own time series.

## Known conscious debt

- Redis runs in compose but no application code uses it yet (future: caching,
  rate limiting).
- `audit_logs` table exists but nothing writes to it yet.
- `internal/ai/` is empty — Claude API integration is a future module.
- Postgres/Redis are not scraped by Prometheus: that needs
  `postgres_exporter` / `redis_exporter` sidecars, so those jobs were left out
  rather than kept as permanently failing targets.

## Production backups (Supabase)

Supabase Free keeps no downloadable backups, so the **"Backup — Database"**
workflow (`.github/workflows/backup.yml`) makes one every night at 03:17
(Brasília). It dumps the app tables (schema `public`), encrypts the dump with
[`age`](https://age-encryption.org) to a **public** key, and keeps it for 30
days as a workflow artifact. Run it by hand from the Actions tab
(*Run workflow*) when needed, e.g. before a risky migration.

- GitHub holds only the public key (repo variable `BACKUP_AGE_RECIPIENT`): it
  can create backups but cannot read them.
- The **private key** is `~/.myroutine/backup-age-key.txt` on the owner's Mac.
  **Keep a copy in a password manager** — without it no backup can be opened.

Restore a backup into a throwaway local Postgres 17 (to inspect it, or as a
restore drill):

```bash
gh run list --workflow "Backup — Database"            # pick a run id
gh run download <run-id> --dir /tmp/myroutine-backup
go run filippo.io/age/cmd/age@v1.2.1 -d \
  -i ~/.myroutine/backup-age-key.txt \
  -o /tmp/myroutine-backup/db.dump /tmp/myroutine-backup/*/myroutine-db-*.dump.age

docker run -d --rm --name myroutine-restore -e POSTGRES_PASSWORD=restore -p 55432:5432 postgres:17-alpine
sleep 5
# Backups taken before migration 015 still reference Supabase's
# extensions.uuid_generate_v4(); recreating that schema keeps them restorable.
docker exec -e PGPASSWORD=restore myroutine-restore psql -U postgres -c \
  'CREATE SCHEMA IF NOT EXISTS extensions; CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA extensions;'
docker run --rm --network host -e PGPASSWORD=restore -v /tmp/myroutine-backup:/in postgres:17-alpine \
  pg_restore -h 127.0.0.1 -p 55432 -U postgres -d postgres --no-owner --exit-on-error /in/db.dump
```

`--exit-on-error` makes a broken restore fail loudly instead of quietly
skipping tables.

Delete `/tmp/myroutine-backup` afterwards: the decrypted dump holds real data.
