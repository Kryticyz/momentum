# Aggregation Spec

Functions in `services/dashboard-backend/aggregator.go`. All are pure — no I/O, no global state. All operate on `[]TimeEntry` slices returned by `store.Entries()`.

## Types

```go
type ProjectStat struct {
    Project string  `json:"project"`
    Minutes int     `json:"minutes"`
    Hours   float64 `json:"hours"`   // round(minutes/60, 2dp)
}

type DayStat struct {
    Date    string  `json:"date"`    // YYYY-MM-DD
    Minutes int     `json:"minutes"`
    Hours   float64 `json:"hours"`
}

type WeekStat struct {
    WeekStart string  `json:"weekStart"` // YYYY-MM-DD of Sunday
    Minutes   int     `json:"minutes"`
    Hours     float64 `json:"hours"`
}
```

---

## `filterByRange(entries []TimeEntry, from, to string) []TimeEntry`

Returns entries whose `Date` falls within `[from, to]` inclusive.

**Algorithm:** plain string comparison — `entry.Date >= from && entry.Date <= to`.

**Why string comparison is correct:** `Date` is always `YYYY-MM-DD`. This format is lexicographically ordered identically to chronological order. No timezone conversion is applied — the plugin writes `date` in the user's local timezone, so treating it as an opaque calendar string is correct.

**Never** parse `Date` as `time.Time` with UTC location. The plugin user's timezone is encoded in which calendar date was chosen, not in a UTC offset.

---

## `aggregateByProject(entries []TimeEntry) []ProjectStat`

Groups entries by `Project`, sums `Minutes`, computes `Hours`.

**Case-sensitivity:** grouping is case-sensitive. `"Project A"` and `"project a"` produce separate buckets. The plugin preserves the original wiki-link leaf case in JSONL output. The plugin's internal `aggregateMinutesByProject` lowercases keys, but that function is only used for weekly note table rendering — not for JSONL export.

**Sort order:** descending by `Minutes`. Ties broken alphabetically by `Project` (ascending) for deterministic output.

**Hours formula:** `math.Round(float64(minutes)/60*100) / 100` — round to 2 decimal places.

**Empty input:** returns an empty (non-nil) slice.

---

## `aggregateByDay(entries []TimeEntry, from, to string) []DayStat`

Groups entries by `Date`, sums `Minutes`, and **zero-fills every calendar day** in `[from, to]`.

**Zero-fill is required.** The frontend `DailyHoursChart` uses Recharts `BarChart` with `dataKey="date"` as the x-axis. If a date is missing from the data array, Recharts omits it from the axis entirely, creating visual gaps. Every date must appear even if `minutes == 0`.

**Algorithm:**
1. Build a `map[string]int` of `Date → sum(Minutes)` from entries.
2. Iterate from `from` to `to` one day at a time using `addDays(d, 1)`.
3. For each date `d`, emit `DayStat{Date: d, Minutes: totals[d]}` (map lookup returns 0 for missing keys).

**`addDays` implementation:** parses `YYYY-MM-DD` as UTC noon (`time.Date(y, m, d, 12, 0, 0, 0, time.UTC)`), calls `AddDate(0, 0, n)`, formats back. Using UTC noon avoids DST edges where midnight can produce an unexpected date in some timezones.

**Month and year boundaries** are handled automatically by `time.Date` — it normalises overflow (e.g. `Date(2026, 1, 32, ...)` becomes `2026-02-01`).

---

## `aggregateByWeek(entries []TimeEntry) []WeekStat`

Groups entries by their Sunday-start week, sums `Minutes`, returns sorted ascending by `WeekStart`.

**`weekStartSunday(dateIso string) string`** — the week-bucketing key:

```go
func weekStartSunday(dateIso string) string {
    parts := strings.SplitN(dateIso, "-", 3)
    y, _ := strconv.Atoi(parts[0])
    m, _ := strconv.Atoi(parts[1])
    d, _ := strconv.Atoi(parts[2])
    t := time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.UTC) // UTC noon
    dow := int(t.Weekday())  // 0 = Sunday
    t = t.AddDate(0, 0, -dow)
    return t.Format("2006-01-02")
}
```

**Why UTC noon:** mirrors `isoToUtcDate` in `src/core/date.ts`:
```typescript
function isoToUtcDate(dateIso: string): Date {
  const [year, month, day] = dateIso.split("-").map(Number);
  return new Date(Date.UTC(year, month - 1, day, 12, 0, 0));
}
```
Midnight UTC can roll back or forward a day when converted to a local timezone. Using noon avoids this for any timezone within UTC−11 to UTC+11.

**Sunday as day 0:** `time.Weekday()` returns 0 for Sunday. Subtracting `dow` from the date always yields a Sunday.

**Year-boundary example:** `2026-01-01` (Thursday) → `dow = 4` → subtract 4 days → `2025-12-28` (Sunday). This is correct.

**Sort order:** ascending by `WeekStart` string (lexicographic, safe for `YYYY-MM-DD`).

**No zero-fill for weeks.** Unlike `/api/days`, weeks without any entries are simply omitted. This is intentional — sparse weekly data is still legible on a line chart, and the exact date range for week-filling is ambiguous when entries span week boundaries outside the query range.

---

## `roundHours(minutes int) float64`

```go
math.Round(float64(minutes)/60*100) / 100
```

Rounds to 2 decimal places. Examples: `35m → 0.58`, `60m → 1.0`, `90m → 1.5`.

---

## Adding a New Aggregation

1. Define the response type in `aggregator.go` with JSON tags.
2. Write a pure function that takes `[]TimeEntry` and returns the new type.
3. Add a handler in `handlers.go` that calls `parseDateRange`, `filterByRange`, then your function.
4. Register the route in `server.go:newMux`.
5. Add tests in `aggregator_test.go` covering: empty input, single entry, multi-entry sum, sort order, and any boundary conditions specific to the new aggregation.
