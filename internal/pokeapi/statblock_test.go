package pokeapi

import (
	"reflect"
	"testing"

	"github.com/leekli/pokedex-go/internal/pokemon"
)

func TestBuildStatBlock(t *testing.T) {
	p := Pokemon{
		ID:       25,
		Name:     "pikachu",
		HeightDm: 4,
		WeightHg: 60,
		Types:    []string{"electric"},
		Stats: map[string]int{
			"hp":              35,
			"attack":          55,
			"defense":         40,
			"speed":           90,
			"special-attack":  50,
			"special-defense": 50,
		},
	}

	want := pokemon.StatBlock{
		DexNumber:      25,
		Name:           "pikachu",
		Types:          []string{"electric"},
		HeightFeet:     1,
		HeightInches:   4,
		WeightPounds:   13.2,
		HP:             35,
		Attack:         55,
		Defense:        40,
		Speed:          90,
		SpecialAttack:  50,
		SpecialDefense: 50,
	}

	got := BuildStatBlock(p)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildStatBlock(%+v) = %+v, want %+v", p, got, want)
	}
}

// TestBuildStatBlock_StatsAreKeyedByName proves the mapping reads PokeAPI's
// stats by name, not by array position, since PokeAPI does not guarantee
// stats[] ordering. Stats is already a map here, but this test locks in
// that guarantee regardless of how the map was populated.
func TestBuildStatBlock_StatsAreKeyedByName(t *testing.T) {
	reorderedStats := map[string]int{
		"special-defense": 1,
		"speed":           2,
		"hp":              3,
		"defense":         4,
		"special-attack":  5,
		"attack":          6,
	}
	p := Pokemon{ID: 1, Name: "test", Types: []string{"normal"}, Stats: reorderedStats}

	got := BuildStatBlock(p)

	if got.HP != 3 || got.Attack != 6 || got.Defense != 4 || got.Speed != 2 ||
		got.SpecialAttack != 5 || got.SpecialDefense != 1 {
		t.Errorf("BuildStatBlock did not correctly key stats by name: got %+v", got)
	}
}

func TestBuildStatBlock_MissingStatsDefaultToZero(t *testing.T) {
	p := Pokemon{ID: 1, Name: "incomplete", Stats: map[string]int{"hp": 10}}

	got := BuildStatBlock(p)

	if got.HP != 10 {
		t.Errorf("HP = %d, want 10", got.HP)
	}
	if got.Attack != 0 || got.Defense != 0 || got.Speed != 0 || got.SpecialAttack != 0 || got.SpecialDefense != 0 {
		t.Errorf("expected missing stats to default to zero, got %+v", got)
	}
}
