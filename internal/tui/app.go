// Package tui is pokedex-go's Bubble Tea layer: the Splash, Search, and
// Result screens and the App model that switches between them. This is the
// only package that imports both bubbletea and the pokeapi/pokemon layers
// below it.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leekli/pokedex-go/internal/pokeapi"
)

// screen identifies which of the three screens App is currently showing.
type screen int

const (
	screenSplash screen = iota
	screenSearch
	screenResult
)

// App is the root Bubble Tea model. It owns the current screen and one
// sub-model per screen, and handles the transitions between them.
type App struct {
	screen screen
	splash splashModel
	search searchModel
	result resultModel
}

// NewApp builds an App that will use client for all PokeAPI lookups.
func NewApp(client *pokeapi.Client) App {
	return App{
		screen: screenSplash,
		splash: newSplashModel(),
		search: newSearchModel(client),
		result: resultModel{},
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
		a.screen = msg.to
		if msg.to == screenSearch {
			a.search = a.search.reset()
		}
		return a, nil

	case showResultMsg:
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
	case screenResult:
		a.result, cmd = a.result.Update(msg)
	}
	return a, cmd
}

func (a App) View() string {
	switch a.screen {
	case screenSplash:
		return a.splash.View()
	case screenSearch:
		return a.search.View()
	case screenResult:
		return a.result.View()
	default:
		return ""
	}
}
