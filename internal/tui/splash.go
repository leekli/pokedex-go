package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var splashPromptStyle = lipgloss.NewStyle().Faint(true)

// splashModel is the Splash Screen: the embedded logo art plus a prompt to
// continue. It holds no state of its own - it's always the same view.
type splashModel struct{}

func newSplashModel() splashModel {
	return splashModel{}
}

// Update handles the Splash Screen's keys directly: Enter advances to the
// Search Screen; Q or Esc quit (there's no free-text input on this screen,
// so unlike the Search Screen, a bare "q" is safe to bind to quit).
// Ctrl+C is handled globally by App, not here.
func (s splashModel) Update(msg tea.Msg) (splashModel, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}

	switch km.Type {
	case tea.KeyEnter:
		return s, switchTo(screenSearch)
	case tea.KeyEsc:
		return s, tea.Quit
	}
	if km.String() == "q" {
		return s, tea.Quit
	}
	return s, nil
}

func (s splashModel) View() string {
	return renderSplashArt() + "\n" + splashPromptStyle.Render("Press Enter to search for Pokémon") + "\n"
}
