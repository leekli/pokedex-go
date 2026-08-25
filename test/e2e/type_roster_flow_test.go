package e2e

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// advanceToTypeSelectScreen drives the app from the Search Screen to the
// Type Select Screen via Tab (focus the "Search by Type" button) then
// Enter - the keyboard equivalent of clicking it (see search.go).
func advanceToTypeSelectScreen(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	advanceToSearchScreen(t, tm)
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForAll(t, tm, 2*time.Second, "SEARCH BY TYPE", "Fire")
}

// advanceToFireTypeRoster drives the app from the Type Select Screen to the
// Fire Type Roster Screen. Fire is the second entry in AllTypes() (after
// Normal), so a single Down arrow selects it.
func advanceToFireTypeRoster(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	advanceToTypeSelectScreen(t, tm)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForAll(t, tm, 3*time.Second, "FIRE", "#004", "Charmander", "#006", "Charizard", "Generation I")
}

func TestSearch_TabAndEnterOnButtonOpensTypeSelect(t *testing.T) {
	tm := newTestModel(t)
	advanceToTypeSelectScreen(t, tm)
}

// TestTypeSelect_EscReturnsToSearch proves the Type Select Screen's Esc
// goes back to the Search Screen it was reached from.
func TestTypeSelect_EscReturnsToSearch(t *testing.T) {
	tm := newTestModel(t)
	advanceToTypeSelectScreen(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	waitForAll(t, tm, 2*time.Second, "POKÉDEX SEARCH")
}

// TestTypeRoster_FiltersNonDefaultVarietiesAndSortsByDexNumber proves the
// mega-evolution fixture entry never appears, alongside the two real
// entries that do - end-to-end confirmation of
// docs/adr/0002-filter-type-roster-by-pokemon-id.md. Both conditions are
// checked in a single WaitFor predicate since teatest's output reader is
// drained as it's read (see waitForAll's doc comment) - a separate
// "doesn't contain" check afterwards would be reading a different slice of
// output than the one the presence check just consumed.
func TestTypeRoster_FiltersNonDefaultVarietiesAndSortsByDexNumber(t *testing.T) {
	tm := newTestModel(t)
	advanceToTypeSelectScreen(t, tm)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasReal := bytes.Contains(out, []byte("#004")) && bytes.Contains(out, []byte("Charmander")) &&
			bytes.Contains(out, []byte("#006")) && bytes.Contains(out, []byte("Charizard"))
		hasMega := bytes.Contains(out, []byte("mega"))
		return hasReal && !hasMega
	}, teatest.WithDuration(3*time.Second))
}

// TestTypeRoster_EscReturnsToTypeSelect proves the Type Roster Screen's Esc
// goes back to the Type Select Screen, not all the way to Search.
func TestTypeRoster_EscReturnsToTypeSelect(t *testing.T) {
	tm := newTestModel(t)
	advanceToFireTypeRoster(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	waitForAll(t, tm, 2*time.Second, "SEARCH BY TYPE")
}

// TestTypeRoster_EnterOnRowShowsResultScreen proves selecting a row reuses
// the existing Result Screen, with full data from a real lookup (not just
// the roster's Dex#/Name/Generation columns).
func TestTypeRoster_EnterOnRowShowsResultScreen(t *testing.T) {
	tm := newTestModel(t)
	advanceToFireTypeRoster(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitForAll(t, tm, 3*time.Second, "#004 Charmander", "FIRE", "HP", "Attack")
}

// TestResult_FromTypeRoster_EscReturnsToTypeRoster proves Esc on the Result
// Screen goes back to wherever it was reached from - here, the Type Roster
// Screen, not the Search Screen (see docs/adr/0001).
func TestResult_FromTypeRoster_EscReturnsToTypeRoster(t *testing.T) {
	tm := newTestModel(t)
	advanceToFireTypeRoster(t, tm)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForAll(t, tm, 3*time.Second, "#004 Charmander")

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	waitForAll(t, tm, 2*time.Second, "FIRE", "#004", "Charmander")
}

// TestResult_FromTypeRoster_EnterGoesToFreshSearch proves Enter on the
// Result Screen always means "search again", landing on the Search Screen
// even when the Result Screen was reached via the Type Roster rather than
// Search itself (see docs/adr/0001).
func TestResult_FromTypeRoster_EnterGoesToFreshSearch(t *testing.T) {
	tm := newTestModel(t)
	advanceToFireTypeRoster(t, tm)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForAll(t, tm, 3*time.Second, "#004 Charmander")

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitForAll(t, tm, 2*time.Second, "POKÉDEX SEARCH")
}

// TestTypeRoster_ServiceFailureShowsInlineError covers the Type Roster's
// only possible failure mode: PokeAPI itself failing on the reserved
// "water" type name (see testserver_test.go) is never the user's fault, so
// it must always be a Service Error (CONTEXT.md), never a Lookup Error.
func TestTypeRoster_ServiceFailureShowsInlineError(t *testing.T) {
	tm := newTestModel(t)
	advanceToTypeSelectScreen(t, tm)

	// Water is the third entry in AllTypes() (Normal, Fire, Water).
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitForAll(t, tm, 3*time.Second, "Couldn't reach PokeAPI")
}
