package tui

import (
	"fmt"
	"image"
	"math"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leekli/pokedex-go/internal/pokemon"
	"github.com/leekli/pokedex-go/internal/spriteart"
)

// spriteMaxWidth caps a single sprite's rendered width (in terminal
// columns) when only one of front/back is available - see renderSprites.
// spriteHalfWidth caps each one's width when both are shown side by side;
// it's deliberately close to spriteMaxWidth (rather than half of it) so
// splitting the sprite row in two doesn't double spriteart.Render's
// downsample scale and destroy small but diagnostic sprite detail (a
// Pikachu's cheek, an eye) - see docs/adr/0005-keep-both-sprites-near-full-resolution.md.
// spriteGap is the blank column gap between the two.
const (
	spriteMaxWidth  = 40
	spriteHalfWidth = 32
	spriteGap       = 2
)

// resultCardWidth is the fixed width the Result Screen's card content is
// centered/padded to, so the card's size doesn't jump around between
// Pokémon with short vs. long names. Wide enough to fit the front+back
// sprite row (2*spriteHalfWidth + spriteGap = 66) with a little margin -
// see docs/adr/0005-keep-both-sprites-near-full-resolution.md for why that
// row needs to be this wide in the first place.
const resultCardWidth = 70

// statBarWidth is the bar's cell width (in block characters) in the base
// stats table; maxBaseStat (255) is the highest possible base stat value
// across all Pokémon, used to scale each bar.
const (
	statBarWidth = 20
	maxBaseStat  = 255
)

var (
	resultTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pokemonYellow))
	resultHintStyle  = lipgloss.NewStyle().Faint(true)
	resultLabelStyle = lipgloss.NewStyle().Bold(true)
	noSpriteStyle    = lipgloss.NewStyle().Faint(true).Italic(true)

	resultCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(pokemonYellow)).
			Padding(0, 1)

	resultCenterStyle = lipgloss.NewStyle().Width(resultCardWidth).Align(lipgloss.Center)
	resultEntryStyle  = lipgloss.NewStyle().Width(resultCardWidth)
	// resultFallbackStyle marks any "no data available" message on the
	// card (no sprite, no Pokédex Entry, no type effectiveness) as
	// distinct from real content.
	resultFallbackStyle = lipgloss.NewStyle().Faint(true).Italic(true)

	statTableBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#30363D"))
	statTableHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6E7681"))
	statTableStripeStyle = lipgloss.NewStyle().Background(lipgloss.Color(statTableStripeColor))
)

// statTableStripeColor shades alternating stat rows. It's threaded directly
// into renderStatBar's own segment styles (rather than left to the table's
// per-cell background) because lipgloss.Style.Render always closes with its
// own reset: a background applied only around the finished bar string would
// be cut off right after the bar's first internally-styled run.
const statTableStripeColor = "#161B23"
const statTableEmptyColor = "#30363D"

// resultFlourishTypes are the types shown in the Result Screen's dot
// flourish - fewer than the Search Screen's (see searchFlourishTypes),
// since CONTEXT.md's Stat Block is a dense data screen where the same
// maximalist brand motif should read as a light touch, not a repeat.
var resultFlourishTypes = []string{"fire", "water", "grass", "electric", "psychic"}

// resultModel is the Result Screen: the looked-up Pokémon's rendered Sprite
// (or a fallback message) and its Stat Block, presented as a single bordered
// card - echoing the Search Screen's input box - with the base stats laid
// out as a table of colored bars.
type resultModel struct {
	stat              pokemon.StatBlock
	spriteFront       image.Image
	spriteBack        image.Image
	pokedexEntry      string
	evolutionChain    *pokemon.EvolutionChain
	typeEffectiveness *pokemon.TypeEffectiveness
}

func newResultModel(stat pokemon.StatBlock, spriteFront, spriteBack image.Image, pokedexEntry string, evolutionChain *pokemon.EvolutionChain, typeEffectiveness *pokemon.TypeEffectiveness) resultModel {
	return resultModel{
		stat:              stat,
		spriteFront:       spriteFront,
		spriteBack:        spriteBack,
		pokedexEntry:      pokedexEntry,
		evolutionChain:    evolutionChain,
		typeEffectiveness: typeEffectiveness,
	}
}

// Update handles the Result Screen's keys directly: Esc goes back to
// whichever screen led here (the Search Screen, or the Type Roster Screen -
// see docs/adr/0001-navigation-history-for-back-navigation.md); Enter has a
// single fixed meaning, "look up another Pokémon," always landing on a
// freshly-reset Search Screen regardless of how this screen was reached. Q
// quits (safe here too - no free-text input on this screen). Ctrl+C is
// handled globally by App.
func (m resultModel) Update(msg tea.Msg) (resultModel, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch km.Type {
	case tea.KeyEsc:
		return m, goBack()
	case tea.KeyEnter:
		return m, searchAgain()
	}
	if km.String() == "q" {
		return m, tea.Quit
	}
	return m, nil
}

func (m resultModel) View() string {
	var b strings.Builder
	b.WriteString(resultCardStyle.Render(renderResultCard(m.stat, m.spriteFront, m.spriteBack, m.pokedexEntry, m.evolutionChain, m.typeEffectiveness)))
	b.WriteString("\n\n")
	b.WriteString(resultHintStyle.Render("Enter/Esc to search again · Q to quit"))
	b.WriteString("\n")
	return b.String()
}

// renderResultCard renders the card's interior: title, sprites (front and
// back, side by side - or a "no sprite" fallback if neither is available),
// type badges, the Evolution Chain (or its own fallback - see
// docs/adr/0006-scope-evolution-condition-text-to-common-cases.md), the
// Pokédex Entry (or its own fallback - see
// docs/adr/0003-prefer-generation-1-pokedex-entry-version.md),
// Weaknesses & Resistances (or its own fallback - see
// docs/adr/0004-all-or-nothing-type-effectiveness.md), height/weight, a
// small dot flourish, and the base stats table - everything CONTEXT.md's
// Stat Block already shows, just restyled, plus the sections alongside it.
func renderResultCard(stat pokemon.StatBlock, spriteFront, spriteBack image.Image, pokedexEntry string, evolutionChain *pokemon.EvolutionChain, typeEffectiveness *pokemon.TypeEffectiveness) string {
	var b strings.Builder

	title := fmt.Sprintf("#%03d %s", stat.DexNumber, capitalize(stat.Name))
	b.WriteString(resultCenterStyle.Render(resultTitleStyle.Render(title)))
	b.WriteString("\n\n")

	b.WriteString(resultCenterStyle.Render(renderSprites(spriteFront, spriteBack)))
	b.WriteString("\n\n")

	b.WriteString(resultCenterStyle.Render(renderTypeBadges(stat.Types)))
	b.WriteString("\n\n")

	b.WriteString(resultCenterStyle.Render(renderEvolutionChain(stat.DexNumber, evolutionChain)))
	b.WriteString("\n\n")

	if pokedexEntry != "" {
		b.WriteString(resultEntryStyle.Render(pokedexEntry))
	} else {
		b.WriteString(resultEntryStyle.Render(resultFallbackStyle.Render("No Pokédex entry available.")))
	}
	b.WriteString("\n\n")

	b.WriteString(renderTypeEffectiveness(typeEffectiveness))
	b.WriteString("\n\n")

	b.WriteString(resultCenterStyle.Render(renderHeightWeight(stat)))
	b.WriteString("\n\n")

	b.WriteString(resultCenterStyle.Render(renderDotFlourish(resultFlourishTypes)))
	b.WriteString("\n\n")

	b.WriteString(resultCenterStyle.Render(renderStatTable(stat)))

	return b.String()
}

// renderSprites renders the front sprite on the left and the back sprite on
// the right, separated by a spriteGap-wide blank column. Either may be nil
// (PokeAPI had none, or its fetch failed - see lookupCmd), in which case
// only the other is rendered; if both are nil, the "no sprite" fallback
// message is shown instead. When both are present, each is capped to
// spriteHalfWidth rather than a naive half of spriteMaxWidth - see
// docs/adr/0005-keep-both-sprites-near-full-resolution.md on why that
// naive split destroys small sprite detail. A lone sprite gets the full
// spriteMaxWidth, matching the Result Screen's pre-back-sprite sizing.
func renderSprites(spriteFront, spriteBack image.Image) string {
	if spriteFront == nil && spriteBack == nil {
		return noSpriteStyle.Render("No sprite available")
	}
	if spriteFront != nil && spriteBack != nil {
		front := spriteart.Render(spriteFront, spriteart.Options{MaxWidth: spriteHalfWidth})
		back := spriteart.Render(spriteBack, spriteart.Options{MaxWidth: spriteHalfWidth})
		return lipgloss.JoinHorizontal(lipgloss.Top, front, strings.Repeat(" ", spriteGap), back)
	}

	img := spriteFront
	if img == nil {
		img = spriteBack
	}
	return spriteart.Render(img, spriteart.Options{MaxWidth: spriteMaxWidth})
}

// evolutionArrow and evolutionSlash separate consecutive names in the
// Evolution Chain breadcrumb: an arrow between stages on the path to the
// currently-viewed Pokémon, a slash between siblings the current stage
// could evolve into (e.g. Eevee's many evolutions).
const (
	evolutionArrow = " → "
	evolutionSlash = " / "
)

// renderEvolutionChain renders the Evolution Chain (see CONTEXT.md) as a
// centered breadcrumb: every stage from the chain's root up to the
// currently-viewed Pokémon (highlighted), followed by what it evolves into
// next (if anything - siblings from a branch like Eevee's are slash-
// separated), with each transition's condition annotated on a second line
// roughly beneath the name it leads into. nil (the chain fetch failed, or
// PokeAPI had none) shows a fallback message instead - best-effort, the
// same as the Sprite and Pokédex Entry (see lookupCmd's doc comment). A
// Pokémon with no evolution relations at all (e.g. Tauros) shows "Does not
// evolve" instead - real information, not a degraded/failure state.
func renderEvolutionChain(dexNumber int, chain *pokemon.EvolutionChain) string {
	if chain == nil {
		return resultFallbackStyle.Render("Evolution data unavailable")
	}

	path := evolutionPathTo(chain.Root, dexNumber, nil)
	if len(path) == 0 {
		// Defensive: the Result Screen's own Pokémon should always be
		// somewhere in its own Evolution Chain.
		return resultFallbackStyle.Render("Evolution data unavailable")
	}

	current := path[len(path)-1]
	if len(path) == 1 && len(current.EvolvesTo) == 0 {
		return "Does not evolve"
	}

	segments := make([]evolutionSegment, 0, len(path)+len(current.EvolvesTo))
	for i, stage := range path {
		segments = append(segments, evolutionSegment{
			name:      stage.Name,
			current:   i == len(path)-1,
			condition: stage.Condition,
		})
	}
	branchStart := len(segments)
	for _, child := range current.EvolvesTo {
		segments = append(segments, evolutionSegment{name: child.Name, condition: child.Condition})
	}

	return renderEvolutionBreadcrumb(segments, branchStart)
}

// evolutionPathTo walks stage's subtree looking for dexNumber, returning
// the path from the original root down to (and including) the matching
// stage, or nil if dexNumber isn't found anywhere in it. ancestors is the
// path so far (nil at the top-level call); a fresh slice is built at each
// step rather than appending into a shared one, so sibling branches never
// alias the same backing array.
func evolutionPathTo(stage pokemon.EvolutionStage, dexNumber int, ancestors []pokemon.EvolutionStage) []pokemon.EvolutionStage {
	path := append(append([]pokemon.EvolutionStage{}, ancestors...), stage)
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

// evolutionSegment is one name shown in the Evolution Chain breadcrumb: a
// stage on the path to the currently-viewed Pokémon, or one of that
// Pokémon's own possible evolutions. current marks the Pokémon actually
// being viewed, rendered highlighted; condition is the text describing the
// transition leading into this segment, "" only for the chain's root
// (which has no incoming transition to describe).
type evolutionSegment struct {
	name      string
	current   bool
	condition string
}

// renderEvolutionBreadcrumb lays segments out left to right - arrow-joined,
// except between branchStart's siblings (a branch's second child onward),
// which are slash-joined instead - with a second line beneath annotating
// each segment's condition directly under its name. The transition from
// the currently-viewed stage into its first child is still an arrow: only
// siblings of that first child (Eevee's second, third, ... evolution) are
// slash-separated from one another.
//
// Each segment reserves a column width of max(name width, condition
// width) - a condition ("high friendship") is very often wider than the
// name above it ("Pikachu"), so sizing columns by name width alone would
// let condition text collide with its neighbor; this keeps both lines the
// same total width, segment by segment, however long either one runs.
// Widths are tracked visibly (utf8.RuneCountInString on the plain name),
// not by string length, so the currently-viewed segment's lipgloss styling
// (which adds invisible ANSI bytes) never throws off alignment.
func renderEvolutionBreadcrumb(segments []evolutionSegment, branchStart int) string {
	var line1, line2 strings.Builder

	for i, seg := range segments {
		if i > 0 {
			sep := evolutionArrow
			if i > branchStart {
				sep = evolutionSlash
			}
			line1.WriteString(sep)
			line2.WriteString(strings.Repeat(" ", utf8.RuneCountInString(sep)))
		}

		display := capitalize(seg.name)
		nameWidth := utf8.RuneCountInString(display)
		conditionWidth := utf8.RuneCountInString(seg.condition)
		colWidth := max(nameWidth, conditionWidth)

		name := display
		if seg.current {
			name = resultTitleStyle.Render(display)
		}
		line1.WriteString(name)
		line1.WriteString(strings.Repeat(" ", colWidth-nameWidth))

		line2.WriteString(seg.condition)
		line2.WriteString(strings.Repeat(" ", colWidth-conditionWidth))
	}

	return line1.String() + "\n" + resultHintStyle.Render(line2.String())
}

func renderHeightWeight(stat pokemon.StatBlock) string {
	height := fmt.Sprintf("%d'%02d\"", stat.HeightFeet, stat.HeightInches)
	weight := fmt.Sprintf("%.1f lbs", stat.WeightPounds)
	return resultLabelStyle.Render("Height") + "   " + height +
		"      " + resultLabelStyle.Render("Weight") + "   " + weight
}

// renderTypeEffectiveness renders the Weaknesses & Resistances section (see
// CONTEXT.md): every type that deals super effective, not very effective,
// or no damage to this Pokémon, color-coded the same way as its Type
// Badges. nil (any of the Pokémon's own types' damage relations failed to
// fetch) shows a fallback message instead of a possibly-wrong partial
// chart - see docs/adr/0004-all-or-nothing-type-effectiveness.md.
func renderTypeEffectiveness(te *pokemon.TypeEffectiveness) string {
	if te == nil {
		return resultEntryStyle.Render(resultFallbackStyle.Render("Weaknesses & resistances unavailable."))
	}

	var b strings.Builder
	b.WriteString(resultLabelStyle.Render("Weak to") + "    " + renderMatchups(te.Weaknesses))
	b.WriteString("\n")
	b.WriteString(resultLabelStyle.Render("Resists") + "    " + renderMatchups(te.Resistances))
	if len(te.Immunities) > 0 {
		b.WriteString("\n")
		b.WriteString(resultLabelStyle.Render("Immune to") + "  " + renderTypeNames(te.Immunities))
	}
	return resultEntryStyle.Render(b.String())
}

// renderMatchups renders a list of TypeMatchups as "Type Nx" pairs, each
// type name colored via pokemon.TypeColor, or "None" for an empty list
// (a Pokémon can genuinely have zero weaknesses, or zero resistances).
func renderMatchups(matchups []pokemon.TypeMatchup) string {
	if len(matchups) == 0 {
		return "None"
	}
	parts := make([]string, len(matchups))
	for i, m := range matchups {
		style := lipgloss.NewStyle().Foreground(pokemon.TypeColor(m.Type))
		parts[i] = style.Render(capitalize(m.Type)) + " " + formatMultiplier(m.Multiplier)
	}
	return strings.Join(parts, ", ")
}

// renderTypeNames renders a list of type names, each colored via
// pokemon.TypeColor - used for Immunities, which (unlike Weaknesses and
// Resistances) has no multiplier to show alongside the name.
func renderTypeNames(types []string) string {
	parts := make([]string, len(types))
	for i, t := range types {
		style := lipgloss.NewStyle().Foreground(pokemon.TypeColor(t))
		parts[i] = style.Render(capitalize(t))
	}
	return strings.Join(parts, ", ")
}

// formatMultiplier renders a type effectiveness multiplier the way real
// Pokédex UIs do (×, ½, ¼) rather than as a raw float. With at most two of
// a Pokémon's own types combined, the multiplier is always one of these
// four values - the default case only guards against a future change to
// BuildTypeEffectiveness's combination math.
func formatMultiplier(m float64) string {
	switch m {
	case 4:
		return "4×"
	case 2:
		return "2×"
	case 0.5:
		return "½×"
	case 0.25:
		return "¼×"
	default:
		return fmt.Sprintf("%g×", m)
	}
}

func renderTypeBadges(types []string) string {
	badges := make([]string, 0, len(types))
	for _, t := range types {
		style := lipgloss.NewStyle().
			Background(pokemon.TypeColor(t)).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 1)
		badges = append(badges, style.Render(strings.ToUpper(t)))
	}
	return strings.Join(badges, " ")
}

// renderStatTable lays out the six base stats as a lipgloss table: STAT
// label, a colored bar scaled out of maxBaseStat, and the numeric VALUE -
// bar color tied to the Pokémon's primary type so the table visually
// belongs to this specific Pokémon. Odd rows get a subtle background stripe
// for readability; no cell borders, just a rule under the header.
func renderStatTable(stat pokemon.StatBlock) string {
	barColor := string(pokemon.TypeColor(stat.Types[0]))

	entries := []struct {
		label string
		value int
	}{
		{"HP", stat.HP},
		{"Attack", stat.Attack},
		{"Defense", stat.Defense},
		{"Sp. Atk", stat.SpecialAttack},
		{"Sp. Def", stat.SpecialDefense},
		{"Speed", stat.Speed},
	}

	t := table.New().
		Headers("STAT", "", "VALUE").
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		BorderHeader(true).
		BorderStyle(statTableBorderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle()
			if col == 0 {
				s = s.PaddingRight(2)
			}
			if col == 2 {
				s = s.PaddingLeft(2).Align(lipgloss.Right)
			}
			switch {
			case row == table.HeaderRow:
				return s.Inherit(statTableHeaderStyle)
			// Column 1 (the bar) is deliberately excluded here - see
			// statTableStripeColor - its striping is baked into
			// renderStatBar's own segments instead.
			case col != 1 && row%2 == 1:
				return s.Inherit(statTableStripeStyle)
			}
			return s
		})

	for i, e := range entries {
		stripe := ""
		if i%2 == 1 {
			stripe = statTableStripeColor
		}
		t.Row(e.label, renderStatBar(e.value, barColor, stripe), fmt.Sprintf("%d", e.value))
	}

	return t.Render()
}

// renderStatBar draws one base-stat bar: a run of filled blocks (in
// fillColor) followed by empty blocks, scaled out of maxBaseStat. When
// stripeBg is non-empty it's set on both runs' own style rather than
// wrapped around the finished string - see statTableStripeColor for why.
func renderStatBar(value int, fillColor, stripeBg string) string {
	filled := int(math.Round(float64(value) / float64(maxBaseStat) * float64(statBarWidth)))
	filled = min(max(filled, 0), statBarWidth)
	empty := statBarWidth - filled

	filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(fillColor))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statTableEmptyColor))
	if stripeBg != "" {
		filledStyle = filledStyle.Background(lipgloss.Color(stripeBg))
		emptyStyle = emptyStyle.Background(lipgloss.Color(stripeBg))
	}
	return filledStyle.Render(strings.Repeat("█", filled)) + emptyStyle.Render(strings.Repeat("░", empty))
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
