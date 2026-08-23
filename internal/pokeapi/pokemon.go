package pokeapi

import "context"

// GetPokemon fetches a PokeAPI pokemon resource by name.
func (c *Client) GetPokemon(ctx context.Context, name string) (Pokemon, error) {
	var dto pokemonDTO
	if err := c.get(ctx, "/pokemon/"+name, name, &dto); err != nil {
		return Pokemon{}, err
	}
	return dto.toDomain(), nil
}
