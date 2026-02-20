# Patch 003: Offline Sync and Conflict Model

## Why this patch exists

Clients must support local-only operation and sync later. Backend behavior must be deterministic when queued mutations from multiple devices arrive out of order.

## Conflict policy

Use **latest-updated-wins** based on backend `updatedAt`:

- Each accepted mutation sets `updatedAt = now()` on server.
- No hard conflict rejection for stale writes (except policy failures like 30-day edit window).
- Last accepted write becomes canonical.

This aligns with tolerance for small time drift and simplifies offline replay.

## Idempotency requirements

Support idempotent mutation replay with `clientMutationId` (UUID):

- Accepted on:
  - `POST /api/v1/entries`
  - `PATCH /api/v1/entries/{id}`
  - `DELETE /api/v1/entries/{id}`
  - `POST /api/v1/timer/start`
  - `POST /api/v1/timer/stop`
- Store processed IDs in `processed_mutations` keyed by `(user_id, client_mutation_id)`.
- Repeated mutation ID returns previous success response (or equivalent canonical state).

## Change feed for incremental sync

Add endpoint:

- `GET /api/v1/sync/changes?since=<RFC3339>&limit=<n>`
  - Returns changed entries (including soft-deleted records) ordered by `updatedAt ASC`.
  - Includes `nextSince` cursor token in `meta`.

This prevents full-range re-fetch after every reconnect.

## Local-only mode interaction

- Backend remains source of truth when reachable.
- Offline-created entries can be posted later using client-generated `id` + `clientMutationId`.
- Widgets continue to show server-synced state only (no backend change required for widget filtering).

## OpenAPI updates

- Add shared `clientMutationId` field to mutation request bodies.
- Add `GET /api/v1/sync/changes` path and response schema.
- Add `meta.nextSince` for paged incremental pull.

## Acceptance criteria

- Replayed queued mutations do not duplicate effects.
- Multiple devices eventually converge to same entry state.
- Sync pull can fetch only records changed since prior cursor.

