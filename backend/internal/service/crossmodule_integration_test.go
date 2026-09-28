package service

import (
	"context"
	"testing"
	"time"

	"github.com/myroutine/backend/internal/appctx"
)

// Health is a consumer module: a workout logged as a habit check-in must show
// up in the Health tab by reference, with no second row anywhere
// (product constitution rule 5, single source of truth per event).
func TestHealthReadsHabitLogsByReference(t *testing.T) {
	healthSvc := NewHealthService(requireDB(t), testLogger)
	habitSvc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	habit, err := habitSvc.Create(ctx, userID, CreateHabitInput{
		Name: "Correr", Icon: "🏃", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
		Category: "health", CheckType: "metric",
		MetricConfig: []MetricField{{Key: "km", Label: "Distância", Unit: "km"}},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	if _, err := habitSvc.CheckIn(ctx, habit.ID, userID, CheckInInput{
		Metrics: map[string]interface{}{"km": 8.2, "rpe": 6, "time_min": 45},
	}); err != nil {
		t.Fatalf("check-in: %v", err)
	}

	activities, err := healthSvc.ListActivities(ctx, userID, userToday(ctx), userToday(ctx))
	if err != nil {
		t.Fatalf("listing activities: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("activity count = %d, want 1", len(activities))
	}
	if activities[0].HabitName != "Correr" {
		t.Errorf("habit_name = %q, want Correr", activities[0].HabitName)
	}
	if activities[0].Metrics["km"] != 8.2 {
		t.Errorf("km = %v, want 8.2", activities[0].Metrics["km"])
	}

	// The event exists exactly once — Health kept no copy.
	var logCount int
	if err := requireDB(t).QueryRow(ctx,
		"SELECT COUNT(*) FROM habit_logs WHERE user_id = $1", userID,
	).Scan(&logCount); err != nil {
		t.Fatalf("counting logs: %v", err)
	}
	if logCount != 1 {
		t.Errorf("habit_logs rows = %d, want 1 (Health must not duplicate events)", logCount)
	}

	summary, err := healthSvc.GetSummary(ctx, userID, 2)
	if err != nil {
		t.Fatalf("health summary: %v", err)
	}
	if len(summary.Weeks) != 1 {
		t.Fatalf("week count = %d, want 1", len(summary.Weeks))
	}
	if summary.Weeks[0].TotalKm != 8.2 {
		t.Errorf("total_km = %v, want 8.2", summary.Weeks[0].TotalKm)
	}
	if summary.Weeks[0].AvgRPE != 6 {
		t.Errorf("avg_rpe = %v, want 6", summary.Weeks[0].AvgRPE)
	}
}

// Habits outside the health category must not leak into the Health tab.
func TestHealthIgnoresNonHealthHabits(t *testing.T) {
	healthSvc := NewHealthService(requireDB(t), testLogger)
	habitSvc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	habit, err := habitSvc.Create(ctx, userID, CreateHabitInput{
		Name: "Ler", Icon: "📚", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
		Category: "general",
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}
	if _, err := habitSvc.CheckIn(ctx, habit.ID, userID, CheckInInput{}); err != nil {
		t.Fatalf("check-in: %v", err)
	}

	activities, err := healthSvc.ListActivities(ctx, userID, userToday(ctx), userToday(ctx))
	if err != nil {
		t.Fatalf("listing activities: %v", err)
	}
	if len(activities) != 0 {
		t.Errorf("activity count = %d, want 0 for a general habit", len(activities))
	}
}

func TestBodyMetricUpsertPerDay(t *testing.T) {
	healthSvc := NewHealthService(requireDB(t), testLogger)
	userID := createTestUser(t)
	ctx := context.Background()

	first, err := healthSvc.UpsertBodyMetric(ctx, userID, today(), floatPtr(72.5), nil)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	second, err := healthSvc.UpsertBodyMetric(ctx, userID, today(), floatPtr(72.1), nil)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if second.ID != first.ID {
		t.Error("re-recording the same day created a second measurement")
	}
	if second.WeightKg == nil || *second.WeightKg != 72.1 {
		t.Errorf("weight = %v, want 72.1", second.WeightKg)
	}

	metrics, err := healthSvc.ListBodyMetrics(ctx, userID, 10)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(metrics) != 1 {
		t.Errorf("measurement count = %d, want 1 per day", len(metrics))
	}
}

// A study session is its own source of truth; linking a habit creates a
// check-in that references the session instead of copying it.
func TestStudySessionCreatesReferencedCheckIn(t *testing.T) {
	studySvc := NewStudyService(requireDB(t), testLogger)
	habitSvc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	habit, err := habitSvc.Create(ctx, userID, CreateHabitInput{
		Name: "Estudar OAC", Icon: "📚", Color: "#AF52DE",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
		Category: "study", CheckType: "timed", TimerMinutes: intPtr(30),
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	session, err := studySvc.CreateSession(ctx, userID, CreateStudySessionInput{
		Subject:         "OAC",
		Topic:           strPtr("Pipeline"),
		DurationMinutes: 50,
		HabitID:         &habit.ID,
	})
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}

	// The session had no date, so it (and its check-in) landed on the user's
	// local day — query that day, not the server's.
	logs, err := habitSvc.GetLogs(ctx, habit.ID, userID, userToday(ctx), userToday(ctx))
	if err != nil {
		t.Fatalf("getting logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(logs))
	}
	if logs[0].SourceType != "study_session" {
		t.Errorf("source_type = %q, want study_session", logs[0].SourceType)
	}
	if logs[0].TimerSeconds == nil || *logs[0].TimerSeconds != 3000 {
		t.Errorf("timer_seconds = %v, want 3000 (50min)", logs[0].TimerSeconds)
	}

	summary, err := studySvc.GetSummary(ctx, userID)
	if err != nil {
		t.Fatalf("study summary: %v", err)
	}
	if summary.TotalMinutes30d != 50 || summary.Sessions30d != 1 {
		t.Errorf("summary = %d min / %d sessions, want 50/1", summary.TotalMinutes30d, summary.Sessions30d)
	}

	// Deleting the session leaves the habit log alone: the check-in already
	// happened, and habit_logs is its own source of truth for that day.
	if err := studySvc.DeleteSession(ctx, session.ID, userID); err != nil {
		t.Fatalf("deleting session: %v", err)
	}
	logs, err = habitSvc.GetLogs(ctx, habit.ID, userID, userToday(ctx), userToday(ctx))
	if err != nil {
		t.Fatalf("getting logs after delete: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("log count = %d, want the check-in preserved", len(logs))
	}
}

// Without a habit_id the session is recorded on its own — no check-in.
func TestStudySessionWithoutHabit(t *testing.T) {
	studySvc := NewStudyService(requireDB(t), testLogger)
	userID := createTestUser(t)
	ctx := context.Background()

	if _, err := studySvc.CreateSession(ctx, userID, CreateStudySessionInput{
		Subject: "Cálculo", DurationMinutes: 90,
	}); err != nil {
		t.Fatalf("creating session: %v", err)
	}

	var logCount int
	if err := requireDB(t).QueryRow(ctx,
		"SELECT COUNT(*) FROM habit_logs WHERE user_id = $1", userID,
	).Scan(&logCount); err != nil {
		t.Fatalf("counting logs: %v", err)
	}
	if logCount != 0 {
		t.Errorf("habit_logs rows = %d, want 0 when no habit is linked", logCount)
	}

	sessions, err := studySvc.ListSessions(ctx, userID, userToday(ctx), userToday(ctx), "")
	if err != nil {
		t.Fatalf("listing sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("session count = %d, want 1", len(sessions))
	}
}

func TestStudySessionDeleteNotFound(t *testing.T) {
	studySvc := NewStudyService(requireDB(t), testLogger)
	userID := createTestUser(t)

	err := studySvc.DeleteSession(context.Background(),
		"00000000-0000-0000-0000-000000000000", userID)
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// The weekly window is anchored on the user's local week: with today =
// Tuesday 2026-03-10 and weeks=1, only Monday 2026-03-09 onward counts.
func TestHealthSummaryUsesUserLocalWeek(t *testing.T) {
	healthSvc := NewHealthService(requireDB(t), testLogger)
	habitSvc := newHabitService(t)
	userID := createTestUser(t)
	ctx, loc := saoPauloCtx(t)

	habit, err := habitSvc.Create(ctx, userID, CreateHabitInput{
		Name: "Correr", Icon: "🏃", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
		Category: "health", CheckType: "metric",
		MetricConfig: []MetricField{{Key: "km", Label: "Distância", Unit: "km"}},
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}
	for date, km := range map[string]float64{"2026-03-08": 3, "2026-03-09": 5} {
		if _, err := habitSvc.CheckIn(ctx, habit.ID, userID, CheckInInput{
			Date: date, Metrics: map[string]interface{}{"km": km},
		}); err != nil {
			t.Fatalf("check-in %s: %v", date, err)
		}
	}

	summary, err := healthSvc.getSummary(ctx, userID, 1, localMidnight(t, "2026-03-10", loc))
	if err != nil {
		t.Fatalf("health summary: %v", err)
	}
	if len(summary.Weeks) != 1 {
		t.Fatalf("weeks = %+v, want exactly the week of 2026-03-09", summary.Weeks)
	}
	if summary.Weeks[0].WeekStart != "2026-03-09" || summary.Weeks[0].TotalKm != 5 {
		t.Errorf("week = %+v, want start 2026-03-09 with 5 km", summary.Weeks[0])
	}
}

// With no date, a body measurement lands on the user's local today. The zone
// is picked so its calendar day differs from the server's, whatever the time.
func TestBodyMetricDefaultsToUserLocalDay(t *testing.T) {
	healthSvc := NewHealthService(requireDB(t), testLogger)
	userID := createTestUser(t)

	serverDay := time.Now().Format("2006-01-02")
	var ctx context.Context
	for _, name := range []string{"Etc/GMT-14", "Etc/GMT+12"} { // UTC+14, UTC-12: 26h apart
		loc, err := time.LoadLocation(name)
		if err != nil {
			t.Fatalf("loading %s: %v", name, err)
		}
		if c := appctx.WithTimezone(context.Background(), loc); userToday(c) != serverDay {
			ctx = c
			break
		}
	}
	if ctx == nil {
		t.Fatal("no candidate timezone differs from the server day")
	}

	m, err := healthSvc.UpsertBodyMetric(ctx, userID, "", floatPtr(70), nil)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if m.MeasuredOn != userToday(ctx) {
		t.Errorf("measured_on = %s, want the user's local day %s", m.MeasuredOn, userToday(ctx))
	}
}
