// Package e2e drives the full pokedex-go TUI through teatest, backed by a
// local httptest fixture server so no test ever touches the real network.
// Color output is pinned to no-color (see TestMain) so assertions can match
// on plain text regardless of the environment the suite runs in.
package e2e

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/leekli/pokedex-go/internal/pokeapi"
	"github.com/leekli/pokedex-go/internal/tui"
)

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.Ascii)
	m.Run()
}

const pikachuPokemonJSONTemplate = `{
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
	"sprites": {"front_default": "%s/sprites/pikachu.png"}
}`

// pikachuSpeciesJSONTemplate includes a Generation I flavor text entry so
// the Search flow can assert the Pokédex Entry actually reaches the
// rendered Result Screen end-to-end - see
// docs/adr/0003-prefer-generation-1-pokedex-entry-version.md. Its
// evolution_chain URL points back at this same server's /evolution-chain/10
// route (see pikachuEvolutionChainJSON), so the Evolution Chain gets the
// same end-to-end coverage.
const pikachuSpeciesJSONTemplate = `{
	"id": 25,
	"name": "pikachu",
	"varieties": [{"is_default": true, "pokemon": {"name": "pikachu"}}],
	"flavor_text_entries": [
		{"flavor_text": "When several of\nthese Pokémon gather,\ntheir electricity could\nbuild and cause lightning\nstorms.", "language": {"name": "en"}, "version": {"name": "red"}}
	],
	"evolution_chain": {"url": "%s/evolution-chain/10/"}
}`

// pikachuEvolutionChainJSON is Pikachu's real evolution family, served for
// /evolution-chain/10: Pichu (root) -> Pikachu (high friendship) -> Raichu
// (Thunder Stone) - see docs/adr/0006-scope-evolution-condition-text-to-common-cases.md.
const pikachuEvolutionChainJSON = `{
	"chain": {
		"species": {"name": "pichu", "url": "https://pokeapi.co/api/v2/pokemon-species/172/"},
		"evolution_details": [],
		"evolves_to": [
			{
				"species": {"name": "pikachu", "url": "https://pokeapi.co/api/v2/pokemon-species/25/"},
				"evolution_details": [{"trigger": {"name": "level-up"}, "min_happiness": 220, "version_group": {"name": "red-blue"}}],
				"evolves_to": [
					{
						"species": {"name": "raichu", "url": "https://pokeapi.co/api/v2/pokemon-species/26/"},
						"evolution_details": [{"trigger": {"name": "use-item"}, "item": {"name": "thunder-stone"}, "version_group": {"name": "red-blue"}}],
						"evolves_to": []
					}
				]
			}
		]
	}
}`

// charmanderPokemonJSON is served for /pokemon/charmander, so the Type
// Roster -> Result flow (see type_roster_flow_test.go) exercises a real
// lookupCmd the same way the Search Screen's flow does, not a shortcut.
const charmanderPokemonJSON = `{
	"id": 4,
	"name": "charmander",
	"height": 6,
	"weight": 85,
	"types": [{"type": {"name": "fire"}}],
	"stats": [
		{"base_stat": 39, "stat": {"name": "hp"}},
		{"base_stat": 52, "stat": {"name": "attack"}},
		{"base_stat": 43, "stat": {"name": "defense"}},
		{"base_stat": 60, "stat": {"name": "special-attack"}},
		{"base_stat": 50, "stat": {"name": "special-defense"}},
		{"base_stat": 65, "stat": {"name": "speed"}}
	],
	"sprites": {"front_default": null}
}`

// fireTypeJSON is served for /type/fire: two real Pokémon (Charmander,
// Charizard) plus one non-default variety (a Mega Charizard form) that the
// Type Roster must filter out - see docs/adr/0002. Its damage_relations is
// Fire's real type chart, so Charmander's Result Screen (reached via
// type_roster_flow_test.go) exercises Weaknesses & Resistances too, sharing
// this same fetch with the Type Roster's own GetPokemonByType call - see
// getTypeDetails.
const fireTypeJSON = `{
	"pokemon": [
		{"pokemon": {"name": "charmander", "url": "%[1]s/pokemon/4/"}},
		{"pokemon": {"name": "charizard-mega-x", "url": "%[1]s/pokemon/10034/"}},
		{"pokemon": {"name": "charizard", "url": "%[1]s/pokemon/6/"}}
	],
	"damage_relations": {
		"double_damage_from": [{"name": "water"}, {"name": "ground"}, {"name": "rock"}],
		"half_damage_from": [{"name": "fire"}, {"name": "grass"}, {"name": "ice"}, {"name": "bug"}, {"name": "steel"}, {"name": "fairy"}],
		"no_damage_from": []
	}
}`

// electricTypeJSON is served for /type/electric: Pikachu's type, so the
// Search flow's Weaknesses & Resistances section has real data (Electric is
// weak to Ground; resists Electric, Flying, Steel).
const electricTypeJSON = `{
	"pokemon": [],
	"damage_relations": {
		"double_damage_from": [{"name": "ground"}],
		"half_damage_from": [{"name": "electric"}, {"name": "flying"}, {"name": "steel"}],
		"no_damage_from": []
	}
}`

// generationOneJSON names Charmander's generation, so the Type Roster's
// Generation column has real data to assert on for at least one row.
const generationOneJSON = `{"name": "generation-i", "pokemon_species": [{"name": "charmander", "url": "%[1]s/pokemon-species/4/"}]}`

// emptyGenerationJSON stands in for the other 8 generations: valid but
// empty, since no e2e fixture Pokémon belongs to them.
const emptyGenerationJSON = `{"name": "generation-empty", "pokemon_species": []}`

func pikachuSpritePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.NRGBA{R: 255, G: 216, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode fixture sprite PNG: %v", err)
	}
	return buf.Bytes()
}

// newFixtureServer serves the small set of PokeAPI routes pokedex-go's e2e
// flows need: pikachu by name and by National Dex Number 25, a reserved
// "servererror" name for exercising the Service Error path, and 404 for
// anything else - matching real PokeAPI's behavior for an unknown Pokémon.
func newFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	var baseURL string
	sprite := pikachuSpritePNG(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon/{name}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("name") {
		case "pikachu":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, pikachuPokemonJSONTemplate, baseURL)
		case "charmander":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(charmanderPokemonJSON))
		case "servererror":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	mux.HandleFunc("/pokemon-species/{id}", func(w http.ResponseWriter, r *http.Request) {
		// Answers both "25" (a DexNumber query's own species resolution, and
		// the Result Screen's Pokédex Entry fetch reusing that same cached
		// call) and "pikachu" (a Name query's separate Pokédex Entry fetch -
		// see lookupCmd's doc comment).
		switch r.PathValue("id") {
		case "25", "pikachu":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, pikachuSpeciesJSONTemplate, baseURL)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	mux.HandleFunc("/evolution-chain/10", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pikachuEvolutionChainJSON))
	})
	mux.HandleFunc("/sprites/pikachu.png", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(sprite)
	})
	mux.HandleFunc("/type/{name}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("name") {
		case "fire":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, fireTypeJSON, baseURL)
		case "electric":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(electricTypeJSON))
		case "water":
			// Reserved, like /pokemon/servererror, for exercising the Type
			// Roster's Service Error path (see type_roster_flow_test.go) -
			// a type-list fetch failing is never the user's mistake, so it
			// can only ever be a Service Error, never a Lookup Error.
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	mux.HandleFunc("/generation/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.PathValue("id") == "1" {
			fmt.Fprintf(w, generationOneJSON, baseURL)
			return
		}
		w.Write([]byte(emptyGenerationJSON))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	baseURL = server.URL
	return server
}

// newTestApp builds a fully-wired tui.App pointed at a fresh fixture server.
func newTestApp(t *testing.T) tea.Model {
	t.Helper()
	server := newFixtureServer(t)
	client := pokeapi.NewClient(pokeapi.WithBaseURL(server.URL))
	return tui.NewApp(client)
}

// newTestModel wraps newTestApp in a teatest.TestModel with a fixed terminal
// size, and registers cleanup so a stuck test can't hang the suite.
func newTestModel(t *testing.T) *teatest.TestModel {
	t.Helper()
	tm := teatest.NewTestModel(t, newTestApp(t), teatest.WithInitialTermSize(100, 40))
	t.Cleanup(func() {
		_ = tm.Quit()
	})
	return tm
}

// waitForAll waits, within timeout, for every one of want to appear in the
// program's output. teatest.WaitFor's underlying reader is drained as it's
// read, so all substrings expected from a single render must be checked
// together in one WaitFor call rather than in separate sequential reads.
func waitForAll(t *testing.T, tm *teatest.TestModel, timeout time.Duration, want ...string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		for _, w := range want {
			if !bytes.Contains(out, []byte(w)) {
				return false
			}
		}
		return true
	}, teatest.WithDuration(timeout))
}
