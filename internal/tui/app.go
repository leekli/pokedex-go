// Package tui is pokedex-go's Bubble Tea layer: the Splash, Search, Type
// Select, Type Roster, and Result screens, and the App model that switches
// between them. This is the only package that imports both bubbletea and
// the pokeapi/pokemon layers below it.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
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

// App is the root Bubble Tea model. It owns the current screen, one
// sub-model per screen, and the navigation history that lets every screen's
// Esc mean "go back one screen" regardless of how it was reached — see
// docs/adr/0001-navigation-history-for-back-navigation.md.
type App struct {
	screen  screen
	history []screen

	splash     splashModel
	search     searchModel
	typeSelect typeSelectModel
	typeRoster typeRosterModel
	result     resultModel

	zones *zone.Manager
}

// NewApp builds an App that will use client for all PokeAPI lookups.
func NewApp(client *pokeapi.Client) App {
	zones := zone.New()
	return App{
		screen:     screenSplash,
		splash:     newSplashModel(),
		search:     newSearchModel(client, zones),
		typeSelect: newTypeSelectModel(zones),
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

	case switchScreenMsg:
		a.history = append(a.history, a.screen)
		a.screen = msg.to
		if msg.to == screenSearch {
			a.search = a.search.reset()
		}
		return a, nil

	case backMsg:
		if len(a.history) == 0 {
			return a, nil
		}
		a.screen = a.history[len(a.history)-1]
		a.history = a.history[:len(a.history)-1]
		return a, nil

	case searchAgainMsg:
		// Enter on the Result Screen has one fixed meaning regardless of
		// how it was reached, so it resets history back to "as if we'd
		// just come from Splash" rather than retracing however we got
		// here (see docs/adr/0001).
		a.history = []screen{screenSplash}
		a.screen = screenSearch
		a.search = a.search.reset()
		return a, nil

	case showTypeRosterMsg:
		a.history = append(a.history, a.screen)
		a.typeRoster = newTypeRosterModel(a.search.client, msg.typeName)
		a.screen = screenTypeRoster
		return a, loadTypeRosterCmd(a.search.client, msg.typeName)

	case showResultMsg:
		a.history = append(a.history, a.screen)
		a.result = newResultModel(msg.stat, msg.sprite)
		a.screen = screenResult
		return a, nil
	}

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
	return a, cmd
}

func (a App) View() string {
	var view string
	switch a.screen {
	case screenSplash:
		view = a.splash.View()
	case screenSearch:
		view = a.search.View()
	case screenTypeSelect:
		view = a.typeSelect.View()
	case screenTypeRoster:
		view = a.typeRoster.View()
	case screenResult:
		view = a.result.View()
	}
	if a.zones == nil {
		return view
	}
	return a.zones.Scan(view)
}
