# dashboard-frontend

React + Vite + Recharts SPA. Displays three charts — project breakdown, daily hours, and weekly trend — for time entries served by `services/dashboard-backend`.

## File Map

```
src/
├── main.tsx                     mount React root into #root
├── App.tsx                      top-level layout; owns range and refreshKey state
├── types.ts                     shared TypeScript types (ProjectStat, DayStat, WeekStat, DateRange)
├── api.ts                       typed fetch wrappers; reads VITE_API_BASE_URL
├── hooks/
│   └── useTimeData.ts           fetches all three endpoints in parallel; manages loading/error state
└── components/
    ├── DateRangePicker.tsx      two <input type="date"> inputs; validates from ≤ to before calling onChange
    ├── RefreshButton.tsx        POST /refresh; disables during flight; shows inline error on failure
    ├── ProjectBreakdown.tsx     horizontal BarChart; sorted descending by minutes
    ├── DailyHoursChart.tsx      vertical BarChart; one bar per day (backend zero-fills gaps)
    └── WeeklyTrendChart.tsx     LineChart; one point per Sunday-start week; ascending
```

## Data Flow

```
App
  range: DateRange          ← DateRangePicker.onChange
  refreshKey: number        ← RefreshButton.onRefresh (increment)

  useTimeData(range, refreshKey)
    → Promise.all([fetchProjects, fetchDays, fetchWeeks])
    → { projects, days, weeks, loading, error }

  ProjectBreakdown(projects)
  DailyHoursChart(days)
  WeeklyTrendChart(weeks)
```

`useTimeData` sets a `cancelled` flag on cleanup to prevent stale state updates when the range changes before a fetch completes.

## API Client (`api.ts`)

| Export | Method | Path |
|---|---|---|
| `fetchProjects(range)` | GET | `/api/projects?from=…&to=…` |
| `fetchDays(range)` | GET | `/api/days?from=…&to=…` |
| `fetchWeeks(range)` | GET | `/api/weeks?from=…&to=…` |
| `postRefresh()` | POST | `/refresh` |

`VITE_API_BASE_URL` is prepended to all paths. When unset (empty string), paths are relative — correct for both the Vite dev proxy and the production single-origin setup.

## Dev Mode

Vite proxies `/api/*`, `/health`, and `/refresh` to `http://localhost:8080`:

```ts
// vite.config.ts
server: {
  proxy: {
    "/api": "http://localhost:8080",
    "/health": "http://localhost:8080",
    "/refresh": "http://localhost:8080",
  },
},
```

No `VITE_API_BASE_URL` needed in dev — just run the Go backend on `:8080` and `npm run dev` on `:5173`.

## Build

```bash
npm install
npm run build
```

Output lands in `../../services/dashboard-backend/frontend/dist/`. The Go service serves these files from `/`.

## Run Dev Server

```bash
npm run dev   # → http://localhost:5173
```

## Test

```bash
npm test
```

31 tests across 5 files in `src/test/`:

| File | Covers |
|---|---|
| `aggregator.test.ts` | `defaultRange`, `rangeParams`, tooltip/tick formatters |
| `DateRangePicker.test.tsx` | render, from/to change, invalid range rejection |
| `RefreshButton.test.tsx` | click, loading state, error display, disabled state |
| `charts.test.tsx` | empty state, render-without-crash for all three charts |
| `useTimeData.test.tsx` | loading transition, data population, error handling, re-fetch on refreshKey/range change |

## Adding a New Chart

1. Add a type to `src/types.ts` if a new response shape is needed.
2. Add a fetch function to `src/api.ts`.
3. Add the fetch call to `useTimeData.ts` inside the `Promise.all`.
4. Create a component in `src/components/`.
5. Render it inside a `<Section>` in `App.tsx`.
6. Add tests to `src/test/`.
