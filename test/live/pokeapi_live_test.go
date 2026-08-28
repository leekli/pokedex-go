//go:build live

// Package live holds a single opt-in smoke test against the real PokeAPI,
// to catch drift between the mocked fixtures used everywhere else in this
// suite and PokeAPI's actual response shape. It's excluded from normal
// builds and test runs by the "live" build tag, and additionally skips
// itself at runtime unless POKEDEX_LIVE_TEST=1 is set - run it explicitly
// with:
//
//	POKEDEX_LIVE_TEST=1 go test -tags=live ./test/live/...
package live

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/leekli/pokedex-go/internal/pokeapi"
)

func TestLive_GetPokemonPikachu(t *testing.T) {
	if os.Getenv("POKEDEX_LIVE_TEST") != "1" {
		t.Skip("set POKEDEX_LIVE_TEST=1 to run this test against the real PokeAPI")
	}

	client := pokeapi.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	got, err := client.GetPokemon(ctx, "pikachu")
	if err != nil {
		t.Fatalf("GetPokemon(pikachu) against real PokeAPI failed: %v", err)
	}

	if got.ID != 25 {
		t.Errorf("pikachu id = %d, want 25 (has PokeAPI's numbering changed?)", got.ID)
	}
	if !contains(got.Types, "electric") {
		t.Errorf("pikachu types = %v, want to contain %q", got.Types, "electric")
	}
	if got.SpriteFrontURL == "" {
		t.Error("pikachu has no front_default sprite URL (has PokeAPI's sprites schema changed?)")
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
