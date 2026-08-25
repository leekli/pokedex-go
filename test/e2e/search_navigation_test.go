package e2e

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestSearch_EscReturnsToSplash covers the Search Screen's other exit path
// (see search_lookup_error_test.go / search_service_error_test.go for the
// lookup outcomes, and result_navigation_test.go for the Result Screen's
// equivalent back-navigation): Esc backs out to the Splash Screen without
// performing a lookup.
func TestSearch_EscReturnsToSplash(t *testing.T) {
	tm := newTestModel(t)
	advanceToSearchScreen(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Press Enter to search for Pokémon"))
	}, teatest.WithDuration(2*time.Second))
}
