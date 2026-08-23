package spriteart

import (
	"image"
	"image/color"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain pins lipgloss's color profile to TrueColor for the whole package,
// so Render's output is deterministic regardless of whether tests run in a
// real terminal or a CI pipe (which lipgloss would otherwise auto-detect as
// a lower/no-color profile).
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	m.Run()
}

func solidImage(w, h int, c color.Color) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestRender_FullyTransparentImageIsBlank(t *testing.T) {
	img := solidImage(2, 2, color.NRGBA{R: 10, G: 20, B: 30, A: 0})

	got := Render(img, Options{})

	if got != "  " {
		t.Errorf("Render(fully transparent) = %q, want %q (two blank spaces, no color codes)", got, "  ")
	}
}

func TestRender_BothPixelsVisible(t *testing.T) {
	red := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	blue := color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 1, 2))
	img.Set(0, 0, red)
	img.Set(0, 1, blue)

	got := Render(img, Options{})

	want := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Background(lipgloss.Color("#0000ff")).Render("▀")
	if got != want {
		t.Errorf("Render(red-over-blue) = %q, want %q", got, want)
	}
}

func TestRender_OnlyTopPixelVisible(t *testing.T) {
	green := color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 1, 2))
	img.Set(0, 0, green)
	img.Set(0, 1, color.NRGBA{A: 0})

	got := Render(img, Options{})

	want := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Render("▀")
	if got != want {
		t.Errorf("Render(top-only) = %q, want %q", got, want)
	}
}

func TestRender_OnlyBottomPixelVisible(t *testing.T) {
	yellow := color.NRGBA{R: 255, G: 255, B: 0, A: 255}
	img := image.NewNRGBA(image.Rect(0, 0, 1, 2))
	img.Set(0, 0, color.NRGBA{A: 0})
	img.Set(0, 1, yellow)

	got := Render(img, Options{})

	want := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Render("▄")
	if got != want {
		t.Errorf("Render(bottom-only) = %q, want %q", got, want)
	}
}

func TestRender_OddHeightImageTreatsMissingRowAsTransparent(t *testing.T) {
	// Height 3: row pair (0,1) is a normal cell, row pair (2,3) has no
	// source row 3 at all - it must be treated as transparent, not panic
	// or read out of bounds.
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	img := solidImage(1, 3, white)

	got := Render(img, Options{})

	rowWithBothPixels := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#ffffff")).Render("▀")
	rowWithOnlyTop := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render("▀")
	want := rowWithBothPixels + "\n" + rowWithOnlyTop

	if got != want {
		t.Errorf("Render(odd height) = %q, want %q", got, want)
	}
}

func TestRender_ZeroSizedImageIsEmpty(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 0, 0))
	if got := Render(img, Options{}); got != "" {
		t.Errorf("Render(zero-sized image) = %q, want empty string", got)
	}
}

func TestRender_MaxWidthDownscales(t *testing.T) {
	// 4x4 so that scale=2 (forced by MaxWidth=2) still has a real source row
	// to sample for both the top and bottom of each output cell.
	red := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	img := solidImage(4, 4, red)

	got := Render(img, Options{MaxWidth: 2})

	wantCell := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Background(lipgloss.Color("#ff0000")).Render("▀")
	want := wantCell + wantCell
	if got != want {
		t.Errorf("Render(4-wide, MaxWidth=2) = %q, want 2 cells %q", got, want)
	}
}
