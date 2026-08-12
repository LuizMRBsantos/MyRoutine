package service

import (
	"context"
	"testing"
)

// Cartão Nubank clássico: fecha 25, vence 5.
func createTestCard(t *testing.T, svc *FinanceService, userID string) *CreditCardDTO {
	t.Helper()
	card, err := svc.CreateCard(context.Background(), userID, CreateCreditCardInput{
		Name: "Nubank", ClosingDay: 25, DueDay: 5,
	})
	if err != nil {
		t.Fatalf("creating card: %v", err)
	}
	return card
}

func TestCardPurchaseGeneratesInstallments(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	card := createTestCard(t, svc, userID)
	ctx := context.Background()

	// R$ 1.200 em 12x, comprado em 12/08 → 1ª parcela vence 05/09.
	parts, err := svc.CreateCardPurchase(ctx, userID, CreateTransactionInput{
		AmountCents:  120000,
		Category:     "eletronicos",
		Description:  "Notebook",
		OccurredOn:   "2026-08-12",
		CreditCardID: &card.ID,
		Installments: 12,
	})
	if err != nil {
		t.Fatalf("creating purchase: %v", err)
	}

	if len(parts) != 12 {
		t.Fatalf("gerou %d parcelas, esperado 12", len(parts))
	}

	var sum int64
	for _, p := range parts {
		sum += p.AmountCents
	}
	if sum != 120000 {
		t.Errorf("soma das parcelas = %d, esperado 120000", sum)
	}

	first := parts[0]
	if first.OccurredOn != "2026-09-05" {
		t.Errorf("1ª parcela vence em %s, esperado 2026-09-05", first.OccurredOn)
	}
	if first.PurchasedOn == nil || *first.PurchasedOn != "2026-08-12" {
		t.Error("purchased_on deveria guardar a data da compra")
	}
	if first.InstallmentNumber == nil || *first.InstallmentNumber != 1 {
		t.Error("installment_number da 1ª parcela deveria ser 1")
	}
	if first.InstallmentTotal == nil || *first.InstallmentTotal != 12 {
		t.Error("installment_total deveria ser 12")
	}
	if first.CreditCardName == nil || *first.CreditCardName != "Nubank" {
		t.Error("o nome do cartão deveria vir junto na listagem")
	}

	// Todas as parcelas compartilham o mesmo grupo.
	group := first.InstallmentGroupID
	if group == nil {
		t.Fatal("installment_group_id não foi preenchido")
	}
	for _, p := range parts {
		if p.InstallmentGroupID == nil || *p.InstallmentGroupID != *group {
			t.Fatal("parcelas da mesma compra deveriam compartilhar o grupo")
		}
	}

	if parts[11].OccurredOn != "2027-08-05" {
		t.Errorf("última parcela vence em %s, esperado 2027-08-05", parts[11].OccurredOn)
	}
}

// Compra depois do fechamento pula para a fatura seguinte.
func TestCardPurchaseAfterClosingDay(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	card := createTestCard(t, svc, userID)

	parts, err := svc.CreateCardPurchase(context.Background(), userID, CreateTransactionInput{
		AmountCents:  50000,
		Description:  "Tênis",
		OccurredOn:   "2026-08-28", // depois do fechamento (25)
		CreditCardID: &card.ID,
		Installments: 1,
	})
	if err != nil {
		t.Fatalf("creating purchase: %v", err)
	}

	if parts[0].OccurredOn != "2026-10-05" {
		t.Errorf("compra pós-fechamento venceu em %s, esperado 2026-10-05", parts[0].OccurredOn)
	}
	// Compra à vista no cartão não vira "1 de 1".
	if parts[0].InstallmentNumber != nil {
		t.Error("compra em 1x não deveria ter numeração de parcela")
	}
}

// A parcela precisa aparecer no resumo do mês em que é cobrada, não no da compra.
func TestInstallmentsLandOnTheMonthTheyAreCharged(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	card := createTestCard(t, svc, userID)
	ctx := context.Background()

	_, err := svc.CreateCardPurchase(ctx, userID, CreateTransactionInput{
		AmountCents:  30000, // R$ 300 em 3x = R$ 100/mês
		Category:     "lazer",
		Description:  "Show",
		OccurredOn:   "2026-08-12",
		CreditCardID: &card.ID,
		Installments: 3,
	})
	if err != nil {
		t.Fatalf("creating purchase: %v", err)
	}

	// Agosto (mês da compra) não deve registrar despesa: nada foi pago ainda.
	august, err := svc.GetSummary(ctx, userID, "2026-08-01")
	if err != nil {
		t.Fatalf("summary agosto: %v", err)
	}
	if august.ExpenseCents != 0 {
		t.Errorf("agosto somou %d, esperado 0 — a compra só é paga a partir de setembro", august.ExpenseCents)
	}

	for _, month := range []string{"2026-09-01", "2026-10-01", "2026-11-01"} {
		summary, err := svc.GetSummary(ctx, userID, month)
		if err != nil {
			t.Fatalf("summary %s: %v", month, err)
		}
		if summary.ExpenseCents != 10000 {
			t.Errorf("%s somou %d, esperado 10000", month, summary.ExpenseCents)
		}
	}

	// Dezembro já não tem mais parcela.
	december, err := svc.GetSummary(ctx, userID, "2026-12-01")
	if err != nil {
		t.Fatalf("summary dezembro: %v", err)
	}
	if december.ExpenseCents != 0 {
		t.Errorf("dezembro somou %d, esperado 0", december.ExpenseCents)
	}
}

func TestDeleteInstallmentGroupRemovesEveryInstallment(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	card := createTestCard(t, svc, userID)
	ctx := context.Background()

	parts, err := svc.CreateCardPurchase(ctx, userID, CreateTransactionInput{
		AmountCents:  60000,
		Description:  "Curso",
		OccurredOn:   "2026-08-10",
		CreditCardID: &card.ID,
		Installments: 6,
	})
	if err != nil {
		t.Fatalf("creating purchase: %v", err)
	}

	group := *parts[0].InstallmentGroupID
	if err := svc.DeleteInstallmentGroup(ctx, group, userID); err != nil {
		t.Fatalf("deleting group: %v", err)
	}

	txs, err := svc.ListTransactions(ctx, userID, "2026-01-01", "2027-12-31", "")
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("sobraram %d parcelas após apagar o grupo", len(txs))
	}

	if err := svc.DeleteInstallmentGroup(ctx, group, userID); err != ErrNotFound {
		t.Errorf("apagar o grupo de novo → %v, esperado ErrNotFound", err)
	}
}

func TestCardIsolatedPerUser(t *testing.T) {
	svc := newFinanceService(t)
	owner := createTestUser(t)
	other := createTestUser(t)
	card := createTestCard(t, svc, owner)

	if _, err := svc.GetCard(context.Background(), card.ID, other); err != ErrNotFound {
		t.Errorf("err = %v, esperado ErrNotFound para cartão de outro usuário", err)
	}
}

func TestPurchaseOnUnknownCardIsRejected(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)
	missing := "00000000-0000-0000-0000-000000000000"

	_, err := svc.CreateCardPurchase(context.Background(), userID, CreateTransactionInput{
		AmountCents:  1000,
		Description:  "x",
		CreditCardID: &missing,
		Installments: 2,
	})
	if err != ErrNotFound {
		t.Errorf("err = %v, esperado ErrNotFound", err)
	}
}

// Transação normal (pix/débito) não pode ser afetada pelas colunas novas.
func TestPlainTransactionStillWorks(t *testing.T) {
	svc := newFinanceService(t)
	userID := createTestUser(t)

	tx, err := svc.CreateTransaction(context.Background(), userID, CreateTransactionInput{
		AmountCents: 4500,
		Category:    "alimentacao",
		Description: "Almoço",
		OccurredOn:  "2026-08-12",
	})
	if err != nil {
		t.Fatalf("creating transaction: %v", err)
	}

	if tx.CreditCardID != nil || tx.InstallmentGroupID != nil || tx.InstallmentNumber != nil {
		t.Error("transação avulsa não deveria ter dados de cartão/parcela")
	}
	if tx.OccurredOn != "2026-08-12" {
		t.Errorf("occurred_on = %s, esperado a própria data", tx.OccurredOn)
	}
}
