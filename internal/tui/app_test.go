package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewApp_StartsOnSplashScreen(t *testing.T) {
	a := NewApp(nil)
	if a.screen != screenSplash {
		t.Errorf("NewApp().screen = %v, want screenSplash", a.screen)
	}
}

func TestApp_Init_StartsSplashSweep(t *testing.T) {
	a := NewApp(nil)
	cmd := a.Init()
	if cmd == nil {
		t.Fatal("App.Init() returned a nil cmd, want the splash sweep's tick cmd")
	}
	if _, ok := cmd().(splashTickMsg); !ok {
		t.Errorf("App.Init() cmd produced %T, want splashTickMsg", cmd())
	}
}

// TestApp_Update_CtrlCQuitsFromAnyScreen proves Ctrl+C is a global quit key
// regardless of which screen is currently showing - including the Search
// Screen, where a bare "q" must remain typeable (see app.go's comment).
func TestApp_Update_CtrlCQuitsFromAnyScreen(t *testing.T) {
	for _, tt := range []struct {
		name   string
		screen screen
	}{
		{"splash", screenSplash},
		{"search", screenSearch},
		{"result", screenResult},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := NewApp(nil)
			a.screen = tt.screen

			model, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

			if cmd == nil {
				t.Fatal("App.Update(Ctrl+C) returned a nil cmd, want tea.Quit")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Errorf("App.Update(Ctrl+C) cmd produced %T, want tea.QuitMsg", cmd())
			}
			if got := model.(App).screen; got != tt.screen {
				t.Errorf("App.Update(Ctrl+C) changed screen to %v, want unchanged %v", got, tt.screen)
			}
		})
	}
}

// TestApp_Update_SwitchScreenMsg_ResetsSearchWhenEnteringSearch proves the
// Search Screen's input/error/loading state is cleared on every transition
// into it, so a stale error or half-typed query from a previous visit never
// leaks into the next one.
func TestApp_Update_SwitchScreenMsg_ResetsSearchWhenEnteringSearch(t *testing.T) {
	a := NewApp(nil)
	a.search.input.SetValue("stale query")
	a.search.errMsg = "stale error"
	a.search.loading = true

	model, _ := a.Update(switchScreenMsg{to: screenSearch})
	got := model.(App)

	if got.screen != screenSearch {
		t.Errorf("screen = %v, want screenSearch", got.screen)
	}
	if got.search.input.Value() != "" {
		t.Errorf("search.input.Value() = %q, want empty after reset", got.search.input.Value())
	}
	if got.search.errMsg != "" {
		t.Errorf("search.errMsg = %q, want empty after reset", got.search.errMsg)
	}
	if got.search.loading {
		t.Error("search.loading = true, want false after reset")
	}
}

// TestApp_Update_SwitchScreenMsg_ToSplashDoesNotResetSearch proves the reset
// is specific to entering the Search Screen - switching to the Splash
// Screen (e.g. from Splash's own Esc/quit path is irrelevant here, but the
// Result->Search transition is covered above) leaves search state alone.
func TestApp_Update_SwitchScreenMsg_ToSplashDoesNotResetSearch(t *testing.T) {
	a := NewApp(nil)
	a.search.input.SetValue("kept")

	model, _ := a.Update(switchScreenMsg{to: screenSplash})
	got := model.(App)

	if got.screen != screenSplash {
		t.Errorf("screen = %v, want screenSplash", got.screen)
	}
	if got.search.input.Value() != "kept" {
		t.Errorf("search.input.Value() = %q, want unchanged %q", got.search.input.Value(), "kept")
	}
}

func TestApp_Update_ShowResultMsg_SwitchesToResultScreen(t *testing.T) {
	a := NewApp(nil)
	stat := testStatBlock()

	model, _ := a.Update(showResultMsg{stat: stat})
	got := model.(App)

	if got.screen != screenResult {
		t.Errorf("screen = %v, want screenResult", got.screen)
	}
	if got.result.stat.Name != stat.Name {
		t.Errorf("result.stat.Name = %q, want %q", got.result.stat.Name, stat.Name)
	}
}

// TestApp_Update_DelegatesToCurrentScreen proves an otherwise-unhandled
// message (here, a keystroke) is routed to whichever screen is active,
// rather than only the three App-level message types being wired up.
func TestApp_Update_DelegatesToCurrentScreen(t *testing.T) {
	a := NewApp(nil)
	a.screen = screenSplash

	model, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := model.(App)

	if got.screen != screenSplash {
		t.Errorf("screen = %v, want unchanged screenSplash (App.Update should delegate, not transition itself)", got.screen)
	}
	if cmd == nil {
		t.Fatal("App.Update(Enter) on Splash returned a nil cmd, want splashModel's switchTo(screenSearch)")
	}
	msg, ok := cmd().(switchScreenMsg)
	if !ok || msg.to != screenSearch {
		t.Errorf("App.Update(Enter) on Splash cmd produced %#v, want switchScreenMsg{to: screenSearch}", cmd())
	}
}

func TestApp_View_DelegatesToCurrentScreen(t *testing.T) {
	for _, tt := range []struct {
		name   string
		screen screen
		want   string
	}{
		{"splash", screenSplash, "Press Enter to search for Pokémon"},
		{"search", screenSearch, "POKÉDEX SEARCH"},
		{"result", screenResult, "Enter/Esc to search again"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := NewApp(nil)
			a.screen = tt.screen
			a.result = newResultModel(testStatBlock(), nil)

			if view := a.View(); !strings.Contains(view, tt.want) {
				t.Errorf("App.View() on %s screen missing %q\ngot:\n%s", tt.name, tt.want, view)
			}
		})
	}
}

// TestApp_View_UnknownScreenRendersEmpty exercises View's defensive default
// case: screen is a private enum with only three valid values, so this can
// only happen via a bug, but View must still degrade safely rather than
// panic.
func TestApp_View_UnknownScreenRendersEmpty(t *testing.T) {
	a := NewApp(nil)
	a.screen = screen(99)

	if got := a.View(); got != "" {
		t.Errorf("App.View() with an unrecognized screen = %q, want empty string", got)
	}
}
