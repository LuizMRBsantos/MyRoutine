package service

import (
	"context"
	"testing"
)

func newHabitService(t *testing.T) *HabitService {
	t.Helper()
	return NewHabitService(requireDB(t), testLogger)
}

func floatPtr(v float64) *float64 { return &v }
func intPtr(v int) *int           { return &v }
func strPtr(v string) *string     { return &v }

// A partial update must never silently drop the check-type configuration —
// this was the bug where PUT /habits reset time_of_day/check_type/metric_config.
func TestHabitUpdatePreservesCheckTypeConfig(t *testing.T) {
	svc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, userID, CreateHabitInput{
		Name:       "Correr",
		Icon:       "🏃",
		Color:      "#0071E3",
		Frequency:  "daily",
		TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
		TimeOfDay:  "morning",
		Category:   "health",
		CheckType:  "metric",
		MetricConfig: []MetricField{
			{Key: "km", Label: "Distância", Unit: "km", IsTarget: true, TargetValue: floatPtr(5)},
		},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	// Update only the name — everything else must survive.
	updated, err := svc.Update(ctx, created.ID, userID, UpdateHabitInput{
		Name: strPtr("Correr de manhã"),
	})
	if err != nil {
		t.Fatalf("updating habit: %v", err)
	}

	if updated.Name != "Correr de manhã" {
		t.Errorf("name = %q, want %q", updated.Name, "Correr de manhã")
	}
	if updated.TimeOfDay != "morning" {
		t.Errorf("time_of_day = %q, want morning (dropped by partial update)", updated.TimeOfDay)
	}
	if updated.CheckType != "metric" {
		t.Errorf("check_type = %q, want metric", updated.CheckType)
	}
	if updated.Category != "health" {
		t.Errorf("category = %q, want health", updated.Category)
	}
	if len(updated.MetricConfig) != 1 {
		t.Fatalf("metric_config length = %d, want 1", len(updated.MetricConfig))
	}
	if !updated.MetricConfig[0].IsTarget || updated.MetricConfig[0].TargetValue == nil {
		t.Error("is_target/target_value were dropped — the frontend sends them")
	}
}

// Switching check type replaces the dependent config as a unit, so a metric
// habit turned into a timed one must not keep stale metric fields.
func TestHabitUpdateSwitchesCheckType(t *testing.T) {
	svc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, userID, CreateHabitInput{
		Name: "Meditar", Icon: "🧘", Color: "#AF52DE", Frequency: "daily",
		TargetDays: []int32{1, 2, 3, 4, 5, 6, 7}, CheckType: "metric",
		MetricConfig: []MetricField{{Key: "min", Label: "Minutos", Unit: "min"}},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	updated, err := svc.Update(ctx, created.ID, userID, UpdateHabitInput{
		CheckType:    strPtr("timed"),
		TimerMinutes: intPtr(20),
	})
	if err != nil {
		t.Fatalf("updating habit: %v", err)
	}

	if updated.CheckType != "timed" {
		t.Fatalf("check_type = %q, want timed", updated.CheckType)
	}
	if updated.TimerMinutes == nil || *updated.TimerMinutes != 20 {
		t.Error("timer_minutes was not persisted")
	}
	if len(updated.MetricConfig) != 0 {
		t.Error("metric_config should be cleared when leaving check_type=metric")
	}
}

func TestHabitUpdateNotFound(t *testing.T) {
	svc := newHabitService(t)
	userID := createTestUser(t)

	_, err := svc.Update(context.Background(),
		"00000000-0000-0000-0000-000000000000", userID,
		UpdateHabitInput{Name: strPtr("x")})
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// A habit belonging to someone else must be invisible, not merely unwritable.
func TestHabitGetByIDIsolatesUsers(t *testing.T) {
	svc := newHabitService(t)
	owner := createTestUser(t)
	other := createTestUser(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, owner, CreateHabitInput{
		Name: "Privado", Icon: "⭐", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	if _, err := svc.GetByID(ctx, created.ID, other); err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound for another user's habit", err)
	}
}

func TestHabitCheckInAndStreak(t *testing.T) {
	svc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, userID, CreateHabitInput{
		Name: "Ler", Icon: "📚", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	log, err := svc.CheckIn(ctx, created.ID, userID, CheckInInput{Notes: "20 páginas"})
	if err != nil {
		t.Fatalf("check-in: %v", err)
	}
	if log.SourceType != "manual" {
		t.Errorf("source_type = %q, want manual by default", log.SourceType)
	}

	habits, err := svc.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("listing habits: %v", err)
	}
	if len(habits) != 1 {
		t.Fatalf("habit count = %d, want 1", len(habits))
	}
	if !habits[0].CompletedToday {
		t.Error("completed_today should be true after check-in")
	}
	if habits[0].CurrentStreak != 1 {
		t.Errorf("current_streak = %d, want 1", habits[0].CurrentStreak)
	}

	stats, err := svc.GetStats(ctx, userID)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalCheckIns != 1 || stats.CompletedToday != 1 {
		t.Errorf("stats totals = %d check-ins / %d today, want 1/1", stats.TotalCheckIns, stats.CompletedToday)
	}
	if stats.CurrentStreak != 1 || stats.BestStreak != 1 {
		t.Errorf("stats streaks = %d/%d, want 1/1 (were hardcoded to 0)", stats.CurrentStreak, stats.BestStreak)
	}
	if len(stats.HabitStats) != 1 {
		t.Errorf("habit_stats length = %d, want 1 (was never populated)", len(stats.HabitStats))
	}
}

// Re-checking the same day updates the existing log instead of failing on the
// unique constraint, and undo removes it.
func TestHabitCheckInIsIdempotentPerDay(t *testing.T) {
	svc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, userID, CreateHabitInput{
		Name: "Água", Icon: "💧", Color: "#5AC8FA",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	first, err := svc.CheckIn(ctx, created.ID, userID, CheckInInput{Notes: "1L"})
	if err != nil {
		t.Fatalf("first check-in: %v", err)
	}
	second, err := svc.CheckIn(ctx, created.ID, userID, CheckInInput{Notes: "2L"})
	if err != nil {
		t.Fatalf("second check-in: %v", err)
	}
	if first.ID != second.ID {
		t.Error("second check-in on the same day created a new log")
	}
	if second.Notes != "2L" {
		t.Errorf("notes = %q, want the updated value", second.Notes)
	}

	if err := svc.UndoCheckIn(ctx, created.ID, userID, today()); err != nil {
		t.Fatalf("undo: %v", err)
	}
	// Undoing again has nothing to delete → 404, not a silent success.
	if err := svc.UndoCheckIn(ctx, created.ID, userID, today()); err != ErrNotFound {
		t.Errorf("second undo err = %v, want ErrNotFound", err)
	}
}

func TestHabitDeleteIsSoftAndReports404(t *testing.T) {
	svc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, userID, CreateHabitInput{
		Name: "Temporário", Icon: "⭐", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	if err := svc.Delete(ctx, created.ID, userID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := svc.Delete(ctx, created.ID, userID); err != ErrNotFound {
		t.Errorf("second delete err = %v, want ErrNotFound", err)
	}

	habits, err := svc.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(habits) != 0 {
		t.Errorf("habit count = %d, want 0 after soft delete", len(habits))
	}
}

// A timed habit rejects a check-in that did not reach the configured minimum.
func TestHabitCheckInValidatesTimer(t *testing.T) {
	svc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, userID, CreateHabitInput{
		Name: "Foco", Icon: "🧠", Color: "#5856D6",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
		CheckType: "timed", TimerMinutes: intPtr(30),
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	if _, err := svc.CheckIn(ctx, created.ID, userID, CheckInInput{TimerSeconds: intPtr(600)}); err == nil {
		t.Error("expected error for a 10min check-in on a 30min habit")
	}
	if _, err := svc.CheckIn(ctx, created.ID, userID, CheckInInput{TimerSeconds: intPtr(1800)}); err != nil {
		t.Errorf("30min check-in should be accepted, got %v", err)
	}
}

// Metrics are optional enrichment and must never block the check-in
// (non-binary-habit-modeling skill).
func TestHabitMetricCheckInWithoutValues(t *testing.T) {
	svc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, userID, CreateHabitInput{
		Name: "Pedalar", Icon: "🚴", Color: "#34C759",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
		CheckType:    "metric",
		MetricConfig: []MetricField{{Key: "km", Label: "Distância", Unit: "km"}},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	if _, err := svc.CheckIn(ctx, created.ID, userID, CheckInInput{}); err != nil {
		t.Errorf("metric habit must accept a check-in with no metrics, got %v", err)
	}
}
