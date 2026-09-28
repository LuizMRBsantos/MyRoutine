package service

import (
	"context"
	"testing"
	"time"

	"github.com/myroutine/backend/internal/appctx"
)

func newReviewService(t *testing.T) *ReviewService {
	t.Helper()
	return NewReviewService(requireDB(t), testLogger)
}

// yesterday is relative to the user's local today (the default timezone for a
// bare context), matching what GetMissedDays and CheckIn use.
func yesterday() string {
	return appctx.Today(context.Background()).AddDate(0, 0, -1).Format("2006-01-02")
}

// The IDOR fix: reviewing a habit that belongs to someone else must fail,
// not silently write a row pointing at their habit.
func TestReviewDayRejectsOtherUsersHabit(t *testing.T) {
	reviewSvc := newReviewService(t)
	habitSvc := newHabitService(t)
	owner := createTestUser(t)
	attacker := createTestUser(t)
	ctx := context.Background()

	habit, err := habitSvc.Create(ctx, owner, CreateHabitInput{
		Name: "Privado", Icon: "⭐", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	_, err = reviewSvc.ReviewDay(ctx, habit.ID, attacker, yesterday(), "discarded")
	if err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound — another user must not review this habit", err)
	}

	// And nothing was written.
	var count int
	if err := requireDB(t).QueryRow(ctx,
		"SELECT COUNT(*) FROM habit_day_reviews WHERE habit_id = $1", habit.ID,
	).Scan(&count); err != nil {
		t.Fatalf("counting reviews: %v", err)
	}
	if count != 0 {
		t.Errorf("review rows = %d, want 0", count)
	}
}

func TestReviewDayUpsertsDecision(t *testing.T) {
	reviewSvc := newReviewService(t)
	habitSvc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	habit, err := habitSvc.Create(ctx, userID, CreateHabitInput{
		Name: "Ler", Icon: "📚", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	first, err := reviewSvc.ReviewDay(ctx, habit.ID, userID, yesterday(), "migrated")
	if err != nil {
		t.Fatalf("first review: %v", err)
	}
	if first.Status != "migrated" {
		t.Errorf("status = %q, want migrated", first.Status)
	}

	// Changing one's mind updates the same row rather than failing on the
	// (habit_id, review_date) unique constraint.
	second, err := reviewSvc.ReviewDay(ctx, habit.ID, userID, yesterday(), "discarded")
	if err != nil {
		t.Fatalf("second review: %v", err)
	}
	if second.ID != first.ID {
		t.Error("re-reviewing the same day created a second row")
	}
	if second.Status != "discarded" {
		t.Errorf("status = %q, want discarded", second.Status)
	}
}

// A missed day is a scheduled day with no log. Days the habit is not
// scheduled for must never show up as missed.
func TestGetMissedDaysOnlyScheduledDays(t *testing.T) {
	reviewSvc := newReviewService(t)
	habitSvc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	// Scheduled only on the ISO weekday of yesterday.
	y, err := time.Parse("2006-01-02", yesterday())
	if err != nil {
		t.Fatalf("parsing yesterday: %v", err)
	}
	if _, err := habitSvc.Create(ctx, userID, CreateHabitInput{
		Name: "Semanal", Icon: "⭐", Color: "#0071E3",
		Frequency: "custom", TargetDays: []int32{isoWeekday(y)},
	}); err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	missed, err := reviewSvc.GetMissedDays(ctx, userID)
	if err != nil {
		t.Fatalf("getting missed days: %v", err)
	}

	// Exactly one missed day in the 7-day window: yesterday.
	if len(missed) != 1 {
		t.Fatalf("missed count = %d, want 1 (only the scheduled weekday)", len(missed))
	}
	if missed[0].Date != yesterday() {
		t.Errorf("missed date = %q, want %q", missed[0].Date, yesterday())
	}
	if missed[0].Review != nil {
		t.Error("day should have no review yet")
	}
}

func TestGetMissedDaysExcludesLoggedDays(t *testing.T) {
	reviewSvc := newReviewService(t)
	habitSvc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	habit, err := habitSvc.Create(ctx, userID, CreateHabitInput{
		Name: "Diário", Icon: "⭐", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	before, err := reviewSvc.GetMissedDays(ctx, userID)
	if err != nil {
		t.Fatalf("getting missed days: %v", err)
	}
	if len(before) != 7 {
		t.Fatalf("missed count = %d, want 7 (the whole window, today excluded)", len(before))
	}

	if _, err := habitSvc.CheckIn(ctx, habit.ID, userID, CheckInInput{Date: yesterday()}); err != nil {
		t.Fatalf("check-in: %v", err)
	}

	after, err := reviewSvc.GetMissedDays(ctx, userID)
	if err != nil {
		t.Fatalf("getting missed days: %v", err)
	}
	if len(after) != 6 {
		t.Errorf("missed count = %d, want 6 after logging yesterday", len(after))
	}
	for _, m := range after {
		if m.Date == yesterday() {
			t.Error("a logged day must not appear as missed")
		}
	}
}

// The 7-day window ends on the user's local yesterday, not the DB server's
// CURRENT_DATE.
func TestGetMissedDaysUsesUserLocalDay(t *testing.T) {
	reviewSvc := newReviewService(t)
	habitSvc := newHabitService(t)
	userID := createTestUser(t)
	ctx, loc := saoPauloCtx(t)

	habit, err := habitSvc.Create(ctx, userID, CreateHabitInput{
		Name: "Diário", Icon: "⭐", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}
	if _, err := habitSvc.CheckIn(ctx, habit.ID, userID, CheckInInput{Date: "2026-03-05"}); err != nil {
		t.Fatalf("check-in: %v", err)
	}

	missed, err := reviewSvc.getMissedDays(ctx, userID, localMidnight(t, "2026-03-10", loc))
	if err != nil {
		t.Fatalf("getting missed days: %v", err)
	}
	want := []string{"2026-03-09", "2026-03-08", "2026-03-07", "2026-03-06", "2026-03-04", "2026-03-03"}
	if len(missed) != len(want) {
		t.Fatalf("missed = %+v, want dates %v", missed, want)
	}
	for i, m := range missed {
		if m.Date != want[i] {
			t.Errorf("missed[%d] = %s, want %s", i, m.Date, want[i])
		}
	}
}
