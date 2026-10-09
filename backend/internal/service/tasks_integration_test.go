package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func newTaskService(t *testing.T) *TaskService {
	t.Helper()
	return NewTaskService(requireDB(t))
}

func patchFields(t *testing.T, body string) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &fields); err != nil {
		t.Fatalf("building patch body: %v", err)
	}
	return fields
}

// PATCH must be a real patch: absent fields are kept, an explicit null clears
// a nullable column, and the date can move (drag-and-drop between days).
func TestTaskUpdateMovesDateAndClearsFields(t *testing.T) {
	svc := newTaskService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, userID, CreateTaskInput{
		Title:     "Treino de corrida",
		Date:      today(),
		StartTime: strPtr("06:30"),
		Category:  "exercise",
		Notes:     strPtr("levar garrafa"),
	})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}
	if created.StartTime == nil || *created.StartTime != "06:30" {
		t.Fatalf("start_time = %v, want 06:30 (HH:MM, not HH:MM:SS)", created.StartTime)
	}

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	updated, err := svc.Update(ctx, created.ID, userID,
		patchFields(t, `{"date":"`+tomorrow+`","start_time":null}`))
	if err != nil {
		t.Fatalf("updating task: %v", err)
	}

	if updated.Date != tomorrow {
		t.Errorf("date = %q, want %q (date was not updatable before)", updated.Date, tomorrow)
	}
	if updated.StartTime != nil {
		t.Errorf("start_time = %v, want nil (explicit null must clear it)", *updated.StartTime)
	}
	if updated.Title != "Treino de corrida" {
		t.Errorf("title = %q, absent fields must be preserved", updated.Title)
	}
	if updated.Notes == nil || *updated.Notes != "levar garrafa" {
		t.Error("notes should have been preserved")
	}
}

func TestTaskUpdateNotFound(t *testing.T) {
	svc := newTaskService(t)
	userID := createTestUser(t)

	_, err := svc.Update(context.Background(),
		"00000000-0000-0000-0000-000000000000", userID,
		patchFields(t, `{"title":"x"}`))
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound (was returning 500)", err)
	}
}

func TestTaskDeleteNotFound(t *testing.T) {
	svc := newTaskService(t)
	userID := createTestUser(t)

	err := svc.Delete(context.Background(), "00000000-0000-0000-0000-000000000000", userID)
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// Advancing a linked task to 'done' must create the habit check-in,
// referencing the task as its source (cross-module-data-flow skill).
func TestTaskAdvanceCreatesLinkedHabitCheckIn(t *testing.T) {
	taskSvc := newTaskService(t)
	habitSvc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	habit, err := habitSvc.Create(ctx, userID, CreateHabitInput{
		Name: "Correr", Icon: "🏃", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7}, Category: "health",
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	task, err := taskSvc.Create(ctx, userID, CreateTaskInput{
		Title:         "Treino",
		Date:          today(),
		Category:      "exercise",
		LinkedHabitID: &habit.ID,
	})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	// planned → in_progress (no check-in yet)
	if _, err := taskSvc.AdvanceStatus(ctx, task.ID, userID); err != nil {
		t.Fatalf("first advance: %v", err)
	}
	logs, err := habitSvc.GetLogs(ctx, habit.ID, userID, today(), today())
	if err != nil {
		t.Fatalf("getting logs: %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("check-in created too early: task is only in_progress")
	}

	// in_progress → done (check-in expected)
	advanced, err := taskSvc.AdvanceStatus(ctx, task.ID, userID)
	if err != nil {
		t.Fatalf("second advance: %v", err)
	}
	if advanced.Status != "done" {
		t.Fatalf("status = %q, want done", advanced.Status)
	}

	logs, err = habitSvc.GetLogs(ctx, habit.ID, userID, today(), today())
	if err != nil {
		t.Fatalf("getting logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("log count = %d, want 1 (linked_habit_id was a dead feature)", len(logs))
	}
	if logs[0].SourceType != "task" {
		t.Errorf("source_type = %q, want task", logs[0].SourceType)
	}
}

// A manual check-in already made by the user is the source of truth and must
// not be overwritten when the linked task is completed later.
func TestTaskAdvanceDoesNotOverwriteManualCheckIn(t *testing.T) {
	taskSvc := newTaskService(t)
	habitSvc := newHabitService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	habit, err := habitSvc.Create(ctx, userID, CreateHabitInput{
		Name: "Correr", Icon: "🏃", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7}, Category: "health",
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}

	manual, err := habitSvc.CheckIn(ctx, habit.ID, userID, CheckInInput{Notes: "corri cedo"})
	if err != nil {
		t.Fatalf("manual check-in: %v", err)
	}

	task, err := taskSvc.Create(ctx, userID, CreateTaskInput{
		// The manual check-in above had no date, so it landed on the user's
		// local day; the task must target that same day.
		Title: "Treino", Date: userToday(ctx), LinkedHabitID: &habit.ID,
	})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := taskSvc.AdvanceStatus(ctx, task.ID, userID); err != nil {
			t.Fatalf("advance %d: %v", i, err)
		}
	}

	logs, err := habitSvc.GetLogs(ctx, habit.ID, userID, userToday(ctx), userToday(ctx))
	if err != nil {
		t.Fatalf("getting logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("log count = %d, want 1 (no duplicate)", len(logs))
	}
	if logs[0].ID != manual.ID || logs[0].Notes != "corri cedo" {
		t.Error("the manual check-in was overwritten by the task")
	}
	if logs[0].SourceType != "manual" {
		t.Errorf("source_type = %q, want manual to be preserved", logs[0].SourceType)
	}
}

func TestTaskListByDateIsolatesUsers(t *testing.T) {
	svc := newTaskService(t)
	owner := createTestUser(t)
	other := createTestUser(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, owner, CreateTaskInput{Title: "Privada", Date: today()}); err != nil {
		t.Fatalf("creating task: %v", err)
	}

	tasks, err := svc.ListByDate(ctx, other, today())
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("task count = %d, want 0 for another user", len(tasks))
	}
}

func TestMonthlyGoalLifecycle(t *testing.T) {
	svc := newTaskService(t)
	userID := createTestUser(t)
	ctx := context.Background()
	month := time.Now().Format("2006-01") + "-01"

	goal, err := svc.CreateGoal(ctx, userID, "Correr 100km", month, nil, strPtr("#34C759"))
	if err != nil {
		t.Fatalf("creating goal: %v", err)
	}
	if goal.Status != "active" {
		t.Errorf("status = %q, want active", goal.Status)
	}

	updated, err := svc.UpdateGoalStatus(ctx, goal.ID, userID, "done")
	if err != nil {
		t.Fatalf("updating goal: %v", err)
	}
	if updated.Status != "done" {
		t.Errorf("status = %q, want done", updated.Status)
	}

	goals, err := svc.ListGoalsByMonth(ctx, userID, month)
	if err != nil {
		t.Fatalf("listing goals: %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("goal count = %d, want 1", len(goals))
	}

	if err := svc.DeleteGoal(ctx, goal.ID, userID); err != nil {
		t.Fatalf("deleting goal: %v", err)
	}
	if err := svc.DeleteGoal(ctx, goal.ID, userID); err != ErrNotFound {
		t.Errorf("second delete err = %v, want ErrNotFound", err)
	}
}

func TestMonthlyGoalStatusNotFound(t *testing.T) {
	svc := newTaskService(t)
	userID := createTestUser(t)

	_, err := svc.UpdateGoalStatus(context.Background(),
		"00000000-0000-0000-0000-000000000000", userID, "done")
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// "Me avisar antes": on by default for appointments, exams and work
// (meetings); off for classes, exercise, waking up and "other". Either can be
// changed per task.
func TestTaskNotifyDefaultsByCategoryAndCanChange(t *testing.T) {
	svc := newTaskService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	for category, want := range map[string]bool{
		"appointment": true, "exam": true, "work": true,
		"study": false, "exercise": false, "wake_up": false, "other": false, "": false,
	} {
		task, err := svc.Create(ctx, userID, CreateTaskInput{Title: "x", Date: today(), StartTime: strPtr("10:00"), Category: category})
		if err != nil {
			t.Fatalf("%q: %v", category, err)
		}
		if task.Notify != want {
			t.Errorf("category %q: notify = %v, want %v", category, task.Notify, want)
		}
	}

	off := false
	exam, err := svc.Create(ctx, userID, CreateTaskInput{Title: "Prova", Date: today(), Category: "exam", Notify: &off})
	if err != nil || exam.Notify {
		t.Fatalf("explicit notify=false on an exam: %+v, err %v", exam, err)
	}
	updated, err := svc.Update(ctx, exam.ID, userID, patchFields(t, `{"notify":true}`))
	if err != nil || !updated.Notify {
		t.Fatalf("update notify=true: %+v, err %v", updated, err)
	}
	if _, err := svc.Update(ctx, exam.ID, userID, patchFields(t, `{"notify":"yes"}`)); err == nil {
		t.Fatal("notify must be a boolean")
	}
}

// task_details (the subject of a class or exam) goes to a jsonb column. In
// production (simple protocol) a []byte parameter used to reach Postgres as
// bytea and every task with details failed with 500.
func TestTaskDetailsSurviveCreateAndUpdate(t *testing.T) {
	svc := newTaskService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	exam, err := svc.Create(ctx, userID, CreateTaskInput{
		Title: "Prova - C2", Date: today(), StartTime: strPtr("08:00"), Category: "exam", Priority: "high",
		TaskDetails: json.RawMessage(`{"subject":"C2"}`),
	})
	if err != nil {
		t.Fatalf("creating exam with details: %v", err)
	}
	if string(exam.TaskDetails) != `{"subject": "C2"}` {
		t.Fatalf("details = %s, want the subject back", exam.TaskDetails)
	}
	updated, err := svc.Update(ctx, exam.ID, userID, patchFields(t, `{"task_details":{"subject":"Cálculo 2"}}`))
	if err != nil || !strings.Contains(string(updated.TaskDetails), "Cálculo 2") {
		t.Fatalf("updating details: %s, err %v", updated.TaskDetails, err)
	}
}
