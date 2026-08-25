package pokeapi

import (
	"context"
	"strconv"
	"sync"
)

// generationCount is how many PokeAPI generation resources exist (I-IX).
// Like the app's fixed 18-type list, this is a small, curated set rather
// than something worth an extra discovery request to look up — bump it
// when a new generation is added to PokeAPI.
const generationCount = 9

// generationDTO is the shape of a GET /generation/{id} response, reduced to
// what's needed to build the dex-id -> generation index: which species
// belong to this generation.
type generationDTO struct {
	Name           string `json:"name"`
	PokemonSpecies []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"pokemon_species"`
}

// GetGenerationIndex builds a National Dex Number -> raw generation name
// (e.g. "generation-i", suitable for pokemon.FormatGeneration) map by
// fetching every generation resource once, in parallel. It's best-effort:
// a generation that fails to fetch is simply missing from the result rather
// than failing the whole call, since a Type Roster's Dex #/Name columns are
// still valuable on their own even if Generation can't be filled in for
// every row — the same reasoning as a failed sprite fetch not failing an
// otherwise-successful lookup (see commands.go).
func (c *Client) GetGenerationIndex(ctx context.Context) map[int]string {
	type result struct {
		dto generationDTO
		err error
	}
	results := make([]result, generationCount)

	var wg sync.WaitGroup
	for i := range generationCount {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var dto generationDTO
			id := strconv.Itoa(i + 1)
			err := c.get(ctx, "/generation/"+id, "generation "+id, &dto)
			results[i] = result{dto: dto, err: err}
		}(i)
	}
	wg.Wait()

	index := make(map[int]string)
	for _, r := range results {
		if r.err != nil {
			continue
		}
		for _, species := range r.dto.PokemonSpecies {
			id, ok := pokemonIDFromURL(species.URL)
			if !ok {
				continue
			}
			index[id] = r.dto.Name
		}
	}
	return index
}
