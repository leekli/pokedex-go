package pokeapi

// The DTOs below mirror only the subset of PokeAPI's JSON response fields
// this app actually uses; unrecognized fields are ignored by encoding/json.

// pokemonDTO is the shape of a GET /pokemon/{name} response.
type pokemonDTO struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Height int    `json:"height"` // decimetres
	Weight int    `json:"weight"` // hectograms
	Types  []struct {
		Type struct {
			Name string `json:"name"`
		} `json:"type"`
	} `json:"types"`
	Stats []struct {
		BaseStat int `json:"base_stat"`
		Stat     struct {
			Name string `json:"name"`
		} `json:"stat"`
	} `json:"stats"`
	Sprites struct {
		FrontDefault *string `json:"front_default"`
		BackDefault  *string `json:"back_default"`
	} `json:"sprites"`
}

// speciesDTO is the shape of a GET /pokemon-species/{id-or-name} response.
// The default variety's Pokémon name resolves a National Dex Number lookup
// into the base-form pokemon resource; flavor_text_entries is the source of
// the Result Screen's Pokédex Entry (see SelectPokedexEntry).
type speciesDTO struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Varieties []struct {
		IsDefault bool `json:"is_default"`
		Pokemon   struct {
			Name string `json:"name"`
		} `json:"pokemon"`
	} `json:"varieties"`
	FlavorTextEntries []struct {
		FlavorText string `json:"flavor_text"`
		Language   struct {
			Name string `json:"name"`
		} `json:"language"`
		Version struct {
			Name string `json:"name"`
		} `json:"version"`
	} `json:"flavor_text_entries"`
}

// Pokemon is the domain representation of a GET /pokemon/{name} response —
// PokeAPI's raw units and unordered stat list normalized into a stable shape.
type Pokemon struct {
	ID            int
	Name          string
	HeightDm      int
	WeightHg      int
	Types         []string
	Stats         map[string]int // keyed by PokeAPI stat name, e.g. "special-attack"
	SpriteURL     string         // empty if PokeAPI returned no front_default sprite
	SpriteBackURL string         // empty if PokeAPI returned no back_default sprite
}

// Species is the domain representation of a GET /pokemon-species/{...}
// response: enough to resolve a National Dex Number, plus every raw
// flavor text entry PokeAPI has for this species (see SelectPokedexEntry
// for picking one to display).
type Species struct {
	ID                 int
	Name               string
	DefaultPokemonName string
	FlavorTextEntries  []FlavorTextEntry
}

// FlavorTextEntry is one language/game-version-specific Pokédex description
// from a pokemon-species resource, before any selection or cleanup - see
// SelectPokedexEntry and docs/adr/0003-prefer-generation-1-pokedex-entry-version.md.
type FlavorTextEntry struct {
	Text     string
	Language string
	Version  string
}

func (d pokemonDTO) toDomain() Pokemon {
	p := Pokemon{
		ID:       d.ID,
		Name:     d.Name,
		HeightDm: d.Height,
		WeightHg: d.Weight,
		Types:    make([]string, 0, len(d.Types)),
		Stats:    make(map[string]int, len(d.Stats)),
	}
	for _, t := range d.Types {
		p.Types = append(p.Types, t.Type.Name)
	}
	for _, s := range d.Stats {
		p.Stats[s.Stat.Name] = s.BaseStat
	}
	if d.Sprites.FrontDefault != nil {
		p.SpriteURL = *d.Sprites.FrontDefault
	}
	if d.Sprites.BackDefault != nil {
		p.SpriteBackURL = *d.Sprites.BackDefault
	}
	return p
}

func (d speciesDTO) toDomain() Species {
	s := Species{ID: d.ID, Name: d.Name}
	for _, v := range d.Varieties {
		if v.IsDefault {
			s.DefaultPokemonName = v.Pokemon.Name
			break
		}
	}
	s.FlavorTextEntries = make([]FlavorTextEntry, 0, len(d.FlavorTextEntries))
	for _, e := range d.FlavorTextEntries {
		s.FlavorTextEntries = append(s.FlavorTextEntries, FlavorTextEntry{
			Text:     e.FlavorText,
			Language: e.Language.Name,
			Version:  e.Version.Name,
		})
	}
	return s
}
