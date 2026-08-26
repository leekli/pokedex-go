package pokeapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// fireTypeJSON mirrors real PokeAPI's /type/fire shape closely enough to
// exercise GetPokemonByType: a mix of real, dex-numbered Pokémon (in
// deliberately unsorted order, to prove the client sorts rather than
// trusting response order) and non-default varieties (mega evolutions),
// which must be filtered out. Its damage_relations is Fire's real type
// chart, reused by typeeffectiveness_test.go's Charizard (Fire/Flying)
// cases.
const fireTypeJSON = `{
	"pokemon": [
		{"pokemon": {"name": "charizard", "url": "https://pokeapi.co/api/v2/pokemon/6/"}},
		{"pokemon": {"name": "charmander", "url": "https://pokeapi.co/api/v2/pokemon/4/"}},
		{"pokemon": {"name": "charizard-mega-x", "url": "https://pokeapi.co/api/v2/pokemon/10034/"}},
		{"pokemon": {"name": "vulpix", "url": "https://pokeapi.co/api/v2/pokemon/37/"}}
	],
	"damage_relations": {
		"double_damage_from": [{"name": "water"}, {"name": "ground"}, {"name": "rock"}],
		"half_damage_from": [{"name": "fire"}, {"name": "grass"}, {"name": "ice"}, {"name": "bug"}, {"name": "steel"}, {"name": "fairy"}],
		"no_damage_from": []
	}
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

// TestGetPokemonByType_CachesAcrossCalls proves a second GetPokemonByType
// call for the same type is served from the Client's cache rather than
// hitting PokeAPI again.
func TestGetPokemonByType_CachesAcrossCalls(t *testing.T) {
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/type/fire", func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fireTypeJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	first, err := client.GetPokemonByType(context.Background(), "fire")
	if err != nil {
		t.Fatalf("first GetPokemonByType returned error: %v", err)
	}
	second, err := client.GetPokemonByType(context.Background(), "fire")
	if err != nil {
		t.Fatalf("second GetPokemonByType returned error: %v", err)
	}

	if hits != 1 {
		t.Errorf("PokeAPI was hit %d times, want 1 (second call should be served from cache)", hits)
	}
	if len(second) != len(first) {
		t.Errorf("cached GetPokemonByType returned %d entries, want %d", len(second), len(first))
	}
}

// TestGetPokemonByType_ErrorsAreNotCached proves a failed GetPokemonByType
// call isn't remembered: a later call for the same type that succeeds must
// still hit PokeAPI rather than replaying the earlier failure.
func TestGetPokemonByType_ErrorsAreNotCached(t *testing.T) {
	fail := true
	mux := http.NewServeMux()
	mux.HandleFunc("/type/fire", func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fireTypeJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	if _, err := client.GetPokemonByType(context.Background(), "fire"); err == nil {
		t.Fatal("first GetPokemonByType returned nil error, want a ServiceError from the 500 fixture")
	}

	fail = false
	got, err := client.GetPokemonByType(context.Background(), "fire")
	if err != nil {
		t.Fatalf("second GetPokemonByType returned error: %v, want success now that the fixture stopped failing", err)
	}
	if len(got) == 0 {
		t.Error("GetPokemonByType after a prior failure returned no entries, want the roster (proves the failed call wasn't cached)")
	}
}

func TestGetTypeDamageRelations_DecodesDamageRelations(t *testing.T) {
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/type/fire": {status: http.StatusOK, body: fireTypeJSON},
	})

	got, err := client.GetTypeDamageRelations(context.Background(), "fire")
	if err != nil {
		t.Fatalf("GetTypeDamageRelations returned error: %v", err)
	}

	wantDoubleFrom := []string{"water", "ground", "rock"}
	wantHalfFrom := []string{"fire", "grass", "ice", "bug", "steel", "fairy"}
	if !reflect.DeepEqual(got.DoubleFrom, wantDoubleFrom) {
		t.Errorf("DoubleFrom = %v, want %v", got.DoubleFrom, wantDoubleFrom)
	}
	if !reflect.DeepEqual(got.HalfFrom, wantHalfFrom) {
		t.Errorf("HalfFrom = %v, want %v", got.HalfFrom, wantHalfFrom)
	}
	if len(got.NoFrom) != 0 {
		t.Errorf("NoFrom = %v, want none", got.NoFrom)
	}
}

// TestGetPokemonByType_AndGetTypeDamageRelations_ShareOneFetch proves both
// methods read the same cached /type/{name} fetch rather than each keeping
// its own - see getTypeDetails and cache's doc comment.
func TestGetPokemonByType_AndGetTypeDamageRelations_ShareOneFetch(t *testing.T) {
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/type/fire", func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fireTypeJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	if _, err := client.GetPokemonByType(context.Background(), "fire"); err != nil {
		t.Fatalf("GetPokemonByType returned error: %v", err)
	}
	if _, err := client.GetTypeDamageRelations(context.Background(), "fire"); err != nil {
		t.Fatalf("GetTypeDamageRelations returned error: %v", err)
	}

	if hits != 1 {
		t.Errorf("/type/fire was hit %d times, want 1 (GetPokemonByType and GetTypeDamageRelations should share one fetch)", hits)
	}
}

// TestGetTypeDamageRelations_ErrorsAreNotCached proves a failed
// GetTypeDamageRelations call isn't remembered, the same guarantee
// GetPokemonByType already has (see TestGetPokemonByType_ErrorsAreNotCached)
// - both share the same underlying fetch.
func TestGetTypeDamageRelations_ErrorsAreNotCached(t *testing.T) {
	fail := true
	mux := http.NewServeMux()
	mux.HandleFunc("/type/fire", func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fireTypeJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	if _, err := client.GetTypeDamageRelations(context.Background(), "fire"); err == nil {
		t.Fatal("first GetTypeDamageRelations returned nil error, want a ServiceError from the 500 fixture")
	}

	fail = false
	got, err := client.GetTypeDamageRelations(context.Background(), "fire")
	if err != nil {
		t.Fatalf("second GetTypeDamageRelations returned error: %v, want success now that the fixture stopped failing", err)
	}
	if len(got.DoubleFrom) == 0 {
		t.Error("GetTypeDamageRelations after a prior failure returned no DoubleFrom entries, want the fixture's data (proves the failed call wasn't cached)")
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
