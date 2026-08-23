package tui

import (
	"image"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leekli/pokedex-go/internal/pokemon"
)

// switchScreenMsg requests a plain screen transition with no accompanying
// data (Splash<->Search, Result->Search).
type switchScreenMsg struct {
	to screen
}

func switchTo(s screen) tea.Cmd {
	return func() tea.Msg { return switchScreenMsg{to: s} }
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
