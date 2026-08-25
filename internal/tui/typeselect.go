package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leekli/pokedex-go/internal/pokemon"
	zone "github.com/lrstanley/bubblezone"
)

var (
	typeSelectTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pokemonYellow))
	typeSelectHintStyle   = lipgloss.NewStyle().Faint(true)
	typeSelectCursorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pokemonYellow))
)

// typeSelectZoneID is the mouse click zone id for the type at index i in
// pokemon.AllTypes() — stable across renders since it's derived from the
// type name itself, not from cursor position.
func typeSelectZoneID(typeName string) string {
	return "type-select-" + typeName
}

// typeSelectModel is the Type Select Screen: a list of every real Pokémon
// type, normalized to a capitalized display form and rendered as a colored
// badge (reusing the same TypeColor palette as the Result Screen's Type
// Badges). There's no scrolling or filtering to manage — exactly 18 types
// always exist and always fit on screen — so this is a plain hand-rolled
// cursor over a static list rather than a stateful bubbles component.
type typeSelectModel struct {
	types  []string
	cursor int
	zones  *zone.Manager
}

func newTypeSelectModel(zones *zone.Manager) typeSelectModel {
	return typeSelectModel{types: pokemon.AllTypes(), zones: zones}
}

func (m typeSelectModel) Update(msg tea.Msg) (typeSelectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyEsc:
			return m, goBack()
		case msg.Type == tea.KeyEnter:
			return m, showTypeRoster(m.types[m.cursor])
		case msg.String() == "up" || msg.String() == "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case msg.String() == "down" || msg.String() == "j":
			if m.cursor < len(m.types)-1 {
				m.cursor++
			}
		}
		return m, nil

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		for i, t := range m.types {
			if zoneInBounds(m.zones, typeSelectZoneID(t), msg) {
				m.cursor = i
				return m, showTypeRoster(t)
			}
		}
		return m, nil
	}
	return m, nil
}

func (m typeSelectModel) View() string {
	var b strings.Builder
	b.WriteString(typeSelectTitleStyle.Render("SEARCH BY TYPE"))
	b.WriteString("\n")
	b.WriteString(typeSelectHintStyle.Render("Choose a type to browse its Pokémon."))
	b.WriteString("\n\n")

	for i, t := range m.types {
		cursor := "  "
		if i == m.cursor {
			cursor = typeSelectCursorStyle.Render("▸ ")
		}
		badge := lipgloss.NewStyle().
			Background(pokemon.TypeColor(t)).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 2).
			Render(capitalize(t))
		row := cursor + badge
		b.WriteString(markZone(m.zones, typeSelectZoneID(t), row))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(typeSelectHintStyle.Render("↑/↓ browse · Enter select · Esc back · Ctrl+C quit"))
	b.WriteString("\n")
	return b.String()
}
