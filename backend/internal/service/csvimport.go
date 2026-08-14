package service

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CSVMapping says which column holds what. Every bank exports a different
// layout, so the mapping is data instead of code — one parser serves all.
type CSVMapping struct {
	DateColumn        int    `json:"date_column"`
	AmountColumn      int    `json:"amount_column"`
	DescriptionColumn int    `json:"description_column"`
	DateFormat        string `json:"date_format"`
	// Extratos costumam trazer despesa como valor negativo. Alguns cartões
	// invertem: a compra vem positiva e o pagamento negativo.
	NegativeIsExpense bool   `json:"negative_is_expense"`
	Delimiter         string `json:"delimiter"`
	HasHeader         bool   `json:"has_header"`
}

// ParsedEntry is one statement line, before any reconciliation.
type ParsedEntry struct {
	OccurredOn     string `json:"occurred_on"`
	AmountCents    int64  `json:"amount_cents"`
	Kind           string `json:"kind"`
	RawDescription string `json:"raw_description"`
}

// ─── Valor monetário ──────────────────────────────────────────────────────────

// parseAmountCents reads the many shapes a statement uses for money and
// returns integer cents, keeping the sign.
//
// The hard part is telling "1.234" (Brazilian thousands → 1234,00) from
// "1.234" (decimal → 1,234). The rule: when both separators appear, the last
// one is the decimal. When only one appears, it is a decimal separator only
// if exactly two digits follow it.
func parseAmountCents(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("valor vazio")
	}

	// Remove símbolo de moeda, espaços não separáveis e parênteses de negativo.
	negative := strings.HasPrefix(s, "-") || (strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")"))
	for _, noise := range []string{"R$", "r$", "(", ")", "-", "+", " ", " "} {
		s = strings.ReplaceAll(s, noise, "")
	}
	if s == "" {
		return 0, fmt.Errorf("valor sem dígitos: %q", raw)
	}

	lastDot := strings.LastIndex(s, ".")
	lastComma := strings.LastIndex(s, ",")

	var decimalSep string
	switch {
	case lastDot >= 0 && lastComma >= 0:
		if lastComma > lastDot {
			decimalSep = ","
		} else {
			decimalSep = "."
		}
	case lastComma >= 0:
		// Exatamente três dígitos depois é separador de milhar ("1.234");
		// qualquer outra quantidade é casa decimal ("45,9" e "45,90").
		if len(s)-lastComma-1 != 3 {
			decimalSep = ","
		}
	case lastDot >= 0:
		if len(s)-lastDot-1 != 3 {
			decimalSep = "."
		}
	}

	var intPart, fracPart string
	if decimalSep == "" {
		// Sem casas decimais: tudo é parte inteira, separadores são milhar.
		intPart = strings.NewReplacer(".", "", ",", "").Replace(s)
		fracPart = "00"
	} else {
		idx := strings.LastIndex(s, decimalSep)
		intPart = strings.NewReplacer(".", "", ",", "").Replace(s[:idx])
		fracPart = s[idx+1:]
	}

	if intPart == "" {
		intPart = "0"
	}
	// Normaliza para exatamente dois dígitos de centavos.
	switch {
	case len(fracPart) == 1:
		fracPart += "0"
	case len(fracPart) > 2:
		fracPart = fracPart[:2]
	case len(fracPart) == 0:
		fracPart = "00"
	}

	units, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("valor inválido %q: %w", raw, err)
	}
	cents, err := strconv.ParseInt(fracPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("centavos inválidos em %q: %w", raw, err)
	}

	total := units*100 + cents
	if negative {
		total = -total
	}
	return total, nil
}

// ─── Data ─────────────────────────────────────────────────────────────────────

// dateFormats are tried in order when the mapping does not name one.
var dateFormats = []string{
	"02/01/2006", "2006-01-02", "02-01-2006", "02/01/06", "01/02/2006",
}

func parseStatementDate(raw, preferred string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("data vazia")
	}

	formats := dateFormats
	if preferred != "" {
		formats = append([]string{preferred}, dateFormats...)
	}

	for _, f := range formats {
		if d, err := time.Parse(f, s); err == nil {
			return d.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("data não reconhecida: %q", raw)
}

// ─── Detecção automática ──────────────────────────────────────────────────────

// detectDelimiter picks the separator by counting candidates in the first line.
func detectDelimiter(content string) string {
	firstLine := content
	if idx := strings.IndexAny(content, "\r\n"); idx >= 0 {
		firstLine = content[:idx]
	}

	best, bestCount := ",", strings.Count(firstLine, ",")
	for _, d := range []string{";", "\t"} {
		if c := strings.Count(firstLine, d); c > bestCount {
			best, bestCount = d, c
		}
	}
	return best
}

// headerHints maps common Brazilian statement headers to their role.
var headerHints = map[string][]string{
	"date":        {"DATA", "DT", "DATA LANCAMENTO", "DATA DA COMPRA", "DATE"},
	"amount":      {"VALOR", "VALOR R", "QUANTIA", "AMOUNT", "MONTANTE", "VALOR BRL"},
	"description": {"DESCRICAO", "HISTORICO", "LANCAMENTO", "ESTABELECIMENTO", "TITULO", "MEMO", "DESCRIPTION"},
}

// SniffMapping guesses the layout from the header row so the user usually has
// nothing to configure. Returns the mapping and whether every column was found.
func SniffMapping(content string) (CSVMapping, bool) {
	delimiter := detectDelimiter(content)
	mapping := CSVMapping{
		DateColumn: -1, AmountColumn: -1, DescriptionColumn: -1,
		Delimiter: delimiter, NegativeIsExpense: true, HasHeader: true,
	}

	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = rune(delimiter[0])
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return mapping, false
	}

	for i, cell := range header {
		normalized := normalizeDescription(cell)
		for role, hints := range headerHints {
			for _, hint := range hints {
				if normalized != hint {
					continue
				}
				switch role {
				case "date":
					if mapping.DateColumn < 0 {
						mapping.DateColumn = i
					}
				case "amount":
					if mapping.AmountColumn < 0 {
						mapping.AmountColumn = i
					}
				case "description":
					if mapping.DescriptionColumn < 0 {
						mapping.DescriptionColumn = i
					}
				}
			}
		}
	}

	complete := mapping.DateColumn >= 0 && mapping.AmountColumn >= 0 && mapping.DescriptionColumn >= 0
	return mapping, complete
}

// ─── Parser ───────────────────────────────────────────────────────────────────

// ParseCSVStatement turns raw file content into entries. Lines that cannot be
// read are skipped and reported, so one malformed row never costs the import.
func ParseCSVStatement(content string, mapping CSVMapping) ([]ParsedEntry, []string, error) {
	if mapping.DateColumn < 0 || mapping.AmountColumn < 0 || mapping.DescriptionColumn < 0 {
		return nil, nil, fmt.Errorf("mapeamento incompleto: informe as colunas de data, valor e descrição")
	}

	delimiter := mapping.Delimiter
	if delimiter == "" {
		delimiter = detectDelimiter(content)
	}

	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = rune(delimiter[0])
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("lendo CSV: %w", err)
	}

	widest := mapping.DateColumn
	for _, c := range []int{mapping.AmountColumn, mapping.DescriptionColumn} {
		if c > widest {
			widest = c
		}
	}

	entries := []ParsedEntry{}
	skipped := []string{}

	for i, row := range records {
		if mapping.HasHeader && i == 0 {
			continue
		}
		if len(row) <= widest {
			if strings.TrimSpace(strings.Join(row, "")) != "" {
				skipped = append(skipped, fmt.Sprintf("linha %d: colunas de menos", i+1))
			}
			continue
		}

		date, err := parseStatementDate(row[mapping.DateColumn], mapping.DateFormat)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("linha %d: %v", i+1, err))
			continue
		}

		amount, err := parseAmountCents(row[mapping.AmountColumn])
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("linha %d: %v", i+1, err))
			continue
		}
		if amount == 0 {
			skipped = append(skipped, fmt.Sprintf("linha %d: valor zero", i+1))
			continue
		}

		kind := "expense"
		if mapping.NegativeIsExpense {
			if amount > 0 {
				kind = "income"
			}
		} else if amount < 0 {
			kind = "income"
		}
		if amount < 0 {
			amount = -amount
		}

		description := strings.TrimSpace(row[mapping.DescriptionColumn])
		if description == "" {
			description = "Lançamento sem descrição"
		}

		entries = append(entries, ParsedEntry{
			OccurredOn:     date,
			AmountCents:    amount,
			Kind:           kind,
			RawDescription: description,
		})
	}

	return entries, skipped, nil
}
