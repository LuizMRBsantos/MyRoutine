package service

import (
	"testing"
	"time"
)

// Wednesday, 2026-08-05 — fixed so tests never depend on the real clock.
var testToday = time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)

func loggedOn(dates ...string) map[string]bool {
	m := make(map[string]bool, len(dates))
	for _, d := range dates {
		m[d] = true
	}
	return m
}

var everyDay = []int32{1, 2, 3, 4, 5, 6, 7}

func TestComputeStreakEmpty(t *testing.T) {
	current, best := computeStreak(map[string]bool{}, everyDay, testToday)
	if current != 0 || best != 0 {
		t.Fatalf("expected 0/0 for no logs, got %d/%d", current, best)
	}
}

func TestComputeStreakCountsToday(t *testing.T) {
	logs := loggedOn("2026-08-03", "2026-08-04", "2026-08-05")
	current, best := computeStreak(logs, everyDay, testToday)
	if current != 3 {
		t.Errorf("current streak = %d, want 3", current)
	}
	if best != 3 {
		t.Errorf("best streak = %d, want 3", best)
	}
}

// An unlogged today is a neutral state, not a failure: the day is not over,
// so it must not break the current streak (product constitution rule 4).
func TestComputeStreakUnloggedTodayDoesNotBreak(t *testing.T) {
	logs := loggedOn("2026-08-03", "2026-08-04")
	current, best := computeStreak(logs, everyDay, testToday)
	if current != 2 {
		t.Errorf("current streak = %d, want 2 (today still open)", current)
	}
	if best != 2 {
		t.Errorf("best streak = %d, want 2", best)
	}
}

func TestComputeStreakGapBreaksCurrent(t *testing.T) {
	// 08-01 and 08-02 logged, 08-03 missed, 08-04/08-05 logged
	logs := loggedOn("2026-08-01", "2026-08-02", "2026-08-04", "2026-08-05")
	current, best := computeStreak(logs, everyDay, testToday)
	if current != 2 {
		t.Errorf("current streak = %d, want 2 (gap on 08-03)", current)
	}
	if best != 2 {
		t.Errorf("best streak = %d, want 2", best)
	}
}

// Days the habit is not scheduled for must be skipped entirely — a weekday
// habit does not lose its streak over the weekend.
func TestComputeStreakSkipsUnscheduledDays(t *testing.T) {
	weekdays := []int32{1, 2, 3, 4, 5} // Mon–Fri
	// Thu 07-30, Fri 07-31 logged; Sat 08-01 and Sun 08-02 not scheduled;
	// Mon 08-03, Tue 08-04, Wed 08-05 logged.
	logs := loggedOn("2026-07-30", "2026-07-31", "2026-08-03", "2026-08-04", "2026-08-05")
	current, best := computeStreak(logs, weekdays, testToday)
	if current != 5 {
		t.Errorf("current streak = %d, want 5 (weekend does not break)", current)
	}
	if best != 5 {
		t.Errorf("best streak = %d, want 5", best)
	}
}

func TestComputeStreakBestExceedsCurrent(t *testing.T) {
	// A 4-day run in July, then a gap, then a 2-day run ending today.
	logs := loggedOn(
		"2026-07-20", "2026-07-21", "2026-07-22", "2026-07-23",
		"2026-08-04", "2026-08-05",
	)
	current, best := computeStreak(logs, everyDay, testToday)
	if current != 2 {
		t.Errorf("current streak = %d, want 2", current)
	}
	if best != 4 {
		t.Errorf("best streak = %d, want 4", best)
	}
}

func TestComputeStreakNoTargetDays(t *testing.T) {
	current, best := computeStreak(loggedOn("2026-08-05"), nil, testToday)
	if current != 0 || best != 0 {
		t.Fatalf("expected 0/0 when habit has no scheduled days, got %d/%d", current, best)
	}
}

func TestCompletionRate(t *testing.T) {
	from := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC) // Thursday
	logs := loggedOn("2026-07-30", "2026-08-01", "2026-08-03")

	// Every day scheduled: 7 days in window (07-30..08-05), 3 logged.
	got := completionRate(logs, everyDay, from, testToday)
	want := 3.0 / 7.0
	if got != want {
		t.Errorf("completion rate = %v, want %v", got, want)
	}
}

func TestCompletionRateNoScheduledDays(t *testing.T) {
	// Habit only scheduled on Sundays; window is Mon–Tue, so nothing is due.
	from := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	if got := completionRate(loggedOn("2026-08-03"), []int32{7}, from, to); got != 0 {
		t.Errorf("completion rate = %v, want 0 when nothing was scheduled", got)
	}
}

func TestIsoWeekday(t *testing.T) {
	cases := map[string]int32{
		"2026-08-03": 1, // Monday
		"2026-08-05": 3, // Wednesday
		"2026-08-08": 6, // Saturday
		"2026-08-09": 7, // Sunday
	}
	for date, want := range cases {
		d, err := time.Parse("2006-01-02", date)
		if err != nil {
			t.Fatalf("parsing %s: %v", date, err)
		}
		if got := isoWeekday(d); got != want {
			t.Errorf("isoWeekday(%s) = %d, want %d", date, got, want)
		}
	}
}
