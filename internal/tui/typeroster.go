package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leekli/pokedex-go/internal/pokeapi"
	"github.com/leekli/pokedex-go/internal/pokemon"
)

var (
	typeRosterHintStyle  = lipgloss.NewStyle().Faint(true)
	typeRosterErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C03028")).Bold(true)
)

// typeRosterTableHeight is the table's visible row count. Some types (e.g.
// Water) have 150+ real Pokémon, far more than fits on screen, hence a
// scrollable table rather than a plain list.
const typeRosterTableHeight = 14

// typeRosterModel is the Type Roster Screen: a header naming the type in
// its conventional color, and a scrollable table of every real Pokémon of
// that type (Dex #, Name, Generation), National Dex Number ascending.
// Selecting a row (Enter, or a mouse wheel to scroll then Enter — see the
// package README for why a precise per-row mouse click isn't supported)
// looks up that Pokémon's full details, the same way the Search Screen
// does, before advancing to the Result Screen.
type typeRosterModel struct {
	client   *pokeapi.Client
	typeName string

	table   table.Model
	spinner spinner.Model

	loading       bool // fetching the roster itself
	loadingDetail bool // fetching a selected row's full Pokémon details
	errMsg        string
}

func newTypeRosterModel(client *pokeapi.Client, typeName string) typeRosterModel {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "#", Width: 6},
			{Title: "Name", Width: 20},
			{Title: "Generation", Width: 16},
		}),
		table.WithHeight(typeRosterTableHeight),
		table.WithFocused(true),
	)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return typeRosterModel{
		client:   client,
		typeName: typeName,
		table:    t,
		spinner:  sp,
		loading:  true,
	}
}

// busy reports whether the screen is mid-fetch and should ignore
// navigation keys, the same guard the Search Screen applies while a lookup
// is in flight (see searchModel.Update).
func (m typeRosterModel) busy() bool {
	return m.loading || m.loadingDetail
}

func (m typeRosterModel) Update(msg tea.Msg) (typeRosterModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.busy() {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyEsc:
			return m, goBack()
		case tea.KeyEnter:
			row := m.table.SelectedRow()
			if row == nil {
				return m, nil
			}
			m.loadingDetail = true
			m.errMsg = ""
			// row[nameColumn] is the capitalized display name; PokeAPI wants
			// the original lowercase slug back, same normalization the
			// Search Screen already applies to typed input.
			name := pokemon.NormalizeName(row[nameColumn])
			return m, tea.Batch(m.spinner.Tick, lookupCmd(m.client, pokemon.Query{Kind: pokemon.Name, Value: name}))
		}
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd

	case tea.MouseMsg:
		if m.busy() {
			return m, nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.table.MoveUp(1)
		case tea.MouseButtonWheelDown:
			m.table.MoveDown(1)
		}
		return m, nil

	case spinner.TickMsg:
		if !m.busy() {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case typeRosterResultMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = errorMessageFor(msg.err, msg.typeName)
			return m, nil
		}
		m.table.SetRows(rosterRows(msg.entries, msg.generations))
		return m, nil

	case lookupResultMsg:
		m.loadingDetail = false
		if msg.err != nil {
			m.errMsg = errorMessageFor(msg.err, m.table.SelectedRow()[nameColumn])
			return m, nil
		}
		return m, showResult(msg.stat, msg.spriteFront, msg.spriteBack, msg.pokedexEntry, msg.typeEffectiveness)
	}
	return m, nil
}

// nameColumn is the index of the Name column within a rosterRows row —
// what SelectedRow()[nameColumn] and lookupCmd's query both key off.
const nameColumn = 1

// rosterRows renders entries (already Dex-ascending) as table rows, filling
// in each one's Generation where the best-effort generations index has it,
// and an em dash otherwise — the same graceful-degradation the Result
// Screen already applies to a missing sprite.
func rosterRows(entries []pokeapi.TypeRosterEntry, generations map[int]string) []table.Row {
	rows := make([]table.Row, len(entries))
	for i, e := range entries {
		generation := "—"
		if name, ok := generations[e.DexNumber]; ok {
			generation = pokemon.FormatGeneration(name)
		}
		rows[i] = table.Row{fmt.Sprintf("#%03d", e.DexNumber), capitalize(e.Name), generation}
	}
	return rows
}

func (m typeRosterModel) View() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Background(pokemon.TypeColor(m.typeName)).
		Foreground(lipgloss.Color("#000000")).
		Bold(true).
		Padding(0, 2).
		Render(strings.ToUpper(m.typeName))
	b.WriteString(header)
	b.WriteString("\n\n")

	switch {
	case m.loading:
		b.WriteString(m.spinner.View())
		fmt.Fprintf(&b, " Loading %s-type Pokémon...\n", capitalize(m.typeName))
	case m.errMsg != "":
		b.WriteString(typeRosterErrorStyle.Render(m.errMsg))
		b.WriteString("\n")
	case m.loadingDetail:
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading...\n")
	default:
		b.WriteString(m.table.View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(typeRosterHintStyle.Render("↑/↓ scroll · Enter select · Esc back · Ctrl+C quit"))
	b.WriteString("\n")
	return b.String()
}
