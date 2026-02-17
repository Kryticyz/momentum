# dashboard-backend

Go HTTP service that reads the plugin's JSONL export and serves aggregated time data to clients. No database; data is kept in memory and refreshed from disk.

## Module Map

| File | Responsibility |
|---|---|
| `main.go` | Entry point: config, initial load, poller startup, HTTP server lifecycle |
| `config.go` | `Config` struct, config loading, CLI overrides |
| `config.json` | Runtime config example |
| `loader.go` | JSONL reader and parser (`[]TimeEntry`) |
| `store.go` | Thread-safe in-memory store and polling reload loop |
| `storage.go` | `EntryStore` interface used by handlers |
| `aggregator.go` | Pure aggregation functions |
| `handlers.go` | HTTP handlers and request validation |
| `server.go` | Route wiring, CORS middleware, SPA fallback handler |

## Configuration

`config.json` is loaded first, then CLI flags override individual fields.

```json
{
  "jsonl_path": "",
  "port": 8080,
  "timezone": "Australia/Sydney",
  "poll_interval_hours": 1,
  "serve_api": true,
  "serve_frontend": true,
  "frontend_dir": "./frontend/dist",
  "cors_allowed_origins": ["http://localhost:5173"],
  "read_timeout_seconds": 15,
  "read_header_timeout_seconds": 10,
  "write_timeout_seconds": 30,
  "idle_timeout_seconds": 60,
  "shutdown_timeout_seconds": 10
}
```

| Field | CLI flag | Default |
|---|---|---|
| `jsonl_path` | `-jsonl` | `""` |
| `port` | `-port` | `8080` |
| `timezone` | `-tz` | `"Australia/Sydney"` |
| `poll_interval_hours` | `-poll` | `1` |
| `serve_api` | `-serve-api` (`true`/`false`) | `true` |
| `serve_frontend` | `-serve-frontend` (`true`/`false`) | `true` |
| `frontend_dir` | `-frontend` | `"./frontend/dist"` |
| `cors_allowed_origins` | `-cors-origins` (comma-separated) | `["http://localhost:5173"]` |
| `read_timeout_seconds` | `-read-timeout` | `15` |
| `read_header_timeout_seconds` | `-read-header-timeout` | `10` |
| `write_timeout_seconds` | `-write-timeout` | `30` |
| `idle_timeout_seconds` | `-idle-timeout` | `60` |
| `shutdown_timeout_seconds` | `-shutdown-timeout` | `10` |

Use `-config /path/to/config.json` to choose a different config file.

## API Endpoints

All `GET` endpoints accept `?from=YYYY-MM-DD&to=YYYY-MM-DD`. If omitted, range defaults to the last 30 days in configured timezone.

### `GET /health`

```json
{
  "status": "ok",
  "entries": 142,
  "lastLoaded": "2026-02-17T19:52:49+11:00"
}
```

### `POST /refresh`

Triggers immediate reload of configured `jsonl_path`.

```json
{ "ok": true, "entries": 142 }
```

### `GET /api/v1/entries`

Raw filtered `[]TimeEntry` for diagnostics.

### `GET /api/v1/projects`

Minutes per project, sorted descending.

### `GET /api/v1/days`

Daily totals across range, zero-filled for missing dates.

### `GET /api/v1/weeks`

Weekly totals (Sunday-start weeks), sorted ascending.

### `GET /api/v1/planned-vs-actual`

Stub endpoint. Returns `501` with:

```json
{ "error": "not implemented" }
```

## Static Frontend Serving

When `serve_frontend=true`, `/` serves static files from `frontend_dir`.

SPA routing fallback behavior:
- Existing files are served directly.
- Missing asset requests (for example `.js`/`.css`) return `404`.
- Unknown client routes (for example `/reports/weekly`) serve `index.html`.

## CORS

CORS allow-list comes from `cors_allowed_origins`.
- `OPTIONS` preflight returns `204` for allowed origins.
- Disallowed preflight origins return `403`.
- Set `["*"]` to allow all origins.

## Server Hardening

- Configurable read/read-header/write/idle HTTP timeouts.
- Signal-based graceful shutdown (`SIGINT`, `SIGTERM`) with configurable shutdown timeout.
- Store poller uses context cancellation and exits on shutdown.

## OpenAPI

- Canonical API spec: `docs/openapi-dashboard-backend.yaml`

## Run

```bash
go run .
# or:
go build -o dashboard .
./dashboard
```

## Test

```bash
go test ./...
```
