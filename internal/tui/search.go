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
	zone "github.com/lrstanley/bubblezone"
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

	searchByTypeButtonStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#5B6270")).
				Foreground(lipgloss.Color("#B7BCC7")).
				Padding(0, 1)

	searchByTypeButtonFocusedStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color(pokemonYellow)).
					Foreground(lipgloss.Color(pokemonYellow)).
					Bold(true).
					Padding(0, 1)

	ballOutlineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(pokemonInk))
	ballRedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(pokemonRed))
	ballWhiteStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(pokemonWhite))
)

// searchInputWidth is the fixed character width of the text input's display
// viewport, so its bordered box stays a stable size as the user types rather
// than growing and shrinking with the query.
const searchInputWidth = 40

// searchByTypeButtonZoneID is the mouse click zone id for the "Search by
// Type" button.
const searchByTypeButtonZoneID = "search-by-type-button"

// searchByTypeButtonLabel is the button's text, shared between the two
// focus-state styles and its zone marker so they always wrap the same
// content.
const searchByTypeButtonLabel = "🔎 Search by Type"

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

// searchFocus identifies which of the Search Screen's two focusable
// controls (the name/number input, or the "Search by Type" button) has
// keyboard focus. Tab/Shift+Tab cycle between them; a mouse click on the
// button jumps focus there directly regardless of the current value.
type searchFocus int

const (
	searchFocusInput searchFocus = iota
	searchFocusButton
)

// searchModel is the Search Screen: a decorative Poké Ball and title, a
// bordered text input for a Pokémon name or National Dex Number, a "Search
// by Type" button beneath it, a loading spinner while a lookup is in
// flight, and an inline Lookup Error / Service Error message on failure
// (see CONTEXT.md).
type searchModel struct {
	client  *pokeapi.Client
	input   textinput.Model
	spinner spinner.Model
	loading bool
	errMsg  string

	focus searchFocus
	zones *zone.Manager
}

func newSearchModel(client *pokeapi.Client, zones *zone.Manager) searchModel {
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

	return searchModel{client: client, input: input, spinner: sp, zones: zones}
}

// reset clears the input and any error/loading/focus state left over from a
// previous visit to this screen - called by App when transitioning back
// here from the Splash, Type Select, or Result Screen.
func (m searchModel) reset() searchModel {
	m.input.SetValue("")
	m.input.Focus()
	m.loading = false
	m.errMsg = ""
	m.focus = searchFocusInput
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
		switch {
		case msg.Type == tea.KeyEsc:
			return m, goBack()
		case msg.Type == tea.KeyTab, msg.Type == tea.KeyShiftTab:
			return m.toggleFocus(), nil
		case msg.Type == tea.KeyEnter && m.focus == searchFocusButton:
			return m, switchTo(screenTypeSelect)
		case msg.Type == tea.KeyEnter:
			return m.submit()
		}
		if m.focus == searchFocusButton {
			// No free-text entry while the button has focus.
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case tea.MouseMsg:
		if m.loading || msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		if zoneInBounds(m.zones, searchByTypeButtonZoneID, msg) {
			return m, switchTo(screenTypeSelect)
		}
		return m, nil

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
		return m, showResult(msg.stat, msg.spriteFront, msg.spriteBack, msg.pokedexEntry, msg.evolutionChain, msg.typeEffectiveness)
	}
	return m, nil
}

// toggleFocus moves keyboard focus between the input and the "Search by
// Type" button, blurring/focusing the textinput sub-model to match so its
// own cursor stops rendering while the button is focused.
func (m searchModel) toggleFocus() searchModel {
	if m.focus == searchFocusInput {
		m.focus = searchFocusButton
		m.input.Blur()
	} else {
		m.focus = searchFocusInput
		m.input.Focus()
	}
	return m
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
	b.WriteString("\n")

	buttonStyle := searchByTypeButtonStyle
	if m.focus == searchFocusButton {
		buttonStyle = searchByTypeButtonFocusedStyle
	}
	b.WriteString(markZone(m.zones, searchByTypeButtonZoneID, buttonStyle.Render(searchByTypeButtonLabel)))
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
	b.WriteString(searchHintStyle.Render("Enter to search · Tab to switch to Search by Type · Esc to go back · Ctrl+C to quit"))
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
