package service

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

// Cartão típico: fecha dia 25, vence dia 5 do mês seguinte.
func TestChargeDateClosingBeforeDue(t *testing.T) {
	cases := []struct {
		purchase string
		want     string
		why      string
	}{
		{"2026-08-12", "2026-09-05", "antes do fechamento → fatura do mês, vence no mês seguinte"},
		{"2026-08-25", "2026-09-05", "no dia do fechamento ainda entra nessa fatura"},
		{"2026-08-26", "2026-10-05", "um dia depois do fechamento → pula para a fatura seguinte"},
		{"2026-08-28", "2026-10-05", "depois do fechamento"},
		{"2026-12-28", "2027-02-05", "virada de ano"},
	}

	for _, c := range cases {
		got := chargeDate(date(c.purchase), 25, 5)
		if got.Format("2006-01-02") != c.want {
			t.Errorf("compra %s → %s, esperado %s (%s)",
				c.purchase, got.Format("2006-01-02"), c.want, c.why)
		}
	}
}

// Cartão que fecha cedo e vence no mesmo mês: fecha dia 2, vence dia 10.
func TestChargeDateDueAfterClosingSameMonth(t *testing.T) {
	cases := []struct{ purchase, want string }{
		{"2026-08-01", "2026-08-10"},
		{"2026-08-02", "2026-08-10"},
		{"2026-08-03", "2026-09-10"},
	}

	for _, c := range cases {
		got := chargeDate(date(c.purchase), 2, 10)
		if got.Format("2006-01-02") != c.want {
			t.Errorf("compra %s → %s, esperado %s", c.purchase, got.Format("2006-01-02"), c.want)
		}
	}
}

// Meses curtos: dia 31 não existe em todo mês e precisa ser ajustado.
func TestChargeDateShortMonths(t *testing.T) {
	// Fecha 31, vence 28. Compra em 30/01: o fechamento de janeiro é 31,
	// então ainda entra nessa fatura; vencimento 28 <= 31 → mês seguinte.
	if got := chargeDate(date("2026-01-30"), 31, 28).Format("2006-01-02"); got != "2026-02-28" {
		t.Errorf("compra 30/01 com fecha 31/vence 28 → %s, esperado 2026-02-28", got)
	}

	// Fevereiro de 2026 tem 28 dias: um fechamento dia 31 vira 28/02, então
	// uma compra em 28/02 ainda entra na fatura de fevereiro.
	if got := chargeDate(date("2026-02-28"), 31, 15).Format("2006-01-02"); got != "2026-03-15" {
		t.Errorf("compra 28/02 com fecha 31/vence 15 → %s, esperado 2026-03-15", got)
	}

	// Vencimento dia 31 em mês de 30 dias cai no último dia.
	if got := chargeDate(date("2026-03-10"), 20, 31).Format("2006-01-02"); got != "2026-03-31" {
		t.Errorf("vencimento 31 em março → %s, esperado 2026-03-31", got)
	}
	if got := chargeDate(date("2026-04-10"), 20, 31).Format("2006-01-02"); got != "2026-04-30" {
		t.Errorf("vencimento 31 em abril → %s, esperado 2026-04-30 (abril tem 30 dias)", got)
	}
}

func TestSplitInstallmentsExact(t *testing.T) {
	parts := splitInstallments(120000, 12) // R$ 1.200 em 12x
	if len(parts) != 12 {
		t.Fatalf("gerou %d parcelas, esperado 12", len(parts))
	}
	for i, p := range parts {
		if p != 10000 {
			t.Errorf("parcela %d = %d, esperado 10000", i+1, p)
		}
	}
}

// O caso que perde dinheiro se feito ingenuamente.
func TestSplitInstallmentsWithRemainder(t *testing.T) {
	parts := splitInstallments(100000, 3) // R$ 1.000 em 3x
	want := []int64{33334, 33333, 33333}

	for i := range want {
		if parts[i] != want[i] {
			t.Errorf("parcela %d = %d, esperado %d", i+1, parts[i], want[i])
		}
	}

	var sum int64
	for _, p := range parts {
		sum += p
	}
	if sum != 100000 {
		t.Fatalf("soma das parcelas = %d, esperado 100000 — sumiu dinheiro no arredondamento", sum)
	}
}

func TestSplitInstallmentsAlwaysSumsToTotal(t *testing.T) {
	totals := []int64{1, 7, 99, 100, 12345, 999999, 100000}
	counts := []int{1, 2, 3, 5, 7, 12, 24}

	for _, total := range totals {
		for _, n := range counts {
			var sum int64
			for _, p := range splitInstallments(total, n) {
				sum += p
			}
			if sum != total {
				t.Errorf("total %d em %dx somou %d", total, n, sum)
			}
		}
	}
}

func TestInstallmentDatesFollowDueDay(t *testing.T) {
	dates := installmentDates(date("2026-08-12"), 25, 5, 4)
	want := []string{"2026-09-05", "2026-10-05", "2026-11-05", "2026-12-05"}

	for i, w := range want {
		if got := dates[i].Format("2006-01-02"); got != w {
			t.Errorf("parcela %d venceu em %s, esperado %s", i+1, got, w)
		}
	}
}

// Um mês curto no meio não pode arrastar o erro para as parcelas seguintes.
func TestInstallmentDatesShortMonthDoesNotDrift(t *testing.T) {
	// Vencimento dia 31, começando em janeiro.
	dates := installmentDates(date("2026-01-05"), 20, 31, 4)
	want := []string{"2026-01-31", "2026-02-28", "2026-03-31", "2026-04-30"}

	for i, w := range want {
		if got := dates[i].Format("2006-01-02"); got != w {
			t.Errorf("parcela %d venceu em %s, esperado %s", i+1, got, w)
		}
	}
}

func TestInstallmentDatesCrossYear(t *testing.T) {
	dates := installmentDates(date("2026-11-10"), 25, 5, 3)
	want := []string{"2026-12-05", "2027-01-05", "2027-02-05"}

	for i, w := range want {
		if got := dates[i].Format("2006-01-02"); got != w {
			t.Errorf("parcela %d venceu em %s, esperado %s", i+1, got, w)
		}
	}
}
