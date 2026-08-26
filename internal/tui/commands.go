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
// lookupResultMsg. Both the sprite fetch and the Pokédex Entry fetch are
// best-effort: a failure in either leaves that field blank (image.Image nil
// / string "") rather than failing the overall lookup, since a missing
// sprite or description is never worse than losing an otherwise-successful
// stat lookup over it. The Pokédex Entry comes from a second, separate
// GetSpecies call keyed by q.Value — for a DexNumber query this reuses the
// species Lookup itself already fetched (and cached) under that same key,
// so it costs no extra network round trip; for a Name query it's a genuine
// second request, since Lookup skips species entirely in that case. See
// docs/adr/0003-prefer-generation-1-pokedex-entry-version.md.
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

		var pokedexEntry string
		if species, speciesErr := client.GetSpecies(ctx, q.Value); speciesErr == nil {
			pokedexEntry = pokeapi.SelectPokedexEntry(species.FlavorTextEntries)
		}

		return lookupResultMsg{stat: stat, sprite: sprite, pokedexEntry: pokedexEntry}
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
