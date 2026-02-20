# Momentum Platform

Backend and client applications for consuming time data exported by the Momentum ecosystem.

Current components:
- `services/dashboard-backend/` — Go API service for loading and aggregating time entries.
- `apps/dashboard-frontend/` — React dashboard that visualizes API data.
- `docs/` — Data contracts, aggregation rules, and platform architecture notes.

## Direction

This repository is intentionally API-first. The web dashboard is one consumer, but the backend API should remain stable and reusable for future clients, including:
- Android application
- iOS application
- macOS widgets
- iPhone widgets

The backend should avoid dashboard-specific assumptions in route design and response models.

## Repository principles

- Treat `docs/backend-contract.md` as the source of truth for entry schema and API boundaries.
- Keep aggregation rules deterministic and documented (`docs/aggregation-spec.md`).
- Keep client-specific presentation logic in each client app, not in shared API contracts.
- Prefer additive API evolution to avoid breaking future mobile/widget clients.

## Quick Start (Docker Compose)

```bash
cp .env.example .env
# Edit .env: set POSTGRES_PASSWORD and optionally API_KEY
docker compose up --build
```

The stack starts Postgres, runs schema migration, and serves the API + frontend through Caddy on port 80.

Verify:

```bash
curl http://localhost/health
# With auth: curl -H "Authorization: Bearer <your-api-key>" http://localhost/health
```

## Quick Start (Local Development)

Backend:

```bash
cd services/dashboard-backend
go test ./...
go run . -jsonl /path/to/time-entries.jsonl
```

Frontend:

```bash
cd apps/dashboard-frontend
bun install
bun test
bun run dev
```

## Migration (JSONL to PostgreSQL)

```bash
./momentum-dashboard -migrate -database-url "postgres://..." -jsonl /path/to/entries.jsonl
```

## Full Build

From the repository root:

```bash
./build.sh
```

Installs frontend dependencies, builds the React app into the backend's static
directory, and compiles the Go binary.

## Configuration

See [docs/configuration.md](docs/configuration.md) for the full reference.

Key environment variables:
- `DATABASE_URL` — PostgreSQL connection (empty = in-memory JSONL mode)
- `API_KEY` — Bearer token for auth (empty = no auth)
- `PORT` — HTTP listen port (default: 8080)
- `TIMEZONE` — IANA timezone (default: Australia/Sydney)
- `CORS_ALLOWED_ORIGINS` — Comma-separated origins (default: http://localhost:5173)
