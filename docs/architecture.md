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
2. Add auth and user scoping if cross-device data synchronization is introduced.
3. Introduce ingestion abstraction to support multiple producers beyond Obsidian exports.
4. Add contract tests so new consumers can validate API compatibility automatically.

## Why frontend and backend remain in one repo

- They evolve together against the same contract.
- Changes to API and charts can be validated in one pull request.
- Lower operational overhead while the platform is still early.

When native clients are active and release cadence diverges, splitting by client can be revisited.

## EntryStore interface evolution for PostgreSQL

The `EntryStore` interface in `services/dashboard-backend/storage.go` is the seam for future database migration.

Current interface:

```go
type EntryStore interface {
    Entries() []TimeEntry
    Count() int
    LastLoaded() time.Time
    Reload() error
}
```

When PostgreSQL is introduced, the interface evolves to:

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

Handlers switch from:
```go
entries := filterByRange(h.store.Entries(), from, to)
```
To:
```go
entries, err := h.store.EntriesInRange(from, to)
```

The in-memory `Store` implements `EntriesInRange` by calling `filterByRange` internally. The PostgreSQL `PgStore` implements it with a `WHERE` clause. `AddEntries` supports both HTTP push and JSONL import ingestion paths.

## Client type generation from OpenAPI

The `docs/openapi-dashboard-backend.yaml` spec is the single source of truth for API types.

- **TypeScript** (web dashboard): Generate types with `openapi-typescript` from the OpenAPI spec.
- **Swift** (macOS app): Generate types with Apple's `swift-openapi-generator` from the same spec.
- **Go** (backend): Structs are authoritative; contract tests validate they match the spec.
