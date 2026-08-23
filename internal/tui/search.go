package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leekli/pokedex-go/internal/pokeapi"
	"github.com/leekli/pokedex-go/internal/pokemon"
)

var (
	searchTitleStyle = lipgloss.NewStyle().Bold(true)
	searchHintStyle  = lipgloss.NewStyle().Faint(true)
	searchErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C03028")).Bold(true)
)

// searchModel is the Search Screen: a text input for a Pokémon name or
// National Dex Number, a loading spinner while a lookup is in flight, and an
// inline Lookup Error / Service Error message on failure (see CONTEXT.md).
type searchModel struct {
	client  *pokeapi.Client
	input   textinput.Model
	spinner spinner.Model
	loading bool
	errMsg  string
}

func newSearchModel(client *pokeapi.Client) searchModel {
	input := textinput.New()
	input.Placeholder = "pikachu, or 25..."
	input.CharLimit = 64
	input.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return searchModel{client: client, input: input, spinner: sp}
}

// reset clears the input and any error/loading state left over from a
// previous visit to this screen - called by App when transitioning back
// here from the Splash or Result Screen.
func (m searchModel) reset() searchModel {
	m.input.SetValue("")
	m.input.Focus()
	m.loading = false
	m.errMsg = ""
	return m
}

func (m searchModel) Update(msg tea.Msg) (searchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			// Ignore keys while a lookup is in flight, to avoid double
			// submits or edits racing the in-flight request.
			return m, nil
		}
		switch msg.Type {
		case tea.KeyEsc:
			return m, switchTo(screenSplash)
		case tea.KeyEnter:
			return m.submit()
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case lookupResultMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = errorMessageFor(msg.err, m.input.Value())
			return m, nil
		}
		return m, showResult(msg.stat, msg.sprite)
	}
	return m, nil
}

func (m searchModel) submit() (searchModel, tea.Cmd) {
	value := strings.TrimSpace(m.input.Value())
	if value == "" {
		return m, nil
	}
	m.loading = true
	m.errMsg = ""
	q := pokemon.ResolveQuery(value)
	return m, tea.Batch(m.spinner.Tick, lookupCmd(m.client, q))
}

// errorMessageFor renders a *pokeapi.LookupError and a *pokeapi.ServiceError
// as distinct messages (CONTEXT.md's Lookup Error vs Service Error), always
// via errors.As rather than string-matching.
func errorMessageFor(err error, query string) string {
	var lookupErr *pokeapi.LookupError
	if errors.As(err, &lookupErr) {
		return fmt.Sprintf("No Pokémon found for %q.", strings.TrimSpace(query))
	}
	return "Couldn't reach PokeAPI — please try again."
}

func (m searchModel) View() string {
	var b strings.Builder
	b.WriteString(searchTitleStyle.Render("Search the Pokédex"))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	switch {
	case m.loading:
		b.WriteString(m.spinner.View() + " Searching...\n")
	case m.errMsg != "":
		b.WriteString(searchErrorStyle.Render(m.errMsg) + "\n")
	default:
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(searchHintStyle.Render("Enter to search · Esc to go back · Ctrl+C to quit"))
	b.WriteString("\n")
	return b.String()
}
