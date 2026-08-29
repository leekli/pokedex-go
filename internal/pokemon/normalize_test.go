package pokemon

import (
	"strings"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"already a slug", "pikachu", "pikachu"},
		{"mixed case", "PiKaChu", "pikachu"},
		{"space separated", "Mr Mime", "mr-mime"},
		{"period and space", "Mr. Mime", "mr-mime"},
		{"apostrophe dropped", "Farfetch'd", "farfetchd"},
		{"nidoran female symbol", "Nidoran♀", "nidoran-f"},
		{"nidoran male symbol", "Nidoran♂", "nidoran-m"},
		{"already hyphenated", "ho-oh", "ho-oh"},
		{"extra whitespace", "  porygon z  ", "porygon-z"},
		{"repeated punctuation collapses", "jangmo--o", "jangmo-o"},
		{"leading and trailing punctuation trimmed", "-pikachu-", "pikachu"},
		{"numeric-looking name", "porygon2", "porygon2"},
		{"empty input", "", ""},
		{"only whitespace", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeName(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// FuzzNormalizeName probes NormalizeName with arbitrary input beyond this
// file's hand-picked table cases — the tricky part of this function is
// messy Unicode/punctuation (accents, emoji, mixed scripts, unusual
// whitespace) that a fixed table of real Pokémon names never exercises,
// where a coverage percentage can't prove every edge case was tried. Seeded
// from the table above. Besides "never panics" (which the fuzzer itself
// enforces), every result must be idempotent and match PokeAPI's slug
// alphabet: lowercase alphanumerics separated by single internal hyphens,
// never leading, trailing, or doubled.
func FuzzNormalizeName(f *testing.F) {
	for _, tt := range []struct {
		name  string
		input string
		want  string
	}{
		{"already a slug", "pikachu", "pikachu"},
		{"mixed case", "PiKaChu", "pikachu"},
		{"space separated", "Mr Mime", "mr-mime"},
		{"period and space", "Mr. Mime", "mr-mime"},
		{"apostrophe dropped", "Farfetch'd", "farfetchd"},
		{"nidoran female symbol", "Nidoran♀", "nidoran-f"},
		{"nidoran male symbol", "Nidoran♂", "nidoran-m"},
		{"already hyphenated", "ho-oh", "ho-oh"},
		{"extra whitespace", "  porygon z  ", "porygon-z"},
		{"repeated punctuation collapses", "jangmo--o", "jangmo-o"},
		{"leading and trailing punctuation trimmed", "-pikachu-", "pikachu"},
		{"numeric-looking name", "porygon2", "porygon2"},
		{"empty input", "", ""},
		{"only whitespace", "   ", ""},
	} {
		f.Add(tt.input)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got := NormalizeName(input)

		if again := NormalizeName(got); again != got {
			t.Errorf("NormalizeName(%q) = %q, not idempotent: NormalizeName(%q) = %q", input, got, got, again)
		}
		if strings.HasPrefix(got, "-") || strings.HasSuffix(got, "-") {
			t.Errorf("NormalizeName(%q) = %q has a leading/trailing hyphen", input, got)
		}
		if strings.Contains(got, "--") {
			t.Errorf("NormalizeName(%q) = %q contains a doubled hyphen", input, got)
		}
		for _, r := range got {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				t.Errorf("NormalizeName(%q) = %q contains disallowed rune %q", input, got, r)
			}
		}
	})
}
