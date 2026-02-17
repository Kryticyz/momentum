# dashboard-backend

Go HTTP service that reads the plugin's JSONL export and serves aggregated time data to the dashboard frontend. No database — in-memory only, reloaded from disk on a configurable interval.

## Module Map

| File | Responsibility |
|---|---|
| `main.go` | Entry point: parse config, initial load, start poller, start server |
| `config.go` | `Config` struct, load from `config.json`, override with CLI flags |
| `config.json` | User-editable runtime config (not committed with real paths) |
| `loader.go` | Read JSONL file line-by-line; skip malformed lines, return `[]TimeEntry` |
| `store.go` | Thread-safe in-memory store (`sync.RWMutex`); `Load()`, `Entries()`, `startPoller()` |
| `aggregator.go` | Pure aggregation functions: `filterByRange`, `aggregateByProject`, `aggregateByDay`, `aggregateByWeek` |
| `handlers.go` | All `http.HandlerFunc` implementations; `parseDateRange` helper; `writeJSON` helper |
| `server.go` | `newMux` wiring all routes; `corsMiddleware`; `spaHandler` static file fallback |

## Configuration

`config.json` is loaded first, then CLI flags override individual fields.

```json
{
  "jsonl_path": "/path/to/vault/.obsidian/project-insights/time-entries.jsonl",
  "port": 8080,
  "timezone": "Australia/Sydney",
  "poll_interval_hours": 1,
  "frontend_dir": "./frontend/dist"
}
```

| Field | CLI flag | Default |
|---|---|---|
| `jsonl_path` | `-jsonl` | `""` (warn on startup, serve empty store) |
| `port` | `-port` | `8080` |
| `timezone` | `-tz` | `"Australia/Sydney"` |
| `poll_interval_hours` | `-poll` | `1` |
| `frontend_dir` | `-frontend` | `"./frontend/dist"` |

To use a different config file: `-config /path/to/config.json`

## API Endpoints

All `GET` endpoints accept `?from=YYYY-MM-DD&to=YYYY-MM-DD`. Both default to the last 30 days in the configured timezone when omitted. Returns `400` on invalid or inverted range.

### `GET /health`

```json
{
  "status": "ok",
  "entries": 142,
  "lastLoaded": "2026-02-17T19:52:49+11:00"
}
```

### `POST /refresh`

Triggers an immediate reload of the JSONL file.

```json
{ "ok": true, "entries": 142 }
```

### `GET /api/entries`

Raw `[]TimeEntry` filtered by date range. Use for debugging; not paginated.

```json
[
  {
    "source": "daily-note",
    "filePath": "2026-02-12.md",
    "date": "2026-02-12",
    "project": "Project A",
    "start": "09:10",
    "end": "09:45",
    "minutes": 35,
    "note": "Deep work",
    "lineNumber": 42
  }
]
```

### `GET /api/projects`

Minutes per project, sorted descending by `minutes`. `hours` is rounded to 2 decimal places.

```json
[
  { "project": "Project A", "minutes": 320, "hours": 5.33 },
  { "project": "Project B", "minutes": 180, "hours": 3.0 }
]
```

### `GET /api/days`

One entry per calendar day in `[from, to]`. **Zero-filled** — days with no entries appear as `minutes: 0`. This guarantees contiguous data for bar charts.

```json
[
  { "date": "2026-02-01", "minutes": 90, "hours": 1.5 },
  { "date": "2026-02-02", "minutes": 0,  "hours": 0.0 },
  { "date": "2026-02-03", "minutes": 60, "hours": 1.0 }
]
```

### `GET /api/weeks`

One entry per Sunday-start week containing entries in range. Sorted ascending by `weekStart`.

```json
[
  { "weekStart": "2026-02-08", "minutes": 640, "hours": 10.67 },
  { "weekStart": "2026-02-15", "minutes": 300, "hours": 5.0 }
]
```

### `GET /api/planned-vs-actual`

Returns `501 Not Implemented`:

```json
{ "error": "not implemented" }
```

### `GET /` (static files)

Serves `frontend/dist/` — the built React SPA. Handles all non-API paths.

## CORS

All responses carry `Access-Control-Allow-Origin: *`. OPTIONS preflight returns `204`. In production (Go serves both API and static files on the same port), CORS is irrelevant; headers are present for dev-mode convenience.

## Key Invariants

**Date filtering** uses plain string comparison (`entry.Date >= from`). This is correct because `YYYY-MM-DD` is lexicographically ordered. Never parse `date` as UTC — it was written in the user's local timezone by the plugin.

**Week-start** uses UTC noon to avoid DST day-boundary drift. This mirrors `src/core/date.ts:getWeekStartSunday` exactly. See `docs/aggregation-spec.md` for the algorithm.

**Project grouping** is case-sensitive. `"Project A"` and `"project a"` are separate buckets. The plugin preserves original case in JSONL output.

**`/api/days` zero-fill** iterates the full `[from, to]` calendar range using `addDays`, not just dates present in the data. The frontend bar chart requires contiguous data.

## Run

```bash
go run .
# or build a binary:
go build -o dashboard .
./dashboard
```

## Test

```bash
go test ./...
```

47 tests across `aggregator_test.go`, `handlers_test.go`, `loader_test.go`.
