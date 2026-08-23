package e2e

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestSplash_QuitKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}},
		{"esc", tea.KeyMsg{Type: tea.KeyEsc}},
		{"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := newTestModel(t)
			teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
				return bytes.Contains(out, []byte("Press Enter to search for Pokémon"))
			}, teatest.WithDuration(2*time.Second))

			tm.Send(tt.key)

			tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
		})
	}
}
