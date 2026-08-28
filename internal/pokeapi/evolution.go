package pokeapi

import (
	"context"
	"strconv"

	"github.com/leekli/pokedex-go/internal/pokemon"
)

// evolutionChainDTO is the shape of a GET /evolution-chain/{id} response:
// a single recursive chain node rooted at the species' earliest known
// stage (e.g. Pichu for Pikachu).
type evolutionChainDTO struct {
	Chain evolutionChainLinkDTO `json:"chain"`
}

// evolutionChainLinkDTO is one species in the chain, its evolution_details
// (one entry per version group PokeAPI has a distinct condition for - see
// evolutionDetailDTO), and its own further evolutions. Branches (e.g.
// Eevee's many evolutions) show up as multiple entries in evolves_to.
type evolutionChainLinkDTO struct {
	Species struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"species"`
	EvolutionDetails []evolutionDetailDTO    `json:"evolution_details"`
	EvolvesTo        []evolutionChainLinkDTO `json:"evolves_to"`
}

// evolutionDetailDTO mirrors only the evolution_details fields common
// enough to describe in the Result Screen's Evolution Chain - see
// pokemon.DescribeEvolutionCondition and
// docs/adr/0006-scope-evolution-condition-text-to-common-cases.md for why
// PokeAPI's less common condition fields (min_affection, needs_overworld_rain,
// relative_physical_stats, region, gender, ...) aren't decoded at all.
type evolutionDetailDTO struct {
	Trigger struct {
		Name string `json:"name"`
	} `json:"trigger"`
	MinLevel     *int              `json:"min_level"`
	MinHappiness *int              `json:"min_happiness"`
	MinBeauty    *int              `json:"min_beauty"`
	Item         *namedResourceDTO `json:"item"`
	HeldItem     *namedResourceDTO `json:"held_item"`
	KnownMove    *namedResourceDTO `json:"known_move"`
	TradeSpecies *namedResourceDTO `json:"trade_species"`
	TimeOfDay    string            `json:"time_of_day"`
	VersionGroup struct {
		Name string `json:"name"`
	} `json:"version_group"`
}

// EvolutionChainNode is one raw evolution-chain link, after JSON decoding
// but before BuildEvolutionChain picks one EvolutionDetail per branch and
// turns it into display text.
type EvolutionChainNode struct {
	DexNumber int
	Name      string
	Details   []EvolutionDetail // empty only for the chain's root, which has no incoming trigger
	EvolvesTo []EvolutionChainNode
}

// EvolutionDetail is one evolution_details entry: the version group it
// applies to, plus the raw condition data - see
// evolutionDetailVersionGroupPriority for how BuildEvolutionChain picks
// one per branch.
type EvolutionDetail struct {
	VersionGroup string
	Condition    pokemon.EvolutionCondition
}

func (d evolutionChainDTO) toDomain() EvolutionChainNode {
	return d.Chain.toDomain()
}

func (l evolutionChainLinkDTO) toDomain() EvolutionChainNode {
	node := EvolutionChainNode{Name: l.Species.Name}
	if id, ok := resourceIDFromURL(l.Species.URL); ok {
		node.DexNumber = id
	}
	for _, det := range l.EvolutionDetails {
		node.Details = append(node.Details, det.toDomain())
	}
	for _, child := range l.EvolvesTo {
		node.EvolvesTo = append(node.EvolvesTo, child.toDomain())
	}
	return node
}

func (d evolutionDetailDTO) toDomain() EvolutionDetail {
	cond := pokemon.EvolutionCondition{
		Trigger:   d.Trigger.Name,
		Happiness: d.MinHappiness != nil,
		Beauty:    d.MinBeauty != nil,
		TimeOfDay: d.TimeOfDay,
	}
	if d.MinLevel != nil {
		cond.Level = *d.MinLevel
	}
	if d.Item != nil {
		cond.Item = d.Item.Name
	}
	if d.HeldItem != nil {
		cond.HeldItem = d.HeldItem.Name
	}
	if d.TradeSpecies != nil {
		cond.TradeSpecies = d.TradeSpecies.Name
	}
	if d.KnownMove != nil {
		cond.KnownMove = d.KnownMove.Name
	}
	return EvolutionDetail{VersionGroup: d.VersionGroup.Name, Condition: cond}
}

// evolutionDetailVersionGroupPriority mirrors pokedexEntryVersionPriority
// (see pokedexentry.go and docs/adr/0003): when a branch's evolution
// condition differs by version group (e.g. Pikachu evolves into Raichu via
// a Thunder Stone in every version group, but only into Alolan Raichu in
// Sun/Moon), BuildEvolutionChain prefers whichever of these appears first,
// falling back to the first entry PokeAPI returned if none match - see
// docs/adr/0006-scope-evolution-condition-text-to-common-cases.md.
var evolutionDetailVersionGroupPriority = []string{"red-blue", "yellow"}

// BuildEvolutionChain maps a raw EvolutionChainNode tree (see
// GetEvolutionChain) into the pokemon.EvolutionChain shown on the Result
// Screen's Evolution Chain (see CONTEXT.md): picking one EvolutionDetail
// per branch (see evolutionDetailVersionGroupPriority) and describing it as
// display text via pokemon.DescribeEvolutionCondition.
func BuildEvolutionChain(root EvolutionChainNode) pokemon.EvolutionChain {
	return pokemon.EvolutionChain{Root: buildEvolutionStage(root, "")}
}

func buildEvolutionStage(node EvolutionChainNode, condition string) pokemon.EvolutionStage {
	stage := pokemon.EvolutionStage{
		DexNumber: node.DexNumber,
		Name:      node.Name,
		Condition: condition,
	}
	for _, child := range node.EvolvesTo {
		childCondition := pokemon.DescribeEvolutionCondition(selectEvolutionDetail(child.Details).Condition)
		stage.EvolvesTo = append(stage.EvolvesTo, buildEvolutionStage(child, childCondition))
	}
	return stage
}

// selectEvolutionDetail picks one EvolutionDetail out of a branch's
// possibly-several version-group-specific entries - see
// evolutionDetailVersionGroupPriority. Returns the zero value (which
// DescribeEvolutionCondition renders as "special condition") if details is
// empty - not expected in practice, since every non-root chain link has at
// least one, but safer than panicking on an unexpected PokeAPI response.
func selectEvolutionDetail(details []EvolutionDetail) EvolutionDetail {
	for _, preferred := range evolutionDetailVersionGroupPriority {
		for _, d := range details {
			if d.VersionGroup == preferred {
				return d
			}
		}
	}
	if len(details) > 0 {
		return details[0]
	}
	return EvolutionDetail{}
}

// GetEvolutionChain fetches a PokeAPI evolution-chain resource by id (see
// Species.EvolutionChainID), caching the result for the rest of the
// Client's lifetime (see cache's doc comment).
func (c *Client) GetEvolutionChain(ctx context.Context, id int) (pokemon.EvolutionChain, error) {
	c.cache.mu.Lock()
	chain, ok := c.cache.evolutionChains[id]
	c.cache.mu.Unlock()
	if ok {
		return chain, nil
	}

	idStr := strconv.Itoa(id)
	var dto evolutionChainDTO
	if err := c.get(ctx, "/evolution-chain/"+idStr, idStr, &dto); err != nil {
		return pokemon.EvolutionChain{}, err
	}
	chain = BuildEvolutionChain(dto.toDomain())

	c.cache.mu.Lock()
	c.cache.evolutionChains[id] = chain
	c.cache.mu.Unlock()
	return chain, nil
}
