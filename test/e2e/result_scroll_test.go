package e2e

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestResult_ScrollRevealsClippedContent is an integration-level
// regression test for the reported bug: on a realistic small terminal
// (80x25), the Result Screen's content - sprites, evolution chain, stat
// table, and more - is taller than the terminal, and before App-level
// scrolling existed there was no way to see what got pushed off-screen.
// The absence-then-presence proof itself lives at the App level (see
// TestApp_View_ClampsResultScreenToTerminalHeight and
// TestApp_Update_DownRevealsClippedContent in internal/tui/app_test.go);
// this test's job is to prove the same mechanism works end-to-end through
// the real program, at the exact terminal size from the bug report.
func TestResult_ScrollRevealsClippedContent(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(t), teatest.WithInitialTermSize(80, 25))
	t.Cleanup(func() { _ = tm.Quit() })

	advanceToResultScreen(t, tm)

	for range 40 {
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Speed is the stat table's last row - on an 80x25 terminal it's
	// pushed well below the fold until scrolled into view.
	waitForAll(t, tm, 2*time.Second, "Speed")
}
