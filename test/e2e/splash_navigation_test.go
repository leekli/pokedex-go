package e2e

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestSplash_EnterAdvancesToSearchScreen(t *testing.T) {
	tm := newTestModel(t)

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Press Enter to search for Pokémon"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Search the Pokédex"))
	}, teatest.WithDuration(2*time.Second))
}
