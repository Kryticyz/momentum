# Platform Architecture

## Goal

Support multiple data consumers over time while keeping one canonical time-entry ingestion and aggregation layer.

## Current state

- Producer: Obsidian plugin exports JSONL time entries.
- Ingestion + API: `services/dashboard-backend`.
- Consumer: `apps/dashboard-frontend`.

## Planned consumer expansion

Planned future consumers include native mobile and widget surfaces. To support these without redesigning the backend:

- Keep endpoint naming generic (`/api/v1/projects`, `/api/v1/days`, `/api/v1/weeks`) and not dashboard-branded.
- Keep response shapes transport-oriented and client-agnostic.
- Keep temporal semantics explicit (`timezone`, date range parameters, week-start rules).
- Avoid embedding chart-specific formatting in API responses.

## Suggested evolution path

1. Maintain backward-compatible evolution within `/api/v1`; introduce `/api/v2` only for breaking changes.
2. User scoping when cross-device data synchronization is introduced.
3. Additional producers beyond Obsidian (mobile apps can push via `POST /api/v1/entries`).
4. Contract tests validate API compatibility automatically (already implemented).

## Why frontend and backend remain in one repo

- They evolve together against the same contract.
- Changes to API and charts can be validated in one pull request.
- Lower operational overhead while the platform is still early.

When native clients are active and release cadence diverges, splitting by client can be revisited.

## Dual-Store Architecture

The `EntryStore` interface in `services/dashboard-backend/storage.go` is the abstraction between handlers and storage:

```go
type EntryStore interface {
    EntriesInRange(from, to string) ([]TimeEntry, error)
    Count() int
    LastLoaded() time.Time
    Reload() error
    AddEntries(entries []TimeEntry) error
    Close() error
}
```

Two implementations:
- **`Store`** (in-memory) — JSONL file backed. `EntriesInRange` calls `filterByRange` on an atomic snapshot. Background poller reloads periodically.
- **`PgStore`** (PostgreSQL) — `EntriesInRange` uses `WHERE date >= $1 AND date <= $2`. `AddEntries` does batch upsert with `ON CONFLICT`.

Store selection in `main.go`: if `DATABASE_URL` is set, Postgres; otherwise, in-memory JSONL.

The optional `Pinger` interface enables the health endpoint to report database connectivity for `PgStore`.

## Authentication Layer

Bearer token authentication middleware sits between CORS and the router:

```
requestLogger → authMiddleware → corsMiddleware → mux
```

- Empty `API_KEY` → pass-through (no auth, for local/open-source mode)
- `/health` and `OPTIONS` always bypass auth
- All other routes require `Authorization: Bearer <key>`

## Ingestion Paths

Data enters the system via three routes:
1. **JSONL file** — Obsidian plugin exports, backend reads on poll or refresh
2. **Push API** (`POST /api/v1/entries`) — JSON array of entries, for real-time ingestion
3. **Import API** (`POST /api/v1/import`) — JSONL body, for bulk file uploads

## Client type generation from OpenAPI

The `docs/openapi-dashboard-backend.yaml` spec is the single source of truth for API types.

- **TypeScript** (web dashboard): Generate types with `openapi-typescript` from the OpenAPI spec.
- **Swift** (macOS app): Generate types with Apple's `swift-openapi-generator` from the same spec.
- **Go** (backend): Structs are authoritative; contract tests validate they match the spec.
