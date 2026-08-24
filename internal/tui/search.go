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

// Brand colors shared across the Search Screen's decorative elements (ball
// art, title, input box) - the same palette already used by the Splash
// Screen's logo (see splash_legend.go).
const (
	pokemonYellow = "#FFCB05"
	pokemonBlue   = "#3B4CCA"
	pokemonRed    = "#EE1515"
	pokemonWhite  = "#FFFFFF"
	pokemonInk    = "#222224"
)

var (
	searchTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pokemonYellow))
	searchSubtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C8FE0"))
	searchHintStyle     = lipgloss.NewStyle().Faint(true)
	searchErrorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#C03028")).Bold(true)
	searchExampleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#B7BCC7"))
	searchDexNumStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pokemonBlue))

	searchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(pokemonYellow)).
			Padding(0, 1)

	ballOutlineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(pokemonInk))
	ballRedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(pokemonRed))
	ballWhiteStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(pokemonWhite))
)

// searchInputWidth is the fixed character width of the text input's display
// viewport, so its bordered box stays a stable size as the user types rather
// than growing and shrinking with the query.
const searchInputWidth = 40

// searchExamples are the sample queries hinted below the input box, cycling
// through the brand's yellow/blue/red accents.
var searchExamples = []struct {
	name   string
	accent string
}{
	{"pikachu", pokemonYellow},
	{"charizard", pokemonBlue},
	{"25", pokemonRed},
	{"mew", pokemonYellow},
	{"snorlax", pokemonBlue},
}

// searchFlourishTypes picks a spread of Pokémon types purely for the
// decorative dot flourish above the input box, reusing pokemon.TypeColor so
// its palette stays in sync with the Result Screen's type badges rather than
// duplicating hex values.
var searchFlourishTypes = []string{
	"fire", "water", "grass", "electric", "psychic", "dragon", "poison", "ice", "fairy",
}

// searchModel is the Search Screen: a decorative Poké Ball and title, a
// bordered text input for a Pokémon name or National Dex Number, a loading
// spinner while a lookup is in flight, and an inline Lookup Error / Service
// Error message on failure (see CONTEXT.md).
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
	input.Width = searchInputWidth
	input.Prompt = "» "
	input.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pokemonYellow))
	input.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5B6270"))
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

	ball := searchBallArt()
	b.WriteString("\n")
	b.WriteString(ball[0])
	b.WriteString("\n")
	b.WriteString(ball[1])
	b.WriteString("\n")
	b.WriteString(ball[2])
	b.WriteString("  ")
	b.WriteString(searchTitleStyle.Render("POKÉDEX SEARCH"))
	b.WriteString("\n")
	b.WriteString(ball[3])
	b.WriteString("  ")
	b.WriteString(searchSubtitleStyle.Render("Find any Pokémon by name or National Dex Number."))
	b.WriteString("\n")
	b.WriteString(ball[4])
	b.WriteString("\n")
	b.WriteString(ball[5])
	b.WriteString("\n")
	b.WriteString(ball[6])
	b.WriteString("\n")
	b.WriteString("\n")

	b.WriteString(renderDotFlourish(searchFlourishTypes))
	b.WriteString("\n")
	b.WriteString(searchBoxStyle.Render(m.input.View()))
	b.WriteString("\n\n")

	b.WriteString(searchDexNumStyle.Render("#001"))
	b.WriteString(" — ")
	b.WriteString(searchDexNumStyle.Render("#1025"))
	b.WriteString(" Pokémon are waiting, Trainer!")
	b.WriteString("\n\n")

	switch {
	case m.loading:
		b.WriteString(m.spinner.View())
		b.WriteString(" Searching...\n")
	case m.errMsg != "":
		b.WriteString(searchErrorStyle.Render(m.errMsg))
		b.WriteString("\n")
	default:
		b.WriteString(renderExamples())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(searchHintStyle.Render("Enter to search · Esc to go back · Ctrl+C to quit"))
	b.WriteString("\n")
	return b.String()
}

// searchBallArt renders a small Poké Ball as colored block art, echoing the
// badge on the Splash Screen (see splash_legend.go) at Search Screen scale.
// Each of the 7 lines is pre-styled and equal width (11 cells) so they line
// up cleanly against the title/subtitle text View places beside them.
func searchBallArt() [7]string {
	top := ballOutlineStyle.Render("  ▄▄▄▄▄▄▄  ")
	band := ballOutlineStyle.Render(" █▄▄▄▄▄▄▄█ ")
	bottom := ballOutlineStyle.Render("  ▀▀▀▀▀▀▀  ")
	redRow := ballOutlineStyle.Render(" █") + ballRedStyle.Render("▓▓▓▓▓▓▓") + ballOutlineStyle.Render("█ ")
	whiteRow := ballOutlineStyle.Render(" █") + ballWhiteStyle.Render("░░░░░░░") + ballOutlineStyle.Render("█ ")
	return [7]string{top, redRow, redRow, band, whiteRow, whiteRow, bottom}
}

// renderDotFlourish draws a decorative dot divider, one dot per entry in
// types, each in that type's conventional color - shared by the Search
// Screen (see searchFlourishTypes) and Result Screen (resultFlourishTypes)
// so both scenes use the same brand motif.
func renderDotFlourish(types []string) string {
	dots := make([]string, len(types))
	for i, t := range types {
		dots[i] = lipgloss.NewStyle().Foreground(pokemon.TypeColor(t)).Render("●")
	}
	return strings.Join(dots, " ")
}

// renderExamples draws the sample-query hints shown below the input box
// whenever the screen isn't loading or showing an error.
func renderExamples() string {
	parts := make([]string, len(searchExamples))
	for i, ex := range searchExamples {
		bullet := lipgloss.NewStyle().Foreground(lipgloss.Color(ex.accent)).Render("✦")
		parts[i] = bullet + " " + searchExampleStyle.Render(ex.name)
	}
	return strings.Join(parts, "   ")
}
