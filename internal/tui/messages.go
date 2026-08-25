package tui

import (
	"image"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leekli/pokedex-go/internal/pokeapi"
	"github.com/leekli/pokedex-go/internal/pokemon"
)

// switchScreenMsg requests a forward screen transition with no accompanying
// data (Splash->Search, Search->Type Select): the current screen is pushed
// onto App's navigation history before switching, so a later backMsg can
// return to it.
type switchScreenMsg struct {
	to screen
}

func switchTo(s screen) tea.Cmd {
	return func() tea.Msg { return switchScreenMsg{to: s} }
}

// backMsg requests "go back one screen": App pops its navigation history
// and switches to whatever it finds there. Every screen's Esc binding uses
// this instead of naming a fixed destination, so "Esc goes back one screen"
// holds regardless of how the current screen was reached (see
// docs/adr/0001-navigation-history-for-back-navigation.md).
type backMsg struct{}

func goBack() tea.Cmd {
	return func() tea.Msg { return backMsg{} }
}

// searchAgainMsg is the Result Screen's Enter binding: unlike Esc, it has a
// single fixed meaning ("look up another Pokémon") regardless of how the
// Result Screen was reached, so it always lands on a freshly-reset Search
// Screen rather than retracing history.
type searchAgainMsg struct{}

func searchAgain() tea.Cmd {
	return func() tea.Msg { return searchAgainMsg{} }
}

// lookupResultMsg is what an in-flight lookupCmd resolves to: either a
// successfully built StatBlock (with an optional sprite image — nil if
// PokeAPI had no sprite, or the sprite fetch itself failed) or an error from
// the pokeapi package (*pokeapi.LookupError or *pokeapi.ServiceError).
type lookupResultMsg struct {
	stat   pokemon.StatBlock
	sprite image.Image
	err    error
}

// showResultMsg carries a successful lookup's data into the Result Screen
// and switches to it. Separate from switchScreenMsg because it's the only
// transition that carries a payload.
type showResultMsg struct {
	stat   pokemon.StatBlock
	sprite image.Image
}

func showResult(stat pokemon.StatBlock, sprite image.Image) tea.Cmd {
	return func() tea.Msg { return showResultMsg{stat: stat, sprite: sprite} }
}

// showTypeRosterMsg carries the selected type into the Type Roster Screen
// and switches to it, pushing the current screen (the Type Select Screen)
// onto App's navigation history first, same as showResultMsg.
type showTypeRosterMsg struct {
	typeName string
}

func showTypeRoster(typeName string) tea.Cmd {
	return func() tea.Msg { return showTypeRosterMsg{typeName: typeName} }
}

// typeRosterResultMsg is what an in-flight loadTypeRosterCmd resolves to:
// either a populated roster (entries, sorted by National Dex Number, plus
// whatever generation data was available — see GetGenerationIndex's
// best-effort contract) or an error from the pokeapi package. Consumed
// entirely by typeRosterModel, the same way lookupResultMsg is consumed
// entirely by searchModel.
type typeRosterResultMsg struct {
	typeName    string
	entries     []pokeapi.TypeRosterEntry
	generations map[int]string
	err         error
}
