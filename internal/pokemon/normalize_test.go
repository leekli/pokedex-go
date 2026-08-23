package pokemon

import "testing"

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
