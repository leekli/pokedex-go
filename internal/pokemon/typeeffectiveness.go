package pokemon

// TypeMatchup is one attacking type and the overall multiplier it deals to
// a Pokémon, after combining all of that Pokémon's own types together.
type TypeMatchup struct {
	Type       string
	Multiplier float64
}

// TypeEffectiveness is a Pokémon's Weaknesses & Resistances: every
// attacking type that deals more than normal damage (Weaknesses), less
// than normal damage (Resistances), or none at all (Immunities). A type
// that deals normal (1x) damage is omitted from all three - a plain data
// holder, the same convention as StatBlock; see pokeapi.BuildTypeEffectiveness
// for how it's built from PokeAPI's raw per-type damage relations.
type TypeEffectiveness struct {
	Weaknesses  []TypeMatchup
	Resistances []TypeMatchup
	Immunities  []string
}
