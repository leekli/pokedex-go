package pokeapi

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

// fireTypeJSON mirrors real PokeAPI's /type/fire shape closely enough to
// exercise GetPokemonByType: a mix of real, dex-numbered Pokémon (in
// deliberately unsorted order, to prove the client sorts rather than
// trusting response order) and non-default varieties (mega evolutions),
// which must be filtered out.
const fireTypeJSON = `{
	"pokemon": [
		{"pokemon": {"name": "charizard", "url": "https://pokeapi.co/api/v2/pokemon/6/"}},
		{"pokemon": {"name": "charmander", "url": "https://pokeapi.co/api/v2/pokemon/4/"}},
		{"pokemon": {"name": "charizard-mega-x", "url": "https://pokeapi.co/api/v2/pokemon/10034/"}},
		{"pokemon": {"name": "vulpix", "url": "https://pokeapi.co/api/v2/pokemon/37/"}}
	]
}`

func TestGetPokemonByType_FiltersAndSortsByDexNumber(t *testing.T) {
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/type/fire": {status: http.StatusOK, body: fireTypeJSON},
	})

	got, err := client.GetPokemonByType(context.Background(), "fire")
	if err != nil {
		t.Fatalf("GetPokemonByType returned error: %v", err)
	}

	want := []TypeRosterEntry{
		{DexNumber: 4, Name: "charmander"},
		{DexNumber: 6, Name: "charizard"},
		{DexNumber: 37, Name: "vulpix"},
	}
	if len(got) != len(want) {
		t.Fatalf("GetPokemonByType returned %d entries, want %d (mega evolution should have been filtered out): %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestGetPokemonByType_NotFound(t *testing.T) {
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/type/notarealtype": {status: http.StatusNotFound},
	})

	_, err := client.GetPokemonByType(context.Background(), "notarealtype")

	var lookupErr *LookupError
	if !errors.As(err, &lookupErr) {
		t.Fatalf("GetPokemonByType error = %v (%T), want *LookupError", err, err)
	}
}

func TestGetPokemonByType_ServiceError(t *testing.T) {
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/type/fire": {status: http.StatusInternalServerError},
	})

	_, err := client.GetPokemonByType(context.Background(), "fire")

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("GetPokemonByType error = %v (%T), want *ServiceError", err, err)
	}
}

func TestPokemonIDFromURL(t *testing.T) {
	tests := []struct {
		url    string
		wantID int
		wantOK bool
	}{
		{"https://pokeapi.co/api/v2/pokemon/25/", 25, true},
		{"https://pokeapi.co/api/v2/pokemon/10100/", 10100, true},
		{"https://pokeapi.co/api/v2/pokemon/1/", 1, true},
		{"https://pokeapi.co/api/v2/pokemon/not-a-number/", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		id, ok := pokemonIDFromURL(tt.url)
		if id != tt.wantID || ok != tt.wantOK {
			t.Errorf("pokemonIDFromURL(%q) = (%d, %v), want (%d, %v)", tt.url, id, ok, tt.wantID, tt.wantOK)
		}
	}
}
