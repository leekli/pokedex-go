package tui

import (
	"context"
	"image"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leekli/pokedex-go/internal/pokeapi"
	"github.com/leekli/pokedex-go/internal/pokemon"
)

// lookupTimeout bounds a single Search Screen submission: the PokeAPI
// lookup plus, on success, the sprite image fetch.
const lookupTimeout = 10 * time.Second

// lookupCmd resolves q against PokeAPI and reports the outcome as a
// lookupResultMsg. A failed sprite fetch is not treated as an overall
// failure — the Result Screen falls back to "No sprite available" instead
// of losing an otherwise-successful stat lookup over a broken image.
func lookupCmd(client *pokeapi.Client, q pokemon.Query) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
		defer cancel()

		p, err := client.Lookup(ctx, q)
		if err != nil {
			return lookupResultMsg{err: err}
		}

		stat := pokeapi.BuildStatBlock(p)
		var sprite image.Image
		if p.SpriteURL != "" {
			if img, spriteErr := client.FetchSprite(ctx, p.SpriteURL); spriteErr == nil {
				sprite = img
			}
		}
		return lookupResultMsg{stat: stat, sprite: sprite}
	}
}

// typeRosterTimeout bounds a Type Roster Screen load: the type's pokemon
// list plus, the first time in a session, the one-off generation index
// (see GetGenerationIndex) — more round-trips than a single Pokémon lookup,
// so it gets a longer budget than lookupTimeout.
const typeRosterTimeout = 20 * time.Second

// loadTypeRosterCmd fetches every real Pokémon of typeName and reports the
// outcome as a typeRosterResultMsg. It always calls GetGenerationIndex,
// but that's cheap after the first call in a session — the Client itself
// caches the generation index (see GetGenerationIndex's doc comment), so
// only the very first Type Roster load in a run of the app actually hits
// PokeAPI for it.
func loadTypeRosterCmd(client *pokeapi.Client, typeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), typeRosterTimeout)
		defer cancel()

		entries, err := client.GetPokemonByType(ctx, typeName)
		if err != nil {
			return typeRosterResultMsg{typeName: typeName, err: err}
		}

		generations := client.GetGenerationIndex(ctx)
		return typeRosterResultMsg{typeName: typeName, entries: entries, generations: generations}
	}
}
