package pokeapi

import (
	"reflect"
	"testing"

	"github.com/leekli/pokedex-go/internal/pokemon"
)

// fireDamageRelations and flyingDamageRelations are Fire's and Flying's
// real type charts (also see fireTypeJSON's fixture, which encodes Fire's
// as JSON) - together they make Charizard, used throughout these tests as
// a real, checkable dual-type example rather than synthetic data.
var (
	fireDamageRelations = TypeDamageRelations{
		DoubleFrom: []string{"water", "ground", "rock"},
		HalfFrom:   []string{"fire", "grass", "ice", "bug", "steel", "fairy"},
	}
	flyingDamageRelations = TypeDamageRelations{
		DoubleFrom: []string{"rock", "electric", "ice"},
		HalfFrom:   []string{"fighting", "bug", "grass"},
		NoFrom:     []string{"ground"},
	}
)

func TestBuildTypeEffectiveness_NoRelationsGivesAllNeutral(t *testing.T) {
	got := BuildTypeEffectiveness()

	if len(got.Weaknesses) != 0 || len(got.Resistances) != 0 || len(got.Immunities) != 0 {
		t.Errorf("BuildTypeEffectiveness() = %+v, want all three empty with no relations to combine", got)
	}
}

// TestBuildTypeEffectiveness_SingleType proves a single type's own damage
// relations pass straight through as the Pokémon's weaknesses, with no
// combination math involved yet.
func TestBuildTypeEffectiveness_SingleType(t *testing.T) {
	got := BuildTypeEffectiveness(fireDamageRelations)

	want := []pokemon.TypeMatchup{
		{Type: "water", Multiplier: 2},
		{Type: "ground", Multiplier: 2},
		{Type: "rock", Multiplier: 2},
	}
	if !reflect.DeepEqual(got.Weaknesses, want) {
		t.Errorf("Weaknesses = %+v, want %+v", got.Weaknesses, want)
	}
	if len(got.Immunities) != 0 {
		t.Errorf("Immunities = %v, want none for a single type with no NoFrom entries", got.Immunities)
	}
}

// TestBuildTypeEffectiveness_DualTypeMultipliesSharedWeakness proves
// Charizard's (Fire/Flying) famous 4x weakness to Rock: Rock appears in
// both types' DoubleFrom, so the multiplier doubles twice rather than once.
func TestBuildTypeEffectiveness_DualTypeMultipliesSharedWeakness(t *testing.T) {
	got := BuildTypeEffectiveness(fireDamageRelations, flyingDamageRelations)

	rock := findMatchup(t, got.Weaknesses, "rock")
	if rock.Multiplier != 4 {
		t.Errorf("rock weakness multiplier = %v, want 4 (Fire and Flying are both weak to Rock)", rock.Multiplier)
	}
}

// TestBuildTypeEffectiveness_ImmunityOverridesWeakness proves Charizard's
// real Ground immunity (from Flying's NoFrom) holds even though Fire alone
// is weak to Ground - a 0x multiplier from either type wins outright over
// any other type's weakness.
func TestBuildTypeEffectiveness_ImmunityOverridesWeakness(t *testing.T) {
	got := BuildTypeEffectiveness(fireDamageRelations, flyingDamageRelations)

	if !reflect.DeepEqual(got.Immunities, []string{"ground"}) {
		t.Errorf("Immunities = %v, want [ground]", got.Immunities)
	}
	for _, w := range got.Weaknesses {
		if w.Type == "ground" {
			t.Errorf("ground appears in Weaknesses = %+v, want it only in Immunities", got.Weaknesses)
		}
	}
}

// TestBuildTypeEffectiveness_CancelsToNeutral proves a weakness from one
// type and a resistance from the other, multiplying out to exactly 1x, are
// omitted from all three buckets - Charizard is famously NOT weak to Ice
// despite Flying being weak to it, because Fire resists it.
func TestBuildTypeEffectiveness_CancelsToNeutral(t *testing.T) {
	got := BuildTypeEffectiveness(fireDamageRelations, flyingDamageRelations)

	for _, w := range got.Weaknesses {
		if w.Type == "ice" {
			t.Errorf("ice appears in Weaknesses = %+v, want it omitted (cancels to 1x neutral)", got.Weaknesses)
		}
	}
	for _, r := range got.Resistances {
		if r.Type == "ice" {
			t.Errorf("ice appears in Resistances = %+v, want it omitted (cancels to 1x neutral)", got.Resistances)
		}
	}
}

// TestBuildTypeEffectiveness_CharizardFullChart locks in Charizard's
// complete, real Fire/Flying type chart end to end, including sort order:
// strongest weakness/resistance first, ties broken by AllTypes' fixed
// order (see BuildTypeEffectiveness's use of sort.SliceStable).
func TestBuildTypeEffectiveness_CharizardFullChart(t *testing.T) {
	got := BuildTypeEffectiveness(fireDamageRelations, flyingDamageRelations)

	wantWeaknesses := []pokemon.TypeMatchup{
		{Type: "rock", Multiplier: 4},
		{Type: "water", Multiplier: 2},
		{Type: "electric", Multiplier: 2},
	}
	wantResistances := []pokemon.TypeMatchup{
		{Type: "grass", Multiplier: 0.25},
		{Type: "bug", Multiplier: 0.25},
		{Type: "fire", Multiplier: 0.5},
		{Type: "fighting", Multiplier: 0.5},
		{Type: "steel", Multiplier: 0.5},
		{Type: "fairy", Multiplier: 0.5},
	}
	wantImmunities := []string{"ground"}

	if !reflect.DeepEqual(got.Weaknesses, wantWeaknesses) {
		t.Errorf("Weaknesses = %+v, want %+v", got.Weaknesses, wantWeaknesses)
	}
	if !reflect.DeepEqual(got.Resistances, wantResistances) {
		t.Errorf("Resistances = %+v, want %+v", got.Resistances, wantResistances)
	}
	if !reflect.DeepEqual(got.Immunities, wantImmunities) {
		t.Errorf("Immunities = %+v, want %+v", got.Immunities, wantImmunities)
	}
}

func findMatchup(t *testing.T, matchups []pokemon.TypeMatchup, typeName string) pokemon.TypeMatchup {
	t.Helper()
	for _, m := range matchups {
		if m.Type == typeName {
			return m
		}
	}
	t.Fatalf("no matchup found for %q in %+v", typeName, matchups)
	return pokemon.TypeMatchup{}
}
