package service

import (
	"context"
	"errors"
	"testing"

	"github.com/myroutine/backend/internal/appctx"
)

func newFinanceService(t *testing.T) *FinanceService {
	t.Helper()
	return NewFinanceService(requireDB(t), testLogger)
}

// currentMonth is the first day of the default user's local month, matching
// today() — never the server clock's month.
func currentMonth() string {
	return firstOfMonth(appctx.Today(context.Background()))
}

// The mobile journal pushes the same row whenever the user re-syncs. The
// unique (source_type, source_id) index must make that an update, never a
// duplicate — this is what keeps "one source of truth per event" true.
func TestTransactionPushIsIdempotentBySource(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	ctx := context.Background()
	sourceID := "11111111-1111-1111-1111-111111111111"

	first, err := svc.CreateTransaction(ctx, userID, CreateTransactionInput{
		AmountCents: 2500,
		Category:    "transporte",
		Description: "Uber",
		OccurredOn:  today(),
		SourceType:  "track_day",
		SourceID:    &sourceID,
	})
	if err != nil {
		t.Fatalf("first push: %v", err)
	}

	// Same source, corrected value — must update the same row.
	second, err := svc.CreateTransaction(ctx, userID, CreateTransactionInput{
		AmountCents: 2600,
		Category:    "transporte",
		Description: "Uber",
		OccurredOn:  today(),
		SourceType:  "track_day",
		SourceID:    &sourceID,
	})
	if err != nil {
		t.Fatalf("second push: %v", err)
	}

	if second.ID != first.ID {
		t.Error("re-pushing the same source_id created a duplicate transaction")
	}
	if second.AmountCents != 2600 {
		t.Errorf("amount_cents = %d, want 2600 (should reflect the correction)", second.AmountCents)
	}

	txs, err := svc.ListTransactions(ctx, userID, today(), today(), "")
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(txs) != 1 {
		t.Errorf("transaction count = %d, want 1", len(txs))
	}
}

// Manual entries have no source_id, so identical ones are legitimately
// distinct rows (two coffees on the same day).
func TestManualTransactionsAreNotDeduplicated(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if _, err := svc.CreateTransaction(ctx, userID, CreateTransactionInput{
			AmountCents: 700, Category: "alimentacao", Description: "Café", OccurredOn: today(),
		}); err != nil {
			t.Fatalf("push %d: %v", i, err)
		}
	}

	txs, err := svc.ListTransactions(ctx, userID, today(), today(), "")
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(txs) != 2 {
		t.Errorf("transaction count = %d, want 2 (manual entries are distinct)", len(txs))
	}
}

func TestFinanceSummaryTotalsAndBudgets(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	ctx := context.Background()
	month := currentMonth()

	inputs := []CreateTransactionInput{
		{AmountCents: 5000, Category: "alimentacao", Description: "Almoço", OccurredOn: today()},
		{AmountCents: 3000, Category: "alimentacao", Description: "Jantar", OccurredOn: today()},
		{AmountCents: 2000, Category: "transporte", Description: "Ônibus", OccurredOn: today()},
		{AmountCents: 300000, Kind: "income", Category: "salario", Description: "Bolsa", OccurredOn: today()},
	}
	for _, in := range inputs {
		if _, err := svc.CreateTransaction(ctx, userID, in); err != nil {
			t.Fatalf("creating transaction: %v", err)
		}
	}

	if _, err := svc.UpsertBudget(ctx, userID, "alimentacao", month, 80000); err != nil {
		t.Fatalf("upserting budget: %v", err)
	}
	// A budget with no spend yet must still appear in the summary.
	if _, err := svc.UpsertBudget(ctx, userID, "lazer", month, 20000); err != nil {
		t.Fatalf("upserting budget: %v", err)
	}

	summary, err := svc.GetSummary(ctx, userID, month)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}

	if summary.IncomeCents != 300000 {
		t.Errorf("income = %d, want 300000", summary.IncomeCents)
	}
	if summary.ExpenseCents != 10000 {
		t.Errorf("expense = %d, want 10000", summary.ExpenseCents)
	}

	byCategory := map[string]CategorySummary{}
	for _, c := range summary.ByCategory {
		byCategory[c.Category] = c
	}

	food := byCategory["alimentacao"]
	if food.SpentCents != 8000 {
		t.Errorf("alimentacao spent = %d, want 8000", food.SpentCents)
	}
	if food.BudgetCents == nil || *food.BudgetCents != 80000 {
		t.Error("alimentacao budget missing from summary")
	}

	leisure, ok := byCategory["lazer"]
	if !ok {
		t.Fatal("a budgeted category with no spend must still appear")
	}
	if leisure.SpentCents != 0 {
		t.Errorf("lazer spent = %d, want 0", leisure.SpentCents)
	}
}

// One budget per (category, month): setting it again replaces the value.
func TestBudgetUpsertReplacesValue(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	ctx := context.Background()
	month := currentMonth()

	first, err := svc.UpsertBudget(ctx, userID, "alimentacao", month, 80000)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	second, err := svc.UpsertBudget(ctx, userID, "alimentacao", month, 90000)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	if second.ID != first.ID {
		t.Error("upsert created a second budget row for the same category/month")
	}
	if second.AmountCents != 90000 {
		t.Errorf("amount_cents = %d, want 90000", second.AmountCents)
	}

	budgets, err := svc.ListBudgets(ctx, userID, month)
	if err != nil {
		t.Fatalf("listing budgets: %v", err)
	}
	if len(budgets) != 1 {
		t.Errorf("budget count = %d, want 1", len(budgets))
	}
}

func TestTransactionUpdateAndDelete(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	tx, err := svc.CreateTransaction(ctx, userID, CreateTransactionInput{
		AmountCents: 5000, Category: "alimentacao", Description: "Almoço",
		OccurredOn: today(), Method: strPtr("pix"),
	})
	if err != nil {
		t.Fatalf("creating: %v", err)
	}

	updated, err := svc.UpdateTransaction(ctx, tx.ID, userID,
		patchFields(t, `{"amount_cents":5500,"method":null}`))
	if err != nil {
		t.Fatalf("updating: %v", err)
	}
	if updated.AmountCents != 5500 {
		t.Errorf("amount_cents = %d, want 5500", updated.AmountCents)
	}
	if updated.Method != nil {
		t.Error("explicit null should have cleared method")
	}
	if updated.Description != "Almoço" {
		t.Error("absent fields must be preserved")
	}

	if err := svc.DeleteTransaction(ctx, tx.ID, userID); err != nil {
		t.Fatalf("deleting: %v", err)
	}
	if err := svc.DeleteTransaction(ctx, tx.ID, userID); err != ErrNotFound {
		t.Errorf("second delete err = %v, want ErrNotFound", err)
	}
}

func TestTransactionIsolatesUsers(t *testing.T) {
	svc := newFinanceService(t)
	owner := createTestUser(t)
	other := createTestUser(t)
	ctx := context.Background()

	tx, err := svc.CreateTransaction(ctx, owner, CreateTransactionInput{
		AmountCents: 5000, Category: "alimentacao", Description: "Privado", OccurredOn: today(),
	})
	if err != nil {
		t.Fatalf("creating: %v", err)
	}

	if _, err := svc.UpdateTransaction(ctx, tx.ID, other,
		patchFields(t, `{"amount_cents":1}`)); err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound for another user's transaction", err)
	}

	txs, err := svc.ListTransactions(ctx, other, today(), today(), "")
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("transaction count = %d, want 0", len(txs))
	}
}

// The source (source_type, source_id) is unique across all users. Pushing a
// source_id that already belongs to another user must never touch their row.
func TestTransactionSourceCannotOverwriteAnotherUsersRow(t *testing.T) {
	svc := newFinanceService(t)
	ctx := context.Background()
	victim := createTestUser(t)
	attacker := createTestUser(t)
	sourceID := "22222222-2222-2222-2222-222222222222"

	orig, err := svc.CreateTransaction(ctx, victim, CreateTransactionInput{
		AmountCents: 5000, Category: "alimentacao", Description: "Mercado",
		OccurredOn: today(), SourceType: "track_day", SourceID: &sourceID,
	})
	if err != nil {
		t.Fatalf("victim push: %v", err)
	}
	t.Cleanup(func() {
		_, _ = requireDB(t).Exec(ctx, "DELETE FROM transactions WHERE source_id = $1", sourceID)
	})

	_, err = svc.CreateTransaction(ctx, attacker, CreateTransactionInput{
		AmountCents: 1, Category: "other", Description: "sobrescrito",
		OccurredOn: today(), SourceType: "track_day", SourceID: &sourceID,
	})
	if !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("foreign source_id: err = %v, want ErrInvalidReference", err)
	}

	txs, err := svc.ListTransactions(ctx, victim, today(), today(), "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, tx := range txs {
		if tx.ID == orig.ID && (tx.AmountCents != 5000 || tx.Description != "Mercado") {
			t.Fatalf("victim's transaction was overwritten: %+v", tx)
		}
	}
}
