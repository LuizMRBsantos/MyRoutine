package service

import "testing"

// O ponto onde bug de dinheiro nasce: cada banco escreve valor de um jeito.
func TestParseAmountCents(t *testing.T) {
	cases := map[string]int64{
		"45,90":       4590,  // formato brasileiro
		"45.90":       4590,  // formato americano
		"1.234,56":    123456, // ponto como milhar, vírgula decimal
		"1,234.56":    123456, // vírgula como milhar, ponto decimal
		"1.234":       123400, // três dígitos após o ponto → milhar, não decimal
		"1234":        123400,
		"R$ 45,90":    4590,
		"-45,90":      -4590,
		"(45,90)":     -4590, // negativo entre parênteses
		"0,05":        5,
		"45,9":        4590,  // uma casa decimal
		"10.000,00":   1000000,
	}

	for raw, want := range cases {
		got, err := parseAmountCents(raw)
		if err != nil {
			t.Errorf("parseAmountCents(%q) devolveu erro: %v", raw, err)
			continue
		}
		if got != want {
			t.Errorf("parseAmountCents(%q) = %d, esperado %d", raw, got, want)
		}
	}
}

func TestParseAmountCentsRejectsGarbage(t *testing.T) {
	for _, raw := range []string{"", "   ", "abc", "R$"} {
		if _, err := parseAmountCents(raw); err == nil {
			t.Errorf("parseAmountCents(%q) deveria falhar", raw)
		}
	}
}

func TestParseStatementDate(t *testing.T) {
	cases := map[string]string{
		"12/08/2026": "2026-08-12",
		"2026-08-12": "2026-08-12",
		"12-08-2026": "2026-08-12",
	}

	for raw, want := range cases {
		got, err := parseStatementDate(raw, "")
		if err != nil {
			t.Errorf("parseStatementDate(%q): %v", raw, err)
			continue
		}
		if got != want {
			t.Errorf("parseStatementDate(%q) = %q, esperado %q", raw, got, want)
		}
	}

	if _, err := parseStatementDate("não é data", ""); err == nil {
		t.Error("data inválida deveria falhar")
	}
}

// Formato ambíguo: 01/02/2026 é 1º de fevereiro no Brasil. O formato
// preferido informado pelo usuário tem que ganhar.
func TestParseStatementDatePreferredFormatWins(t *testing.T) {
	got, err := parseStatementDate("01/02/2026", "01/02/2006") // mês/dia
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if got != "2026-01-02" {
		t.Errorf("com formato mês/dia = %q, esperado 2026-01-02", got)
	}
}

func TestDetectDelimiter(t *testing.T) {
	if d := detectDelimiter("Data;Valor;Descrição\n12/08/2026;45,90;IFOOD"); d != ";" {
		t.Errorf("delimitador = %q, esperado ;", d)
	}
	if d := detectDelimiter("Data,Valor,Descrição\n12/08/2026,45.90,IFOOD"); d != "," {
		t.Errorf("delimitador = %q, esperado ,", d)
	}
}

func TestSniffMappingFindsBrazilianHeaders(t *testing.T) {
	content := "Data;Valor;Histórico\n12/08/2026;-45,90;IFOOD *RESTAURANTE"

	mapping, complete := SniffMapping(content)
	if !complete {
		t.Fatal("deveria reconhecer todas as colunas")
	}
	if mapping.DateColumn != 0 || mapping.AmountColumn != 1 || mapping.DescriptionColumn != 2 {
		t.Errorf("mapeamento errado: %+v", mapping)
	}
	if mapping.Delimiter != ";" {
		t.Errorf("delimitador = %q, esperado ;", mapping.Delimiter)
	}
}

func TestSniffMappingReportsIncomplete(t *testing.T) {
	content := "Coluna A;Coluna B;Coluna C\n1;2;3"
	if _, complete := SniffMapping(content); complete {
		t.Error("cabeçalho irreconhecível não deveria ser dado como completo")
	}
}

func TestParseCSVStatement(t *testing.T) {
	content := `Data;Valor;Histórico
12/08/2026;-45,90;IFOOD *RESTAURANTE
13/08/2026;-120,00;POSTO IPIRANGA
01/08/2026;5.000,00;SALARIO`

	mapping, complete := SniffMapping(content)
	if !complete {
		t.Fatal("mapeamento deveria ser detectado")
	}

	entries, skipped, err := ParseCSVStatement(content, mapping)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if len(skipped) != 0 {
		t.Errorf("pulou linhas inesperadamente: %v", skipped)
	}
	if len(entries) != 3 {
		t.Fatalf("leu %d lançamentos, esperado 3", len(entries))
	}

	first := entries[0]
	if first.OccurredOn != "2026-08-12" || first.AmountCents != 4590 || first.Kind != "expense" {
		t.Errorf("primeiro lançamento errado: %+v", first)
	}
	if first.RawDescription != "IFOOD *RESTAURANTE" {
		t.Errorf("descrição = %q", first.RawDescription)
	}

	// Valor positivo num extrato onde negativo é despesa → receita.
	last := entries[2]
	if last.Kind != "income" || last.AmountCents != 500000 {
		t.Errorf("salário deveria ser receita de 500000: %+v", last)
	}
}

// Uma linha ruim não pode custar a importação inteira.
func TestParseCSVStatementSkipsBadRowsAndKeepsGoing(t *testing.T) {
	content := `Data;Valor;Histórico
12/08/2026;-45,90;IFOOD
linha quebrada
xx/xx/xxxx;-10,00;DATA RUIM
13/08/2026;abc;VALOR RUIM
14/08/2026;-30,00;VALIDA`

	mapping := CSVMapping{
		DateColumn: 0, AmountColumn: 1, DescriptionColumn: 2,
		Delimiter: ";", NegativeIsExpense: true, HasHeader: true,
	}

	entries, skipped, err := ParseCSVStatement(content, mapping)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("leu %d lançamentos válidos, esperado 2", len(entries))
	}
	if len(skipped) != 3 {
		t.Errorf("reportou %d linhas puladas, esperado 3: %v", len(skipped), skipped)
	}
}

func TestParseCSVStatementRequiresMapping(t *testing.T) {
	_, _, err := ParseCSVStatement("a;b", CSVMapping{DateColumn: -1})
	if err == nil {
		t.Error("mapeamento incompleto deveria falhar")
	}
}
