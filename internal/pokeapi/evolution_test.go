package pokeapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leekli/pokedex-go/internal/pokemon"
)

// --- BuildEvolutionChain: literal-struct tests, no JSON/HTTP involved ---
// (the same separation typeeffectiveness_test.go uses for
// BuildTypeEffectiveness) - see TestGetEvolutionChain_DecodesRealShapedJSON
// below for the JSON-decoding boundary itself.

// pichuPikachuRaichuNode is Pikachu's real evolution family, used
// throughout these tests as a real, checkable linear example rather than
// synthetic data - the exact family used in the Result Screen's own
// Evolution Chain design.
func pichuPikachuRaichuNode() EvolutionChainNode {
	return EvolutionChainNode{
		DexNumber: 172,
		Name:      "pichu",
		EvolvesTo: []EvolutionChainNode{
			{
				DexNumber: 25,
				Name:      "pikachu",
				Details: []EvolutionDetail{
					{VersionGroup: "red-blue", Condition: pokemon.EvolutionCondition{Trigger: "level-up", Happiness: true}},
				},
				EvolvesTo: []EvolutionChainNode{
					{
						DexNumber: 26,
						Name:      "raichu",
						Details: []EvolutionDetail{
							{VersionGroup: "red-blue", Condition: pokemon.EvolutionCondition{Trigger: "use-item", Item: "thunder-stone"}},
						},
					},
				},
			},
		},
	}
}

func TestBuildEvolutionChain_LinearChain(t *testing.T) {
	got := BuildEvolutionChain(pichuPikachuRaichuNode())

	if got.Root.DexNumber != 172 || got.Root.Name != "pichu" {
		t.Fatalf("Root = %+v, want Pichu (172)", got.Root)
	}
	if got.Root.Condition != "" {
		t.Errorf("Root.Condition = %q, want empty (the chain's root has no incoming transition)", got.Root.Condition)
	}
	if len(got.Root.EvolvesTo) != 1 {
		t.Fatalf("Root.EvolvesTo = %+v, want exactly 1 (Pikachu)", got.Root.EvolvesTo)
	}

	pikachu := got.Root.EvolvesTo[0]
	if pikachu.Name != "pikachu" || pikachu.Condition != "high friendship" {
		t.Errorf("pikachu stage = %+v, want name pikachu, condition %q", pikachu, "high friendship")
	}
	if len(pikachu.EvolvesTo) != 1 {
		t.Fatalf("pikachu.EvolvesTo = %+v, want exactly 1 (Raichu)", pikachu.EvolvesTo)
	}

	raichu := pikachu.EvolvesTo[0]
	if raichu.Name != "raichu" || raichu.Condition != "use Thunder Stone" {
		t.Errorf("raichu stage = %+v, want name raichu, condition %q", raichu, "use Thunder Stone")
	}
}

// TestBuildEvolutionChain_Branching proves a species with multiple
// evolutions (like Eevee) produces multiple entries under the same
// parent's EvolvesTo, each with its own condition - not just the first one.
func TestBuildEvolutionChain_Branching(t *testing.T) {
	root := EvolutionChainNode{
		DexNumber: 133,
		Name:      "eevee",
		EvolvesTo: []EvolutionChainNode{
			{
				DexNumber: 134,
				Name:      "vaporeon",
				Details:   []EvolutionDetail{{VersionGroup: "red-blue", Condition: pokemon.EvolutionCondition{Trigger: "use-item", Item: "water-stone"}}},
			},
			{
				DexNumber: 135,
				Name:      "jolteon",
				Details:   []EvolutionDetail{{VersionGroup: "red-blue", Condition: pokemon.EvolutionCondition{Trigger: "use-item", Item: "thunder-stone"}}},
			},
			{
				DexNumber: 136,
				Name:      "flareon",
				Details:   []EvolutionDetail{{VersionGroup: "red-blue", Condition: pokemon.EvolutionCondition{Trigger: "use-item", Item: "fire-stone"}}},
			},
		},
	}

	got := BuildEvolutionChain(root)

	if len(got.Root.EvolvesTo) != 3 {
		t.Fatalf("Root.EvolvesTo = %+v, want 3 siblings (Vaporeon, Jolteon, Flareon)", got.Root.EvolvesTo)
	}
	names := []string{got.Root.EvolvesTo[0].Name, got.Root.EvolvesTo[1].Name, got.Root.EvolvesTo[2].Name}
	want := []string{"vaporeon", "jolteon", "flareon"}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("EvolvesTo[%d].Name = %q, want %q (branch order preserved)", i, names[i], want[i])
		}
	}
}

// TestBuildEvolutionChain_PrefersRedBlueVersionGroup proves that when a
// branch has several version-group-specific evolution_details (e.g.
// Pikachu evolving into Raichu via Thunder Stone in every version group,
// but only into Alolan Raichu in Sun/Moon), red-blue is picked over a
// later, non-priority version group - see
// evolutionDetailVersionGroupPriority and
// docs/adr/0006-scope-evolution-condition-text-to-common-cases.md.
func TestBuildEvolutionChain_PrefersRedBlueVersionGroup(t *testing.T) {
	root := EvolutionChainNode{
		Name: "pikachu",
		EvolvesTo: []EvolutionChainNode{
			{
				Name: "raichu",
				Details: []EvolutionDetail{
					{VersionGroup: "sun-moon", Condition: pokemon.EvolutionCondition{Trigger: "use-item", Item: "thunder-stone"}},
					{VersionGroup: "red-blue", Condition: pokemon.EvolutionCondition{Trigger: "use-item", Item: "thunder-stone"}},
					{VersionGroup: "yellow", Condition: pokemon.EvolutionCondition{Trigger: "use-item", Item: "thunder-stone"}},
				},
			},
		},
	}

	got := BuildEvolutionChain(root)

	// All three details actually describe the identical condition here, so
	// prove the selection by ordering (red-blue first) with a case where it
	// differs instead.
	if got.Root.EvolvesTo[0].Condition != "use Thunder Stone" {
		t.Fatalf("condition = %q, want %q", got.Root.EvolvesTo[0].Condition, "use Thunder Stone")
	}

	root.EvolvesTo[0].Details = []EvolutionDetail{
		{VersionGroup: "sun-moon", Condition: pokemon.EvolutionCondition{Trigger: "level-up", Level: 99}},
		{VersionGroup: "red-blue", Condition: pokemon.EvolutionCondition{Trigger: "use-item", Item: "thunder-stone"}},
	}
	got = BuildEvolutionChain(root)
	if got.Root.EvolvesTo[0].Condition != "use Thunder Stone" {
		t.Errorf("condition = %q, want the red-blue entry (%q) preferred over sun-moon", got.Root.EvolvesTo[0].Condition, "use Thunder Stone")
	}
}

// TestBuildEvolutionChain_FallsBackToFirstDetailWhenNoPreferredVersionGroup
// mirrors SelectPokedexEntry's own fallback (see docs/adr/0003 and
// pokedexentry.go): a species introduced after Generation I (e.g. Feebas)
// has no red-blue or yellow entry to prefer, so the first entry PokeAPI
// listed is used instead of guessing.
func TestBuildEvolutionChain_FallsBackToFirstDetailWhenNoPreferredVersionGroup(t *testing.T) {
	root := EvolutionChainNode{
		Name: "feebas",
		EvolvesTo: []EvolutionChainNode{
			{
				Name: "milotic",
				Details: []EvolutionDetail{
					{VersionGroup: "ruby-sapphire", Condition: pokemon.EvolutionCondition{Trigger: "level-up", Beauty: true}},
					{VersionGroup: "black-white", Condition: pokemon.EvolutionCondition{Trigger: "trade", HeldItem: "prism-scale"}},
				},
			},
		},
	}

	got := BuildEvolutionChain(root)

	if got.Root.EvolvesTo[0].Condition != "high beauty" {
		t.Errorf("condition = %q, want the first-listed entry's condition (%q)", got.Root.EvolvesTo[0].Condition, "high beauty")
	}
}

// --- GetEvolutionChain: real fetch/cache/error behavior over HTTP ---

// pikachuEvolutionChainJSON mirrors real PokeAPI's GET /evolution-chain/10
// shape closely enough to exercise the DTO decode boundary end to end:
// Pichu (root, no evolution_details) -> Pikachu (level-up, high
// friendship) -> Raichu (use Thunder Stone).
const pikachuEvolutionChainJSON = `{
	"chain": {
		"species": {"name": "pichu", "url": "https://pokeapi.co/api/v2/pokemon-species/172/"},
		"evolution_details": [],
		"evolves_to": [
			{
				"species": {"name": "pikachu", "url": "https://pokeapi.co/api/v2/pokemon-species/25/"},
				"evolution_details": [
					{
						"trigger": {"name": "level-up"},
						"min_happiness": 220,
						"version_group": {"name": "red-blue"}
					}
				],
				"evolves_to": [
					{
						"species": {"name": "raichu", "url": "https://pokeapi.co/api/v2/pokemon-species/26/"},
						"evolution_details": [
							{
								"trigger": {"name": "use-item"},
								"item": {"name": "thunder-stone"},
								"version_group": {"name": "red-blue"}
							}
						],
						"evolves_to": []
					}
				]
			}
		]
	}
}`

// TestGetEvolutionChain_DecodesRealShapedJSON proves the JSON decode
// boundary itself: real PokeAPI field names (species.url, min_happiness,
// item.name, trigger.name, version_group.name) map into the right
// EvolutionCondition fields and produce the right display text end to end.
func TestGetEvolutionChain_DecodesRealShapedJSON(t *testing.T) {
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/evolution-chain/10": {status: http.StatusOK, body: pikachuEvolutionChainJSON},
	})

	got, err := client.GetEvolutionChain(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetEvolutionChain returned error: %v", err)
	}

	if got.Root.DexNumber != 172 || got.Root.Name != "pichu" {
		t.Fatalf("Root = %+v, want Pichu (172)", got.Root)
	}
	pikachu := got.Root.EvolvesTo[0]
	if pikachu.DexNumber != 25 || pikachu.Condition != "high friendship" {
		t.Errorf("pikachu stage = %+v, want dex 25, condition %q", pikachu, "high friendship")
	}
	raichu := pikachu.EvolvesTo[0]
	if raichu.DexNumber != 26 || raichu.Condition != "use Thunder Stone" {
		t.Errorf("raichu stage = %+v, want dex 26, condition %q", raichu, "use Thunder Stone")
	}
}

// TestGetEvolutionChain_CachesAcrossCalls proves a second GetEvolutionChain
// call for the same id is served from the Client's cache rather than
// hitting PokeAPI again.
func TestGetEvolutionChain_CachesAcrossCalls(t *testing.T) {
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/evolution-chain/10", func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(pikachuEvolutionChainJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	first, err := client.GetEvolutionChain(context.Background(), 10)
	if err != nil {
		t.Fatalf("first GetEvolutionChain returned error: %v", err)
	}
	second, err := client.GetEvolutionChain(context.Background(), 10)
	if err != nil {
		t.Fatalf("second GetEvolutionChain returned error: %v", err)
	}

	if hits != 1 {
		t.Errorf("PokeAPI was hit %d times, want 1 (second call should be served from cache)", hits)
	}
	if second.Root.Name != first.Root.Name {
		t.Errorf("cached GetEvolutionChain = %+v, want %+v", second, first)
	}
}

// TestGetEvolutionChain_ErrorsAreNotCached proves a failed GetEvolutionChain
// call isn't remembered: a later call for the same id that succeeds must
// still hit PokeAPI rather than replaying the earlier failure.
func TestGetEvolutionChain_ErrorsAreNotCached(t *testing.T) {
	fail := true
	mux := http.NewServeMux()
	mux.HandleFunc("/evolution-chain/10", func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(pikachuEvolutionChainJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	if _, err := client.GetEvolutionChain(context.Background(), 10); err == nil {
		t.Fatal("first GetEvolutionChain returned nil error, want a ServiceError from the 500 fixture")
	}

	fail = false
	got, err := client.GetEvolutionChain(context.Background(), 10)
	if err != nil {
		t.Fatalf("second GetEvolutionChain returned error: %v, want success now that the fixture stopped failing", err)
	}
	if got.Root.Name != "pichu" {
		t.Errorf("GetEvolutionChain after a prior failure = %+v, want pichu (proves the failed call wasn't cached)", got)
	}
}

// TestGetEvolutionChain_ServerError proves a non-200 response is classified
// as a ServiceError, the same way every other pokeapi fetch is.
func TestGetEvolutionChain_ServerError(t *testing.T) {
	_, client := newFixtureServer(t, map[string]fixtureResponse{
		"/evolution-chain/10": {status: http.StatusInternalServerError},
	})

	_, err := client.GetEvolutionChain(context.Background(), 10)

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("GetEvolutionChain error = %v (%T), want *ServiceError", err, err)
	}
}
