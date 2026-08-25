package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

// markZone wraps content in a mouse-click zone marker when zones is
// available, so a screen can be built and tested without one (e.g. a nil
// *zone.Manager, same convention as passing a nil *pokeapi.Client to a
// screen constructor that doesn't need it for a given test) — a
// *zone.Manager method call on a nil receiver panics, so this guards every
// call site rather than relying on each screen to remember to check.
func markZone(zones *zone.Manager, id, content string) string {
	if zones == nil {
		return content
	}
	return zones.Mark(id, content)
}

// zoneInBounds reports whether msg falls within the named zone, false if
// zones is nil or the zone isn't known yet (e.g. before the first render).
func zoneInBounds(zones *zone.Manager, id string, msg tea.MouseMsg) bool {
	if zones == nil {
		return false
	}
	return zones.Get(id).InBounds(msg)
}
