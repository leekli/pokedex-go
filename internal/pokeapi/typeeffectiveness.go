package pokeapi

import (
	"sort"

	"github.com/leekli/pokedex-go/internal/pokemon"
)

// BuildTypeEffectiveness combines a Pokémon's own types' TypeDamageRelations
// (one per type - one for a single-type Pokémon, two for dual-type) into
// its overall Weaknesses & Resistances, by multiplying each type's
// contribution together: e.g. Charizard (Fire/Flying) is 4x weak to Rock
// because Rock appears in both Fire's and Flying's DoubleFrom list. See
// docs/adr/0004-all-or-nothing-type-effectiveness.md for why the caller
// (lookupCmd) treats any one relation fetch failing as the whole result
// being unavailable, rather than combining whatever succeeded.
func BuildTypeEffectiveness(relations ...TypeDamageRelations) pokemon.TypeEffectiveness {
	allTypes := pokemon.AllTypes()

	multiplier := make(map[string]float64, len(allTypes))
	for _, t := range allTypes {
		multiplier[t] = 1
	}

	for _, r := range relations {
		for _, t := range r.DoubleFrom {
			multiplier[t] *= 2
		}
		for _, t := range r.HalfFrom {
			multiplier[t] *= 0.5
		}
		for _, t := range r.NoFrom {
			multiplier[t] = 0
		}
	}

	var result pokemon.TypeEffectiveness
	for _, t := range allTypes {
		switch m := multiplier[t]; {
		case m == 0:
			result.Immunities = append(result.Immunities, t)
		case m > 1:
			result.Weaknesses = append(result.Weaknesses, pokemon.TypeMatchup{Type: t, Multiplier: m})
		case m < 1:
			result.Resistances = append(result.Resistances, pokemon.TypeMatchup{Type: t, Multiplier: m})
		}
		// m == 1: neutral, omitted from all three.
	}

	// SliceStable (not Slice) so types tied at the same multiplier keep
	// AllTypes' fixed relative order rather than an unspecified one.
	sort.SliceStable(result.Weaknesses, func(i, j int) bool {
		return result.Weaknesses[i].Multiplier > result.Weaknesses[j].Multiplier
	})
	sort.SliceStable(result.Resistances, func(i, j int) bool {
		return result.Resistances[i].Multiplier < result.Resistances[j].Multiplier
	})
	sort.Strings(result.Immunities)

	return result
}
