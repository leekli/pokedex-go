package pokeapi

import (
	"image"
	"sync"

	"github.com/leekli/pokedex-go/internal/pokemon"
)

// cache holds every successful PokeAPI response a Client has already
// fetched, for the rest of that Client's lifetime. PokeAPI's data changes on
// the timescale of years, not a single run of this app, so nothing here
// ever expires or is invalidated — caching just avoids re-fetching (and, for
// sprites, re-decoding) the same resource twice in one session. See
// GetPokemon, GetSpecies, FetchSprite, GetGenerationIndex, and
// GetEvolutionChain for where each map is read and populated; types is
// shared by GetPokemonByType and GetTypeDamageRelations, which both read
// the same /type/{name} resource - see getTypeDetails.
//
// A failed request is never cached: a *LookupError may be a typo the user
// is about to correct, and a *ServiceError is transient by definition —
// caching either would risk permanently remembering a mistake or an outage
// that's since passed.
//
// The mutex only protects individual map reads/writes; two concurrent
// requests for the same not-yet-cached key can both miss and both fetch.
// That's fine for this app — the TUI never has two lookups or two
// type-roster loads in flight at once (see searchModel/typeRosterModel's
// loading flags) — but would need a singleflight-style guard in a context
// where concurrent duplicate requests were possible.
type cache struct {
	mu              sync.Mutex
	pokemon         map[string]Pokemon
	species         map[string]Species
	types           map[string]typeDetails
	sprites         map[string]image.Image
	generations     map[int]string // nil until GetGenerationIndex has run once
	evolutionChains map[int]pokemon.EvolutionChain
}

func newCache() *cache {
	return &cache{
		pokemon:         make(map[string]Pokemon),
		species:         make(map[string]Species),
		types:           make(map[string]typeDetails),
		sprites:         make(map[string]image.Image),
		evolutionChains: make(map[int]pokemon.EvolutionChain),
	}
}
