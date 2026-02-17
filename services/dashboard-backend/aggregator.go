package main

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ProjectStat is the response element for /api/projects.
type ProjectStat struct {
	Project string  `json:"project"`
	Minutes int     `json:"minutes"`
	Hours   float64 `json:"hours"`
}

// DayStat is the response element for /api/days.
type DayStat struct {
	Date    string `json:"date"`
	Minutes int    `json:"minutes"`
	Hours   float64 `json:"hours"`
}

// WeekStat is the response element for /api/weeks.
type WeekStat struct {
	WeekStart string  `json:"weekStart"`
	Minutes   int     `json:"minutes"`
	Hours     float64 `json:"hours"`
}

// filterByRange returns entries whose Date falls within [from, to] inclusive.
// String comparison is safe because YYYY-MM-DD is lexicographically ordered.
func filterByRange(entries []TimeEntry, from, to string) []TimeEntry {
	var out []TimeEntry
	for _, e := range entries {
		if e.Date >= from && e.Date <= to {
			out = append(out, e)
		}
	}
	return out
}

// aggregateByProject groups entries by project (original case), sums minutes,
// computes hours, and returns sorted descending by minutes.
func aggregateByProject(entries []TimeEntry) []ProjectStat {
	totals := make(map[string]int)
	for _, e := range entries {
		totals[e.Project] += e.Minutes
	}

	stats := make([]ProjectStat, 0, len(totals))
	for project, minutes := range totals {
		stats = append(stats, ProjectStat{
			Project: project,
			Minutes: minutes,
			Hours:   roundHours(minutes),
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Minutes != stats[j].Minutes {
			return stats[i].Minutes > stats[j].Minutes
		}
		return stats[i].Project < stats[j].Project
	})

	return stats
}

// aggregateByDay groups entries by date, sums minutes, and fills zeros for
// every day in [from, to] that has no entries. This ensures the bar chart
// receives a contiguous sequence of dates.
func aggregateByDay(entries []TimeEntry, from, to string) []DayStat {
	totals := make(map[string]int)
	for _, e := range entries {
		totals[e.Date] += e.Minutes
	}

	var stats []DayStat
	for d := from; d <= to; d = addDays(d, 1) {
		mins := totals[d]
		stats = append(stats, DayStat{
			Date:    d,
			Minutes: mins,
			Hours:   roundHours(mins),
		})
	}

	return stats
}

// aggregateByWeek groups entries by their Sunday-start week, sums minutes,
// and returns sorted ascending by weekStart.
func aggregateByWeek(entries []TimeEntry) []WeekStat {
	totals := make(map[string]int)
	for _, e := range entries {
		ws := weekStartSunday(e.Date)
		totals[ws] += e.Minutes
	}

	stats := make([]WeekStat, 0, len(totals))
	for weekStart, minutes := range totals {
		stats = append(stats, WeekStat{
			WeekStart: weekStart,
			Minutes:   minutes,
			Hours:     roundHours(minutes),
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].WeekStart < stats[j].WeekStart
	})

	return stats
}

// weekStartSunday returns the YYYY-MM-DD of the Sunday that starts the ISO week
// containing dateIso. Mirrors the plugin's getWeekStartSunday / isoToUtcDate:
// parse as UTC noon to avoid DST day-boundary drift.
func weekStartSunday(dateIso string) string {
	parts := strings.SplitN(dateIso, "-", 3)
	if len(parts) != 3 {
		return dateIso
	}
	y, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	d, _ := strconv.Atoi(parts[2])

	// UTC noon — mirrors isoToUtcDate in date.ts
	t := time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.UTC)
	dow := int(t.Weekday()) // 0 = Sunday
	t = t.AddDate(0, 0, -dow)
	return t.Format("2006-01-02")
}

// addDays returns the date dateIso + n days as a YYYY-MM-DD string.
func addDays(dateIso string, n int) string {
	parts := strings.SplitN(dateIso, "-", 3)
	if len(parts) != 3 {
		return dateIso
	}
	y, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	d, _ := strconv.Atoi(parts[2])
	t := time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.UTC)
	t = t.AddDate(0, 0, n)
	return t.Format("2006-01-02")
}

// roundHours converts minutes to hours rounded to 2 decimal places.
func roundHours(minutes int) float64 {
	return math.Round(float64(minutes)/60*100) / 100
}
