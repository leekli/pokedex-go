// Package pokemon holds pure domain logic for pokedex-go: input normalization,
// query classification, unit conversion, type colors, and stat-block mapping.
// Nothing in this package performs I/O.
package pokemon

import "strings"

// genderSymbolReplacer rewrites the Nidoran gender symbols to the suffix
// PokeAPI's slugs use (nidoran-f / nidoran-m) before general normalization runs.
var genderSymbolReplacer = strings.NewReplacer("♀", "-f", "♂", "-m")

// NormalizeName converts free-form user input for a Pokémon name into the
// slug PokeAPI expects: lowercase, trimmed, spaces and punctuation collapsed
// to single hyphens, apostrophes dropped (Farfetch'd -> farfetchd), and the
// Nidoran gender symbols rewritten to their -f/-m suffix.
func NormalizeName(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	s = genderSymbolReplacer.Replace(s)

	var b strings.Builder
	lastWasHyphen := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastWasHyphen = false
		case r == '\'':
			// Dropped, not treated as a word boundary.
		default:
			// Whitespace, literal hyphens, and other punctuation all collapse
			// to a single hyphen boundary.
			if !lastWasHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastWasHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
