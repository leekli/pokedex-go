package e2e

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestSearch_ServiceFailureShowsServiceError covers the Service Error path
// from CONTEXT.md: the fixture server's reserved "servererror" name returns
// HTTP 500, which is not the user's fault, so the message must be
// distinguishable from the Lookup Error case (see search_lookup_error_test.go).
func TestSearch_ServiceFailureShowsServiceError(t *testing.T) {
	tm := newTestModel(t)
	advanceToSearchScreen(t, tm)

	tm.Type("servererror")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	waitForAll(t, tm, 3*time.Second, "Couldn't reach PokeAPI")
}
