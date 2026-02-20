# Patch 005: Authentication Evolution (API Key to OAuth)

## Why this patch exists

Current backend supports API key bearer auth only. Product direction requires OAuth and future multi-user support while preserving current single-user deployment.

## Migration goals

- Keep current API key flow working for local/self-host setup.
- Add OAuth support without breaking existing clients.
- Introduce user scoping in storage and handlers.

## Phase model

## Phase 1: Dual-mode auth (non-breaking)

- Keep `Authorization: Bearer <api-key>` behavior when `API_KEY` is configured.
- Add optional OAuth validation path when OAuth config is present:
  - issuer URL
  - audience
  - JWKS endpoint

If OAuth token is valid, derive `user_id` from token `sub` claim.
If API key auth is used, assign fixed `user_id = local-user`.

## Phase 2: User-aware data access

- Scope all mutable/read queries by `user_id`.
- Ensure timer uniqueness, preferences, and sync cursors are per user.

## Phase 3: OAuth-required mode (future)

- Config flag to disable API key auth in hosted/multi-user deployments.

## API and contract notes

- No endpoint path changes required for auth migration.
- Add auth mode info to `/health` payload (non-sensitive), for example:
  - `authMode: "none" | "api-key" | "dual" | "oauth"`

## Security and operational notes

- Enforce HTTPS in production deployments.
- Keep HTTP allowed for local development/testing.
- Add structured auth failure logs without token contents.

## Acceptance criteria

- Existing API-key clients continue functioning unchanged.
- OAuth clients can access same endpoints with user-isolated data.
- Mixed mode can run during transition without data leakage across users.

