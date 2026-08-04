package service

import "time"

// isoWeekday returns the ISO weekday (1=Monday … 7=Sunday), matching the
// habits.target_days convention.
func isoWeekday(t time.Time) int32 {
	wd := int32(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

// computeStreak calculates the current and best streaks for one habit.
//
// logged is the set of check-in dates ("YYYY-MM-DD"); targetDays are the ISO
// weekdays the habit is scheduled for. Non-scheduled days never break a streak.
// Today counts toward the current streak when logged, but an unlogged today
// does not break it — the day is not over yet (product constitution: an
// unlogged day is neutral, not a failure).
func computeStreak(logged map[string]bool, targetDays []int32, today time.Time) (current, best int) {
	if len(logged) == 0 || len(targetDays) == 0 {
		return 0, 0
	}

	target := make(map[int32]bool, len(targetDays))
	for _, d := range targetDays {
		target[d] = true
	}

	// Find the earliest logged date to bound both walks.
	earliest := today
	for key := range logged {
		if d, err := time.Parse("2006-01-02", key); err == nil && d.Before(earliest) {
			earliest = d
		}
	}

	// Current streak: walk backwards from today.
	day := today
	if target[isoWeekday(day)] && logged[day.Format("2006-01-02")] {
		current++
	}
	for day = today.AddDate(0, 0, -1); !day.Before(earliest); day = day.AddDate(0, 0, -1) {
		if !target[isoWeekday(day)] {
			continue
		}
		if !logged[day.Format("2006-01-02")] {
			break
		}
		current++
	}

	// Best streak: walk forward over the whole window.
	run := 0
	for day = earliest; !day.After(today); day = day.AddDate(0, 0, 1) {
		if !target[isoWeekday(day)] {
			continue
		}
		if logged[day.Format("2006-01-02")] {
			run++
			if run > best {
				best = run
			}
		} else if !day.Equal(today) {
			run = 0
		}
	}

	return current, best
}

// completionRate returns logged scheduled days ÷ scheduled days over the
// window [from, to] (inclusive). Returns 0 when nothing was scheduled.
func completionRate(logged map[string]bool, targetDays []int32, from, to time.Time) float64 {
	target := make(map[int32]bool, len(targetDays))
	for _, d := range targetDays {
		target[d] = true
	}

	scheduled, done := 0, 0
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		if !target[isoWeekday(day)] {
			continue
		}
		scheduled++
		if logged[day.Format("2006-01-02")] {
			done++
		}
	}
	if scheduled == 0 {
		return 0
	}
	return float64(done) / float64(scheduled)
}
