package tui

import (
	_ "embed"

	"github.com/charmbracelet/lipgloss"
)

//go:embed assets/splash.txt
var splashArt string

// splashLegend maps each semantic-role character in assets/splash.txt to a
// lipgloss style. Rendering through lipgloss (rather than embedding raw ANSI
// escapes in the asset) means the splash art participates in the same
// terminal-capability color degradation as the rest of the UI.
var splashLegend = map[rune]lipgloss.Style{
	'Y': lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCB05")), // Pokémon yellow
	'B': lipgloss.NewStyle().Foreground(lipgloss.Color("#3B4CCA")), // Pokémon blue
}

// blockGlyph is drawn, in the appropriate color, for every recognized role
// character in the shape file - a solid block reads as pixel art, where
// printing the literal 'Y'/'B' role letters would just look like a wall of
// text.
const blockGlyph = "█"

// renderSplashArt applies splashLegend to splashArt, rendering each
// recognized role rune as a colored block and any other rune (space,
// newline) verbatim.
func renderSplashArt() string {
	var out []byte
	for _, r := range splashArt {
		if r == '\n' {
			out = append(out, '\n')
			continue
		}
		if style, ok := splashLegend[r]; ok {
			out = append(out, style.Render(blockGlyph)...)
			continue
		}
		out = append(out, string(r)...)
	}
	return string(out)
}
