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

- `GET /health`
- `POST /refresh`
- `GET /api/v1/entries?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `GET /api/v1/projects?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `GET /api/v1/days?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `GET /api/v1/weeks?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `GET /api/v1/planned-vs-actual?from=YYYY-MM-DD&to=YYYY-MM-DD` (stub)

`from` and `to` default to last 30 days in configured timezone if omitted.

## Response Envelope

All successful responses use a standard envelope:

```json
{
  "data": <payload>,
  "meta": {
    "count": 2,
    "lastLoaded": "2026-02-19T10:00:00+11:00",
    "version": "16e4a2b3c5d00"
  }
}
```

- `data`: The response payload (array for data endpoints, object for health/refresh).
- `meta.count`: Number of items in the data payload.
- `meta.lastLoaded`: RFC 3339 timestamp of the most recent data reload.
- `meta.version`: Opaque string that changes when the underlying data changes. Compare across responses to detect changes without HTTP caching.

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
    "entries": 142
  },
  "meta": { "count": 142, "lastLoaded": "2026-02-19T10:00:00+11:00" }
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

## Conditional Requests (ETag)

All GET endpoints return `ETag` and `Last-Modified` headers when data has been loaded at least once.

Clients can send `If-None-Match: <etag>` on subsequent requests. If the store has not reloaded since the previous response, the server returns `304 Not Modified` with an empty body.

The ETag is derived from the store's version (which changes on each reload). It is a coarse "did the data change" signal — not per-query. If the store has not reloaded, no query's results can have changed.

Example flow:
1. `GET /api/v1/projects` → `200` with `ETag: "16e4a2b3c5d00"`
2. `GET /api/v1/projects` with `If-None-Match: "16e4a2b3c5d00"` → `304` (empty body)
3. `POST /refresh` (data reloads, version changes)
4. `GET /api/v1/projects` with `If-None-Match: "16e4a2b3c5d00"` → `200` with new ETag

## Data Consistency Model

1. **Eventual consistency** — data refreshes on poll interval (`poll_interval_hours`) or manual `POST /refresh`. Clients may see data up to one poll interval old.
2. **Atomic reloads** — the store replaces all entries under a write lock. No partial-update state.
3. **Read isolation** — `Entries()` returns a copy. Concurrent reads during a reload see either the old or new data, never a torn read.
4. **Change detection** — use `ETag` / `If-None-Match` to skip unchanged data. The `meta.version` field provides the same signal for clients that don't use HTTP-level caching.
5. **Single writer** — the JSONL file is written by the Obsidian plugin. The backend only reads it.

## Native Client Notes

- Native apps (macOS, iOS) do not send `Origin` headers, so CORS does not apply to them.
- Recommended polling interval: 30–60 seconds with conditional requests (`If-None-Match`). This avoids re-processing unchanged data.
- Set a distinctive `User-Agent` header (e.g., `Momentum-macOS/1.0`) for log differentiation.
- Backend discovery: configure `host:port` manually. Hit `GET /health` to verify the server is a Momentum backend (check `data.status == "ok"`).

## Operational Notes

- CORS is allow-list based (`cors_allowed_origins`), with `["*"]` as wildcard option.
- Server supports API-only, frontend-only, or combined mode (`serve_api`, `serve_frontend`).
- Frontend SPA fallback serves `index.html` for client routes, but not for missing static assets.
- All requests are logged with method, path, status code, duration, and User-Agent.
