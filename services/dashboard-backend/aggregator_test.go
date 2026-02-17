package main

import (
	"testing"
)

// --- helpers ---

func makeEntry(date, project string, minutes int) TimeEntry {
	return TimeEntry{
		Source:  "daily-note",
		Date:    date,
		Project: project,
		Minutes: minutes,
	}
}

// --- filterByRange ---

func TestFilterByRange_IncludesEndpoints(t *testing.T) {
	entries := []TimeEntry{
		makeEntry("2026-02-01", "A", 30),
		makeEntry("2026-02-15", "B", 60),
		makeEntry("2026-02-28", "C", 90),
	}
	got := filterByRange(entries, "2026-02-01", "2026-02-28")
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
}

func TestFilterByRange_ExcludesOutside(t *testing.T) {
	entries := []TimeEntry{
		makeEntry("2026-01-31", "A", 30),
		makeEntry("2026-02-01", "B", 60),
		makeEntry("2026-03-01", "C", 90),
	}
	got := filterByRange(entries, "2026-02-01", "2026-02-28")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %+v", len(got), got)
	}
	if got[0].Project != "B" {
		t.Errorf("expected project B, got %s", got[0].Project)
	}
}

func TestFilterByRange_Empty(t *testing.T) {
	got := filterByRange(nil, "2026-02-01", "2026-02-28")
	if len(got) != 0 {
		t.Fatalf("expected 0, got %d", len(got))
	}
}

func TestFilterByRange_SameDay(t *testing.T) {
	entries := []TimeEntry{
		makeEntry("2026-02-15", "A", 30),
		makeEntry("2026-02-16", "B", 60),
	}
	got := filterByRange(entries, "2026-02-15", "2026-02-15")
	if len(got) != 1 || got[0].Project != "A" {
		t.Errorf("expected only entry A, got %+v", got)
	}
}

// --- aggregateByProject ---

func TestAggregateByProject_SumsAndSorts(t *testing.T) {
	entries := []TimeEntry{
		makeEntry("2026-02-01", "Alpha", 60),
		makeEntry("2026-02-02", "Beta", 120),
		makeEntry("2026-02-03", "Alpha", 30),
	}
	stats := aggregateByProject(entries)
	if len(stats) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(stats))
	}
	// Beta (120) should be first, Alpha (90) second.
	if stats[0].Project != "Beta" || stats[0].Minutes != 120 {
		t.Errorf("expected Beta=120 first, got %+v", stats[0])
	}
	if stats[1].Project != "Alpha" || stats[1].Minutes != 90 {
		t.Errorf("expected Alpha=90 second, got %+v", stats[1])
	}
}

func TestAggregateByProject_PreservesCase(t *testing.T) {
	entries := []TimeEntry{
		makeEntry("2026-02-01", "My Project", 30),
		makeEntry("2026-02-02", "my project", 30), // different case → different bucket
	}
	stats := aggregateByProject(entries)
	if len(stats) != 2 {
		t.Errorf("expected 2 separate project buckets (case-sensitive), got %d: %+v", len(stats), stats)
	}
}

func TestAggregateByProject_HoursRounded(t *testing.T) {
	entries := []TimeEntry{makeEntry("2026-02-01", "A", 35)}
	stats := aggregateByProject(entries)
	// 35 / 60 = 0.5833... rounded to 2dp = 0.58
	if stats[0].Hours != 0.58 {
		t.Errorf("expected hours=0.58, got %f", stats[0].Hours)
	}
}

func TestAggregateByProject_Empty(t *testing.T) {
	stats := aggregateByProject(nil)
	if len(stats) != 0 {
		t.Fatalf("expected 0, got %d", len(stats))
	}
}

// --- aggregateByDay ---

func TestAggregateByDay_ZeroFillsGaps(t *testing.T) {
	entries := []TimeEntry{
		makeEntry("2026-02-01", "A", 60),
		makeEntry("2026-02-03", "B", 30),
	}
	stats := aggregateByDay(entries, "2026-02-01", "2026-02-03")
	if len(stats) != 3 {
		t.Fatalf("expected 3 day stats (zero-filled), got %d", len(stats))
	}
	if stats[0].Date != "2026-02-01" || stats[0].Minutes != 60 {
		t.Errorf("day 1: %+v", stats[0])
	}
	if stats[1].Date != "2026-02-02" || stats[1].Minutes != 0 {
		t.Errorf("day 2 should be zero-filled: %+v", stats[1])
	}
	if stats[2].Date != "2026-02-03" || stats[2].Minutes != 30 {
		t.Errorf("day 3: %+v", stats[2])
	}
}

func TestAggregateByDay_SumsMultipleEntriesPerDay(t *testing.T) {
	entries := []TimeEntry{
		makeEntry("2026-02-01", "A", 30),
		makeEntry("2026-02-01", "B", 45),
	}
	stats := aggregateByDay(entries, "2026-02-01", "2026-02-01")
	if len(stats) != 1 || stats[0].Minutes != 75 {
		t.Errorf("expected 75 minutes, got %+v", stats)
	}
}

func TestAggregateByDay_SingleDay(t *testing.T) {
	entries := []TimeEntry{makeEntry("2026-02-15", "A", 90)}
	stats := aggregateByDay(entries, "2026-02-15", "2026-02-15")
	if len(stats) != 1 || stats[0].Minutes != 90 {
		t.Errorf("expected 90 minutes, got %+v", stats)
	}
}

func TestAggregateByDay_MonthBoundary(t *testing.T) {
	// Feb 28 to Mar 2 (non-leap 2026)
	entries := []TimeEntry{makeEntry("2026-03-01", "A", 60)}
	stats := aggregateByDay(entries, "2026-02-28", "2026-03-02")
	if len(stats) != 3 {
		t.Fatalf("expected 3 days, got %d", len(stats))
	}
	if stats[0].Date != "2026-02-28" {
		t.Errorf("first date wrong: %s", stats[0].Date)
	}
	if stats[2].Date != "2026-03-02" {
		t.Errorf("last date wrong: %s", stats[2].Date)
	}
}

// --- aggregateByWeek ---

func TestAggregateByWeek_GroupsBySundayStart(t *testing.T) {
	// 2026-02-08 is a Sunday. 2026-02-09 is Monday (same week).
	// 2026-02-15 is the next Sunday.
	entries := []TimeEntry{
		makeEntry("2026-02-08", "A", 60),
		makeEntry("2026-02-09", "B", 30),
		makeEntry("2026-02-15", "C", 90),
	}
	stats := aggregateByWeek(entries)
	if len(stats) != 2 {
		t.Fatalf("expected 2 weeks, got %d: %+v", len(stats), stats)
	}
	if stats[0].WeekStart != "2026-02-08" || stats[0].Minutes != 90 {
		t.Errorf("week 1: %+v", stats[0])
	}
	if stats[1].WeekStart != "2026-02-15" || stats[1].Minutes != 90 {
		t.Errorf("week 2: %+v", stats[1])
	}
}

func TestAggregateByWeek_SortedAscending(t *testing.T) {
	entries := []TimeEntry{
		makeEntry("2026-02-15", "A", 60),
		makeEntry("2026-02-01", "B", 30),
	}
	stats := aggregateByWeek(entries)
	if len(stats) < 2 {
		t.Fatal("expected at least 2 weeks")
	}
	if stats[0].WeekStart >= stats[1].WeekStart {
		t.Errorf("not sorted ascending: %s >= %s", stats[0].WeekStart, stats[1].WeekStart)
	}
}

func TestAggregateByWeek_Empty(t *testing.T) {
	stats := aggregateByWeek(nil)
	if len(stats) != 0 {
		t.Fatalf("expected 0, got %d", len(stats))
	}
}

// --- weekStartSunday ---

func TestWeekStartSunday_Sunday(t *testing.T) {
	// 2026-02-08 is a Sunday — should return itself.
	got := weekStartSunday("2026-02-08")
	if got != "2026-02-08" {
		t.Errorf("Sunday should map to itself, got %s", got)
	}
}

func TestWeekStartSunday_Monday(t *testing.T) {
	// 2026-02-09 is a Monday — week starts 2026-02-08.
	got := weekStartSunday("2026-02-09")
	if got != "2026-02-08" {
		t.Errorf("expected 2026-02-08, got %s", got)
	}
}

func TestWeekStartSunday_Saturday(t *testing.T) {
	// 2026-02-14 is a Saturday — week starts 2026-02-08.
	got := weekStartSunday("2026-02-14")
	if got != "2026-02-08" {
		t.Errorf("expected 2026-02-08, got %s", got)
	}
}

func TestWeekStartSunday_CrossesMonthBoundary(t *testing.T) {
	// 2026-03-04 is a Wednesday — week starts 2026-03-01 (Sunday).
	got := weekStartSunday("2026-03-04")
	if got != "2026-03-01" {
		t.Errorf("expected 2026-03-01, got %s", got)
	}
}

func TestWeekStartSunday_CrossesYearBoundary(t *testing.T) {
	// 2026-01-01 is a Thursday — week starts 2025-12-28 (Sunday).
	got := weekStartSunday("2026-01-01")
	if got != "2025-12-28" {
		t.Errorf("expected 2025-12-28, got %s", got)
	}
}

// --- roundHours ---

func TestRoundHours(t *testing.T) {
	cases := []struct {
		minutes int
		want    float64
	}{
		{60, 1.00},
		{90, 1.50},
		{35, 0.58},
		{0, 0.00},
		{1, 0.02},
	}
	for _, tc := range cases {
		got := roundHours(tc.minutes)
		if got != tc.want {
			t.Errorf("roundHours(%d) = %f, want %f", tc.minutes, got, tc.want)
		}
	}
}

// --- addDays ---

func TestAddDays_CrossesMonth(t *testing.T) {
	got := addDays("2026-01-31", 1)
	if got != "2026-02-01" {
		t.Errorf("expected 2026-02-01, got %s", got)
	}
}

func TestAddDays_CrossesYear(t *testing.T) {
	got := addDays("2025-12-31", 1)
	if got != "2026-01-01" {
		t.Errorf("expected 2026-01-01, got %s", got)
	}
}
