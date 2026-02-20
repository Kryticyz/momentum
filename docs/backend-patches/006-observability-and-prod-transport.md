# Patch 006: Observability and Production Transport

## Why this patch exists

App Store deployments need predictable transport/security behavior and clear diagnostics for cross-device issues.

## Transport policy

- Development: allow HTTP (current behavior).
- Production mode: require HTTPS and reject plaintext requests unless explicitly overridden.

Suggested config:

- `allow_insecure_http` (default `true` for local)
- `production_mode` (default `false`; when `true`, require HTTPS unless override flag is set)

## Request correlation

- Generate or forward `X-Request-Id` for every request.
- Include `requestId` in success `meta` and error responses.
- Log request ID with method/path/status/duration/user-agent.

## Client diagnostics support

- Add `backendVersion` and `authMode` to `/health` response data.
- Continue exposing database health state (`ok`, `unreachable`, `n/a`).

## Optional ingestion endpoint for self-host telemetry

If third-party telemetry is not desired, add:

- `POST /api/v1/client-events` for non-PII diagnostic events

This endpoint is optional and not required for core app functionality.

## Acceptance criteria

- Production mode does not serve API over HTTP by default.
- Every response has a request identifier for support correlation.
- Health endpoint provides enough metadata for client capability checks.

