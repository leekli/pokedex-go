package e2e

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestSearch_UnknownNameShowsLookupError covers the Lookup Error path from
// CONTEXT.md: an unresolved name is the user's mistake, so the app stays on
// the Search Screen and shows an inline message naming what was searched.
func TestSearch_UnknownNameShowsLookupError(t *testing.T) {
	tm := newTestModel(t)
	advanceToSearchScreen(t, tm)

	tm.Type("notarealpokemon")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// The Lookup Error message only ever renders from the Search Screen
	// (see search.go), so seeing it also confirms the app stayed there
	// rather than navigating away.
	waitForAll(t, tm, 3*time.Second, `No Pokémon found for "notarealpokemon"`)
}
