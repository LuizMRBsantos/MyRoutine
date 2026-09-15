package service

import "testing"

func TestNormalizeDescription(t *testing.T) {
	cases := map[string]string{
		"IFOOD  *RESTAURANTE-SP": "IFOOD RESTAURANTE SP",
		"ALIMENTAÇÃO":            "ALIMENTACAO",
		"posto ipiranga  123":    "POSTO IPIRANGA 123",
		"PAG*JoãoDaSilva":        "PAG JOAODASILVA",
		"  espaços   demais  ":   "ESPACOS DEMAIS",
	}

	for raw, want := range cases {
		if got := normalizeDescription(raw); got != want {
			t.Errorf("normalizeDescription(%q) = %q, esperado %q", raw, got, want)
		}
	}
}

// Acento e caixa não podem gerar impressões digitais diferentes para a
// mesma linha — senão reimportar o arquivo duplicaria tudo.
func TestEntryFingerprintIsStable(t *testing.T) {
	a := entryFingerprint("2026-08-12", 4500, "IFOOD *RESTAURANTE")
	b := entryFingerprint("2026-08-12", 4500, "ifood  *restaurante")
	if a != b {
		t.Error("descrições equivalentes geraram impressões diferentes")
	}

	if entryFingerprint("2026-08-12", 4500, "IFOOD") == entryFingerprint("2026-08-13", 4500, "IFOOD") {
		t.Error("datas diferentes deveriam gerar impressões diferentes")
	}
	if entryFingerprint("2026-08-12", 4500, "IFOOD") == entryFingerprint("2026-08-12", 4600, "IFOOD") {
		t.Error("valores diferentes deveriam gerar impressões diferentes")
	}
}

func TestSuggestCategory(t *testing.T) {
	rules := []CategoryRule{
		{Pattern: "IFOOD", Category: "alimentacao"},
		{Pattern: "POSTO", Category: "transporte"},
		{Pattern: "UBER", Category: "transporte"},
	}

	if got := suggestCategory("IFOOD *RESTAURANTE SP", rules); got != "alimentacao" {
		t.Errorf("categoria = %q, esperado alimentacao", got)
	}
	if got := suggestCategory("COMPRA DESCONHECIDA", rules); got != "" {
		t.Errorf("sem regra deveria devolver vazio, veio %q", got)
	}
}

// Regra mais específica ganha da mais genérica.
func TestSuggestCategoryPrefersLongerPattern(t *testing.T) {
	rules := []CategoryRule{
		{Pattern: "UBER", Category: "transporte"},
		{Pattern: "UBER EATS", Category: "alimentacao"},
	}

	if got := suggestCategory("UBER EATS SAO PAULO", rules); got != "alimentacao" {
		t.Errorf("categoria = %q, esperado alimentacao (padrão mais específico)", got)
	}
	if got := suggestCategory("UBER TRIP 1234", rules); got != "transporte" {
		t.Errorf("categoria = %q, esperado transporte", got)
	}
}

// A regra aprendida precisa ser o nome do estabelecimento, não a linha
// inteira — senão nunca casaria com a próxima compra no mesmo lugar.
func TestLearnPatternKeepsMerchantDropsNoise(t *testing.T) {
	cases := map[string]string{
		"IFOOD *RESTAURANTE SP 1234":   "IFOOD RESTAURANTE",
		"POSTO IPIRANGA 4455 SAOPAULO": "POSTO IPIRANGA",
		"UBER *TRIP":                   "UBER TRIP",
		"NETFLIX":                      "NETFLIX",
		"12345 SEM NOME":               "12345 SEM NOME",
	}

	for raw, want := range cases {
		if got := learnPattern(raw); got != want {
			t.Errorf("learnPattern(%q) = %q, esperado %q", raw, got, want)
		}
	}
}

func TestFindMatchRequiresExactAmount(t *testing.T) {
	candidates := []MatchCandidate{
		{ID: "a", AmountCents: 4500, OccurredOn: "2026-08-12", Description: "Almoço"},
	}

	if m := findMatch("2026-08-12", 4500, "IFOOD", candidates); m == nil || m.ID != "a" {
		t.Error("deveria casar com o mesmo valor no mesmo dia")
	}
	if m := findMatch("2026-08-12", 4600, "IFOOD", candidates); m != nil {
		t.Error("valor diferente não é o mesmo evento e não deveria casar")
	}
}

// A data do extrato quase nunca bate com a do lançamento manual.
func TestFindMatchToleratesDateDrift(t *testing.T) {
	candidates := []MatchCandidate{
		{ID: "a", AmountCents: 4500, OccurredOn: "2026-08-11", Description: "Almoço"},
	}

	if m := findMatch("2026-08-13", 4500, "IFOOD", candidates); m == nil {
		t.Error("2 dias de diferença deveria casar")
	}
	if m := findMatch("2026-08-20", 4500, "IFOOD", candidates); m != nil {
		t.Error("9 dias de diferença é longe demais para ser o mesmo evento")
	}
}

func TestFindMatchPrefersClosestDate(t *testing.T) {
	candidates := []MatchCandidate{
		{ID: "longe", AmountCents: 4500, OccurredOn: "2026-08-10", Description: "Almoço"},
		{ID: "perto", AmountCents: 4500, OccurredOn: "2026-08-12", Description: "Almoço"},
	}

	m := findMatch("2026-08-12", 4500, "IFOOD", candidates)
	if m == nil || m.ID != "perto" {
		t.Error("deveria escolher o candidato de data mais próxima")
	}
}

// Empate na data é desfeito pela descrição mais parecida.
func TestFindMatchBreaksTieByDescription(t *testing.T) {
	candidates := []MatchCandidate{
		{ID: "outro", AmountCents: 4500, OccurredOn: "2026-08-11", Description: "Estacionamento"},
		{ID: "certo", AmountCents: 4500, OccurredOn: "2026-08-13", Description: "IFOOD pedido"},
	}

	m := findMatch("2026-08-12", 4500, "IFOOD RESTAURANTE", candidates)
	if m == nil || m.ID != "certo" {
		t.Errorf("deveria desempatar pela descrição, veio %v", m)
	}
}

func TestFindMatchWithoutCandidates(t *testing.T) {
	if m := findMatch("2026-08-12", 4500, "IFOOD", nil); m != nil {
		t.Error("sem candidatos não há casamento")
	}
}
