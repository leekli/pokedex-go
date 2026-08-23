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

const pikachuSpeciesJSON = `{
	"id": 25,
	"name": "pikachu",
	"varieties": [{"is_default": true, "pokemon": {"name": "pikachu"}}]
}`

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
		case "servererror":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	mux.HandleFunc("/pokemon-species/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "25" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(pikachuSpeciesJSON))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/sprites/pikachu.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(sprite)
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
