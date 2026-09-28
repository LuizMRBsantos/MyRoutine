package service

import (
	"context"
	"testing"
	"time"

	"github.com/myroutine/backend/internal/appctx"
)

// ─── User timezone in finance and study (T0.3) ───────────────────────────────

// localDayAt is the user's local calendar day (midnight in loc) at the given
// UTC instant — what appctx.Today returns when the clock reads that instant.
func localDayAt(t *testing.T, instant string, loc *time.Location) time.Time {
	t.Helper()
	now := utcInstant(t, instant).In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
}

// offDayCtx returns a context whose user timezone is on a different calendar
// day from the server clock, whatever time the suite runs at: UTC+14 and
// UTC-12 are 26 hours apart, so at least one of them differs.
func offDayCtx(t *testing.T) context.Context {
	t.Helper()
	serverDay := time.Now().Format("2006-01-02")
	for _, name := range []string{"Etc/GMT-14", "Etc/GMT+12"} {
		loc, err := time.LoadLocation(name)
		if err != nil {
			t.Fatalf("loading %s: %v", name, err)
		}
		if c := appctx.WithTimezone(context.Background(), loc); userToday(c) != serverDay {
			return c
		}
	}
	t.Fatal("no candidate timezone differs from the server day")
	return nil
}

func derefStr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

// 23:00 in São Paulo is already 02:00 UTC of the next day; a transaction with
// no date must land on the user's local day.
func TestTransactionDefaultsToUserLocalDay(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	ctx, loc := saoPauloCtx(t)

	today := localDayAt(t, "2026-03-10T02:00:00Z", loc) // 2026-03-09 23:00 local
	tx, err := svc.createTransaction(ctx, userID, CreateTransactionInput{
		AmountCents: 1500, Category: "alimentacao", Description: "Pizza",
	}, today)
	if err != nil {
		t.Fatalf("creating: %v", err)
	}
	if tx.OccurredOn != "2026-03-09" {
		t.Errorf("occurred_on = %q, want 2026-03-09 (local day, not the UTC day)", tx.OccurredOn)
	}
}

// The public entry point reads the user's timezone from the context.
func TestTransactionPublicDefaultUsesContextTimezone(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	ctx := offDayCtx(t)

	tx, err := svc.CreateTransaction(ctx, userID, CreateTransactionInput{
		AmountCents: 1500, Category: "alimentacao", Description: "Pizza",
	})
	if err != nil {
		t.Fatalf("creating: %v", err)
	}
	if tx.OccurredOn != userToday(ctx) {
		t.Errorf("occurred_on = %q, want the user's local day %s", tx.OccurredOn, userToday(ctx))
	}
}

// Card closing on the 25th, due on the 5th. At 23:00 on 25/01 in São Paulo
// (26/01 in UTC) the purchase is still inside January's bill → due 05/02.
// Using the UTC day would push it to the next bill (05/03).
func TestCardPurchaseDefaultsToUserLocalDay(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	card := createTestCard(t, svc, userID)
	ctx, loc := saoPauloCtx(t)

	today := localDayAt(t, "2026-01-26T02:00:00Z", loc) // 2026-01-25 23:00 local
	parts, err := svc.createCardPurchase(ctx, userID, CreateTransactionInput{
		AmountCents: 30000, Category: "eletronicos", Description: "Fone",
		CreditCardID: &card.ID, Installments: 3,
	}, today)
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}
	if len(parts) != 3 {
		t.Fatalf("installments = %d, want 3", len(parts))
	}
	if got := derefStr(parts[0].PurchasedOn); got != "2026-01-25" {
		t.Errorf("purchased_on = %q, want 2026-01-25", got)
	}
	for i, want := range []string{"2026-02-05", "2026-03-05", "2026-04-05"} {
		if parts[i].OccurredOn != want {
			t.Errorf("installment %d = %s, want %s (first bill closes 25/01)", i+1, parts[i].OccurredOn, want)
		}
	}
}

// The public card purchase reads the user's timezone from the context.
func TestCardPurchasePublicDefaultUsesContextTimezone(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	card := createTestCard(t, svc, userID)
	ctx := offDayCtx(t)

	parts, err := svc.CreateCardPurchase(ctx, userID, CreateTransactionInput{
		AmountCents: 10000, Category: "eletronicos", Description: "Cabo",
		CreditCardID: &card.ID, Installments: 1,
	})
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}
	if got := derefStr(parts[0].PurchasedOn); got != userToday(ctx) {
		t.Errorf("purchased_on = %q, want the user's local day %s", got, userToday(ctx))
	}
}

// 31/01 23:00 in São Paulo is already February in UTC; with no month the
// summary and the budget list must still report January.
func TestSummaryAndBudgetsDefaultToUserLocalMonth(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	ctx, loc := saoPauloCtx(t)

	if _, err := svc.CreateTransaction(ctx, userID, CreateTransactionInput{
		AmountCents: 4000, Category: "alimentacao", Description: "Mercado", OccurredOn: "2026-01-31",
	}); err != nil {
		t.Fatalf("creating: %v", err)
	}
	if _, err := svc.UpsertBudget(ctx, userID, "alimentacao", "2026-01-01", 50000); err != nil {
		t.Fatalf("budget: %v", err)
	}

	today := localDayAt(t, "2026-02-01T02:00:00Z", loc) // 2026-01-31 23:00 local

	summary, err := svc.getSummary(ctx, userID, "", today)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.Month != "2026-01-01" {
		t.Errorf("summary month = %q, want 2026-01-01", summary.Month)
	}
	if summary.ExpenseCents != 4000 {
		t.Errorf("expense = %d, want 4000 from January", summary.ExpenseCents)
	}

	budgets, err := svc.listBudgets(ctx, userID, "", today)
	if err != nil {
		t.Fatalf("budgets: %v", err)
	}
	if len(budgets) != 1 || budgets[0].Month != "2026-01-01" {
		t.Errorf("budgets = %+v, want January's budget", budgets)
	}
}

// An explicit month always wins over the default.
func TestSummaryExplicitMonthIsKept(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	ctx, loc := saoPauloCtx(t)

	summary, err := svc.getSummary(ctx, userID, "2025-12-01", localMidnight(t, "2026-01-31", loc))
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.Month != "2025-12-01" {
		t.Errorf("summary month = %q, want the explicit 2025-12-01", summary.Month)
	}
}

// A study session with no date lands on the user's local day: 23:00 in São
// Paulo is already the next day in UTC.
func TestStudySessionDefaultsToUserLocalDay(t *testing.T) {
	studySvc := NewStudyService(requireDB(t), testLogger)
	userID := createTestUser(t)
	ctx, loc := saoPauloCtx(t)

	today := localDayAt(t, "2026-03-10T02:00:00Z", loc) // 2026-03-09 23:00 local
	sess, err := studySvc.createSession(ctx, userID, CreateStudySessionInput{
		Subject: "OAC", DurationMinutes: 40,
	}, today)
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}
	if sess.StudiedOn != "2026-03-09" {
		t.Errorf("studied_on = %q, want 2026-03-09 (local day, not the UTC day)", sess.StudiedOn)
	}
}

// The public entry point reads the user's timezone from the context.
func TestStudySessionPublicDefaultUsesContextTimezone(t *testing.T) {
	studySvc := NewStudyService(requireDB(t), testLogger)
	userID := createTestUser(t)
	ctx := offDayCtx(t)

	sess, err := studySvc.CreateSession(ctx, userID, CreateStudySessionInput{
		Subject: "OAC", DurationMinutes: 40,
	})
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}
	if sess.StudiedOn != userToday(ctx) {
		t.Errorf("studied_on = %q, want the user's local day %s", sess.StudiedOn, userToday(ctx))
	}
}

// The 30-day summary is anchored on the user's local day: with today =
// 2026-03-10, 2026-02-08 is the first day in and 2026-02-07 is out.
func TestStudySummaryUsesUserLocalDay(t *testing.T) {
	studySvc := NewStudyService(requireDB(t), testLogger)
	userID := createTestUser(t)
	ctx, loc := saoPauloCtx(t)

	for date, minutes := range map[string]int{"2026-02-07": 10, "2026-02-08": 20, "2026-03-10": 30} {
		if _, err := studySvc.CreateSession(ctx, userID, CreateStudySessionInput{
			Subject: "Cálculo", StudiedOn: date, DurationMinutes: minutes,
		}); err != nil {
			t.Fatalf("session %s: %v", date, err)
		}
	}

	today := localDayAt(t, "2026-03-11T02:00:00Z", loc) // 2026-03-10 23:00 local
	summary, err := studySvc.getSummary(ctx, userID, today)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.TotalMinutes30d != 50 || summary.Sessions30d != 2 {
		t.Errorf("summary = %d min / %d sessions, want 50/2 (02-08 and 03-10 only)",
			summary.TotalMinutes30d, summary.Sessions30d)
	}
}
