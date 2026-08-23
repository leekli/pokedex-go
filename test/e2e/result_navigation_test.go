package e2e

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestResult_BackToSearch(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"esc", tea.KeyMsg{Type: tea.KeyEsc}},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := newTestModel(t)
			advanceToResultScreen(t, tm)

			tm.Send(tt.key)

			// The Search Screen's title only ever renders there, and its
			// re-appearance also confirms the input was reset for a fresh
			// lookup (see searchModel.reset in internal/tui/search.go).
			waitForAll(t, tm, 2*time.Second, "Search the Pokédex")
		})
	}
}

// advanceToResultScreen drives the app from launch through a successful
// pikachu lookup to the Result Screen.
func advanceToResultScreen(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	advanceToSearchScreen(t, tm)
	tm.Type("pikachu")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForAll(t, tm, 3*time.Second, "#025 Pikachu")
}
