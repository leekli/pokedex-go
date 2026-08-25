package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leekli/pokedex-go/internal/pokeapi"
)

func TestNewTypeRosterModel_StartsLoading(t *testing.T) {
	m := newTypeRosterModel(nil, "fire", nil)
	if !m.loading {
		t.Error("newTypeRosterModel().loading = false, want true")
	}
	if m.typeName != "fire" {
		t.Errorf("newTypeRosterModel().typeName = %q, want fire", m.typeName)
	}
}

func TestTypeRosterModel_View_ShowsLoadingSpinner(t *testing.T) {
	m := newTypeRosterModel(nil, "fire", nil)
	view := m.View()

	if !strings.Contains(view, "Loading Fire-type Pokémon") {
		t.Errorf("typeRosterModel.View() while loading missing loading text\ngot:\n%s", view)
	}
	if !strings.Contains(view, "FIRE") {
		t.Errorf("typeRosterModel.View() missing the type header\ngot:\n%s", view)
	}
}

func TestTypeRosterModel_Update_ResultPopulatesTable(t *testing.T) {
	m := newTypeRosterModel(nil, "fire", nil)

	m, _ = m.Update(typeRosterResultMsg{
		typeName: "fire",
		entries: []pokeapi.TypeRosterEntry{
			{DexNumber: 4, Name: "charmander"},
			{DexNumber: 6, Name: "charizard"},
		},
		generations: map[int]string{4: "generation-i"},
	})

	if m.loading {
		t.Error("loading = true after typeRosterResultMsg, want false")
	}
	if len(m.table.Rows()) != 2 {
		t.Fatalf("table has %d rows, want 2", len(m.table.Rows()))
	}
	view := m.View()
	for _, want := range []string{"#004", "Charmander", "Generation I", "#006", "Charizard", "—"} {
		if !strings.Contains(view, want) {
			t.Errorf("typeRosterModel.View() after load missing %q\ngot:\n%s", want, view)
		}
	}
}

func TestTypeRosterModel_Update_ResultError(t *testing.T) {
	m := newTypeRosterModel(nil, "fire", nil)

	m, cmd := m.Update(typeRosterResultMsg{typeName: "fire", err: &pokeapi.ServiceError{}})

	if m.loading {
		t.Error("loading = true after a failed fetch, want false")
	}
	if m.errMsg == "" {
		t.Error("errMsg is empty after a failed fetch, want a rendered Service Error message")
	}
	if cmd != nil {
		t.Error("Update(typeRosterResultMsg{err}) returned a non-nil cmd, want nil")
	}
	if !strings.Contains(m.View(), "Couldn't reach PokeAPI") {
		t.Errorf("typeRosterModel.View() after a failed fetch missing the Service Error message\ngot:\n%s", m.View())
	}
}

// TestTypeRosterModel_Update_KeysIgnoredWhileLoading proves navigation keys
// (including Esc) are blocked until the roster has actually loaded,
// mirroring the Search Screen's same guard.
func TestTypeRosterModel_Update_KeysIgnoredWhileLoading(t *testing.T) {
	m := newTypeRosterModel(nil, "fire", nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd != nil {
		t.Error("Update(Esc) while loading returned a non-nil cmd, want nil")
	}
}

func TestTypeRosterModel_Update_EscGoesBackOnceLoaded(t *testing.T) {
	m := loadedTypeRosterModel(t)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("Update(Esc) returned a nil cmd, want goBack()")
	}
	if _, ok := cmd().(backMsg); !ok {
		t.Errorf("Update(Esc) cmd produced %T, want backMsg", cmd())
	}
}

// TestTypeRosterModel_Update_ArrowKeysDelegateToTable proves keyboard
// navigation is delegated to the underlying bubbles/table for its built-in
// scrolling/cursor behavior.
func TestTypeRosterModel_Update_ArrowKeysDelegateToTable(t *testing.T) {
	m := loadedTypeRosterModel(t)
	if m.table.Cursor() != 0 {
		t.Fatalf("test setup: table cursor = %d, want 0", m.table.Cursor())
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	if m.table.Cursor() != 1 {
		t.Errorf("table cursor after Down = %d, want 1", m.table.Cursor())
	}
}

func TestTypeRosterModel_Update_MouseWheelMovesSelection(t *testing.T) {
	m := loadedTypeRosterModel(t)

	m, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	if m.table.Cursor() != 1 {
		t.Errorf("table cursor after wheel down = %d, want 1", m.table.Cursor())
	}

	m, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	if m.table.Cursor() != 0 {
		t.Errorf("table cursor after wheel up = %d, want 0", m.table.Cursor())
	}
}

// TestTypeRosterModel_Update_EnterLooksUpSelectedRow proves Enter fires the
// same lookupCmd machinery the Search Screen uses, keyed off the selected
// row's name, and enters a "loading detail" state that blocks further keys.
func TestTypeRosterModel_Update_EnterLooksUpSelectedRow(t *testing.T) {
	m := loadedTypeRosterModel(t)

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !m.loadingDetail {
		t.Error("loadingDetail = false after Enter, want true")
	}
	if cmd == nil {
		t.Fatal("Update(Enter) returned a nil cmd, want a lookupCmd batch")
	}

	_, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd2 != nil {
		t.Error("Update(Esc) while loadingDetail returned a non-nil cmd, want nil (blocked like other loading states)")
	}
}

// TestTypeRosterModel_Update_EnterQueriesTheOriginalSlugNotTheDisplayName
// is a regression test: rosterRows renders each name capitalized for
// display ("Charmander"), but PokeAPI only recognizes the lowercase slug
// ("charmander") - Enter must normalize the selected row back before
// querying, not send the capitalized string straight to lookupCmd.
func TestTypeRosterModel_Update_EnterQueriesTheOriginalSlugNotTheDisplayName(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	m := newTypeRosterModel(pokeapi.NewClient(pokeapi.WithBaseURL(server.URL)), "fire", nil)
	m, _ = m.Update(typeRosterResultMsg{
		typeName: "fire",
		entries:  []pokeapi.TypeRosterEntry{{DexNumber: 4, Name: "charmander"}},
	})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// cmd is tea.Batch(m.spinner.Tick, lookupCmd(...)); invoking it returns
	// a BatchMsg of the two sub-commands rather than running them, so run
	// the lookupCmd one (index 1 - see typeroster.go's Update) ourselves to
	// actually issue the HTTP request. The spinner tick (index 0) is left
	// alone: it sleeps for its own tick duration, which would only slow
	// this test down for no benefit here.
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatalf("Update(Enter) cmd produced %#v, want a 2-element tea.BatchMsg", cmd())
	}
	batch[1]()

	if requestedPath != "/pokemon/charmander" {
		t.Errorf("requested path = %q, want /pokemon/charmander (the original slug, not the capitalized display name)", requestedPath)
	}
}

func TestTypeRosterModel_Update_LookupResultMsg_SuccessShowsResult(t *testing.T) {
	m := loadedTypeRosterModel(t)
	m.loadingDetail = true

	_, cmd := m.Update(lookupResultMsg{stat: testStatBlock()})

	if cmd == nil {
		t.Fatal("Update(lookupResultMsg) returned a nil cmd, want showResult(...)")
	}
	msg, ok := cmd().(showResultMsg)
	if !ok || msg.stat.Name != testStatBlock().Name {
		t.Errorf("Update(lookupResultMsg) cmd produced %#v, want showResultMsg", cmd())
	}
}

func TestTypeRosterModel_Update_LookupResultMsg_ErrorShowsInline(t *testing.T) {
	m := loadedTypeRosterModel(t)
	m.loadingDetail = true

	m, cmd := m.Update(lookupResultMsg{err: &pokeapi.LookupError{Query: "charmander"}})

	if m.loadingDetail {
		t.Error("loadingDetail = true after a failed detail fetch, want false")
	}
	if m.errMsg == "" {
		t.Error("errMsg is empty after a failed detail fetch, want a rendered message")
	}
	if cmd != nil {
		t.Error("Update(lookupResultMsg{err}) returned a non-nil cmd, want nil (no screen transition on error)")
	}
}

// loadedTypeRosterModel builds a typeRosterModel already past its initial
// load, with two rows, for tests that only care about post-load behavior.
func loadedTypeRosterModel(t *testing.T) typeRosterModel {
	t.Helper()
	m := newTypeRosterModel(nil, "fire", nil)
	m, _ = m.Update(typeRosterResultMsg{
		typeName: "fire",
		entries: []pokeapi.TypeRosterEntry{
			{DexNumber: 4, Name: "charmander"},
			{DexNumber: 6, Name: "charizard"},
		},
	})
	return m
}

// TestTypeRosterModel_Update_EnterWithNoRowsIsNoOp proves Enter on an
// (unexpected, but possible) empty roster doesn't panic indexing a
// nonexistent selected row.
func TestTypeRosterModel_Update_EnterWithNoRowsIsNoOp(t *testing.T) {
	m := newTypeRosterModel(nil, "fire", nil)
	m, _ = m.Update(typeRosterResultMsg{typeName: "fire", entries: nil})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		t.Error("Update(Enter) on an empty roster returned a non-nil cmd, want nil")
	}
}

// TestTypeRosterModel_Update_MouseWheelIgnoredWhileBusy proves scrolling is
// blocked mid-fetch, the same guard applied to keyboard input.
func TestTypeRosterModel_Update_MouseWheelIgnoredWhileBusy(t *testing.T) {
	m := loadedTypeRosterModel(t)
	m.loadingDetail = true

	m, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})

	if m.table.Cursor() != 0 {
		t.Errorf("table cursor after a wheel scroll while busy = %d, want unchanged 0", m.table.Cursor())
	}
}

// TestTypeRosterModel_Update_SpinnerTickIgnoredWhenNotBusy proves a stray
// spinner tick once the screen is idle (loaded, no detail fetch running)
// is a no-op.
func TestTypeRosterModel_Update_SpinnerTickIgnoredWhenNotBusy(t *testing.T) {
	m := loadedTypeRosterModel(t)

	_, cmd := m.Update(spinner.TickMsg{})

	if cmd != nil {
		t.Error("Update(spinner.TickMsg) while idle returned a non-nil cmd, want nil")
	}
}

// TestTypeRosterModel_Update_UnknownMsgIsNoOp proves a message type none of
// the cases match falls through to a plain no-op rather than panicking.
func TestTypeRosterModel_Update_UnknownMsgIsNoOp(t *testing.T) {
	m := loadedTypeRosterModel(t)

	_, cmd := m.Update(struct{}{})

	if cmd != nil {
		t.Error("Update(unknown msg) returned a non-nil cmd, want nil")
	}
}

func TestRosterRows_FillsGenerationOrDash(t *testing.T) {
	entries := []pokeapi.TypeRosterEntry{
		{DexNumber: 1, Name: "bulbasaur"},
		{DexNumber: 999, Name: "unknown-gen-mon"},
	}
	generations := map[int]string{1: "generation-i"}

	rows := rosterRows(entries, generations)

	want := []table.Row{
		{"#001", "Bulbasaur", "Generation I"},
		{"#999", "Unknown-gen-mon", "—"},
	}
	if len(rows) != len(want) {
		t.Fatalf("rosterRows returned %d rows, want %d", len(rows), len(want))
	}
	for i := range want {
		if rows[i][0] != want[i][0] || rows[i][1] != want[i][1] || rows[i][2] != want[i][2] {
			t.Errorf("rosterRows[%d] = %v, want %v", i, rows[i], want[i])
		}
	}
}
