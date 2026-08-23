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
	} `json:"sprites"`
}

// speciesDTO is the shape of a GET /pokemon-species/{id-or-name} response.
// Only the default variety's Pokémon name is needed, to resolve a National
// Dex Number lookup into the base-form pokemon resource.
type speciesDTO struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Varieties []struct {
		IsDefault bool `json:"is_default"`
		Pokemon   struct {
			Name string `json:"name"`
		} `json:"pokemon"`
	} `json:"varieties"`
}

// Pokemon is the domain representation of a GET /pokemon/{name} response —
// PokeAPI's raw units and unordered stat list normalized into a stable shape.
type Pokemon struct {
	ID        int
	Name      string
	HeightDm  int
	WeightHg  int
	Types     []string
	Stats     map[string]int // keyed by PokeAPI stat name, e.g. "special-attack"
	SpriteURL string         // empty if PokeAPI returned no front_default sprite
}

// Species is the domain representation of a GET /pokemon-species/{...}
// response, reduced to what's needed to resolve a National Dex Number.
type Species struct {
	ID                 int
	Name               string
	DefaultPokemonName string
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
	return s
}
