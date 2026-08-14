package service

import (
	"context"
	"testing"
)

func newImportService(t *testing.T) *ImportService {
	t.Helper()
	return NewImportService(requireDB(t), testLogger)
}

const sampleStatement = `Data;Valor;Histórico
12/08/2026;-45,90;IFOOD *RESTAURANTE SP
13/08/2026;-120,00;POSTO IPIRANGA 4455
01/08/2026;5.000,00;SALARIO EMPRESA`

func importSample(t *testing.T, svc *ImportService, userID, content string) *ImportBatchDTO {
	t.Helper()
	mapping, complete := SniffMapping(content)
	if !complete {
		t.Fatal("mapeamento deveria ser detectado no arquivo de teste")
	}
	batch, err := svc.CreateBatch(context.Background(), userID, CreateImportInput{
		Filename: "extrato.csv", Content: content, Mapping: mapping,
	})
	if err != nil {
		t.Fatalf("importando: %v", err)
	}
	return batch
}

func TestImportCreatesStagedEntriesOnly(t *testing.T) {
	svc := newImportService(t)
	finance := newFinanceService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	batch := importSample(t, svc, userID, sampleStatement)

	if batch.NewCount != 3 {
		t.Errorf("importou %d lançamentos, esperado 3", batch.NewCount)
	}
	if len(batch.Entries) != 3 {
		t.Fatalf("batch trouxe %d entradas, esperado 3", len(batch.Entries))
	}

	// O ponto central: nada virou transação ainda.
	txs, err := finance.ListTransactions(ctx, userID, "2026-01-01", "2026-12-31", "")
	if err != nil {
		t.Fatalf("listando transações: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("%d transações criadas — o extrato não pode entrar antes da aprovação", len(txs))
	}

	for _, e := range batch.Entries {
		if e.Status != "pending" {
			t.Errorf("entrada %s com status %q, esperado pending", e.RawDescription, e.Status)
		}
	}
}

// Reimportar o mesmo arquivo não pode trazer as linhas de novo.
func TestImportIsIdempotentAcrossFiles(t *testing.T) {
	svc := newImportService(t)
	userID := createTestUser(t)

	first := importSample(t, svc, userID, sampleStatement)
	if first.NewCount != 3 || first.DuplicateCount != 0 {
		t.Fatalf("primeira importação: %d novas / %d duplicatas", first.NewCount, first.DuplicateCount)
	}

	second := importSample(t, svc, userID, sampleStatement)
	if second.NewCount != 0 {
		t.Errorf("segunda importação criou %d lançamentos, esperado 0", second.NewCount)
	}
	if second.DuplicateCount != 3 {
		t.Errorf("segunda importação viu %d duplicatas, esperado 3", second.DuplicateCount)
	}
}

// A linha do extrato que corresponde a algo já lançado à mão precisa vir
// sinalizada, senão o mês conta o gasto duas vezes.
func TestImportFlagsLikelyDuplicateOfManualEntry(t *testing.T) {
	svc := newImportService(t)
	finance := newFinanceService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	// Você já lançou o almoço na mão, um dia antes do que o extrato registra.
	manual, err := finance.CreateTransaction(ctx, userID, CreateTransactionInput{
		AmountCents: 4590, Category: "alimentacao",
		Description: "Almoço", OccurredOn: "2026-08-11",
	})
	if err != nil {
		t.Fatalf("criando lançamento manual: %v", err)
	}

	batch := importSample(t, svc, userID, sampleStatement)

	var ifood *ImportEntryDTO
	for i := range batch.Entries {
		if batch.Entries[i].AmountCents == 4590 {
			ifood = &batch.Entries[i]
		}
	}
	if ifood == nil {
		t.Fatal("não achou a entrada do IFOOD")
	}

	if ifood.MatchedTransactionID == nil {
		t.Fatal("a linha deveria vir marcada como possível duplicata")
	}
	if *ifood.MatchedTransactionID != manual.ID {
		t.Errorf("casou com a transação errada: %s", *ifood.MatchedTransactionID)
	}
	if ifood.MatchedTransactionDescription == nil || *ifood.MatchedTransactionDescription != "Almoço" {
		t.Error("deveria trazer a descrição do lançamento existente para comparação")
	}
	if batch.MatchedCount != 1 {
		t.Errorf("batch reportou %d casamentos, esperado 1", batch.MatchedCount)
	}
}

func TestApproveEntryCreatesTransactionAndLearnsRule(t *testing.T) {
	svc := newImportService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	batch := importSample(t, svc, userID, sampleStatement)

	var ifood ImportEntryDTO
	for _, e := range batch.Entries {
		if e.AmountCents == 4590 {
			ifood = e
		}
	}

	tx, err := svc.ApproveEntry(ctx, ifood.ID, userID, "alimentacao")
	if err != nil {
		t.Fatalf("aprovando: %v", err)
	}

	if tx.Category != "alimentacao" || tx.AmountCents != 4590 {
		t.Errorf("transação criada errada: %+v", tx)
	}
	if tx.SourceType != "import" {
		t.Errorf("source_type = %q, esperado import", tx.SourceType)
	}
	if tx.SourceID == nil || *tx.SourceID != ifood.ID {
		t.Error("source_id deveria apontar para a linha importada")
	}

	// Aprovar de novo não pode duplicar.
	if _, err := svc.ApproveEntry(ctx, ifood.ID, userID, "alimentacao"); err == nil {
		t.Error("aprovar duas vezes deveria falhar")
	}

	// E a regra aprendida precisa existir.
	rules, err := svc.ListRules(ctx, userID)
	if err != nil {
		t.Fatalf("listando regras: %v", err)
	}
	found := false
	for _, r := range rules {
		if r.Category == "alimentacao" && r.Pattern == "IFOOD RESTAURANTE" {
			found = true
		}
	}
	if !found {
		t.Errorf("não aprendeu a regra do IFOOD; regras: %+v", rules)
	}
}

// O ganho real: a segunda importação já vem categorizada.
func TestSecondImportUsesLearnedRules(t *testing.T) {
	svc := newImportService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	batch := importSample(t, svc, userID, sampleStatement)
	var ifood ImportEntryDTO
	for _, e := range batch.Entries {
		if e.AmountCents == 4590 {
			ifood = e
		}
	}
	if _, err := svc.ApproveEntry(ctx, ifood.ID, userID, "alimentacao"); err != nil {
		t.Fatalf("aprovando: %v", err)
	}

	// Extrato do mês seguinte, mesmo estabelecimento, outro valor e data.
	next := `Data;Valor;Histórico
12/09/2026;-38,00;IFOOD *RESTAURANTE RJ`

	second := importSample(t, svc, userID, next)
	if len(second.Entries) != 1 {
		t.Fatalf("esperava 1 entrada, veio %d", len(second.Entries))
	}

	entry := second.Entries[0]
	if entry.SuggestedCategory == nil {
		t.Fatal("a categoria deveria vir sugerida pela regra aprendida")
	}
	if *entry.SuggestedCategory != "alimentacao" {
		t.Errorf("sugestão = %q, esperado alimentacao", *entry.SuggestedCategory)
	}
}

func TestDecideEntryIgnoresWithoutCreatingTransaction(t *testing.T) {
	svc := newImportService(t)
	finance := newFinanceService(t)
	userID := createTestUser(t)
	ctx := context.Background()

	batch := importSample(t, svc, userID, sampleStatement)
	entry := batch.Entries[0]

	if err := svc.DecideEntry(ctx, entry.ID, userID, "ignored"); err != nil {
		t.Fatalf("ignorando: %v", err)
	}

	txs, err := finance.ListTransactions(ctx, userID, "2026-01-01", "2026-12-31", "")
	if err != nil {
		t.Fatalf("listando: %v", err)
	}
	if len(txs) != 0 {
		t.Error("ignorar não pode criar transação")
	}

	// Decidir de novo falha: a linha já saiu da caixa de entrada.
	if err := svc.DecideEntry(ctx, entry.ID, userID, "ignored"); err != ErrNotFound {
		t.Errorf("segunda decisão = %v, esperado ErrNotFound", err)
	}

	pending, err := svc.ListPendingEntries(ctx, userID)
	if err != nil {
		t.Fatalf("listando pendentes: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("sobraram %d pendentes, esperado 2", len(pending))
	}
}

func TestDecideEntryRejectsInvalidStatus(t *testing.T) {
	svc := newImportService(t)
	userID := createTestUser(t)
	batch := importSample(t, svc, userID, sampleStatement)

	if err := svc.DecideEntry(context.Background(), batch.Entries[0].ID, userID, "imported"); err == nil {
		t.Error("status 'imported' não pode ser decidido por essa via — só por aprovação")
	}
}

func TestImportEntriesAreIsolatedPerUser(t *testing.T) {
	svc := newImportService(t)
	owner := createTestUser(t)
	other := createTestUser(t)
	ctx := context.Background()

	batch := importSample(t, svc, owner, sampleStatement)

	if _, err := svc.ApproveEntry(ctx, batch.Entries[0].ID, other, "alimentacao"); err != ErrNotFound {
		t.Errorf("aprovar entrada de outro usuário = %v, esperado ErrNotFound", err)
	}

	pending, err := svc.ListPendingEntries(ctx, other)
	if err != nil {
		t.Fatalf("listando: %v", err)
	}
	if len(pending) != 0 {
		t.Error("um usuário não pode ver a importação do outro")
	}
}
