package e2e

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestSearch_SuccessfulLookupByNameShowsResultScreen(t *testing.T) {
	tm := newTestModel(t)
	advanceToSearchScreen(t, tm)

	tm.Type("pikachu")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitForAll(t, tm, 3*time.Second,
		"#025 Pikachu", "ELECTRIC", "HP", "Attack", "Defense", "Speed", `1'04"`, "13.2 lbs")
}

func TestSearch_SuccessfulLookupByDexNumberShowsResultScreen(t *testing.T) {
	tm := newTestModel(t)
	advanceToSearchScreen(t, tm)

	tm.Type("25")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("#025 Pikachu"))
	}, teatest.WithDuration(3*time.Second))
}

// advanceToSearchScreen presses Enter on the Splash Screen and waits for the
// Search Screen to render, so each flow test doesn't repeat that setup.
func advanceToSearchScreen(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Press Enter to search for Pokémon"))
	}, teatest.WithDuration(2*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Search the Pokédex"))
	}, teatest.WithDuration(2*time.Second))
}
