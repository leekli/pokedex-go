package tui

import (
	"fmt"
	"image"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leekli/pokedex-go/internal/pokemon"
	"github.com/leekli/pokedex-go/internal/spriteart"
)

const spriteMaxWidth = 40

var (
	resultTitleStyle = lipgloss.NewStyle().Bold(true)
	resultHintStyle  = lipgloss.NewStyle().Faint(true)
	statLabelStyle   = lipgloss.NewStyle().Bold(true).Width(16)
	noSpriteStyle    = lipgloss.NewStyle().Faint(true).Italic(true)
)

// resultModel is the Result Screen: the looked-up Pokémon's rendered Sprite
// (or a fallback message) and its Stat Block.
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

	title := fmt.Sprintf("#%03d %s", m.stat.DexNumber, capitalize(m.stat.Name))
	b.WriteString(resultTitleStyle.Render(title))
	b.WriteString("\n\n")

	if m.sprite != nil {
		b.WriteString(spriteart.Render(m.sprite, spriteart.Options{MaxWidth: spriteMaxWidth}))
	} else {
		b.WriteString(noSpriteStyle.Render("No sprite available"))
	}
	b.WriteString("\n\n")

	b.WriteString(renderTypeBadges(m.stat.Types))
	b.WriteString("\n\n")

	b.WriteString(statRow("Height", fmt.Sprintf("%d'%02d\"", m.stat.HeightFeet, m.stat.HeightInches)))
	b.WriteString(statRow("Weight", fmt.Sprintf("%.1f lbs", m.stat.WeightPounds)))
	b.WriteString("\n")
	b.WriteString(statRow("HP", fmt.Sprintf("%d", m.stat.HP)))
	b.WriteString(statRow("Attack", fmt.Sprintf("%d", m.stat.Attack)))
	b.WriteString(statRow("Defense", fmt.Sprintf("%d", m.stat.Defense)))
	b.WriteString(statRow("Sp. Atk", fmt.Sprintf("%d", m.stat.SpecialAttack)))
	b.WriteString(statRow("Sp. Def", fmt.Sprintf("%d", m.stat.SpecialDefense)))
	b.WriteString(statRow("Speed", fmt.Sprintf("%d", m.stat.Speed)))

	b.WriteString("\n")
	b.WriteString(resultHintStyle.Render("Enter/Esc to search again · Q to quit"))
	b.WriteString("\n")

	return b.String()
}

func statRow(label, value string) string {
	return statLabelStyle.Render(label) + value + "\n"
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

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
