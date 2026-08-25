package pokeapi

import (
	"context"
	"sort"
	"strconv"
	"strings"
)

// typeDTO is the shape of a GET /type/{name} response, reduced to what's
// needed to build a Type Roster: the list of Pokémon of that type.
type typeDTO struct {
	Pokemon []struct {
		Pokemon struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"pokemon"`
	} `json:"pokemon"`
}

// TypeRosterEntry is one row of a Type Roster: a single real, dex-numbered
// Pokémon of the requested type (see the ADR on filtering by pokémon id for
// why alternate forms — megas, regional forms, formes — never appear here).
type TypeRosterEntry struct {
	DexNumber int
	Name      string
}

// nonDefaultVarietyIDFloor is the pokémon resource id above which PokeAPI
// assigns non-default varieties (mega evolutions, regional forms, battle
// formes, etc.) rather than real, dex-numbered species — see
// docs/adr/0002-filter-type-roster-by-pokemon-id.md for how this was
// verified and why it's used instead of an is_default check per entry.
const nonDefaultVarietyIDFloor = 10000

// GetPokemonByType fetches every real, dex-numbered Pokémon of the given
// type (as returned by PokeAPI's own /type/{name} name, e.g. "fire"),
// sorted by National Dex Number ascending.
func (c *Client) GetPokemonByType(ctx context.Context, typeName string) ([]TypeRosterEntry, error) {
	var dto typeDTO
	if err := c.get(ctx, "/type/"+typeName, typeName, &dto); err != nil {
		return nil, err
	}

	entries := make([]TypeRosterEntry, 0, len(dto.Pokemon))
	for _, p := range dto.Pokemon {
		id, ok := pokemonIDFromURL(p.Pokemon.URL)
		if !ok || id >= nonDefaultVarietyIDFloor {
			continue
		}
		entries = append(entries, TypeRosterEntry{DexNumber: id, Name: p.Pokemon.Name})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].DexNumber < entries[j].DexNumber })
	return entries, nil
}

// pokemonIDFromURL extracts the trailing numeric id from a PokeAPI resource
// URL, e.g. "https://pokeapi.co/api/v2/pokemon/25/" -> 25, false if the URL
// doesn't end in a numeric path segment.
func pokemonIDFromURL(url string) (int, bool) {
	trimmed := strings.TrimSuffix(url, "/")
	segment := trimmed[strings.LastIndex(trimmed, "/")+1:]
	id, err := strconv.Atoi(segment)
	if err != nil {
		return 0, false
	}
	return id, true
}
