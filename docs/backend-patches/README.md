# Backend Patches for Native macOS + iOS Clients

This folder defines backend contract changes required to support:

- Native macOS app with menu bar timer controls
- Native iOS app
- Display-only widgets
- One active timer globally across devices
- Entry editing/deletion with guardrails
- Future multi-user auth (OAuth), while supporting current single-user mode

## Scope and principles

- Keep existing `/api/v1` endpoints backward compatible where possible.
- Add new fields/endpoints in additive form under `/api/v1`.
- Reserve `/api/v2` only for truly breaking changes.
- Keep responses client-agnostic (no chart-format payloads).
- Treat PostgreSQL as required for mutation and global timer features.

## Confirmed product constraints

- One active timer globally across devices.
- Backend is source of truth for active timer state.
- Timer precision can drift by a few seconds.
- Soft delete for entries.
- Entry edit/delete allowed only for entries in the last 30 days.
- Conflict strategy is latest-updated-wins.
- Offline clients sync queued changes when reconnected.
- Widgets must show only server-synced data.
- Project colors are user customizable.
- Current mode is single profile/user; leave auth seams for OAuth.

## Patch order

1. `001-entry-identity-and-lifecycle.md`
2. `002-global-active-timer.md`
3. `003-sync-and-conflict-model.md`
4. `004-project-preferences.md`
5. `005-auth-evolution.md`
6. `006-observability-and-prod-transport.md`

## Out of scope for backend patches

- Widget UI implementation
- macOS/iOS UI structure
- Local device analytics/crash SDK selection

## Test expectations (from day one)

- Unit tests for:
  - timer lifecycle state transitions
  - 30-day mutation policy enforcement
  - latest-updated-wins write semantics
  - soft-delete filtering in aggregate queries
- Integration tests for:
  - cross-device timer start/stop flows
  - offline replay idempotency via `clientMutationId`
  - sync cursor pagination and change feed correctness
  - auth mode behavior (`none`, `api-key`, `dual`, `oauth`)
- Contract tests:
  - OpenAPI coverage for new endpoints/fields/error responses
