package pokemon

import (
	"fmt"
	"strings"
)

// EvolutionStage is one species in an Evolution Chain (see CONTEXT.md): its
// identity plus, for every stage but the chain's root, the human-readable
// condition that triggers its evolution from its parent stage. Branches
// (e.g. Eevee's many evolutions) appear as multiple stages under the same
// parent's EvolvesTo.
type EvolutionStage struct {
	DexNumber int
	Name      string
	Condition string // "" only for the chain's root stage, which has no incoming trigger
	EvolvesTo []EvolutionStage
}

// EvolutionChain is a species' full family tree of evolutions, rooted at
// its earliest known stage (e.g. Pichu for Pikachu) - a plain data holder,
// the same convention as StatBlock and TypeEffectiveness; see
// pokeapi.BuildEvolutionChain for how one is assembled from PokeAPI's raw
// evolution-chain resource.
type EvolutionChain struct {
	Root EvolutionStage
}

// EvolutionCondition is the plain data pokeapi.BuildEvolutionChain extracts
// from one PokeAPI evolution_details entry, before DescribeEvolutionCondition
// turns it into display text. Kept as plain values rather than a pokeapi DTO
// type so this package's no-network-imports rule holds - pokeapi already
// depends on this package, so the reverse dependency would be an import
// cycle (the same reasoning as StatBlock's doc comment).
type EvolutionCondition struct {
	Trigger      string // PokeAPI's evolution-trigger name: "level-up", "trade", "use-item", "shed", ...
	Level        int    // 0 if this isn't a plain level-based trigger
	Happiness    bool   // a minimum friendship threshold applies
	Beauty       bool   // a minimum beauty threshold applies (e.g. Feebas -> Milotic)
	Item         string // item consumed to trigger the evolution, "" if none
	HeldItem     string // item that must be held (typically while trading), "" if none
	TradeSpecies string // species that must be traded for, "" if any trade will do
	KnownMove    string // move that must be known, "" if none
	TimeOfDay    string // "day", "night", or "" if not time-restricted
}

// DescribeEvolutionCondition renders c as the short phrase shown next to an
// evolution arrow (e.g. "Lv. 16", "use Thunder Stone", "trade"). PokeAPI
// supports far more evolution triggers than the common ones covered here
// (battle-style moves, recoil damage, region-specific mechanics...) - an
// unrecognized combination falls back to "special condition" rather than
// guessing at unfamiliar fields - see
// docs/adr/0006-scope-evolution-condition-text-to-common-cases.md.
func DescribeEvolutionCondition(c EvolutionCondition) string {
	switch {
	case c.Trigger == "trade" && c.TradeSpecies != "":
		return "trade for " + humanizeSlug(c.TradeSpecies)
	case c.Trigger == "trade" && c.HeldItem != "":
		return "trade holding " + humanizeSlug(c.HeldItem)
	case c.Trigger == "trade":
		return "trade"
	case c.Trigger == "shed":
		return "spare Poké Ball & party space"
	case c.Item != "":
		return "use " + humanizeSlug(c.Item)
	case c.Level > 0 && c.Happiness:
		return fmt.Sprintf("Lv. %d, high friendship", c.Level)
	case c.Level > 0 && c.TimeOfDay != "":
		return fmt.Sprintf("Lv. %d (%s)", c.Level, c.TimeOfDay)
	case c.Level > 0:
		return fmt.Sprintf("Lv. %d", c.Level)
	case c.Happiness && c.TimeOfDay != "":
		return "high friendship (" + c.TimeOfDay + ")"
	case c.Happiness:
		return "high friendship"
	case c.Beauty:
		return "high beauty"
	case c.KnownMove != "":
		return "knows " + humanizeSlug(c.KnownMove)
	case c.TimeOfDay != "":
		return "level up (" + c.TimeOfDay + ")"
	default:
		return "special condition"
	}
}

// humanizeSlug turns a PokeAPI kebab-case resource name (e.g.
// "thunder-stone") into its display form ("Thunder Stone").
func humanizeSlug(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}
