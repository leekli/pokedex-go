package pokemon

import "strings"

// Kind distinguishes whether a Query targets a Pokémon by name or by its
// National Dex Number.
type Kind int

const (
	// Name means Value is an already-normalized PokeAPI name slug.
	Name Kind = iota
	// DexNumber means Value is a National Dex Number, as a decimal string.
	DexNumber
)

// Query is the resolved, normalized form of whatever the user typed on the
// Search Screen: either a name slug or a National Dex Number.
type Query struct {
	Kind  Kind
	Value string
}

// ResolveQuery classifies and normalizes raw Search Screen input into a
// Query. Input consisting only of ASCII digits (after trimming) is treated
// as a National Dex Number; anything else is treated as a name and run
// through NormalizeName. Names that happen to end in digits (e.g.
// "porygon2") are still names, since National Dex input must be digits only.
func ResolveQuery(input string) Query {
	trimmed := strings.TrimSpace(input)
	if trimmed != "" && isAllDigits(trimmed) {
		return Query{Kind: DexNumber, Value: trimmed}
	}
	return Query{Kind: Name, Value: NormalizeName(input)}
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
