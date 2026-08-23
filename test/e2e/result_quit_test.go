package e2e

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestResult_QuitKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}},
		{"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := newTestModel(t)
			advanceToResultScreen(t, tm)

			tm.Send(tt.key)

			tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
		})
	}
}
