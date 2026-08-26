package pokeapi

import (
	"context"
	"fmt"
)

// GetSpecies fetches a PokeAPI pokemon-species resource by National Dex
// Number (as a decimal string) or by species name, caching the result for
// the rest of the Client's lifetime (see cache's doc comment).
func (c *Client) GetSpecies(ctx context.Context, idOrName string) (Species, error) {
	c.cache.mu.Lock()
	s, ok := c.cache.species[idOrName]
	c.cache.mu.Unlock()
	if ok {
		return s, nil
	}

	var dto speciesDTO
	if err := c.get(ctx, "/pokemon-species/"+idOrName, idOrName, &dto); err != nil {
		return Species{}, err
	}
	species := dto.toDomain()
	if species.DefaultPokemonName == "" {
		return Species{}, &ServiceError{Err: fmt.Errorf("species %q has no default variety", idOrName)}
	}

	c.cache.mu.Lock()
	c.cache.species[idOrName] = species
	c.cache.mu.Unlock()
	return species, nil
}
