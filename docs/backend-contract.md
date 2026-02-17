# Backend Contract (Go Dashboard Service)

This defines the handoff between the Obsidian plugin and a local self-hosted Go service.

## Input Source
Default export file from plugin:
- `.obsidian/project-insights/time-entries.jsonl`

Refresh behavior:
- Poll every 1 hour (configurable)
- Manual refresh endpoint for immediate re-read

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
- `date` is local note date (timezone-aware from plugin setting)
- `minutes` is authoritative for aggregation
- `project` is project note name (wiki-link target leaf)

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

## Aggregations Needed
- Per project total minutes (range)
- Per day total minutes (range)
- Weekly trend (Sunday-start week)

Timezone:
- Default `Australia/Sydney`
- Configurable in backend config

## Suggested API (v1)
- `GET /health`
- `POST /refresh` (manual reload of JSONL)
- `GET /api/entries?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `GET /api/projects?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `GET /api/days?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `GET /api/weeks?from=YYYY-MM-DD&to=YYYY-MM-DD`

## Response Shape Suggestions
### `/api/projects`
```json
[
  {
    "project": "Project A",
    "minutes": 320,
    "hours": 5.33
  }
]
```

### `/api/days`
```json
[
  {
    "date": "2026-02-12",
    "minutes": 180
  }
]
```

### `/api/weeks`
```json
[
  {
    "weekStart": "2026-02-08",
    "minutes": 640
  }
]
```

## Planned vs Actual (Stub)
Keep endpoint placeholder for future goals/plan ingestion:
- `GET /api/planned-vs-actual?from=YYYY-MM-DD&to=YYYY-MM-DD`

For now, return an empty array or `501 Not Implemented`.
