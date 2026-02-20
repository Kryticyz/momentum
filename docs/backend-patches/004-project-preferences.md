# Patch 004: Project Preferences (User-Customizable Colors)

## Why this patch exists

Project colors must be user customizable and consistent across macOS, iOS, and widgets.

## Required data model

Create `project_preferences`:

- `user_id` (`text`, default `local-user`)
- `project` (`text`)
- `color_hex` (`text`, e.g. `#3A7AFE`)
- `updated_at` (`timestamptz`)

Primary key:

- `(user_id, project)`

## API additions

- `GET /api/v1/preferences/projects`
  - Returns per-project preference map for authenticated user.

- `PUT /api/v1/preferences/projects/{project}`
  - Upsert color preference.
  - Request body: `{ "colorHex": "#3A7AFE" }`

- `DELETE /api/v1/preferences/projects/{project}`
  - Remove custom color and fall back to default mapping.

## Behavior rules

- Aggregate endpoints remain project-name based and unchanged.
- If no preference exists for a project, clients apply deterministic fallback color.
- Backend stores only user preferences; it does not rename or manage project lifecycle.

## Forward compatibility for tags

To leave room for future Obsidian tag sync without breaking changes:

- Add optional `tags: string[]` to `TimeEntry` now (read/write optional).
- Tags may be empty/omitted until project strategy is finalized.

## OpenAPI updates

- Add preferences endpoints and schemas.
- Add optional `tags` in `TimeEntry` schema.

## Acceptance criteria

- Color preference edited on one device is visible on another after sync.
- Deleting a preference reverts to deterministic fallback.
- Existing clients remain compatible if they ignore `tags`.

