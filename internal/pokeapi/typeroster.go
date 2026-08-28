package pokeapi

import (
	"context"
	"sort"
	"strconv"
	"strings"
)

// namedResourceDTO is PokeAPI's common {name, url} resource reference shape,
// reused across typeDTO's pokemon list and its damage_relations lists.
type namedResourceDTO struct {
	Name string `json:"name"`
}

// typeDTO is the shape of a GET /type/{name} response, reduced to what's
// needed to build a Type Roster (the list of Pokémon of that type) and a
// Pokémon's Weaknesses & Resistances (damage_relations - what other types
// deal double, half, or no damage to this type).
type typeDTO struct {
	Pokemon []struct {
		Pokemon struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"pokemon"`
	} `json:"pokemon"`
	DamageRelations struct {
		DoubleDamageFrom []namedResourceDTO `json:"double_damage_from"`
		HalfDamageFrom   []namedResourceDTO `json:"half_damage_from"`
		NoDamageFrom     []namedResourceDTO `json:"no_damage_from"`
	} `json:"damage_relations"`
}

// TypeRosterEntry is one row of a Type Roster: a single real, dex-numbered
// Pokémon of the requested type (see the ADR on filtering by pokémon id for
// why alternate forms — megas, regional forms, formes — never appear here).
type TypeRosterEntry struct {
	DexNumber int
	Name      string
}

// TypeDamageRelations is one type's own weaknesses/resistances/immunities,
// before combining with a Pokémon's second type — see BuildTypeEffectiveness,
// which does that combining, and docs/adr/0004-all-or-nothing-type-effectiveness.md.
type TypeDamageRelations struct {
	DoubleFrom []string // types that deal double damage to this type
	HalfFrom   []string // types that deal half damage to this type
	NoFrom     []string // types that deal no damage to this type
}

// typeDetails is everything one /type/{name} fetch yields, cached together
// so GetPokemonByType and GetTypeDamageRelations share a single fetch per
// type per Client lifetime, rather than each keeping its own cache of the
// same underlying resource — see getTypeDetails.
type typeDetails struct {
	roster          []TypeRosterEntry
	damageRelations TypeDamageRelations
}

// nonDefaultVarietyIDFloor is the pokémon resource id above which PokeAPI
// assigns non-default varieties (mega evolutions, regional forms, battle
// formes, etc.) rather than real, dex-numbered species — see
// docs/adr/0002-filter-type-roster-by-pokemon-id.md for how this was
// verified and why it's used instead of an is_default check per entry.
const nonDefaultVarietyIDFloor = 10000

// getTypeDetails fetches and caches GET /type/{name} once per Client
// lifetime, backing both GetPokemonByType and GetTypeDamageRelations so
// browsing a Type Roster and looking up a Pokémon of that type never fetch
// the same resource twice (see cache's doc comment).
func (c *Client) getTypeDetails(ctx context.Context, typeName string) (typeDetails, error) {
	c.cache.mu.Lock()
	cached, ok := c.cache.types[typeName]
	c.cache.mu.Unlock()
	if ok {
		return cached, nil
	}

	var dto typeDTO
	if err := c.get(ctx, "/type/"+typeName, typeName, &dto); err != nil {
		return typeDetails{}, err
	}

	roster := make([]TypeRosterEntry, 0, len(dto.Pokemon))
	for _, p := range dto.Pokemon {
		id, ok := resourceIDFromURL(p.Pokemon.URL)
		if !ok || id >= nonDefaultVarietyIDFloor {
			continue
		}
		roster = append(roster, TypeRosterEntry{DexNumber: id, Name: p.Pokemon.Name})
	}
	sort.Slice(roster, func(i, j int) bool { return roster[i].DexNumber < roster[j].DexNumber })

	details := typeDetails{
		roster: roster,
		damageRelations: TypeDamageRelations{
			DoubleFrom: namedResourceNames(dto.DamageRelations.DoubleDamageFrom),
			HalfFrom:   namedResourceNames(dto.DamageRelations.HalfDamageFrom),
			NoFrom:     namedResourceNames(dto.DamageRelations.NoDamageFrom),
		},
	}

	c.cache.mu.Lock()
	c.cache.types[typeName] = details
	c.cache.mu.Unlock()
	return details, nil
}

func namedResourceNames(entries []namedResourceDTO) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}

// GetPokemonByType fetches every real, dex-numbered Pokémon of the given
// type (as returned by PokeAPI's own /type/{name} name, e.g. "fire"),
// sorted by National Dex Number ascending. The result is cached for the
// rest of the Client's lifetime, shared with GetTypeDamageRelations (see
// cache's doc comment).
func (c *Client) GetPokemonByType(ctx context.Context, typeName string) ([]TypeRosterEntry, error) {
	details, err := c.getTypeDetails(ctx, typeName)
	if err != nil {
		return nil, err
	}
	return details.roster, nil
}

// GetTypeDamageRelations fetches what other types deal double, half, or no
// damage to the given type — i.e. that type's own weaknesses, resistances,
// and immunities, before combining with a Pokémon's second type (see
// BuildTypeEffectiveness). Cached for the rest of the Client's lifetime,
// shared with GetPokemonByType (see cache's doc comment).
func (c *Client) GetTypeDamageRelations(ctx context.Context, typeName string) (TypeDamageRelations, error) {
	details, err := c.getTypeDetails(ctx, typeName)
	if err != nil {
		return TypeDamageRelations{}, err
	}
	return details.damageRelations, nil
}

// resourceIDFromURL extracts the trailing numeric id from a PokeAPI
// resource URL, e.g. "https://pokeapi.co/api/v2/pokemon/25/" -> 25, or
// "https://pokeapi.co/api/v2/evolution-chain/10/" -> 10 - false if the URL
// doesn't end in a numeric path segment. Shared by every resource type
// whose id pokedex-go needs from an embedded {name, url} reference, rather
// than PokeAPI's own numeric ids (pokémon, generation species, evolution
// chains); see GetGenerationIndex and GetEvolutionChain's callers for the
// others.
func resourceIDFromURL(url string) (int, bool) {
	trimmed := strings.TrimSuffix(url, "/")
	segment := trimmed[strings.LastIndex(trimmed, "/")+1:]
	id, err := strconv.Atoi(segment)
	if err != nil {
		return 0, false
	}
	return id, true
}
