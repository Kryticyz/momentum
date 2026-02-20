# Configuration Reference

The dashboard backend supports three layers of configuration, applied in order of increasing priority:

1. **Defaults** — sensible values for local development
2. **Config file** (`config.json`) — persistent settings
3. **Environment variables** — container/server overrides
4. **CLI flags** — highest priority, for one-off runs

## Configuration Options

| JSON Key | Env Var | CLI Flag | Type | Default | Description |
|---|---|---|---|---|---|
| `jsonl_path` | `JSONL_PATH` | `-jsonl` | string | `""` | Path to JSONL time entries file. Required for in-memory mode. |
| `database_url` | `DATABASE_URL` | `-database-url` | string | `""` | PostgreSQL connection URL. When set, uses Postgres instead of JSONL. |
| `api_key` | `API_KEY` | `-api-key` | string | `""` | Bearer token for API auth. Empty disables authentication. |
| `port` | `PORT` | `-port` | int | `8080` | HTTP listen port. |
| `bind_address` | `BIND_ADDRESS` | `-bind` | string | `""` | Bind address (e.g., `0.0.0.0`, `127.0.0.1`). Empty binds all interfaces. |
| `timezone` | `TIMEZONE` | `-tz` | string | `Australia/Sydney` | IANA timezone for date defaults and display. |
| `poll_interval_hours` | `POLL_INTERVAL_HOURS` | `-poll` | int | `1` | JSONL reload interval in hours. `0` disables polling. Only applies to in-memory mode. |
| `frontend_dir` | `FRONTEND_DIR` | `-frontend` | string | `./frontend/dist` | Path to frontend build output directory. |
| `serve_api` | `SERVE_API` | `-serve-api` | bool | `true` | Enable API routes. |
| `serve_frontend` | `SERVE_FRONTEND` | `-serve-frontend` | bool | `true` | Enable frontend static file serving and SPA fallback. |
| `cors_allowed_origins` | `CORS_ALLOWED_ORIGINS` | `-cors-origins` | string[] | `["http://localhost:5173"]` | Comma-separated list of allowed CORS origins. Use `*` to allow all. |
| `read_timeout_seconds` | — | `-read-timeout` | int | `15` | HTTP server read timeout. |
| `read_header_timeout_seconds` | — | `-read-header-timeout` | int | `10` | HTTP server read header timeout. |
| `write_timeout_seconds` | — | `-write-timeout` | int | `30` | HTTP server write timeout. |
| `idle_timeout_seconds` | — | `-idle-timeout` | int | `60` | HTTP server idle timeout. |
| `shutdown_timeout_seconds` | — | `-shutdown-timeout` | int | `10` | Graceful shutdown deadline. |
| — | — | `-migrate` | bool | `false` | Run schema migration (and optional JSONL import) then exit. |
| — | — | `-config` | string | `config.json` | Path to config JSON file. |

## Operating Modes

### In-Memory (JSONL) Mode

Set `jsonl_path` to use the file-backed in-memory store. The server reads the JSONL file on startup and periodically reloads it.

```bash
./momentum-dashboard -jsonl /path/to/time-entries.jsonl
```

### PostgreSQL Mode

Set `database_url` to switch to PostgreSQL. The JSONL poller is disabled in this mode.

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/momentum"
export API_KEY="your-secret-key"
./momentum-dashboard
```

### Migration

Import an existing JSONL file into PostgreSQL:

```bash
./momentum-dashboard -migrate -database-url "postgres://..." -jsonl /path/to/entries.jsonl
```

## Example Config File

```json
{
  "jsonl_path": "",
  "database_url": "postgres://user:pass@localhost:5432/momentum",
  "api_key": "your-secret-key",
  "port": 8080,
  "timezone": "Australia/Sydney",
  "poll_interval_hours": 1,
  "frontend_dir": "./frontend/dist",
  "serve_api": true,
  "serve_frontend": true,
  "cors_allowed_origins": ["http://localhost:5173"]
}
```

## Example Docker Compose Environment

```yaml
environment:
  DATABASE_URL: postgres://momentum:secret@db:5432/momentum
  API_KEY: ${API_KEY}
  PORT: "8080"
  TIMEZONE: Australia/Sydney
  CORS_ALLOWED_ORIGINS: "https://your-domain.com"
  SERVE_FRONTEND: "true"
  SERVE_API: "true"
```
