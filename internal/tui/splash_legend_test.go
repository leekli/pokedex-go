package tui

import (
	"strings"
	"testing"
)

func TestWidestLine(t *testing.T) {
	tests := map[string]int{
		"":           0,
		"abc":        3,
		"a\nbb\nccc": 3,
		"ccc\nbb\na": 3,
		"same\nsame": 4,
		"\n\n":       0,
		"trailing\n": 8,
		"日本語\nab":    3, // multi-byte runes counted as runes, not bytes
	}
	for input, want := range tests {
		if got := widestLine(input); got != want {
			t.Errorf("widestLine(%q) = %d, want %d", input, got, want)
		}
	}
}

// TestBlendHex_ClampsAtEndpoints proves t<=0 and t>=1 return the endpoint
// colors exactly, without going through the float lerp math (and its
// associated rounding) at all.
func TestBlendHex_ClampsAtEndpoints(t *testing.T) {
	if got := blendHex("#112233", "#445566", 0); got != "#112233" {
		t.Errorf("blendHex(t=0) = %q, want %q", got, "#112233")
	}
	if got := blendHex("#112233", "#445566", -5); got != "#112233" {
		t.Errorf("blendHex(t=-5) = %q, want %q", got, "#112233")
	}
	if got := blendHex("#112233", "#445566", 1); got != "#445566" {
		t.Errorf("blendHex(t=1) = %q, want %q", got, "#445566")
	}
	if got := blendHex("#112233", "#445566", 5); got != "#445566" {
		t.Errorf("blendHex(t=5) = %q, want %q", got, "#445566")
	}
}

func TestBlendHex_Midpoint(t *testing.T) {
	got := blendHex("#000000", "#FFFFFF", 0.5)
	// lerpByte(0, 255, 0.5) truncates to 127, for every channel.
	if got != "#7F7F7F" {
		t.Errorf("blendHex(black, white, 0.5) = %q, want %q", got, "#7F7F7F")
	}
}

func TestHexRGB(t *testing.T) {
	tests := []struct {
		hex     string
		r, g, b uint8
	}{
		{"#FFFFFF", 255, 255, 255},
		{"#000000", 0, 0, 0},
		{"#FF8000", 255, 128, 0},
		{"112233", 0x11, 0x22, 0x33}, // works without a leading '#' too
	}
	for _, tt := range tests {
		r, g, b := hexRGB(tt.hex)
		if r != tt.r || g != tt.g || b != tt.b {
			t.Errorf("hexRGB(%q) = (%d,%d,%d), want (%d,%d,%d)", tt.hex, r, g, b, tt.r, tt.g, tt.b)
		}
	}
}

func TestLerpByte(t *testing.T) {
	tests := []struct {
		a, b uint8
		t    float64
		want uint8
	}{
		{0, 255, 0, 0},
		{0, 255, 1, 255},
		{0, 100, 0.5, 50},
		{100, 0, 0.5, 50},
		{50, 50, 0.75, 50},
	}
	for _, tt := range tests {
		if got := lerpByte(tt.a, tt.b, tt.t); got != tt.want {
			t.Errorf("lerpByte(%d, %d, %v) = %d, want %d", tt.a, tt.b, tt.t, got, tt.want)
		}
	}
}

// TestSweepColor_AtAndBeyondBand proves a column at or past splashSweepBand
// distance from the sweep's leading edge is left at its resting color,
// entirely unblended - the highlight only touches nearby columns.
func TestSweepColor_AtAndBeyondBand(t *testing.T) {
	base := "#112233"
	if got := sweepColor(base, 0, splashSweepBand); got != base {
		t.Errorf("sweepColor at exactly splashSweepBand distance = %q, want unblended base %q", got, base)
	}
	if got := sweepColor(base, 0, splashSweepBand*10); got != base {
		t.Errorf("sweepColor far beyond splashSweepBand = %q, want unblended base %q", got, base)
	}
}

// TestSweepColor_AtLeadingEdgeIsFullyHighlighted proves a column exactly at
// the sweep's leading edge (distance 0) is blended all the way to the
// highlight color.
func TestSweepColor_AtLeadingEdgeIsFullyHighlighted(t *testing.T) {
	got := sweepColor("#112233", 5, 5)
	if got != splashSweepHighlight {
		t.Errorf("sweepColor at the leading edge = %q, want the highlight color %q", got, splashSweepHighlight)
	}
}

// TestSweepColor_SymmetricAroundEdge proves the shine fades the same way on
// either side of the leading edge (the doc comment promises "on either
// side").
func TestSweepColor_SymmetricAroundEdge(t *testing.T) {
	ahead := sweepColor("#112233", 2, 5)  // column 2, front at 5: distance 3
	behind := sweepColor("#112233", 8, 5) // column 8, front at 5: distance 3
	if ahead != behind {
		t.Errorf("sweepColor is not symmetric around the leading edge: ahead=%q behind=%q", ahead, behind)
	}
}

// TestRenderSplashArtSweep_NoPanicAcrossFullRange is a smoke test across the
// animation's full progress range, proving every role rune still renders as
// exactly one blockGlyph and structural characters (newlines, spaces) are
// passed through unchanged, at both extremes and mid-sweep.
func TestRenderSplashArtSweep_NoPanicAcrossFullRange(t *testing.T) {
	roleRuneCount := 0
	for _, r := range splashArt {
		if _, ok := splashRoleColor[r]; ok {
			roleRuneCount++
		}
	}

	for _, progress := range []float64{0, 0.25, 0.5, 0.75, 1} {
		out := renderSplashArtSweep(progress)
		if got := strings.Count(out, blockGlyph); got != roleRuneCount {
			t.Errorf("renderSplashArtSweep(%v) contains %d block glyphs, want %d (one per role rune)", progress, got, roleRuneCount)
		}
		if wantLines, gotLines := strings.Count(splashArt, "\n"), strings.Count(out, "\n"); gotLines != wantLines {
			t.Errorf("renderSplashArtSweep(%v) has %d newlines, want %d (matching splashArt's line count)", progress, gotLines, wantLines)
		}
	}
}
