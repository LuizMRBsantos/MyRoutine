package service

import (
	"context"
	"errors"
	"testing"
)

func TestWeeklyGoalsBelongToTheirWeek(t *testing.T) {
	svc := NewWeeklyGoalService(requireDB(t))
	userID := createTestUser(t)
	ctx := context.Background()

	// Any day of the week files the goal under that week's Monday.
	goal, err := svc.Create(ctx, userID, "Entregar o trabalho de BD", "2030-03-14") // Thursday
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if goal.Week != "2030-03-11" || goal.Done {
		t.Fatalf("goal = %+v, want week 2030-03-11 (Monday), not done", goal)
	}
	if _, err := svc.Create(ctx, userID, "Outra semana", "2030-03-18"); err != nil {
		t.Fatal(err)
	}

	got, err := svc.List(ctx, userID, "2030-03-17") // Sunday, same week
	if err != nil || len(got) != 1 || got[0].ID != goal.ID {
		t.Fatalf("list week of 2030-03-11 = %+v, err %v; want only the BD goal", got, err)
	}

	done, err := svc.SetDone(ctx, goal.ID, userID, true)
	if err != nil || !done.Done {
		t.Fatalf("set done: %+v, err %v", done, err)
	}
	undone, err := svc.SetDone(ctx, goal.ID, userID, false)
	if err != nil || undone.Done {
		t.Fatalf("undo: %+v, err %v", undone, err)
	}
}

func TestWeeklyGoalsAreValidatedAndPrivate(t *testing.T) {
	svc := NewWeeklyGoalService(requireDB(t))
	owner, other := createTestUser(t), createTestUser(t)
	ctx := context.Background()

	for _, title := range []string{"", "   "} {
		if _, err := svc.Create(ctx, owner, title, "2030-03-11"); !errors.Is(err, ErrInvalidWeeklyGoal) {
			t.Errorf("title %q: err = %v, want ErrInvalidWeeklyGoal", title, err)
		}
	}
	if _, err := svc.Create(ctx, owner, "x", "not-a-date"); !errors.Is(err, ErrInvalidWeeklyGoal) {
		t.Errorf("bad date: err = %v, want ErrInvalidWeeklyGoal", err)
	}

	goal, err := svc.Create(ctx, owner, "Correr no sábado", "2030-03-11")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetDone(ctx, goal.ID, other, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user marking done: err = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, goal.ID, other); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user deleting: err = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, goal.ID, owner); err != nil {
		t.Fatalf("owner deleting: %v", err)
	}
	if got, _ := svc.List(ctx, owner, "2030-03-11"); len(got) != 0 {
		t.Fatalf("after delete: %+v", got)
	}
}
