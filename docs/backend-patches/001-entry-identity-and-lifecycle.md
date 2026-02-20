# Patch 001: Entry Identity, Edit/Delete, and Lifecycle Rules

## Why this patch exists

Current entries do not expose stable IDs and cannot be edited or deleted via API. Native apps require:

- Stable entry identity across sync cycles
- Edit and soft delete operations
- Enforcement of a 30-day mutation window

## Required data model changes

Add fields to persisted entries:

- `id` (`uuid`, primary key, stable)
- `created_at` (`timestamptz`, server set)
- `updated_at` (`timestamptz`, server set on every mutation)
- `deleted_at` (`timestamptz`, nullable; soft delete marker)
- `user_id` (`text` for now; default `local-user`, future OAuth subject)

Keep existing ingestion identity (`date`, `file_path`, `line_number`) for dedupe/upsert of Obsidian exports.

## API additions and changes

## `TimeEntry` response shape (additive)

Add the following response fields (do not remove existing fields):

- `id: string (uuid)`
- `createdAt: string (RFC3339)`
- `updatedAt: string (RFC3339)`
- `deletedAt: string | null (RFC3339)`

## New endpoints

- `PATCH /api/v1/entries/{id}`
  - Partial update for mutable fields (`project`, `start`, `end`, `minutes`, `note`)
  - Reject if entry date is older than 30 days in backend timezone

- `DELETE /api/v1/entries/{id}`
  - Soft delete only (`deletedAt` set)
  - Reject if entry date is older than 30 days in backend timezone

## Existing endpoint updates

- `GET /api/v1/entries`
  - Add query param: `includeDeleted=false` (default false)

- `GET /api/v1/projects`, `GET /api/v1/days`, `GET /api/v1/weeks`
  - Exclude soft-deleted entries by default

- `POST /api/v1/entries`
  - Accept optional `id` for client-created offline entries
  - If `id` exists, apply upsert semantics

## Validation and policy

- Mutation window: only entries where `date >= (today - 30 days)` are editable/deletable.
- On violation, return `422` with:
  - `error: "edit window exceeded"`
  - `code: "edit_window_exceeded"`
- Preserve existing `401` and `400` behaviors.

## Suggested schema migration (PostgreSQL)

1. Add new columns with nullable defaults.
2. Backfill:
   - `id = gen_random_uuid()` for existing rows
   - `created_at = now()` where null
   - `updated_at = created_at` where null
   - `user_id = 'local-user'` where null
3. Enforce non-null constraints for `id`, `created_at`, `updated_at`, `user_id`.
4. Add index on `(user_id, date)` and `(user_id, updated_at)`.
5. Keep existing dedupe index on `(date, file_path, line_number)` where applicable.

## OpenAPI updates

- Extend `TimeEntry` schema with the additive fields above.
- Add path schemas for `PATCH /api/v1/entries/{id}` and `DELETE /api/v1/entries/{id}`.
- Add `includeDeleted` query parameter to entries route.
- Define error code shape for `422` policy failures.

## Acceptance criteria

- Entry IDs remain stable across repeated reads/imports.
- Soft-deleted entries are hidden from aggregate endpoints.
- Attempts to edit/delete older than 30 days are rejected with `422`.
- Contract tests cover new fields and mutation policies.

