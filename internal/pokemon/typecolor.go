package pokemon

// typeColors maps each of the 18 Pokémon types to its conventional color, as
// used across official and fan Pokédex UIs alike. Colors are plain hex
// strings rather than a terminal-styling library's color type, so this pure
// package stays free of any TUI dependency — see CLAUDE.md's layering rule.
// Callers that need a lipgloss.Color wrap the result themselves.
var typeColors = map[string]string{
	"normal":   "#A8A878",
	"fire":     "#F08030",
	"water":    "#6890F0",
	"electric": "#F8D030",
	"grass":    "#78C850",
	"ice":      "#98D8D8",
	"fighting": "#C03028",
	"poison":   "#A040A0",
	"ground":   "#E0C068",
	"flying":   "#A890F0",
	"psychic":  "#F85888",
	"bug":      "#A8B820",
	"rock":     "#B8A038",
	"ghost":    "#705898",
	"dragon":   "#7038F8",
	"dark":     "#705848",
	"steel":    "#B8B8D0",
	"fairy":    "#EE99AC",
}

// defaultTypeColor is used for any type name not in typeColors, so an
// unrecognized or future type never causes a crash — just a neutral badge.
const defaultTypeColor = "#68A090"

// TypeColor returns the conventional color for a Pokémon type name (as
// returned by PokeAPI, e.g. "fire", "water"), as a hex string. Unknown type
// names return defaultTypeColor rather than an error.
func TypeColor(typeName string) string {
	if c, ok := typeColors[typeName]; ok {
		return c
	}
	return defaultTypeColor
}

// allTypes is the canonical, stable-ordered list of the 18 real Pokémon
// types (i.e. typeColors' keys, in a fixed display order) — Go map
// iteration order is randomized, so anything that needs to list "every
// type" (e.g. the Type Select Screen) must go through AllTypes rather than
// ranging over typeColors directly.
var allTypes = []string{
	"normal", "fire", "water", "electric", "grass", "ice",
	"fighting", "poison", "ground", "flying", "psychic", "bug",
	"rock", "ghost", "dragon", "dark", "steel", "fairy",
}

// AllTypes returns the 18 real Pokémon types, normalized (lowercase, as
// PokeAPI names them) and in a fixed display order. The caller owns the
// returned slice.
func AllTypes() []string {
	out := make([]string, len(allTypes))
	copy(out, allTypes)
	return out
}
