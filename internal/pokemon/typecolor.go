package pokemon

import "github.com/charmbracelet/lipgloss"

// typeColors maps each of the 18 Pokémon types to its conventional color, as
// used across official and fan Pokédex UIs alike.
var typeColors = map[string]lipgloss.Color{
	"normal":   lipgloss.Color("#A8A878"),
	"fire":     lipgloss.Color("#F08030"),
	"water":    lipgloss.Color("#6890F0"),
	"electric": lipgloss.Color("#F8D030"),
	"grass":    lipgloss.Color("#78C850"),
	"ice":      lipgloss.Color("#98D8D8"),
	"fighting": lipgloss.Color("#C03028"),
	"poison":   lipgloss.Color("#A040A0"),
	"ground":   lipgloss.Color("#E0C068"),
	"flying":   lipgloss.Color("#A890F0"),
	"psychic":  lipgloss.Color("#F85888"),
	"bug":      lipgloss.Color("#A8B820"),
	"rock":     lipgloss.Color("#B8A038"),
	"ghost":    lipgloss.Color("#705898"),
	"dragon":   lipgloss.Color("#7038F8"),
	"dark":     lipgloss.Color("#705848"),
	"steel":    lipgloss.Color("#B8B8D0"),
	"fairy":    lipgloss.Color("#EE99AC"),
}

// defaultTypeColor is used for any type name not in typeColors, so an
// unrecognized or future type never causes a crash — just a neutral badge.
const defaultTypeColor = lipgloss.Color("#68A090")

// TypeColor returns the conventional color for a Pokémon type name (as
// returned by PokeAPI, e.g. "fire", "water"). Unknown type names return
// defaultTypeColor rather than an error.
func TypeColor(typeName string) lipgloss.Color {
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
