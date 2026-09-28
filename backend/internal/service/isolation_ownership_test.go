package service

import (
	"context"
	"errors"
	"testing"
)

// Two-user isolation: nothing user B sends may point at rows owned by user A,
// and nothing of A may surface in B's reads.

func createIsolationHabit(t *testing.T, userID string) *HabitDTO {
	t.Helper()
	habit, err := newHabitService(t).Create(context.Background(), userID, CreateHabitInput{
		Name: "Ler", Icon: "📚", Color: "#0071E3",
		Frequency: "daily", TargetDays: []int32{1, 2, 3, 4, 5, 6, 7},
		Category: "general",
	})
	if err != nil {
		t.Fatalf("creating habit: %v", err)
	}
	return habit
}

func TestImportRejectsCardOfAnotherUser(t *testing.T) {
	svc := newImportService(t)
	finance := newFinanceService(t)
	userA := createTestUser(t)
	userB := createTestUser(t)
	ctx := context.Background()

	cardA := createTestCard(t, finance, userA)

	mapping, _ := SniffMapping(sampleStatement)
	_, err := svc.CreateBatch(ctx, userB, CreateImportInput{
		Filename: "extrato.csv", Content: sampleStatement, Mapping: mapping,
		CreditCardID: &cardA.ID,
	})
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("B importou com o cartão de A: err = %v, esperado ErrInvalidReference", err)
	}

	batches, err := svc.ListBatches(ctx, userB)
	if err != nil {
		t.Fatalf("listing batches: %v", err)
	}
	if len(batches) != 0 {
		t.Errorf("remessa rejeitada deixou %d batches para B", len(batches))
	}
}

func TestImportAcceptsOwnCard(t *testing.T) {
	svc := newImportService(t)
	finance := newFinanceService(t)
	userID := createTestUser(t)
	card := createTestCard(t, finance, userID)

	mapping, _ := SniffMapping(sampleStatement)
	batch, err := svc.CreateBatch(context.Background(), userID, CreateImportInput{
		Filename: "extrato.csv", Content: sampleStatement, Mapping: mapping,
		CreditCardID: &card.ID,
	})
	if err != nil {
		t.Fatalf("importando com o próprio cartão: %v", err)
	}
	if batch.CreditCardID == nil || *batch.CreditCardID != card.ID {
		t.Errorf("batch.CreditCardID = %v, esperado %s", batch.CreditCardID, card.ID)
	}
}

// A batch that somehow points at another user's card (legacy rows written
// before the check existed) must not turn into a transaction carrying it.
func TestApproveRejectsBatchWithCardOfAnotherUser(t *testing.T) {
	pool := requireDB(t)
	svc := newImportService(t)
	finance := newFinanceService(t)
	userA := createTestUser(t)
	userB := createTestUser(t)
	ctx := context.Background()

	cardA := createTestCard(t, finance, userA)
	batch := importSample(t, svc, userB, sampleStatement)

	if _, err := pool.Exec(ctx,
		"UPDATE import_batches SET credit_card_id = $1 WHERE id = $2", cardA.ID, batch.ID,
	); err != nil {
		t.Fatalf("forcing legacy batch: %v", err)
	}

	_, err := svc.ApproveEntry(ctx, batch.Entries[0].ID, userB, "food")
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("aprovação com cartão de A: err = %v, esperado ErrInvalidReference", err)
	}

	txs, err := finance.ListTransactions(ctx, userB, "2026-01-01", "2026-12-31", "")
	if err != nil {
		t.Fatalf("listing transactions: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("aprovação rejeitada criou %d transações", len(txs))
	}
}

// Even if a transaction of B references A's card, A's card name never shows up
// in B's listing.
func TestTransactionsNeverShowCardNameOfAnotherUser(t *testing.T) {
	pool := requireDB(t)
	finance := newFinanceService(t)
	userA := createTestUser(t)
	userB := createTestUser(t)
	ctx := context.Background()

	cardA, err := finance.CreateCard(ctx, userA, CreateCreditCardInput{
		Name: "Cartao Secreto de A", ClosingDay: 25, DueDay: 5,
	})
	if err != nil {
		t.Fatalf("creating card: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO transactions (user_id, amount_cents, kind, category, description,
		                          occurred_on, source_type, credit_card_id)
		VALUES ($1, 1000, 'expense', 'other', 'legado', '2026-08-10', 'manual', $2)`,
		userB, cardA.ID,
	); err != nil {
		t.Fatalf("inserting legacy transaction: %v", err)
	}

	txs, err := finance.ListTransactions(ctx, userB, "2026-01-01", "2026-12-31", "")
	if err != nil {
		t.Fatalf("listing transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("B tem %d transações, esperado 1", len(txs))
	}
	if txs[0].CreditCardName != nil {
		t.Errorf("nome do cartão de A vazou para B: %q", *txs[0].CreditCardName)
	}
}

func TestTaskRejectsLinkedHabitOfAnotherUser(t *testing.T) {
	tasks := newTaskService(t)
	userA := createTestUser(t)
	userB := createTestUser(t)
	ctx := context.Background()

	habitA := createIsolationHabit(t, userA)

	_, err := tasks.Create(ctx, userB, CreateTaskInput{
		Title: "Ler", Date: "2026-08-10", LinkedHabitID: &habitA.ID,
	})
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("create com hábito de A: err = %v, esperado ErrInvalidReference", err)
	}

	own, err := tasks.Create(ctx, userB, CreateTaskInput{Title: "Ler", Date: "2026-08-10"})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	_, err = tasks.Update(ctx, own.ID, userB, patchFields(t, `{"linked_habit_id":"`+habitA.ID+`"}`))
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("update com hábito de A: err = %v, esperado ErrInvalidReference", err)
	}

	// O próprio hábito continua aceito, e null continua limpando.
	habitB := createIsolationHabit(t, userB)
	linked, err := tasks.Update(ctx, own.ID, userB, patchFields(t, `{"linked_habit_id":"`+habitB.ID+`"}`))
	if err != nil {
		t.Fatalf("linking own habit: %v", err)
	}
	if linked.LinkedHabitID == nil || *linked.LinkedHabitID != habitB.ID {
		t.Errorf("linked_habit_id = %v, esperado %s", linked.LinkedHabitID, habitB.ID)
	}
	cleared, err := tasks.Update(ctx, own.ID, userB, patchFields(t, `{"linked_habit_id":null}`))
	if err != nil {
		t.Fatalf("clearing link: %v", err)
	}
	if cleared.LinkedHabitID != nil {
		t.Errorf("linked_habit_id deveria ser nulo, veio %v", *cleared.LinkedHabitID)
	}
}

func TestTaskRejectsMalformedLinkedHabit(t *testing.T) {
	tasks := newTaskService(t)
	userID := createTestUser(t)

	bogus := "not-a-uuid"
	_, err := tasks.Create(context.Background(), userID, CreateTaskInput{
		Title: "Ler", Date: "2026-08-10", LinkedHabitID: &bogus,
	})
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("create com id malformado: err = %v, esperado ErrInvalidReference", err)
	}
}

func TestStudySessionRejectsTaskOfAnotherUser(t *testing.T) {
	study := NewStudyService(requireDB(t), testLogger)
	tasks := newTaskService(t)
	userA := createTestUser(t)
	userB := createTestUser(t)
	ctx := context.Background()

	taskA, err := tasks.Create(ctx, userA, CreateTaskInput{Title: "Estudar", Date: "2026-08-10"})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	_, err = study.CreateSession(ctx, userB, CreateStudySessionInput{
		Subject: "OAC", StudiedOn: "2026-08-10", DurationMinutes: 30, TaskID: &taskA.ID,
	})
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("sessão com task de A: err = %v, esperado ErrInvalidReference", err)
	}

	sessions, err := study.ListSessions(ctx, userB, "2026-01-01", "2026-12-31", "")
	if err != nil {
		t.Fatalf("listing sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("sessão rejeitada foi gravada (%d)", len(sessions))
	}

	taskB, err := tasks.Create(ctx, userB, CreateTaskInput{Title: "Estudar", Date: "2026-08-10"})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}
	sess, err := study.CreateSession(ctx, userB, CreateStudySessionInput{
		Subject: "OAC", StudiedOn: "2026-08-10", DurationMinutes: 30, TaskID: &taskB.ID,
	})
	if err != nil {
		t.Fatalf("sessão com a própria task: %v", err)
	}
	if sess.TaskID == nil || *sess.TaskID != taskB.ID {
		t.Errorf("task_id = %v, esperado %s", sess.TaskID, taskB.ID)
	}
}
