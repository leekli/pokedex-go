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

// EvolutionPath describes how one specific Pokémon fits into its
// EvolutionChain: the stages from the chain's root down to (and including)
// it, plus what it can evolve into next (its own EvolvesTo — a branch, e.g.
// Eevee's, means more than one). Plain data, like EvolutionChain itself — a
// renderer decides how to lay it out (arrows, branches, which stage is
// highlighted).
type EvolutionPath struct {
	Stages    []EvolutionStage
	EvolvesTo []EvolutionStage
}

// PathTo finds dexNumber within c's tree and returns the path from the
// chain's root down to (and including) that stage, along with what it
// evolves into next. The zero value (nil Stages) means dexNumber wasn't
// found anywhere in the chain — shouldn't happen for the chain's own
// species, but callers should still check rather than assume.
func (c EvolutionChain) PathTo(dexNumber int) EvolutionPath {
	stages := evolutionPathTo(c.Root, dexNumber, nil)
	if len(stages) == 0 {
		return EvolutionPath{}
	}
	return EvolutionPath{Stages: stages, EvolvesTo: stages[len(stages)-1].EvolvesTo}
}

// DoesNotEvolve reports whether p represents a Pokémon with no evolution
// relations at all (e.g. Tauros) — real information, not a degraded/failure
// state, distinct from PathTo not finding dexNumber at all.
func (p EvolutionPath) DoesNotEvolve() bool {
	return len(p.Stages) == 1 && len(p.EvolvesTo) == 0
}

// evolutionPathTo walks stage's subtree looking for dexNumber, returning
// the path from the original root down to (and including) the matching
// stage, or nil if dexNumber isn't found anywhere in it. ancestors is the
// path so far (nil at the top-level call); a fresh slice is built at each
// step rather than appending into a shared one, so sibling branches never
// alias the same backing array.
func evolutionPathTo(stage EvolutionStage, dexNumber int, ancestors []EvolutionStage) []EvolutionStage {
	path := append(append([]EvolutionStage{}, ancestors...), stage)
	if stage.DexNumber == dexNumber {
		return path
	}
	for _, child := range stage.EvolvesTo {
		if found := evolutionPathTo(child, dexNumber, path); found != nil {
			return found
		}
	}
	return nil
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

// Evolution trigger names, as PokeAPI names them, that DescribeEvolutionCondition
// gives specific text to rather than falling back to "special condition".
const (
	triggerTrade = "trade"
	triggerShed  = "shed"
)

// Condition display text reused across more than one DescribeEvolutionCondition
// branch below. Kept separate from the trigger constants above even where a
// value happens to match (triggerTrade / conditionTrade are both "trade") -
// one is PokeAPI's trigger vocabulary, the other is this app's display text,
// and conflating them would make a future trigger-name change silently
// change display text too.
const (
	conditionTrade            = "trade"
	conditionHighFriendship   = "high friendship"
	conditionSpecialCondition = "special condition"
)

// DescribeEvolutionCondition renders c as the short phrase shown next to an
// evolution arrow (e.g. "Lv. 16", "use Thunder Stone", "trade"). PokeAPI
// supports far more evolution triggers than the common ones covered here
// (battle-style moves, recoil damage, region-specific mechanics...) - an
// unrecognized combination falls back to "special condition" rather than
// guessing at unfamiliar fields - see
// docs/adr/0006-scope-evolution-condition-text-to-common-cases.md.
func DescribeEvolutionCondition(c EvolutionCondition) string {
	switch {
	case c.Trigger == triggerTrade && c.TradeSpecies != "":
		return "trade for " + humanizeSlug(c.TradeSpecies)
	case c.Trigger == triggerTrade && c.HeldItem != "":
		return "trade holding " + humanizeSlug(c.HeldItem)
	case c.Trigger == triggerTrade:
		return conditionTrade
	case c.Trigger == triggerShed:
		return "spare Poké Ball & party space"
	case c.Item != "":
		return "use " + humanizeSlug(c.Item)
	case c.Level > 0 && c.Happiness:
		return fmt.Sprintf("Lv. %d, %s", c.Level, conditionHighFriendship)
	case c.Level > 0 && c.TimeOfDay != "":
		return fmt.Sprintf("Lv. %d (%s)", c.Level, c.TimeOfDay)
	case c.Level > 0:
		return fmt.Sprintf("Lv. %d", c.Level)
	case c.Happiness && c.TimeOfDay != "":
		return conditionHighFriendship + " (" + c.TimeOfDay + ")"
	case c.Happiness:
		return conditionHighFriendship
	case c.Beauty:
		return "high beauty"
	case c.KnownMove != "":
		return "knows " + humanizeSlug(c.KnownMove)
	case c.TimeOfDay != "":
		return "level up (" + c.TimeOfDay + ")"
	default:
		return conditionSpecialCondition
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
