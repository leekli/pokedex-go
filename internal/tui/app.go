// Package tui is pokedex-go's Bubble Tea layer: the Splash, Search, Type
// Select, Type Roster, and Result screens, and the App model that switches
// between them. This is the only package that imports both bubbletea and
// the pokeapi/pokemon layers below it.
package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leekli/pokedex-go/internal/pokeapi"
	zone "github.com/lrstanley/bubblezone"
)

// screen identifies which screen App is currently showing.
type screen int

const (
	screenSplash screen = iota
	screenSearch
	screenTypeSelect
	screenTypeRoster
	screenResult
)

// viewportChromeReservation is the number of terminal rows App reserves
// below a viewport-wrapped screen's content for the scroll indicator (see
// App.View) - reserved unconditionally, whether or not the indicator
// actually renders on a given frame, so toggling it on/off never shifts
// the viewport's own height/scroll math mid-session.
const viewportChromeReservation = 1

// viewportPagerKeyMap is the key set App's shared viewport responds to on
// the Splash, Search, and Result Screens: literal arrow/page keys only -
// deliberately no vim letters (j/k/f/b/u/d), no space, no half-page
// bindings, and no left/right. Vim letters and space are real substrings
// of Pokémon names the Search Screen's text input must stay free to
// receive (e.g. "pikachu" contains both "u" and "k"); left/right stay
// with the Search Screen's own text cursor; horizontal scrolling is out
// of scope entirely - see
// docs/adr/0007-app-level-viewport-with-restricted-keymap-for-scrolling.md.
var viewportPagerKeyMap = viewport.KeyMap{
	Up:       key.NewBinding(key.WithKeys("up")),
	Down:     key.NewBinding(key.WithKeys("down")),
	PageUp:   key.NewBinding(key.WithKeys("pgup")),
	PageDown: key.NewBinding(key.WithKeys("pgdown")),
}

// viewportPagingOnlyKeyMap is viewportPagerKeyMap without Up/Down, used on
// the Type Select Screen, where those keys must stay dedicated to cursor
// movement (see typeSelectModel.Update) rather than compete with it for
// the same keys - see docs/adr/0007.
var viewportPagingOnlyKeyMap = viewport.KeyMap{
	PageUp:   key.NewBinding(key.WithKeys("pgup")),
	PageDown: key.NewBinding(key.WithKeys("pgdown")),
}

var viewportScrollIndicatorStyle = lipgloss.NewStyle().Faint(true)

// App is the root Bubble Tea model. It owns the current screen, one
// sub-model per screen, and the navigation history that lets every screen's
// Esc mean "go back one screen" regardless of how it was reached — see
// docs/adr/0001-navigation-history-for-back-navigation.md. It also owns the
// terminal size (from the last tea.WindowSizeMsg) and a single viewport
// shared across every screen that needs to scroll - see
// docs/adr/0007-app-level-viewport-with-restricted-keymap-for-scrolling.md
// on why that lives here rather than inside each screen model.
type App struct {
	screen  screen
	history []screen

	splash     splashModel
	search     searchModel
	typeSelect typeSelectModel
	typeRoster typeRosterModel
	result     resultModel

	width, height int
	viewport      viewport.Model

	zones *zone.Manager
}

// NewApp builds an App that will use client for all PokeAPI lookups.
func NewApp(client *pokeapi.Client) App {
	zones := zone.New()

	// viewport.New (not a bare viewport.Model{} literal) is required here,
	// and a.viewport must never be reassigned/rebuilt after this: viewport's
	// own Update resets KeyMap back to viewport.DefaultKeyMap() the first
	// time it's used unless the model was constructed via viewport.New - see
	// docs/adr/0007 for why silently losing this override would matter (it
	// would let vim letters and space reach the viewport instead of the
	// Search Screen's text input).
	vp := viewport.New(0, 0)
	vp.KeyMap = viewportPagerKeyMap

	return App{
		screen:     screenSplash,
		splash:     newSplashModel(),
		search:     newSearchModel(client, zones),
		typeSelect: newTypeSelectModel(zones),
		viewport:   vp,
		zones:      zones,
	}
}

// Close releases resources NewApp allocated (currently, the mouse zone
// manager's background worker). Not part of tea.Model — call it once after
// the Bubble Tea program has finished running.
func (a App) Close() {
	if a.zones != nil {
		a.zones.Close()
	}
}

func (a App) Init() tea.Cmd {
	return a.splash.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// topCmd carries a cmd a transition case below needs to fire (currently
	// only showTypeRosterMsg's load) forward past the per-screen dispatch
	// and viewport refresh that now unconditionally follow every case (see
	// their doc comments below on why none of them return early anymore).
	var topCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			// Ctrl+C always quits, on every screen, including while the
			// Search Screen's text input is focused - unlike a bare "q",
			// which must remain typeable (many Pokémon names contain it,
			// e.g. Squirtle) and so is only a quit key on screens with no
			// free-text entry (see splash.go, result.go).
			return a, tea.Quit
		}

	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.viewport.Height = max(0, a.height-viewportChromeReservation)
		// a.viewport.Width is deliberately never set - it stays 0, which
		// tells viewport not to horizontally crop content at all, leaving
		// that to the terminal's own soft-wrap as today. Horizontal
		// responsiveness is out of scope - see docs/adr/0007.
		a.typeRoster = a.typeRoster.resize(a.height)

	case switchScreenMsg:
		a.history = append(a.history, a.screen)
		a.screen = msg.to
		if msg.to == screenSearch {
			a.search = a.search.reset()
		}
		a.viewport.GotoTop()

	case backMsg:
		if len(a.history) == 0 {
			return a, nil
		}
		a.screen = a.history[len(a.history)-1]
		a.history = a.history[:len(a.history)-1]
		a.viewport.GotoTop()

	case searchAgainMsg:
		// Enter on the Result Screen has one fixed meaning regardless of
		// how it was reached, so it resets history back to "as if we'd
		// just come from Splash" rather than retracing however we got
		// here (see docs/adr/0001).
		a.history = []screen{screenSplash}
		a.screen = screenSearch
		a.search = a.search.reset()
		a.viewport.GotoTop()

	case showTypeRosterMsg:
		a.history = append(a.history, a.screen)
		a.typeRoster = newTypeRosterModel(a.search.client, msg.typeName, a.height)
		a.screen = screenTypeRoster
		topCmd = loadTypeRosterCmd(a.search.client, msg.typeName)

	case showResultMsg:
		a.history = append(a.history, a.screen)
		a.result = newResultModel(msg.stat, msg.spriteFront, msg.spriteBack, msg.pokedexEntry, msg.evolutionChain, msg.typeEffectiveness)
		a.screen = screenResult
		a.viewport.GotoTop()
	}

	// None of the transition cases above return early anymore (past the
	// screen-swap/GotoTop bookkeeping they own): every one of them still
	// needs the newly-active screen's own Update to run once against this
	// same msg (harmless for these message types - none of the per-screen
	// models special-case switchScreenMsg/backMsg/searchAgainMsg/
	// showResultMsg, so each just falls through to its own no-op default),
	// and critically still needs the viewport-refresh block below to run,
	// so the viewport's content is set to the *new* screen immediately
	// rather than staying frozen on the screen just left until some
	// unrelated later message (a cursor blink, a spinner tick) happens to
	// trigger a refresh - see docs/adr/0007.
	var cmd tea.Cmd
	switch a.screen {
	case screenSplash:
		a.splash, cmd = a.splash.Update(msg)
	case screenSearch:
		a.search, cmd = a.search.Update(msg)
	case screenTypeSelect:
		a.typeSelect, cmd = a.typeSelect.Update(msg)
	case screenTypeRoster:
		a.typeRoster, cmd = a.typeRoster.Update(msg)
	case screenResult:
		a.result, cmd = a.result.Update(msg)
	}
	cmd = tea.Batch(topCmd, cmd)

	if a.width > 0 && a.height > 0 && isViewportWrapped(a.screen) {
		a.viewport.KeyMap = viewportKeyMapFor(a.screen)
		a.viewport.SetContent(a.activeScreenView())
		if a.screen == screenTypeSelect {
			// Up/Down/j/k stay dedicated to cursor movement on this screen
			// (viewportPagingOnlyKeyMap above) - this is what keeps the
			// cursor's row on screen instead, the same "ensure visible"
			// behavior bubbles/list gives for free (see typeselect.go's
			// doc comment on why this screen isn't a bubbles/list).
			a.viewport = followCursor(a.viewport, typeSelectHeaderLines+a.typeSelect.cursor)
		}

		var vpCmd tea.Cmd
		a.viewport, vpCmd = a.viewport.Update(msg)
		cmd = tea.Batch(cmd, vpCmd)
	}

	return a, cmd
}

// isViewportWrapped reports whether s is rendered through App's shared
// viewport (see Update and View) rather than directly. The Type Roster
// Screen is excluded - its bubbles/table already scrolls its own rows
// (see typeRosterTableHeight), so wrapping it again would double up
// scrolling behavior for no benefit.
func isViewportWrapped(s screen) bool {
	switch s {
	case screenSplash, screenSearch, screenResult, screenTypeSelect:
		return true
	default:
		return false
	}
}

// viewportKeyMapFor returns the key set App's shared viewport should use
// while s is active - see viewportPagerKeyMap and viewportPagingOnlyKeyMap's
// doc comments for why the Type Select Screen gets the narrower one.
func viewportKeyMapFor(s screen) viewport.KeyMap {
	if s == screenTypeSelect {
		return viewportPagingOnlyKeyMap
	}
	return viewportPagerKeyMap
}

// activeScreenView renders whichever screen is currently active - shared
// by View (which may further wrap it in App's viewport) and Update (which
// feeds it to the viewport's SetContent so scroll keys act against fresh
// content).
func (a App) activeScreenView() string {
	switch a.screen {
	case screenSplash:
		return a.splash.View()
	case screenSearch:
		return a.search.View()
	case screenTypeSelect:
		return a.typeSelect.View()
	case screenTypeRoster:
		return a.typeRoster.View()
	case screenResult:
		return a.result.View()
	}
	return ""
}

func (a App) View() string {
	view := a.activeScreenView()

	if a.width > 0 && a.height > 0 && isViewportWrapped(a.screen) {
		view = a.viewport.View()
		if a.viewport.TotalLineCount() > a.viewport.VisibleLineCount() {
			view += "\n" + viewportScrollIndicatorStyle.Render("↑/↓ scroll for more")
		}
	}

	if a.zones == nil {
		return view
	}
	return a.zones.Scan(view)
}
