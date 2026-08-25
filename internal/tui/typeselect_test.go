package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leekli/pokedex-go/internal/pokemon"
	zone "github.com/lrstanley/bubblezone"
)

func TestNewTypeSelectModel_ListsAllTypes(t *testing.T) {
	m := newTypeSelectModel(nil)
	if len(m.types) != 18 {
		t.Fatalf("newTypeSelectModel().types has %d entries, want 18", len(m.types))
	}
	if m.cursor != 0 {
		t.Errorf("newTypeSelectModel().cursor = %d, want 0", m.cursor)
	}
}

func TestTypeSelectModel_View_ShowsEveryTypeCapitalized(t *testing.T) {
	m := newTypeSelectModel(nil)
	view := m.View()

	for _, typ := range pokemon.AllTypes() {
		want := capitalize(typ)
		if !strings.Contains(view, want) {
			t.Errorf("typeSelectModel.View() missing %q\ngot:\n%s", want, view)
		}
	}
	for _, want := range []string{"SEARCH BY TYPE", "↑/↓ browse · Enter select · Esc back · Ctrl+C quit"} {
		if !strings.Contains(view, want) {
			t.Errorf("typeSelectModel.View() missing %q", want)
		}
	}
}

func TestTypeSelectModel_Update_ArrowKeysMoveCursor(t *testing.T) {
	m := newTypeSelectModel(nil)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("cursor after Down = %d, want 1", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("cursor after Up = %d, want 0", m.cursor)
	}
}

func TestTypeSelectModel_Update_CursorClampsAtBounds(t *testing.T) {
	m := newTypeSelectModel(nil)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("cursor after Up at top = %d, want 0 (clamped)", m.cursor)
	}

	for range len(m.types) + 5 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.cursor != len(m.types)-1 {
		t.Errorf("cursor after repeated Down = %d, want %d (clamped)", m.cursor, len(m.types)-1)
	}
}

func TestTypeSelectModel_Update_JKAlsoMoveCursor(t *testing.T) {
	m := newTypeSelectModel(nil)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 1 {
		t.Errorf("cursor after j = %d, want 1", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("cursor after k = %d, want 0", m.cursor)
	}
}

func TestTypeSelectModel_Update_EnterSelectsCurrentType(t *testing.T) {
	m := newTypeSelectModel(nil)
	m.cursor = 3

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Update(Enter) returned a nil cmd, want showTypeRoster")
	}
	msg, ok := cmd().(showTypeRosterMsg)
	if !ok || msg.typeName != m.types[3] {
		t.Errorf("Update(Enter) cmd produced %#v, want showTypeRosterMsg{typeName: %q}", cmd(), m.types[3])
	}
}

func TestTypeSelectModel_Update_EscGoesBack(t *testing.T) {
	m := newTypeSelectModel(nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("Update(Esc) returned a nil cmd, want goBack()")
	}
	if _, ok := cmd().(backMsg); !ok {
		t.Errorf("Update(Esc) cmd produced %T, want backMsg", cmd())
	}
}

// TestTypeSelectModel_Update_MouseClickSelectsAndAdvances proves clicking a
// type badge both moves the cursor there and immediately advances to the
// Type Roster Screen - the spec's own wording ("when a user clicks a type
// ... a new screen should load") describes a single-step action, unlike the
// Type Roster Screen's own click handling (see typeroster.go).
func TestTypeSelectModel_Update_MouseClickSelectsAndAdvances(t *testing.T) {
	zones := zone.New()
	t.Cleanup(zones.Close)
	m := newTypeSelectModel(zones)
	zones.Scan(m.View())
	time.Sleep(15 * time.Millisecond)

	targetIndex := 5
	targetZone := zones.Get(typeSelectZoneID(m.types[targetIndex]))
	if targetZone.IsZero() {
		t.Fatalf("zone for %q was not registered", m.types[targetIndex])
	}
	click := tea.MouseMsg{X: targetZone.StartX, Y: targetZone.StartY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	m, cmd := m.Update(click)

	if m.cursor != targetIndex {
		t.Errorf("cursor after click = %d, want %d", m.cursor, targetIndex)
	}
	if cmd == nil {
		t.Fatal("Update(click) returned a nil cmd, want showTypeRoster")
	}
	msg, ok := cmd().(showTypeRosterMsg)
	if !ok || msg.typeName != m.types[targetIndex] {
		t.Errorf("Update(click) cmd produced %#v, want showTypeRosterMsg{typeName: %q}", cmd(), m.types[targetIndex])
	}
}

func TestTypeSelectModel_Update_MouseClickOutsideAnyZoneIsNoOp(t *testing.T) {
	zones := zone.New()
	t.Cleanup(zones.Close)
	m := newTypeSelectModel(zones)
	zones.Scan(m.View())
	time.Sleep(15 * time.Millisecond)

	click := tea.MouseMsg{X: 9999, Y: 9999, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	got, cmd := m.Update(click)

	if cmd != nil {
		t.Error("Update(click outside any zone) returned a non-nil cmd, want nil")
	}
	if got.cursor != m.cursor {
		t.Error("Update(click outside any zone) changed the cursor, want unchanged")
	}
}

func TestTypeSelectModel_Update_NonPressMouseIsIgnored(t *testing.T) {
	m := newTypeSelectModel(nil)

	_, cmd := m.Update(tea.MouseMsg{Action: tea.MouseActionMotion})

	if cmd != nil {
		t.Error("Update(mouse motion) returned a non-nil cmd, want nil")
	}
}

// TestTypeSelectModel_Update_UnknownMsgIsNoOp proves a message type none of
// the cases match falls through to a plain no-op rather than panicking.
func TestTypeSelectModel_Update_UnknownMsgIsNoOp(t *testing.T) {
	m := newTypeSelectModel(nil)

	_, cmd := m.Update(struct{}{})

	if cmd != nil {
		t.Error("Update(unknown msg) returned a non-nil cmd, want nil")
	}
}
