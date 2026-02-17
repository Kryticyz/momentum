# Platform Architecture

## Goal

Support multiple data consumers over time while keeping one canonical time-entry ingestion and aggregation layer.

## Current state

- Producer: Obsidian plugin exports JSONL time entries.
- Ingestion + API: `services/dashboard-backend`.
- Consumer: `apps/dashboard-frontend`.

## Planned consumer expansion

Planned future consumers include native mobile and widget surfaces. To support these without redesigning the backend:

- Keep endpoint naming generic (`/api/projects`, `/api/days`, `/api/weeks`) and not dashboard-branded.
- Keep response shapes transport-oriented and client-agnostic.
- Keep temporal semantics explicit (`timezone`, date range parameters, week-start rules).
- Avoid embedding chart-specific formatting in API responses.

## Suggested evolution path (not implemented)

1. Introduce versioned API namespace (for example `/api/v1`).
2. Add auth and user scoping if cross-device data synchronization is introduced.
3. Introduce ingestion abstraction to support multiple producers beyond Obsidian exports.
4. Add contract tests so new consumers can validate API compatibility automatically.

## Why frontend and backend remain in one repo

- They evolve together against the same contract.
- Changes to API and charts can be validated in one pull request.
- Lower operational overhead while the platform is still early.

When native clients are active and release cadence diverges, splitting by client can be revisited.
