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
	os.Exit(run())
}

// run wires up the Client and App, runs the Bubble Tea program, and returns
// the process exit code — kept separate from main so app.Close() always
// runs via defer before exiting, which os.Exit called directly from main
// would skip.
func run() int {
	client := pokeapi.NewClient()
	app := tui.NewApp(client)
	defer app.Close()

	if _, err := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "pokedex-go:", err)
		return 1
	}
	return 0
}
