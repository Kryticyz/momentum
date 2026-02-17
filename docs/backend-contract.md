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

## Response Shapes

### `/api/v1/projects`

```json
[
  {
    "project": "Project A",
    "minutes": 320,
    "hours": 5.33
  }
]
```

### `/api/v1/days`

```json
[
  {
    "date": "2026-02-12",
    "minutes": 180
  }
]
```

### `/api/v1/weeks`

```json
[
  {
    "weekStart": "2026-02-08",
    "minutes": 640
  }
]
```

### `/api/v1/planned-vs-actual`

Current response:

```json
{ "error": "not implemented" }
```

## Operational Notes

- CORS is allow-list based (`cors_allowed_origins`), with `["*"]` as wildcard option.
- Server supports API-only, frontend-only, or combined mode (`serve_api`, `serve_frontend`).
- Frontend SPA fallback serves `index.html` for client routes, but not for missing static assets.
