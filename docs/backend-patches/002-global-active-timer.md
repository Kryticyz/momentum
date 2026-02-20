# Patch 002: Global Active Timer (Cross-Device Source of Truth)

## Why this patch exists

The product requires one active timer globally across devices, with backend authority for state. Current backend has no timer session resource.

## Required data model

Create `timer_sessions`:

- `id` (`uuid`, primary key)
- `user_id` (`text`, default `local-user`)
- `project` (`text`, required)
- `note` (`text`, nullable)
- `started_at` (`timestamptz`, required)
- `stopped_at` (`timestamptz`, nullable)
- `created_at` (`timestamptz`)
- `updated_at` (`timestamptz`)
- `source_device_id` (`text`, nullable)

Indexes:

- Unique partial index for one active timer per user:
  - `UNIQUE (user_id) WHERE stopped_at IS NULL`
- Index on `(user_id, updated_at)`

## API additions

- `GET /api/v1/timer/active`
  - Returns active timer or `null`.

- `POST /api/v1/timer/start`
  - Request:
    - `project` (required)
    - `note` (optional)
    - `startedAt` (optional; defaults server time)
    - `sourceDeviceId` (optional)
    - `clientMutationId` (recommended for idempotency)
  - Behavior:
    - If no active timer: create active session.
    - If active timer exists: return `409` with active session payload.

- `POST /api/v1/timer/stop`
  - Request:
    - `stoppedAt` (optional; defaults server time)
    - `note` (optional override/append policy)
    - `clientMutationId` (recommended)
  - Behavior:
    - Stop active session.
    - Materialize a `TimeEntry` record.
    - If no active timer exists: return `409`.

## Entry materialization rules on stop

- `date` is the local date of `startedAt` in backend timezone.
- `start` and `end` remain `HH:mm` fields for compatibility.
- `minutes` uses exact elapsed minutes (no rounding to 5/15 blocks).
- If timer crosses midnight, keep a single entry (no split).

## State behavior expectations

- Timer continues through app termination, restart, and machine sleep because source state is persisted server-side.
- Clients recompute elapsed UI from `startedAt`; slight clock drift is acceptable.

## OpenAPI updates

- Add `TimerSession` schema.
- Add the three timer endpoints above.
- Add `409` responses for start/stop conflict conditions.

## Acceptance criteria

- Starting a timer on one device is visible to another device on next fetch.
- Only one active timer is possible per user at any time.
- Stopping a timer creates exactly one entry.
- Cross-midnight timers remain single entries.

