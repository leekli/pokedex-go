package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewApp_StartsOnSplashScreen(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	if a.screen != screenSplash {
		t.Errorf("NewApp().screen = %v, want screenSplash", a.screen)
	}
}

func TestApp_Init_StartsSplashSweep(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
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
			t.Cleanup(a.Close)
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
	t.Cleanup(a.Close)
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
	t.Cleanup(a.Close)
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
	t.Cleanup(a.Close)
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
	t.Cleanup(a.Close)
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
			t.Cleanup(a.Close)
			a.screen = tt.screen
			a.result = newResultModel(testStatBlock(), nil, "")

			if view := a.View(); !strings.Contains(view, tt.want) {
				t.Errorf("App.View() on %s screen missing %q\ngot:\n%s", tt.name, tt.want, view)
			}
		})
	}
}

// TestApp_View_UnknownScreenRendersEmpty exercises View's defensive default
// case: screen is a private enum with a small set of valid values, so this
// can only happen via a bug, but View must still degrade safely rather than
// panic.
func TestApp_View_UnknownScreenRendersEmpty(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	a.screen = screen(99)

	if got := a.View(); got != "" {
		t.Errorf("App.View() with an unrecognized screen = %q, want empty string", got)
	}
}

// TestApp_Update_SwitchScreenMsg_PushesHistory proves a forward transition
// records where it came from, so a later backMsg can return there (see
// docs/adr/0001-navigation-history-for-back-navigation.md).
func TestApp_Update_SwitchScreenMsg_PushesHistory(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	a.screen = screenSplash

	model, _ := a.Update(switchScreenMsg{to: screenSearch})
	got := model.(App)

	if len(got.history) != 1 || got.history[0] != screenSplash {
		t.Errorf("history = %v, want [screenSplash]", got.history)
	}
}

// TestApp_Update_BackMsg_PopsHistory proves backMsg returns to whatever was
// most recently pushed, regardless of which screen is currently showing.
func TestApp_Update_BackMsg_PopsHistory(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	a.screen = screenTypeRoster
	a.history = []screen{screenSplash, screenSearch, screenTypeSelect}

	model, cmd := a.Update(backMsg{})
	got := model.(App)

	if got.screen != screenTypeSelect {
		t.Errorf("screen = %v, want screenTypeSelect (the top of history)", got.screen)
	}
	if len(got.history) != 2 {
		t.Errorf("history = %v, want length 2 after popping", got.history)
	}
	if cmd != nil {
		t.Errorf("Update(backMsg) returned a non-nil cmd, want nil")
	}
}

// TestApp_Update_BackMsg_EmptyHistoryIsNoOp proves popping an empty history
// (which shouldn't normally happen - every screen but Splash always has
// something pushed before it - see app.go's Update) doesn't panic or change
// the screen.
func TestApp_Update_BackMsg_EmptyHistoryIsNoOp(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	a.screen = screenSearch
	a.history = nil

	model, _ := a.Update(backMsg{})
	got := model.(App)

	if got.screen != screenSearch {
		t.Errorf("screen = %v, want unchanged screenSearch", got.screen)
	}
}

// TestApp_Update_SearchAgainMsg_ResetsToFreshSearch proves Enter on the
// Result Screen always lands on a freshly-reset Search Screen with history
// collapsed back to just Splash, regardless of how deep the navigation
// history was (e.g. reached via the Type Roster Screen) - see
// docs/adr/0001.
func TestApp_Update_SearchAgainMsg_ResetsToFreshSearch(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	a.screen = screenResult
	a.history = []screen{screenSplash, screenSearch, screenTypeSelect, screenTypeRoster}
	a.search.input.SetValue("stale")
	a.search.errMsg = "stale error"

	model, _ := a.Update(searchAgainMsg{})
	got := model.(App)

	if got.screen != screenSearch {
		t.Errorf("screen = %v, want screenSearch", got.screen)
	}
	if len(got.history) != 1 || got.history[0] != screenSplash {
		t.Errorf("history = %v, want [screenSplash] (collapsed, not retracing the Type Roster path)", got.history)
	}
	if got.search.input.Value() != "" || got.search.errMsg != "" {
		t.Errorf("search state not reset: input=%q errMsg=%q", got.search.input.Value(), got.search.errMsg)
	}
}

// TestApp_Update_ShowTypeRosterMsg_PushesHistoryAndTriggersLoad proves
// selecting a type pushes history, switches to the Type Roster Screen with
// a fresh (loading) model for that type, and returns the load command.
func TestApp_Update_ShowTypeRosterMsg_PushesHistoryAndTriggersLoad(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	a.screen = screenTypeSelect

	model, cmd := a.Update(showTypeRosterMsg{typeName: "fire"})
	got := model.(App)

	if got.screen != screenTypeRoster {
		t.Errorf("screen = %v, want screenTypeRoster", got.screen)
	}
	if len(got.history) != 1 || got.history[0] != screenTypeSelect {
		t.Errorf("history = %v, want [screenTypeSelect]", got.history)
	}
	if got.typeRoster.typeName != "fire" {
		t.Errorf("typeRoster.typeName = %q, want fire", got.typeRoster.typeName)
	}
	if !got.typeRoster.loading {
		t.Error("typeRoster.loading = false, want true (a load should be in flight)")
	}
	if cmd == nil {
		t.Fatal("Update(showTypeRosterMsg) returned a nil cmd, want loadTypeRosterCmd")
	}
}

// TestApp_Close_NilZonesIsSafe proves Close doesn't panic on a zero-value
// App (no zone manager), matching the nil-safety the rest of the mouse
// support relies on (see zones.go).
func TestApp_Close_NilZonesIsSafe(t *testing.T) {
	var a App
	a.Close()
}

// TestApp_View_NilZonesIsSafe proves View doesn't panic on a zero-value App
// (no zone manager) - the same nil-safety Close relies on, but for the
// zones.Scan call in View instead.
func TestApp_View_NilZonesIsSafe(t *testing.T) {
	a := App{screen: screenSplash, splash: newSplashModel()}

	if view := a.View(); !strings.Contains(view, "Press Enter to search for Pokémon") {
		t.Errorf("App.View() with nil zones = %q, want the splash prompt still rendered", view)
	}
}
