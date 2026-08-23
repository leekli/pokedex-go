package pokeapi

import "github.com/leekli/pokedex-go/internal/pokemon"

// BuildStatBlock maps a Pokemon into the pokemon.StatBlock shown on the
// Result Screen, converting units to imperial and reading base stats by
// name — PokeAPI does not guarantee the order of the stats array, so index
// position must never be relied on.
func BuildStatBlock(p Pokemon) pokemon.StatBlock {
	feet, inches := pokemon.DecimetresToFeetInches(p.HeightDm)

	return pokemon.StatBlock{
		DexNumber:      p.ID,
		Name:           p.Name,
		Types:          p.Types,
		HeightFeet:     feet,
		HeightInches:   inches,
		WeightPounds:   pokemon.HectogramsToPounds(p.WeightHg),
		HP:             p.Stats["hp"],
		Attack:         p.Stats["attack"],
		Defense:        p.Stats["defense"],
		Speed:          p.Stats["speed"],
		SpecialAttack:  p.Stats["special-attack"],
		SpecialDefense: p.Stats["special-defense"],
	}
}
