package pokeapi

import (
	"context"
	"fmt"
)

// GetSpecies fetches a PokeAPI pokemon-species resource by National Dex
// Number (as a decimal string) or by species name.
func (c *Client) GetSpecies(ctx context.Context, idOrName string) (Species, error) {
	var dto speciesDTO
	if err := c.get(ctx, "/pokemon-species/"+idOrName, idOrName, &dto); err != nil {
		return Species{}, err
	}
	species := dto.toDomain()
	if species.DefaultPokemonName == "" {
		return Species{}, &ServiceError{Err: fmt.Errorf("species %q has no default variety", idOrName)}
	}
	return species, nil
}
