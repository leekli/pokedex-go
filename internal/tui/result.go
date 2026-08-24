package tui

import (
	"fmt"
	"image"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leekli/pokedex-go/internal/pokemon"
	"github.com/leekli/pokedex-go/internal/spriteart"
)

const spriteMaxWidth = 40

// resultCardWidth is the fixed width the Result Screen's card content is
// centered/padded to, so the card's size doesn't jump around between
// Pokémon with short vs. long names.
const resultCardWidth = 52

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
	stat   pokemon.StatBlock
	sprite image.Image
}

func newResultModel(stat pokemon.StatBlock, sprite image.Image) resultModel {
	return resultModel{stat: stat, sprite: sprite}
}

// Update handles the Result Screen's keys directly: Esc or Enter return to
// the Search Screen; Q quits (safe here too - no free-text input on this
// screen). Ctrl+C is handled globally by App.
func (m resultModel) Update(msg tea.Msg) (resultModel, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch km.Type {
	case tea.KeyEsc, tea.KeyEnter:
		return m, switchTo(screenSearch)
	}
	if km.String() == "q" {
		return m, tea.Quit
	}
	return m, nil
}

func (m resultModel) View() string {
	var b strings.Builder
	b.WriteString(resultCardStyle.Render(renderResultCard(m.stat, m.sprite)))
	b.WriteString("\n\n")
	b.WriteString(resultHintStyle.Render("Enter/Esc to search again · Q to quit"))
	b.WriteString("\n")
	return b.String()
}

// renderResultCard renders the card's interior: title, sprite (or its "no
// sprite" fallback), type badges, height/weight, a small dot flourish, and
// the base stats table - everything CONTEXT.md's Stat Block already shows,
// just restyled.
func renderResultCard(stat pokemon.StatBlock, sprite image.Image) string {
	var b strings.Builder

	title := fmt.Sprintf("#%03d %s", stat.DexNumber, capitalize(stat.Name))
	b.WriteString(resultCenterStyle.Render(resultTitleStyle.Render(title)))
	b.WriteString("\n\n")

	if sprite != nil {
		art := spriteart.Render(sprite, spriteart.Options{MaxWidth: spriteMaxWidth})
		b.WriteString(resultCenterStyle.Render(art))
	} else {
		b.WriteString(resultCenterStyle.Render(noSpriteStyle.Render("No sprite available")))
	}
	b.WriteString("\n\n")

	b.WriteString(resultCenterStyle.Render(renderTypeBadges(stat.Types)))
	b.WriteString("\n\n")

	b.WriteString(resultCenterStyle.Render(renderHeightWeight(stat)))
	b.WriteString("\n\n")

	b.WriteString(resultCenterStyle.Render(renderDotFlourish(resultFlourishTypes)))
	b.WriteString("\n\n")

	b.WriteString(renderStatTable(stat))

	return b.String()
}

func renderHeightWeight(stat pokemon.StatBlock) string {
	height := fmt.Sprintf("%d'%02d\"", stat.HeightFeet, stat.HeightInches)
	weight := fmt.Sprintf("%.1f lbs", stat.WeightPounds)
	return resultLabelStyle.Render("Height") + "   " + height +
		"      " + resultLabelStyle.Render("Weight") + "   " + weight
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
