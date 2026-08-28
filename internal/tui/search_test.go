package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leekli/pokedex-go/internal/pokeapi"
	zone "github.com/lrstanley/bubblezone"
)

// TestSearchBallArt_LinesAreAligned guards against the misaligned box/art
// bug class this design went through during mockup review: every line of
// the ball art must render at the same cell width, or the ball distorts and
// the title/subtitle text placed beside it (see searchModel.View) drifts
// out of column.
func TestSearchBallArt_LinesAreAligned(t *testing.T) {
	lines := searchBallArt()
	want := lipgloss.Width(lines[0])
	for i, line := range lines {
		if got := lipgloss.Width(line); got != want {
			t.Errorf("searchBallArt()[%d] width = %d, want %d (line %q)", i, got, want, line)
		}
	}
}

// TestSearchBallArt_BandRowHasButtonDot proves the band row (index 3, the
// black belt between the ball's red and white halves) carries a white
// button dot, echoing the Splash Screen badge's own button pixels (see
// assets/splash.txt) rather than a plain unbroken band.
func TestSearchBallArt_BandRowHasButtonDot(t *testing.T) {
	lines := searchBallArt()
	band := lines[3]

	if !strings.Contains(band, "●") {
		t.Errorf("searchBallArt()[3] (the band row) = %q, want it to contain a button dot (●)", band)
	}
	for i, line := range lines {
		if i == 3 {
			continue
		}
		if strings.Contains(line, "●") {
			t.Errorf("searchBallArt()[%d] = %q, the button dot should only appear on the band row", i, line)
		}
	}
}

// TestSearchModel_View_ShowsExpectedCopy is a smoke test for the Playful
// Maximalist redesign's static content: the title, instruction, dex-range
// hint, and example queries should all appear on the idle Search Screen.
func TestSearchModel_View_ShowsExpectedCopy(t *testing.T) {
	m := newSearchModel(nil, nil)
	view := m.View()

	for _, want := range []string{
		"POKÉDEX SEARCH",
		"Find any Pokémon by name or National Dex Number.",
		"#001", "#1025",
		"pikachu", "charizard", "snorlax",
		searchByTypeButtonLabel,
		"Enter to search · Tab to switch to Search by Type · Esc to go back · Ctrl+C to quit",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("searchModel.View() missing %q\ngot:\n%s", want, view)
		}
	}
}

// TestSearchModel_View_LoadingAndErrorStates confirms the dynamic status
// line still switches correctly between the spinner and the two distinct
// error messages (CONTEXT.md's Lookup Error vs Service Error) now that it
// shares its line with the examples row instead of always being blank.
func TestSearchModel_View_LoadingAndErrorStates(t *testing.T) {
	m := newSearchModel(nil, nil)

	m.loading = true
	if view := m.View(); !strings.Contains(view, "Searching...") {
		t.Errorf("loading searchModel.View() missing spinner text\ngot:\n%s", view)
	}
	if view := m.View(); strings.Contains(view, "charizard") {
		t.Errorf("loading searchModel.View() should not show examples\ngot:\n%s", view)
	}

	m.loading = false
	m.errMsg = `No Pokémon found for "notarealpokemon".`
	view := m.View()
	if !strings.Contains(view, m.errMsg) {
		t.Errorf("error searchModel.View() missing error message\ngot:\n%s", view)
	}
	if strings.Contains(view, "Searching...") {
		t.Errorf("error searchModel.View() should not show spinner\ngot:\n%s", view)
	}
}

func TestSearchModel_Update_EscGoesBack(t *testing.T) {
	m := newSearchModel(nil, nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("Update(Esc) returned a nil cmd, want goBack()")
	}
	if _, ok := cmd().(backMsg); !ok {
		t.Errorf("Update(Esc) cmd produced %T, want backMsg", cmd())
	}
}

// TestSearchModel_Update_IgnoresKeysWhileLoading proves keystrokes are
// dropped while a lookup is in flight, so a fast typist can't double-submit
// or edit the query out from under an in-flight request (see the doc
// comment on searchModel.Update).
func TestSearchModel_Update_IgnoresKeysWhileLoading(t *testing.T) {
	m := newSearchModel(nil, nil)
	m.input.SetValue("pikachu")
	m.loading = true

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if cmd != nil {
		t.Errorf(`Update("x") while loading returned a non-nil cmd, want nil`)
	}
	if got.input.Value() != "pikachu" {
		t.Errorf("Update(%q) while loading changed input to %q, want unchanged %q", "x", got.input.Value(), "pikachu")
	}
}

// TestSearchModel_Update_EscIgnoredWhileLoading proves even the Esc
// shortcut is dropped while loading, consistent with every other key.
func TestSearchModel_Update_EscIgnoredWhileLoading(t *testing.T) {
	m := newSearchModel(nil, nil)
	m.loading = true

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd != nil {
		t.Errorf("Update(Esc) while loading returned a non-nil cmd, want nil")
	}
}

// TestSearchModel_Update_SpinnerTickIgnoredWhenNotLoading proves a stray
// spinner tick (e.g. one already in flight when a lookup finishes) is a
// no-op once loading has gone back to false.
func TestSearchModel_Update_SpinnerTickIgnoredWhenNotLoading(t *testing.T) {
	m := newSearchModel(nil, nil)
	m.loading = false

	got, cmd := m.Update(spinner.TickMsg{})

	if cmd != nil {
		t.Errorf("Update(spinner.TickMsg) while not loading returned a non-nil cmd, want nil")
	}
	if got.spinner.View() != m.spinner.View() {
		t.Error("Update(spinner.TickMsg) while not loading changed the spinner, want unchanged")
	}
}

// TestSearchModel_Update_SpinnerTickAdvancesWhileLoading proves a spinner
// tick is delegated to the spinner sub-model, and rescheduled, while a
// lookup is in flight.
func TestSearchModel_Update_SpinnerTickAdvancesWhileLoading(t *testing.T) {
	m := newSearchModel(nil, nil)
	m.loading = true

	_, cmd := m.Update(spinner.TickMsg{})

	if cmd == nil {
		t.Fatal("Update(spinner.TickMsg) while loading returned a nil cmd, want the spinner's next tick")
	}
}

// TestSearchModel_Update_LookupResultMsg_Error proves a failed lookup
// clears the loading state and sets a rendered error message, without
// transitioning screens.
func TestSearchModel_Update_LookupResultMsg_Error(t *testing.T) {
	m := newSearchModel(nil, nil)
	m.loading = true
	m.input.SetValue("notarealpokemon")

	got, cmd := m.Update(lookupResultMsg{err: &pokeapi.LookupError{Query: "notarealpokemon"}})

	if got.loading {
		t.Error("Update(lookupResultMsg{err}) left loading = true, want false")
	}
	if got.errMsg == "" {
		t.Error("Update(lookupResultMsg{err}) left errMsg empty, want a rendered message")
	}
	if cmd != nil {
		t.Errorf("Update(lookupResultMsg{err}) returned a non-nil cmd, want nil (no screen transition on error)")
	}
}

// TestSearchModel_Update_LookupResultMsg_Success proves a successful lookup
// clears loading and returns a cmd that switches to the Result Screen.
func TestSearchModel_Update_LookupResultMsg_Success(t *testing.T) {
	m := newSearchModel(nil, nil)
	m.loading = true
	stat := testStatBlock()

	got, cmd := m.Update(lookupResultMsg{stat: stat})

	if got.loading {
		t.Error("Update(lookupResultMsg{stat}) left loading = true, want false")
	}
	if cmd == nil {
		t.Fatal("Update(lookupResultMsg{stat}) returned a nil cmd, want showResult(...)")
	}
	msg, ok := cmd().(showResultMsg)
	if !ok || msg.stat.Name != stat.Name {
		t.Errorf("Update(lookupResultMsg{stat}) cmd produced %#v, want showResultMsg carrying %q", cmd(), stat.Name)
	}
}

// TestSearchModel_Submit_EmptyInputNoOps proves submitting a blank query
// (or one that's only whitespace) does nothing - no loading state, no
// lookup command - rather than firing a doomed PokeAPI request.
func TestSearchModel_Submit_EmptyInputNoOps(t *testing.T) {
	for _, value := range []string{"", "   ", "\t"} {
		t.Run("input="+value, func(t *testing.T) {
			m := newSearchModel(nil, nil)
			m.input.SetValue(value)

			got, cmd := m.submit()

			if cmd != nil {
				t.Errorf("submit() with input %q returned a non-nil cmd, want nil", value)
			}
			if got.loading {
				t.Errorf("submit() with input %q set loading = true, want false", value)
			}
		})
	}
}

// TestSearchModel_Reset_ClearsState proves reset clears input, error, and
// loading state left over from a previous visit, and refocuses the input -
// called by App on every transition into the Search Screen.
func TestSearchModel_Reset_ClearsState(t *testing.T) {
	m := newSearchModel(nil, nil)
	m.input.SetValue("stale")
	m.input.Blur()
	m.errMsg = "stale error"
	m.loading = true

	got := m.reset()

	if got.input.Value() != "" {
		t.Errorf("reset().input.Value() = %q, want empty", got.input.Value())
	}
	if !got.input.Focused() {
		t.Error("reset().input.Focused() = false, want true")
	}
	if got.errMsg != "" {
		t.Errorf("reset().errMsg = %q, want empty", got.errMsg)
	}
	if got.loading {
		t.Error("reset().loading = true, want false")
	}
}

// TestErrorMessageFor distinguishes CONTEXT.md's Lookup Error and Service
// Error messages directly, complementing the e2e coverage of the same
// distinction end-to-end.
func TestErrorMessageFor(t *testing.T) {
	lookupMsg := errorMessageFor(&pokeapi.LookupError{Query: "notarealpokemon"}, "notarealpokemon")
	if !strings.Contains(lookupMsg, "No Pokémon found") || !strings.Contains(lookupMsg, "notarealpokemon") {
		t.Errorf("errorMessageFor(LookupError) = %q, want it to name the query as a Lookup Error", lookupMsg)
	}

	serviceMsg := errorMessageFor(&pokeapi.ServiceError{}, "pikachu")
	if !strings.Contains(serviceMsg, "Couldn't reach PokeAPI") {
		t.Errorf("errorMessageFor(ServiceError) = %q, want a Service Error message", serviceMsg)
	}

	if lookupMsg == serviceMsg {
		t.Error("errorMessageFor produced the same message for a LookupError and a ServiceError, want them distinguishable (CONTEXT.md)")
	}
}

// TestSearchModel_Update_TabCyclesFocus proves Tab moves focus from the
// input to the "Search by Type" button and back, blurring/focusing the
// textinput sub-model to match.
func TestSearchModel_Update_TabCyclesFocus(t *testing.T) {
	m := newSearchModel(nil, nil)
	if m.focus != searchFocusInput || !m.input.Focused() {
		t.Fatal("newSearchModel should start with the input focused")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focus != searchFocusButton {
		t.Error("Update(Tab) did not move focus to the button")
	}
	if m.input.Focused() {
		t.Error("Update(Tab) left the input focused, want it blurred once the button has focus")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.focus != searchFocusInput {
		t.Error("Update(Shift+Tab) did not move focus back to the input")
	}
	if !m.input.Focused() {
		t.Error("Update(Shift+Tab) left the input blurred, want it focused again")
	}
}

// TestSearchModel_Update_EnterOnButtonGoesToTypeSelect proves Enter, while
// the button has focus, advances to the Type Select Screen instead of
// submitting the (irrelevant, possibly empty) input.
func TestSearchModel_Update_EnterOnButtonGoesToTypeSelect(t *testing.T) {
	m := newSearchModel(nil, nil)
	m.focus = searchFocusButton

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Update(Enter) with the button focused returned a nil cmd, want switchTo(screenTypeSelect)")
	}
	msg, ok := cmd().(switchScreenMsg)
	if !ok || msg.to != screenTypeSelect {
		t.Errorf("Update(Enter) with the button focused produced %#v, want switchScreenMsg{to: screenTypeSelect}", cmd())
	}
}

// TestSearchModel_Update_ButtonFocusedIgnoresTyping proves rune keys don't
// leak into the input while the button has focus.
func TestSearchModel_Update_ButtonFocusedIgnoresTyping(t *testing.T) {
	m := newSearchModel(nil, nil)
	m.focus = searchFocusButton

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if cmd != nil {
		t.Error(`Update("x") with the button focused returned a non-nil cmd, want nil`)
	}
	if got.input.Value() != "" {
		t.Errorf("Update(%q) with the button focused wrote into the input: %q", "x", got.input.Value())
	}
}

// TestSearchModel_Update_MouseClickOnButtonGoesToTypeSelect proves clicking
// the button's zone advances to the Type Select Screen, using a real
// *zone.Manager the same way App wires one up, rather than nil.
func TestSearchModel_Update_MouseClickOnButtonGoesToTypeSelect(t *testing.T) {
	zones := zone.New()
	t.Cleanup(zones.Close)
	m := newSearchModel(nil, zones)
	zones.Scan(m.View())
	// zone registration happens on a background worker (see bubblezone's
	// own tests, which use the same pattern), so a freshly-scanned zone
	// isn't necessarily visible to Get() the instant Scan() returns.
	time.Sleep(15 * time.Millisecond)
	buttonZone := zones.Get(searchByTypeButtonZoneID)
	if buttonZone.IsZero() {
		t.Fatal("button zone was not registered by the scan; is it still wrapped in markZone?")
	}
	click := tea.MouseMsg{X: buttonZone.StartX, Y: buttonZone.StartY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, cmd := m.Update(click)

	if cmd == nil {
		t.Fatal("Update(click on button) returned a nil cmd, want switchTo(screenTypeSelect)")
	}
	msg, ok := cmd().(switchScreenMsg)
	if !ok || msg.to != screenTypeSelect {
		t.Errorf("Update(click on button) produced %#v, want switchScreenMsg{to: screenTypeSelect}", cmd())
	}
}

// TestSearchModel_Update_MouseIgnoredWhileLoading proves a click can't
// double-submit or jump screens while a lookup is in flight, mirroring the
// existing keyboard guard.
func TestSearchModel_Update_MouseIgnoredWhileLoading(t *testing.T) {
	zones := zone.New()
	t.Cleanup(zones.Close)
	m := newSearchModel(nil, zones)
	m.loading = true
	zones.Scan(m.View())
	time.Sleep(15 * time.Millisecond)
	buttonZone := zones.Get(searchByTypeButtonZoneID)
	click := tea.MouseMsg{X: buttonZone.StartX, Y: buttonZone.StartY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, cmd := m.Update(click)

	if cmd != nil {
		t.Error("Update(click on button) while loading returned a non-nil cmd, want nil")
	}
}

// TestSearchModel_Update_MouseClickOutsideButtonIsNoOp proves a click that
// doesn't land on the button's zone is simply ignored.
func TestSearchModel_Update_MouseClickOutsideButtonIsNoOp(t *testing.T) {
	zones := zone.New()
	t.Cleanup(zones.Close)
	m := newSearchModel(nil, zones)
	zones.Scan(m.View())
	time.Sleep(15 * time.Millisecond)

	click := tea.MouseMsg{X: 9999, Y: 9999, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	_, cmd := m.Update(click)

	if cmd != nil {
		t.Error("Update(click outside the button) returned a non-nil cmd, want nil")
	}
}

// TestSearchModel_Update_MouseClickWithNilZonesIsSafe proves a click is
// simply ignored (not a panic) when the model has no zone manager - the
// same convention as passing a nil *pokeapi.Client where it's not needed.
func TestSearchModel_Update_MouseClickWithNilZonesIsSafe(t *testing.T) {
	m := newSearchModel(nil, nil)

	click := tea.MouseMsg{X: 5, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	_, cmd := m.Update(click)

	if cmd != nil {
		t.Error("Update(click) with nil zones returned a non-nil cmd, want nil")
	}
}
