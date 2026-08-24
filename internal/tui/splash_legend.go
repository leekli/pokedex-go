package tui

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

//go:embed assets/splash.txt
var splashArt string

// splashRoleColor maps each semantic-role character in assets/splash.txt to
// its resting hex color. Rendering through lipgloss (rather than embedding
// raw ANSI escapes in the asset) means the splash art participates in the
// same terminal-capability color degradation as the rest of the UI.
var splashRoleColor = map[rune]string{
	'Y': "#FFCB05", // Pokémon yellow (wordmark fill)
	'B': "#3B4CCA", // Pokémon blue (wordmark outline)
	'R': "#EE1515", // Poké Ball red
	'W': "#FFFFFF", // Poké Ball white
	'K': "#222224", // Poké Ball band and outline
}

// blockGlyph is drawn, in the appropriate color, for every recognized role
// character in the shape file - a solid block reads as pixel art, where
// printing the literal role letters would just look like a wall of text.
const blockGlyph = "█"

// splashSweepHighlight is the bright "shine" color that sweeps across the
// splash art on startup, before each pixel settles into its resting
// splashRoleColor.
const splashSweepHighlight = "#FFFFFF"

// splashSweepBand is how many columns wide the shine is, on each side of its
// leading edge.
const splashSweepBand = 8

// splashArtWidth is the widest line in splashArt, in runes - the horizontal
// distance the startup sweep travels across.
var splashArtWidth = widestLine(splashArt)

func widestLine(art string) int {
	width := 0
	for line := range strings.SplitSeq(art, "\n") {
		if n := utf8.RuneCountInString(line); n > width {
			width = n
		}
	}
	return width
}

// renderSplashArtSweep renders splashArt mid-way through its startup
// animation: a bright shine sweeps left to right across the art as progress
// goes from 0 (the sweep hasn't reached it yet) to 1 (the sweep has fully
// passed and every pixel is at rest in its final color). Every role rune is
// always drawn in its own resting color, just brightened towards
// splashSweepHighlight the closer the sweep's leading edge is to its column.
func renderSplashArtSweep(progress float64) string {
	// front travels from -splashSweepBand (fully off-canvas, left) to
	// splashArtWidth+splashSweepBand (fully off-canvas, right), so every
	// column - including the first and last - passes fully through the
	// shine and settles back to its resting color by progress 1.
	front := progress*float64(splashArtWidth+2*splashSweepBand) - float64(splashSweepBand)

	var out strings.Builder
	col := 0
	for _, r := range splashArt {
		if r == '\n' {
			out.WriteByte('\n')
			col = 0
			continue
		}
		if base, ok := splashRoleColor[r]; ok {
			out.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(sweepColor(base, col, front))).
				Render(blockGlyph))
		} else {
			out.WriteRune(r)
		}
		col++
	}
	return out.String()
}

// sweepColor blends base towards splashSweepHighlight based on how close
// column col is to the sweep's leading edge: fully highlighted at the edge,
// fading back to base over splashSweepBand columns on either side.
func sweepColor(base string, col int, front float64) string {
	distance := front - float64(col)
	if distance < 0 {
		distance = -distance
	}
	if distance >= splashSweepBand {
		return base
	}
	return blendHex(base, splashSweepHighlight, 1-distance/splashSweepBand)
}

// blendHex linearly interpolates between two "#RRGGBB" colors: t=0 is a,
// t=1 is b.
func blendHex(a, b string, t float64) string {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	ar, ag, ab := hexRGB(a)
	br, bg, bb := hexRGB(b)
	return fmt.Sprintf("#%02X%02X%02X",
		lerpByte(ar, br, t), lerpByte(ag, bg, t), lerpByte(ab, bb, t))
}

func hexRGB(hex string) (r, g, b uint8) {
	v, _ := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	return uint8(v >> 16), uint8(v >> 8), uint8(v)
}

func lerpByte(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*t)
}
