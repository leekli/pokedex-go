package pokeapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leekli/pokedex-go/internal/pokemon"
)

const pikachuPokemonJSON = `{
	"id": 25,
	"name": "pikachu",
	"height": 4,
	"weight": 60,
	"types": [{"type": {"name": "electric"}}],
	"stats": [
		{"base_stat": 35, "stat": {"name": "hp"}},
		{"base_stat": 55, "stat": {"name": "attack"}},
		{"base_stat": 40, "stat": {"name": "defense"}},
		{"base_stat": 50, "stat": {"name": "special-attack"}},
		{"base_stat": 50, "stat": {"name": "special-defense"}},
		{"base_stat": 90, "stat": {"name": "speed"}}
	],
	"sprites": {"front_default": "https://example.invalid/sprites/pikachu.png"}
}`

const pikachuSpeciesJSON = `{
	"id": 25,
	"name": "pikachu",
	"varieties": [
		{"is_default": true, "pokemon": {"name": "pikachu"}}
	]
}`

// newFixtureServer builds an httptest.Server that serves canned JSON bodies
// for specific paths, and a Client pointed at it. Tests never touch the
// real network.
func newFixtureServer(t *testing.T, routes map[string]fixtureResponse) (*httptest.Server, *Client) {
	t.Helper()
	mux := http.NewServeMux()
	for path, resp := range routes {
		resp := resp
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if resp.delay > 0 {
				time.Sleep(resp.delay)
			}
			w.WriteHeader(resp.status)
			if resp.body != "" {
				w.Write([]byte(resp.body))
			}
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := NewClient(WithBaseURL(server.URL))
	return server, client
}

type fixtureResponse struct {
	status int
	body   string
	delay  time.Duration
}

func TestGetPokemon_Success(t *testing.T) {
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/pokemon/pikachu": {status: http.StatusOK, body: pikachuPokemonJSON},
	})

	got, err := client.GetPokemon(context.Background(), "pikachu")
	if err != nil {
		t.Fatalf("GetPokemon returned error: %v", err)
	}
	if got.ID != 25 || got.Name != "pikachu" {
		t.Errorf("GetPokemon = %+v, want id 25 name pikachu", got)
	}
	if got.Stats["special-attack"] != 50 {
		t.Errorf("GetPokemon stats not decoded correctly: %+v", got.Stats)
	}
}

func TestGetPokemon_NotFound(t *testing.T) {
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/pokemon/notarealpokemon": {status: http.StatusNotFound},
	})

	_, err := client.GetPokemon(context.Background(), "notarealpokemon")

	var lookupErr *LookupError
	if !errors.As(err, &lookupErr) {
		t.Fatalf("GetPokemon error = %v (%T), want *LookupError", err, err)
	}
}

func TestGetSpecies_ServerError(t *testing.T) {
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/pokemon-species/25": {status: http.StatusInternalServerError},
	})

	_, err := client.GetSpecies(context.Background(), "25")

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("GetSpecies error = %v (%T), want *ServiceError", err, err)
	}
}

func TestGetPokemon_Timeout(t *testing.T) {
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/pokemon/pikachu": {status: http.StatusOK, body: pikachuPokemonJSON, delay: 100 * time.Millisecond},
	})
	client.httpClient = &http.Client{Timeout: 10 * time.Millisecond}

	_, err := client.GetPokemon(context.Background(), "pikachu")

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("GetPokemon error = %v (%T), want *ServiceError on timeout", err, err)
	}
}

func TestGetPokemon_MalformedJSON(t *testing.T) {
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/pokemon/pikachu": {status: http.StatusOK, body: "{not valid json"},
	})

	_, err := client.GetPokemon(context.Background(), "pikachu")

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("GetPokemon error = %v (%T), want *ServiceError on malformed body", err, err)
	}
}

func TestLookup_DexNumberChainsSpeciesThenPokemon(t *testing.T) {
	var hitSpecies, hitPokemonAfterSpecies bool

	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon-species/25", func(w http.ResponseWriter, r *http.Request) {
		hitSpecies = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(pikachuSpeciesJSON))
	})
	mux.HandleFunc("/pokemon/pikachu", func(w http.ResponseWriter, r *http.Request) {
		if !hitSpecies {
			t.Error("GetPokemon was called before GetSpecies for a DexNumber query")
		}
		hitPokemonAfterSpecies = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(pikachuPokemonJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	got, err := client.Lookup(context.Background(), pokemon.Query{Kind: pokemon.DexNumber, Value: "25"})
	if err != nil {
		t.Fatalf("Lookup returned error: %v", err)
	}
	if !hitSpecies || !hitPokemonAfterSpecies {
		t.Fatalf("Lookup did not chain species -> pokemon as expected (species hit=%v, pokemon hit=%v)", hitSpecies, hitPokemonAfterSpecies)
	}
	if got.Name != "pikachu" {
		t.Errorf("Lookup result name = %q, want pikachu", got.Name)
	}
}

// TestLookup_DexNumberSpeciesLookupFails proves a DexNumber query short-
// circuits on a failed species lookup: GetPokemon must never be called, and
// the species error (unmodified) is what Lookup returns.
func TestLookup_DexNumberSpeciesLookupFails(t *testing.T) {
	pokemonCalled := false
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon-species/9999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/pokemon/", func(w http.ResponseWriter, r *http.Request) {
		pokemonCalled = true
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	_, err := client.Lookup(context.Background(), pokemon.Query{Kind: pokemon.DexNumber, Value: "9999"})

	var lookupErr *LookupError
	if !errors.As(err, &lookupErr) {
		t.Fatalf("Lookup error = %v (%T), want *LookupError from the failed species lookup", err, err)
	}
	if pokemonCalled {
		t.Error("Lookup called /pokemon/ after GetSpecies failed; it should have short-circuited")
	}
}

func TestGetSpecies_NoDefaultVariety(t *testing.T) {
	const speciesWithNoDefaultJSON = `{
		"id": 25,
		"name": "pikachu",
		"varieties": [
			{"is_default": false, "pokemon": {"name": "pikachu-gmax"}}
		]
	}`
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/pokemon-species/25": {status: http.StatusOK, body: speciesWithNoDefaultJSON},
	})

	_, err := client.GetSpecies(context.Background(), "25")

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("GetSpecies error = %v (%T), want *ServiceError when no variety is marked default", err, err)
	}
}

// TestGetPokemon_CachesAcrossCalls proves a second GetPokemon call for the
// same name is served from the Client's cache rather than hitting PokeAPI
// again - see cache's doc comment on why this is safe given PokeAPI data
// effectively never changes.
func TestGetPokemon_CachesAcrossCalls(t *testing.T) {
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon/pikachu", func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(pikachuPokemonJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	first, err := client.GetPokemon(context.Background(), "pikachu")
	if err != nil {
		t.Fatalf("first GetPokemon returned error: %v", err)
	}
	second, err := client.GetPokemon(context.Background(), "pikachu")
	if err != nil {
		t.Fatalf("second GetPokemon returned error: %v", err)
	}

	if hits != 1 {
		t.Errorf("PokeAPI was hit %d times, want 1 (second call should be served from cache)", hits)
	}
	if second.ID != first.ID || second.Name != first.Name {
		t.Errorf("cached GetPokemon = %+v, want the same result as the first call %+v", second, first)
	}
}

// TestGetPokemon_ErrorsAreNotCached proves a failed GetPokemon call isn't
// remembered: a later call for the same name that succeeds must still hit
// PokeAPI rather than replaying the earlier failure.
func TestGetPokemon_ErrorsAreNotCached(t *testing.T) {
	fail := true
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon/pikachu", func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(pikachuPokemonJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	if _, err := client.GetPokemon(context.Background(), "pikachu"); err == nil {
		t.Fatal("first GetPokemon returned nil error, want a LookupError from the 404 fixture")
	}

	fail = false
	got, err := client.GetPokemon(context.Background(), "pikachu")
	if err != nil {
		t.Fatalf("second GetPokemon returned error: %v, want success now that the fixture stopped failing", err)
	}
	if got.Name != "pikachu" {
		t.Errorf("GetPokemon after a prior failure = %+v, want pikachu (proves the failed call wasn't cached)", got)
	}
}

// TestGetSpecies_CachesAcrossCalls proves a second GetSpecies call for the
// same id/name is served from the Client's cache rather than hitting
// PokeAPI again.
func TestGetSpecies_CachesAcrossCalls(t *testing.T) {
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon-species/25", func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(pikachuSpeciesJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	first, err := client.GetSpecies(context.Background(), "25")
	if err != nil {
		t.Fatalf("first GetSpecies returned error: %v", err)
	}
	second, err := client.GetSpecies(context.Background(), "25")
	if err != nil {
		t.Fatalf("second GetSpecies returned error: %v", err)
	}

	if hits != 1 {
		t.Errorf("PokeAPI was hit %d times, want 1 (second call should be served from cache)", hits)
	}
	if second != first {
		t.Errorf("cached GetSpecies = %+v, want %+v", second, first)
	}
}

func TestLookup_NameQueryCallsPokemonDirectly(t *testing.T) {
	speciesCalled := false
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon-species/", func(w http.ResponseWriter, r *http.Request) {
		speciesCalled = true
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/pokemon/pikachu", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(pikachuPokemonJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	_, err := client.Lookup(context.Background(), pokemon.Query{Kind: pokemon.Name, Value: "pikachu"})
	if err != nil {
		t.Fatalf("Lookup returned error: %v", err)
	}
	if speciesCalled {
		t.Error("Lookup called the species endpoint for a Name query; it should call /pokemon/{name} directly")
	}
}
