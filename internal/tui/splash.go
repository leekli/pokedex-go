package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var splashPromptStyle = lipgloss.NewStyle().Faint(true)

// splashSweepFrame is the tick interval driving the startup sweep animation
// (see renderSplashArtSweep) - roughly 24fps, smooth without being wasteful.
const splashSweepFrame = 40 * time.Millisecond

// splashSweepDuration is how long the sweep takes to fully cross the art and
// come to rest.
const splashSweepDuration = 700 * time.Millisecond

// splashTickMsg drives one frame of the startup sweep animation.
type splashTickMsg time.Time

func splashTick() tea.Cmd {
	return tea.Tick(splashSweepFrame, func(t time.Time) tea.Msg {
		return splashTickMsg(t)
	})
}

// splashModel is the Splash Screen: the embedded logo art (which sweeps in
// on startup, then sits at rest) plus a prompt to continue.
type splashModel struct {
	startedAt time.Time
}

func newSplashModel() splashModel {
	return splashModel{startedAt: time.Now()}
}

// Init starts the startup sweep animation.
func (s splashModel) Init() tea.Cmd {
	return splashTick()
}

// sweepProgress reports how far through the startup sweep we are: 0 at
// startedAt, 1 once splashSweepDuration has elapsed.
func (s splashModel) sweepProgress() float64 {
	elapsed := time.Since(s.startedAt)
	if elapsed >= splashSweepDuration {
		return 1
	}
	return float64(elapsed) / float64(splashSweepDuration)
}

// Update handles the Splash Screen's keys directly: Enter advances to the
// Search Screen; Q or Esc quit (there's no free-text input on this screen,
// so unlike the Search Screen, a bare "q" is safe to bind to quit).
// Ctrl+C is handled globally by App, not here. It also drives the startup
// sweep animation forward on each splashTickMsg, stopping once it's done so
// the idle splash screen doesn't keep re-rendering for no reason.
func (s splashModel) Update(msg tea.Msg) (splashModel, tea.Cmd) {
	switch m := msg.(type) {
	case splashTickMsg:
		if s.sweepProgress() >= 1 {
			return s, nil
		}
		return s, splashTick()

	case tea.KeyMsg:
		switch m.Type {
		case tea.KeyEnter:
			return s, switchTo(screenSearch)
		case tea.KeyEsc:
			return s, tea.Quit
		}
		if m.String() == "q" {
			return s, tea.Quit
		}
	}
	return s, nil
}

func (s splashModel) View() string {
	return renderSplashArtSweep(s.sweepProgress()) + "\n" + splashPromptStyle.Render("Press Enter to search for Pokémon") + "\n"
}
