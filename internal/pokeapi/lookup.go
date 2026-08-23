package pokeapi

import (
	"context"

	"github.com/leekli/pokedex-go/internal/pokemon"
)

// Lookup resolves a Search Screen Query into a Pokemon. A DexNumber query is
// resolved via the species' default variety (see CONTEXT.md's National Dex
// Number entry); a Name query fetches the pokemon resource directly.
func (c *Client) Lookup(ctx context.Context, q pokemon.Query) (Pokemon, error) {
	if q.Kind == pokemon.DexNumber {
		species, err := c.GetSpecies(ctx, q.Value)
		if err != nil {
			return Pokemon{}, err
		}
		return c.GetPokemon(ctx, species.DefaultPokemonName)
	}
	return c.GetPokemon(ctx, q.Value)
}
