package service

import "time"

// clampDay builds a date, capping the day at the month's last day: a card
// closing on the 31st closes on the 28th in February. Month values outside
// 1..12 are normalized by time.Date, so callers can do plain month arithmetic
// instead of AddDate — which would turn "31 Jan + 1 month" into March 3rd.
func clampDay(year int, month time.Month, day int) time.Time {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// chargeDate answers: a purchase made on this date, with this card, is paid when?
//
// Two steps. First, which bill catches the purchase: the one closing this month
// if the purchase happened on or before the closing day, otherwise next month's.
// Second, when that bill is due: in the same month it closed when the due day
// comes after the closing day, otherwise in the following month.
//
// Example — card closing on the 25th, due on the 5th:
//
//	purchase 12/08 → closes 25/08 → due 05/09
//	purchase 28/08 → closes 25/09 → due 05/10
func chargeDate(purchasedOn time.Time, closingDay, dueDay int) time.Time {
	year, month := purchasedOn.Year(), purchasedOn.Month()

	closing := clampDay(year, month, closingDay)
	if purchasedOn.Day() > closing.Day() {
		closing = clampDay(year, month+1, closingDay)
	}

	dueYear, dueMonth := closing.Year(), closing.Month()
	if dueDay <= closingDay {
		// O vencimento vem depois do fechamento, já no mês seguinte.
		dueMonth++
	}

	return clampDay(dueYear, dueMonth, dueDay)
}

// splitInstallments divides a total into n installments without losing a cent.
// The remainder goes to the first installment, matching how Brazilian issuers
// present it: R$ 1.000 in 3x becomes 333,34 + 333,33 + 333,33.
func splitInstallments(totalCents int64, n int) []int64 {
	if n <= 1 {
		return []int64{totalCents}
	}

	base := totalCents / int64(n)
	remainder := totalCents - base*int64(n)

	parts := make([]int64, n)
	for i := range parts {
		parts[i] = base
	}
	parts[0] += remainder
	return parts
}

// installmentDates returns the charge date of each installment: the first
// follows the card's billing rule, the rest fall on the same due day of the
// following months. Each date is derived from the base month rather than from
// the previous one, so a short month never drags the error forward.
func installmentDates(purchasedOn time.Time, closingDay, dueDay, n int) []time.Time {
	first := chargeDate(purchasedOn, closingDay, dueDay)
	baseYear, baseMonth := first.Year(), first.Month()

	dates := make([]time.Time, n)
	for i := 0; i < n; i++ {
		dates[i] = clampDay(baseYear, baseMonth+time.Month(i), dueDay)
	}
	return dates
}
