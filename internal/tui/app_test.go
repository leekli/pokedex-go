package tui

import (
	"image"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// tallResultModel builds a resultModel with enough content - full-size
// front/back sprites, an evolution chain, and weaknesses/resistances - to
// exceed a small terminal's height, for the scrolling tests below.
func tallResultModel() resultModel {
	front := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	back := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	chain := pichuPikachuRaichuChain()
	return newResultModel(testStatBlock(), front, back, "A long Pokédex entry describing this Pokémon in some detail.", &chain, charizardTypeEffectiveness())
}

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
			a.result = newResultModel(testStatBlock(), nil, nil, "", nil, nil)

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
func TestApp_Close_NilZonesIsSafe(_ *testing.T) {
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

// TestApp_Update_WindowSizeMsg_StoresDimensions proves App captures the
// terminal size and sizes its shared viewport from it - the foundation
// every other scrolling test below depends on.
func TestApp_Update_WindowSizeMsg_StoresDimensions(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)

	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 25})
	got := model.(App)

	if got.width != 80 || got.height != 25 {
		t.Errorf("width, height = %d, %d, want 80, 25", got.width, got.height)
	}
	if got.viewport.Height != 24 {
		t.Errorf("viewport.Height = %d, want 24 (25 minus the scroll-indicator reservation)", got.viewport.Height)
	}
}

// TestApp_View_ClampsResultScreenToTerminalHeight is the direct regression
// test for the reported bug: on a small terminal, the Result Screen's
// sprites and stat table were pushed off the top with no way to scroll up
// and see them. "Speed" (the stat table's last row) stands in for that
// clipped content.
func TestApp_View_ClampsResultScreenToTerminalHeight(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	a.screen = screenResult
	a.result = tallResultModel()

	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	a = model.(App)

	view := a.View()
	if lines := strings.Count(view, "\n") + 1; lines > a.height {
		t.Errorf("App.View() rendered %d lines, want <= terminal height %d\ngot:\n%s", lines, a.height, view)
	}
	if strings.Contains(view, "Speed") {
		t.Errorf("App.View() on a 10-row terminal already shows \"Speed\" (the last stat row) - want it clipped below the fold\ngot:\n%s", view)
	}
}

// TestApp_Update_DownRevealsClippedContent proves the content
// TestApp_View_ClampsResultScreenToTerminalHeight found clipped actually
// becomes visible by scrolling - the other half of the reported bug fix.
func TestApp_Update_DownRevealsClippedContent(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	a.screen = screenResult
	a.result = tallResultModel()
	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	a = model.(App)
	if strings.Contains(a.View(), "Speed") {
		t.Fatal("test setup: \"Speed\" already visible before scrolling - fixture isn't tall enough to exercise clipping")
	}

	for range 60 {
		model, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})
		a = model.(App)
	}

	if !strings.Contains(a.View(), "Speed") {
		t.Errorf("App.View() after scrolling down still missing \"Speed\" - the previously clipped content never became visible\ngot:\n%s", a.View())
	}
}

// TestApp_Update_ScreenTransitionResetsScrollPosition proves each newly
// shown screen starts scrolled to the top, rather than inheriting wherever
// the previous screen happened to be scrolled to.
func TestApp_Update_ScreenTransitionResetsScrollPosition(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	a.screen = screenResult
	a.result = tallResultModel()
	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	a = model.(App)
	for range 10 {
		model, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})
		a = model.(App)
	}
	if a.viewport.YOffset == 0 {
		t.Fatal("test setup: scrolling down never moved the viewport")
	}

	model, _ = a.Update(searchAgainMsg{})
	a = model.(App)

	if a.viewport.YOffset != 0 {
		t.Errorf("viewport.YOffset = %d after searchAgainMsg, want 0 (a fresh screen should start scrolled to top)", a.viewport.YOffset)
	}
}

// TestApp_Update_SearchScreen_RestrictedKeymapDoesNotSwallowLetters is a
// required regression test: App's shared viewport must never fall back to
// viewport.DefaultKeyMap() (which binds vim letters j/k/f/b/u/d and space
// to scrolling), or typing a Pokémon name containing any of those letters
// - "pikachu" contains both "u" and "k" - would corrupt the Search
// Screen's text input. See
// docs/adr/0007-app-level-viewport-with-restricted-keymap-for-scrolling.md.
func TestApp_Update_SearchScreen_RestrictedKeymapDoesNotSwallowLetters(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 25})
	a = model.(App)
	a.screen = screenSearch

	for _, r := range "pikachu" {
		model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		a = model.(App)
	}

	if got := a.search.input.Value(); got != "pikachu" {
		t.Errorf("search.input.Value() = %q after typing \"pikachu\", want unchanged \"pikachu\" (a letter was swallowed by the viewport's keymap)", got)
	}
}

// TestApp_Update_TypeSelect_ArrowsStillMoveCursorNotViewport proves the
// Type Select Screen's Up/Down cursor navigation still works once it's
// wrapped in App's shared viewport - the viewport must not compete with it
// for those keys (see viewportPagingOnlyKeyMap).
func TestApp_Update_TypeSelect_ArrowsStillMoveCursorNotViewport(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	a = model.(App)
	a.screen = screenTypeSelect

	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})
	a = model.(App)

	if a.typeSelect.cursor != 1 {
		t.Errorf("typeSelect.cursor = %d after Down, want 1", a.typeSelect.cursor)
	}
}

// TestApp_Update_TypeSelect_CursorAutoFollowsOffscreenRow proves the
// viewport follows the cursor down the Type Select Screen's 18-type list
// on a terminal too short to show all of them at once, so the selection
// never scrolls silently out of view (see followCursor in typeselect.go).
func TestApp_Update_TypeSelect_CursorAutoFollowsOffscreenRow(t *testing.T) {
	a := NewApp(nil)
	t.Cleanup(a.Close)
	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	a = model.(App)
	a.screen = screenTypeSelect

	for range 15 {
		model, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})
		a = model.(App)
	}

	line := typeSelectHeaderLines + a.typeSelect.cursor
	if line < a.viewport.YOffset || line >= a.viewport.YOffset+a.viewport.Height {
		t.Errorf("cursor line %d is outside the visible window [%d, %d) after scrolling past it", line, a.viewport.YOffset, a.viewport.YOffset+a.viewport.Height)
	}
}
