package pokemon

import "strings"

// FormatGeneration renders a PokeAPI generation resource name (e.g.
// "generation-i") as the Type Roster Screen's display form (e.g.
// "Generation I"). Any name that doesn't fit that shape is returned
// unchanged, so an unrecognized future PokeAPI name degrades to plain text
// rather than mangled output.
func FormatGeneration(name string) string {
	const prefix = "generation-"
	numeral, ok := strings.CutPrefix(name, prefix)
	if !ok || numeral == "" {
		return name
	}
	return "Generation " + strings.ToUpper(numeral)
}
