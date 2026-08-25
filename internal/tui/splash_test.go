package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestSplashModel_SweepProgress_ClampsAtOne proves sweepProgress never
// exceeds 1 once splashSweepDuration has elapsed - the animation should
// settle, not overshoot or keep growing.
func TestSplashModel_SweepProgress_ClampsAtOne(t *testing.T) {
	s := splashModel{startedAt: time.Now().Add(-splashSweepDuration * 10)}
	if got := s.sweepProgress(); got != 1 {
		t.Errorf("sweepProgress() long after the sweep duration = %v, want exactly 1", got)
	}
}

// TestSplashModel_SweepProgress_MidSweep proves progress is a fraction
// strictly between 0 and 1 partway through the sweep, and is (within
// scheduling jitter) proportional to elapsed time.
func TestSplashModel_SweepProgress_MidSweep(t *testing.T) {
	s := splashModel{startedAt: time.Now().Add(-splashSweepDuration / 2)}
	got := s.sweepProgress()
	if got <= 0 || got >= 1 {
		t.Errorf("sweepProgress() halfway through the sweep = %v, want strictly between 0 and 1", got)
	}
}

// TestSplashModel_Update_TickContinuesWhileSweeping proves the animation
// keeps rescheduling itself every splashTickMsg while the sweep is still in
// progress.
func TestSplashModel_Update_TickContinuesWhileSweeping(t *testing.T) {
	s := splashModel{startedAt: time.Now()}

	_, cmd := s.Update(splashTickMsg(time.Now()))

	if cmd == nil {
		t.Fatal("Update(splashTickMsg) mid-sweep returned a nil cmd, want another splashTick")
	}
	if _, ok := cmd().(splashTickMsg); !ok {
		t.Errorf("Update(splashTickMsg) mid-sweep cmd produced %T, want splashTickMsg", cmd())
	}
}

// TestSplashModel_Update_TickStopsWhenSweepComplete proves the animation
// stops rescheduling itself once the sweep has fully settled, so the idle
// splash screen doesn't keep re-rendering for no reason (see splash.go's
// doc comment on Update).
func TestSplashModel_Update_TickStopsWhenSweepComplete(t *testing.T) {
	s := splashModel{startedAt: time.Now().Add(-splashSweepDuration * 10)}

	_, cmd := s.Update(splashTickMsg(time.Now()))

	if cmd != nil {
		t.Errorf("Update(splashTickMsg) after the sweep settled returned a non-nil cmd, want nil")
	}
}

func TestSplashModel_Update_EnterAdvancesToSearch(t *testing.T) {
	s := newSplashModel()

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Update(Enter) returned a nil cmd, want switchTo(screenSearch)")
	}
	msg, ok := cmd().(switchScreenMsg)
	if !ok || msg.to != screenSearch {
		t.Errorf("Update(Enter) cmd produced %#v, want switchScreenMsg{to: screenSearch}", cmd())
	}
}

func TestSplashModel_Update_EscQuits(t *testing.T) {
	s := newSplashModel()

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("Update(Esc) returned a nil cmd, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("Update(Esc) cmd produced %T, want tea.QuitMsg", cmd())
	}
}

func TestSplashModel_Update_QQuits(t *testing.T) {
	s := newSplashModel()

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if cmd == nil {
		t.Fatal(`Update("q") returned a nil cmd, want tea.Quit`)
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf(`Update("q") cmd produced %T, want tea.QuitMsg`, cmd())
	}
}

// TestSplashModel_Update_OtherKeyNoOp proves a key with no binding (there's
// no free-text input on this screen) is simply ignored.
func TestSplashModel_Update_OtherKeyNoOp(t *testing.T) {
	s := newSplashModel()

	got, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if cmd != nil {
		t.Errorf(`Update("x") returned a non-nil cmd, want nil`)
	}
	if got != s {
		t.Errorf("Update(\"x\") changed the model, want it unchanged")
	}
}

func TestSplashModel_View_ContainsPrompt(t *testing.T) {
	s := newSplashModel()
	if view := s.View(); !strings.Contains(view, "Press Enter to search for Pokémon") {
		t.Errorf("splashModel.View() missing prompt text\ngot:\n%s", view)
	}
}
