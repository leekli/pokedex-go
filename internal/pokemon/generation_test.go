package pokemon

import "testing"

func TestFormatGeneration(t *testing.T) {
	tests := map[string]string{
		"generation-i":     "Generation I",
		"generation-ii":    "Generation II",
		"generation-iii":   "Generation III",
		"generation-ix":    "Generation IX",
		"":                 "",
		"generation-":      "generation-", // no numeral to format; return unchanged
		"not-a-generation": "not-a-generation",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := FormatGeneration(input); got != want {
				t.Errorf("FormatGeneration(%q) = %q, want %q", input, got, want)
			}
		})
	}
}
