// Command pokedex-go is a terminal Pokédex: look up a Pokémon by name or
// National Dex Number and see its sprite and stats, styled after the
// Generation 1 Game Boy Pokédex. See CONTEXT.md for the app's vocabulary.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leekli/pokedex-go/internal/pokeapi"
	"github.com/leekli/pokedex-go/internal/tui"
)

func main() {
	client := pokeapi.NewClient()
	app := tui.NewApp(client)

	if _, err := tea.NewProgram(app, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "pokedex-go:", err)
		os.Exit(1)
	}
}
