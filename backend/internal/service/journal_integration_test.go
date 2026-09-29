package service

import (
	"context"
	"errors"
	"testing"
)

// Track Day: the journal is the source; expenses and workouts registered from
// it live in their own modules and point back with source_type='track_day'.

const journalDay = "2026-03-10"

func newJournal(t *testing.T) *JournalService {
	t.Helper()
	return NewJournalService(requireDB(t), testLogger)
}

func createHealthHabit(t *testing.T, userID, name string) string {
	t.Helper()
	var id string
	if err := requireDB(t).QueryRow(context.Background(),
		`INSERT INTO habits (user_id, name, category) VALUES ($1, $2, 'health') RETURNING id::text`,
		userID, name,
	).Scan(&id); err != nil {
		t.Fatalf("create habit: %v", err)
	}
	return id
}

func TestJournalSaveAndLineIDs(t *testing.T) {
	j := newJournal(t)
	userID := createTestUser(t)
	ctx := context.Background()

	empty, err := j.Get(ctx, userID, journalDay)
	if err != nil || empty.Content != "" || len(empty.LineIDs) != 0 || empty.UpdatedAt != nil {
		t.Fatalf("empty day = %+v, err = %v", empty, err)
	}

	got, err := j.Save(ctx, userID, journalDay, "• Comprar pão\n$ 50 Almoço #alimentacao\n\n")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if got.Content != "• Comprar pão\n$ 50 Almoço #alimentacao\n\n" || got.UpdatedAt == nil {
		t.Fatalf("saved = %+v", got)
	}
	if len(got.LineIDs) != 2 {
		t.Fatalf("line ids = %v, want 2 (blank lines ignored)", got.LineIDs)
	}

	// Spacing does not change a line's identity; its text does.
	respaced, _ := j.Save(ctx, userID, journalDay, "  $  50   Almoço #alimentacao ")
	if respaced.LineIDs["$ 50 Almoço #alimentacao"] != got.LineIDs["$ 50 Almoço #alimentacao"] {
		t.Fatal("extra spaces must not change the line id")
	}
	edited, _ := j.Save(ctx, userID, journalDay, "$ 55 Almoço #alimentacao")
	if edited.LineIDs["$ 55 Almoço #alimentacao"] == got.LineIDs["$ 50 Almoço #alimentacao"] {
		t.Fatal("a different line must get a different id")
	}

	if _, err := j.Save(ctx, userID, "10/03/2026", "x"); !errors.Is(err, ErrInvalidDate) {
		t.Fatalf("bad date: err = %v, want ErrInvalidDate", err)
	}
}

func TestJournalRegisterExpenseIsIdempotentAndUndoable(t *testing.T) {
	j := newJournal(t)
	userID := createTestUser(t)
	ctx := context.Background()
	line := "$ 50 Almoço #alimentacao"

	if _, err := j.Save(ctx, userID, journalDay, line); err != nil {
		t.Fatalf("save: %v", err)
	}
	in := RegisterExpenseInput{AmountCents: 5000, Category: "alimentacao", Description: "Almoço"}
	got, err := j.RegisterExpense(ctx, userID, journalDay, line, in)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].Kind != JournalKindExpense || got.Items[0].SourceID != got.LineIDs[line] {
		t.Fatalf("items = %+v, line ids = %v", got.Items, got.LineIDs)
	}

	// Registering the same line again updates, never duplicates.
	in.AmountCents = 5200
	again, err := j.RegisterExpense(ctx, userID, journalDay, line, in)
	if err != nil {
		t.Fatalf("register again: %v", err)
	}
	if len(again.Items) != 1 || *again.Items[0].AmountCents != 5200 {
		t.Fatalf("after re-register items = %+v, want one of 5200", again.Items)
	}

	// It is a real transaction of that day, visible in Finance.
	txs, err := NewFinanceService(requireDB(t), testLogger).ListTransactions(ctx, userID, journalDay, journalDay, "")
	if err != nil || len(txs) != 1 || txs[0].SourceType != "track_day" {
		t.Fatalf("finance sees %+v (err %v), want the journal expense", txs, err)
	}

	undone, err := j.Unregister(ctx, userID, journalDay, again.Items[0].SourceID)
	if err != nil || len(undone.Items) != 0 {
		t.Fatalf("undo: items = %+v, err = %v", undone.Items, err)
	}
	if _, err := j.Unregister(ctx, userID, journalDay, again.Items[0].SourceID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second undo: err = %v, want ErrNotFound", err)
	}
}

func TestJournalRegisterWorkoutNeverOverwritesOtherCheckIn(t *testing.T) {
	j := newJournal(t)
	pool := requireDB(t)
	userID := createTestUser(t)
	ctx := context.Background()
	habitID := createHealthHabit(t, userID, "Correr")
	minutes := 30
	in := RegisterWorkoutInput{HabitID: habitID, Metrics: map[string]any{"km": 5.0, "time_min": 30}, TimeMinutes: &minutes}

	got, err := j.RegisterWorkout(ctx, userID, journalDay, "corrida 5km 30min", in)
	if err != nil {
		t.Fatalf("register workout: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].Kind != JournalKindWorkout || got.Items[0].Label != "Correr" {
		t.Fatalf("items = %+v", got.Items)
	}
	// Same line again: same check-in, fine.
	if _, err := j.RegisterWorkout(ctx, userID, journalDay, "corrida 5km 30min", in); err != nil {
		t.Fatalf("re-register same line: %v", err)
	}

	// A manual check-in on another day must never be overwritten.
	otherDay := "2026-03-11"
	if _, err := pool.Exec(ctx,
		`INSERT INTO habit_logs (habit_id, user_id, logged_date, notes) VALUES ($1, $2, $3, 'feito à mão')`,
		habitID, userID, otherDay,
	); err != nil {
		t.Fatalf("manual check-in: %v", err)
	}
	if _, err := j.RegisterWorkout(ctx, userID, otherDay, "corrida 8km", in); !errors.Is(err, ErrAlreadyLogged) {
		t.Fatalf("over a manual check-in: err = %v, want ErrAlreadyLogged", err)
	}
	var notes string
	if err := pool.QueryRow(ctx, "SELECT notes FROM habit_logs WHERE habit_id = $1 AND logged_date = $2", habitID, otherDay).Scan(&notes); err != nil || notes != "feito à mão" {
		t.Fatalf("manual check-in notes = %q (err %v), must be untouched", notes, err)
	}

	undone, err := j.Unregister(ctx, userID, journalDay, got.Items[0].SourceID)
	if err != nil || len(undone.Items) != 0 {
		t.Fatalf("undo workout: items = %+v, err = %v", undone.Items, err)
	}
}

func TestJournalIsolatesUsers(t *testing.T) {
	j := newJournal(t)
	ctx := context.Background()
	owner := createTestUser(t)
	other := createTestUser(t)
	ownersHabit := createHealthHabit(t, owner, "Correr")
	line := "$ 10 Café"

	got, err := j.RegisterExpense(ctx, owner, journalDay, line, RegisterExpenseInput{AmountCents: 1000, Category: "alimentacao", Description: "Café"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := j.Unregister(ctx, other, journalDay, got.Items[0].SourceID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user undo: err = %v, want ErrNotFound", err)
	}
	if _, err := j.RegisterWorkout(ctx, other, journalDay, "corrida", RegisterWorkoutInput{HabitID: ownersHabit}); !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("other user's habit: err = %v, want ErrInvalidReference", err)
	}
	theirs, err := j.Get(ctx, other, journalDay)
	if err != nil || len(theirs.Items) != 0 {
		t.Fatalf("other user sees items %+v (err %v)", theirs.Items, err)
	}
}
