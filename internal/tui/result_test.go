package tui

import "testing"

func TestCapitalize(t *testing.T) {
	tests := map[string]string{
		"pikachu": "Pikachu",
		"":        "",
		"x":       "X",
	}
	for input, want := range tests {
		if got := capitalize(input); got != want {
			t.Errorf("capitalize(%q) = %q, want %q", input, got, want)
		}
	}
}
