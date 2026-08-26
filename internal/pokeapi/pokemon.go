package pokeapi

import "context"

// GetPokemon fetches a PokeAPI pokemon resource by name, caching the result
// for the rest of the Client's lifetime (see cache's doc comment).
func (c *Client) GetPokemon(ctx context.Context, name string) (Pokemon, error) {
	c.cache.mu.Lock()
	p, ok := c.cache.pokemon[name]
	c.cache.mu.Unlock()
	if ok {
		return p, nil
	}

	var dto pokemonDTO
	if err := c.get(ctx, "/pokemon/"+name, name, &dto); err != nil {
		return Pokemon{}, err
	}
	p = dto.toDomain()

	c.cache.mu.Lock()
	c.cache.pokemon[name] = p
	c.cache.mu.Unlock()
	return p, nil
}
