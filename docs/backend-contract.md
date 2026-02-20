# Backend Contract (Go Dashboard Service)

Defines the data/API contract between producers (Obsidian export) and consumers (dashboard and future clients).

## Input Source

Default export file from plugin:
- `.obsidian/momentum/time-entries.jsonl`

Refresh behavior:
- Optional polling reload (`poll_interval_hours`)
- Manual refresh via `POST /refresh`

## JSONL Record Schema

Each line is one JSON object.

```json
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
```

Field notes:
- `date` is local note date (timezone-aware from plugin setting).
- `minutes` is authoritative for aggregation.
- `project` is the project note title (wiki-link target leaf).

## Go In-Memory Model

```go
type TimeEntry struct {
    Source     string `json:"source"`
    FilePath   string `json:"filePath"`
    Date       string `json:"date"` // YYYY-MM-DD
    Project    string `json:"project"`
    Start      string `json:"start"` // HH:mm
    End        string `json:"end"`   // HH:mm
    Minutes    int    `json:"minutes"`
    Note       string `json:"note"`
    LineNumber int    `json:"lineNumber"`
}
```

## API (v1)

- `GET /health` — service health and database connectivity
- `POST /refresh` — reload data from backing source
- `GET /api/v1/entries?from=YYYY-MM-DD&to=YYYY-MM-DD` — raw entries
- `POST /api/v1/entries` — push JSON array of entries
- `GET /api/v1/projects?from=YYYY-MM-DD&to=YYYY-MM-DD` — project aggregates
- `GET /api/v1/days?from=YYYY-MM-DD&to=YYYY-MM-DD` — daily aggregates (zero-filled)
- `GET /api/v1/weeks?from=YYYY-MM-DD&to=YYYY-MM-DD` — weekly aggregates
- `POST /api/v1/import` — import JSONL body of entries
- `GET /api/v1/planned-vs-actual?from=YYYY-MM-DD&to=YYYY-MM-DD` (stub)

`from` and `to` default to last 30 days in configured timezone if omitted.

## Authentication

When `API_KEY` is configured, all endpoints except `GET /health` and `OPTIONS` require a Bearer token:

```
Authorization: Bearer <api-key>
```

Missing or invalid tokens return `401 Unauthorized`. When `API_KEY` is empty, authentication is disabled (open-source local mode).

## Response Envelope

All successful responses use a standard envelope:

```json
{
  "data": <payload>,
  "meta": {
    "count": 2,
    "lastLoaded": "2026-02-19T10:00:00+11:00"
  }
}
```

- `data`: The response payload (array for data endpoints, object for health/refresh).
- `meta.count`: Number of items in the data payload.
- `meta.lastLoaded`: RFC 3339 timestamp of the most recent data reload.

All error responses are JSON with at minimum an `"error"` field:

```json
{ "error": "descriptive error message" }
```

Error responses do not use the envelope — they are bare objects.

## Response Shapes

### `/health`

```json
{
  "data": {
    "status": "ok",
    "entries": 142,
    "database": "ok",
    "listenAddress": ":8080"
  },
  "meta": { "count": 142, "lastLoaded": "2026-02-19T10:00:00+11:00" }
}
```

The `database` field reports: `"ok"` (Postgres connected), `"unreachable"` (Postgres ping failed), or `"n/a"` (in-memory JSONL mode).

### `POST /api/v1/entries`

Request body: JSON array of `TimeEntry` objects. Returns 201:

```json
{
  "data": { "accepted": 5 },
  "meta": { "count": 5, "lastLoaded": "..." }
}
```

### `POST /api/v1/import`

Request body: JSONL text (one `TimeEntry` JSON per line). Returns 201:

```json
{
  "data": { "imported": 10 },
  "meta": { "count": 10, "lastLoaded": "..." }
}
```

### `/refresh`

```json
{
  "data": {
    "ok": true,
    "entries": 142
  },
  "meta": { "count": 142, "lastLoaded": "2026-02-19T10:00:00+11:00" }
}
```

### `/api/v1/projects`

```json
{
  "data": [
    {
      "project": "Project A",
      "minutes": 320,
      "hours": 5.33
    }
  ],
  "meta": { "count": 1, "lastLoaded": "2026-02-19T10:00:00+11:00" }
}
```

### `/api/v1/days`

```json
{
  "data": [
    {
      "date": "2026-02-12",
      "minutes": 180,
      "hours": 3.0
    }
  ],
  "meta": { "count": 1, "lastLoaded": "2026-02-19T10:00:00+11:00" }
}
```

### `/api/v1/weeks`

```json
{
  "data": [
    {
      "weekStart": "2026-02-08",
      "minutes": 640,
      "hours": 10.67
    }
  ],
  "meta": { "count": 1, "lastLoaded": "2026-02-19T10:00:00+11:00" }
}
```

### `/api/v1/planned-vs-actual`

Current response:

```json
{ "error": "not implemented" }
```

## Entry Validation

Entries submitted via `POST /api/v1/entries` or `POST /api/v1/import` are validated before storage:

- `date`: required, must match `YYYY-MM-DD` format
- `project`: required, must be non-empty
- `minutes`: required, must be `> 0`

All other fields (`source`, `filePath`, `start`, `end`, `note`, `lineNumber`) are accepted as-is without validation.

## Week Aggregation

The `/api/v1/weeks` endpoint groups entries using **Sunday-start weeks** (matching the Obsidian plugin's `getWeekStartSunday` convention). This is _not_ ISO-8601, which uses Monday as the week start. The `weekStart` field in responses is always the date of the Sunday that begins each week.

## CORS Behaviour

CORS origin matching is case-sensitive and normalises trailing slashes — `http://example.com/` and `http://example.com` are treated as the same origin.

## Data Consistency Model

### In-Memory (JSONL) Mode

1. **Eventual consistency** — data refreshes on poll interval (`poll_interval_hours`) or manual `POST /refresh`. Clients may see data up to one poll interval old.
2. **Atomic reloads** — the store replaces all entries atomically. No partial-update state.
3. **Read isolation** — reads use an atomic pointer to an immutable snapshot. Concurrent reads during a reload see either the old or new data, never a torn read.
4. **Multiple writers** — the JSONL file is written by the Obsidian plugin. Push and import APIs append to the in-memory snapshot.

### PostgreSQL Mode

1. **Immediate consistency** — queries hit Postgres directly. Data is available as soon as `AddEntries` commits.
2. **Upsert deduplication** — entries are upserted on the composite key `(date, file_path, line_number)`. Re-importing the same JSONL file is idempotent and safe to repeat.
3. **Connection pooling** — `pgxpool` manages connections. Health check pings the database.

## Dual Store Architecture

The backend selects its store based on configuration:
- **`DATABASE_URL` set** → PostgreSQL (`PgStore`). JSONL poller is disabled.
- **`DATABASE_URL` empty** → In-memory JSONL (`Store`). File polling enabled if `jsonl_path` is configured.

## Native Client Notes

- Native apps (macOS, iOS) do not send `Origin` headers, so CORS does not apply to them.
- Recommended polling interval: 30–60 seconds.
- Set a distinctive `User-Agent` header (e.g., `Momentum-macOS/1.0`) for log differentiation.
- Backend discovery: configure `host:port` manually. Hit `GET /health` to verify the server is a Momentum backend (check `data.status == "ok"`).

## Operational Notes

- CORS is allow-list based (`cors_allowed_origins`), with `["*"]` as wildcard option.
- Server supports API-only, frontend-only, or combined mode (`serve_api`, `serve_frontend`).
- Frontend SPA fallback serves `index.html` for client routes, but not for missing static assets.
- All requests are logged with method, path, status code, duration, and User-Agent.
