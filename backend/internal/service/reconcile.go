package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// ─── Normalização ─────────────────────────────────────────────────────────────

var nonAlphanumeric = regexp.MustCompile(`[^A-Z0-9 ]+`)
var repeatedSpaces = regexp.MustCompile(` +`)

// normalizeDescription turns a raw statement line into something comparable:
// "IFOOD  *RESTAURANTE-SP  12/08" becomes "IFOOD RESTAURANTE SP 12 08".
// Accents are stripped so "ALIMENTAÇÃO" and "ALIMENTACAO" match.
func normalizeDescription(raw string) string {
	stripAccents := transform.Chain(
		norm.NFD,
		runes.Remove(runes.In(unicode.Mn)),
		norm.NFC,
	)
	folded, _, err := transform.String(stripAccents, raw)
	if err != nil {
		folded = raw
	}

	upper := strings.ToUpper(folded)
	cleaned := nonAlphanumeric.ReplaceAllString(upper, " ")
	return strings.TrimSpace(repeatedSpaces.ReplaceAllString(cleaned, " "))
}

// entryFingerprint identifies a statement line so re-importing the same file
// never brings it back. Built from the normalized description plus date and
// amount — the three things that stay stable across exports of the same data.
func entryFingerprint(occurredOn string, amountCents int64, rawDescription string) string {
	seed := fmt.Sprintf("%s|%d|%s", occurredOn, amountCents, normalizeDescription(rawDescription))
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])
}

// ─── Sugestão de categoria ────────────────────────────────────────────────────

// CategoryRule maps a chunk of a description to a category.
type CategoryRule struct {
	ID       string `json:"id"`
	Pattern  string `json:"pattern"`
	Category string `json:"category"`
	HitCount int    `json:"hit_count"`
}

// suggestCategory picks the rule whose pattern appears in the description.
// Longer patterns win: "UBER EATS" should beat a broader "UBER".
func suggestCategory(rawDescription string, rules []CategoryRule) string {
	normalized := normalizeDescription(rawDescription)

	best := ""
	bestLen := 0
	for _, rule := range rules {
		pattern := normalizeDescription(rule.Pattern)
		if pattern == "" || !strings.Contains(normalized, pattern) {
			continue
		}
		if len(pattern) > bestLen {
			best, bestLen = rule.Category, len(pattern)
		}
	}
	return best
}

// learnPattern derives the rule to remember when the user categorizes a line.
// It keeps the leading words of the description — the merchant name — and drops
// the trailing noise (store number, city, date) that never repeats.
func learnPattern(rawDescription string) string {
	words := strings.Fields(normalizeDescription(rawDescription))

	kept := make([]string, 0, 2)
	for _, w := range words {
		// Números isolados são ruído: número da loja, parcela, data.
		if _, err := strconv.Atoi(w); err == nil {
			break
		}
		kept = append(kept, w)
		if len(kept) == 2 {
			break
		}
	}

	if len(kept) == 0 {
		return normalizeDescription(rawDescription)
	}
	return strings.Join(kept, " ")
}

// ─── Casamento com o que já existe ────────────────────────────────────────────

// MatchCandidate is an existing transaction an imported line might duplicate.
type MatchCandidate struct {
	ID          string
	AmountCents int64
	OccurredOn  string
	Description string
}

// matchWindowDays is how far apart the two dates may be. A card statement
// rarely carries the same date you typed: the purchase is logged when it
// happens, the statement when it is processed.
const matchWindowDays = 3

// findMatch looks for the existing transaction an imported line most likely
// duplicates. The amount must be exact — a different amount is a different
// event — and the date must fall inside the window. Among candidates, the
// closest date wins, then the most similar description.
func findMatch(occurredOn string, amountCents int64, rawDescription string, candidates []MatchCandidate) *MatchCandidate {
	entryDate, err := time.Parse("2006-01-02", occurredOn)
	if err != nil {
		return nil
	}
	normalized := normalizeDescription(rawDescription)

	var best *MatchCandidate
	bestDistance := matchWindowDays + 1
	bestOverlap := -1

	for i := range candidates {
		c := candidates[i]
		if c.AmountCents != amountCents {
			continue
		}

		candidateDate, err := time.Parse("2006-01-02", c.OccurredOn)
		if err != nil {
			continue
		}

		distance := int(entryDate.Sub(candidateDate).Hours() / 24)
		if distance < 0 {
			distance = -distance
		}
		if distance > matchWindowDays {
			continue
		}

		overlap := sharedWordCount(normalized, normalizeDescription(c.Description))

		if distance < bestDistance || (distance == bestDistance && overlap > bestOverlap) {
			best = &candidates[i]
			bestDistance = distance
			bestOverlap = overlap
		}
	}

	return best
}

// sharedWordCount counts words present in both descriptions, ignoring very
// short ones that carry no signal.
func sharedWordCount(a, b string) int {
	inB := make(map[string]bool)
	for _, w := range strings.Fields(b) {
		if len(w) > 2 {
			inB[w] = true
		}
	}

	shared := 0
	for _, w := range strings.Fields(a) {
		if len(w) > 2 && inB[w] {
			shared++
		}
	}
	return shared
}
