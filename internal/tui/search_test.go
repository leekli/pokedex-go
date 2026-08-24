package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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

// TestSearchModel_View_ShowsExpectedCopy is a smoke test for the Playful
// Maximalist redesign's static content: the title, instruction, dex-range
// hint, and example queries should all appear on the idle Search Screen.
func TestSearchModel_View_ShowsExpectedCopy(t *testing.T) {
	m := newSearchModel(nil)
	view := m.View()

	for _, want := range []string{
		"POKÉDEX SEARCH",
		"Find any Pokémon by name or National Dex Number.",
		"#001", "#1025",
		"pikachu", "charizard", "snorlax",
		"Enter to search · Esc to go back · Ctrl+C to quit",
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
	m := newSearchModel(nil)

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
