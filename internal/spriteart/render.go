// Package spriteart renders a decoded image as colored terminal text, using
// lipgloss so the same color-profile detection/degradation used elsewhere in
// the app also governs sprite rendering.
package spriteart

import (
	"image"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// alphaVisibleThreshold is the minimum alpha (out of 0xFFFF, per Go's
// color.Color convention) a pixel needs before it's rendered as a colored
// block rather than a blank space. Keeps transparent sprite backgrounds
// from rendering as a solid black square.
const alphaVisibleThreshold = 0x8000

// Options configures Render.
type Options struct {
	// MaxWidth caps the rendered width in terminal columns. If the source
	// image is wider than MaxWidth, it's downscaled with nearest-neighbor
	// sampling (keeping the blocky, retro pixel-art look). Zero means no cap.
	MaxWidth int
}

// Render converts img into a string of ANSI-colored half-block characters,
// two source pixels tall per line of output (rendered as '▀', foreground =
// top pixel, background = bottom pixel). img may be nil-checked by the
// caller before calling Render; Render itself assumes a non-nil image.
func Render(img image.Image, opts Options) string {
	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW == 0 || srcH == 0 {
		return ""
	}

	scale := 1
	if opts.MaxWidth > 0 && srcW > opts.MaxWidth {
		scale = (srcW + opts.MaxWidth - 1) / opts.MaxWidth
	}

	var out strings.Builder
	for y := 0; y < srcH; y += 2 * scale {
		for x := 0; x < srcW; x += scale {
			top := samplePixel(img, bounds, x, y)
			bottom := samplePixel(img, bounds, x, y+scale)
			out.WriteString(renderCell(top, bottom))
		}
		out.WriteByte('\n')
	}
	return strings.TrimSuffix(out.String(), "\n")
}

type rgba struct {
	r, g, b uint8
	visible bool
}

// samplePixel reads the pixel at (x, y) relative to bounds.Min, treating any
// coordinate at or beyond the image's height as fully transparent — this is
// how an odd-height source image's missing "bottom" row on its last output
// line is handled, without a separate half-row special case.
func samplePixel(img image.Image, bounds image.Rectangle, x, y int) rgba {
	if y >= bounds.Dy() {
		return rgba{}
	}
	r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
	if a < alphaVisibleThreshold {
		return rgba{}
	}
	return rgba{r: uint8(r >> 8), g: uint8(g >> 8), b: uint8(b >> 8), visible: true}
}

func renderCell(top, bottom rgba) string {
	if !top.visible && !bottom.visible {
		return " "
	}

	style := lipgloss.NewStyle()
	if top.visible {
		style = style.Foreground(lipgloss.Color(hex(top)))
	}
	if bottom.visible {
		style = style.Background(lipgloss.Color(hex(bottom)))
	}

	if !top.visible {
		// No top pixel to draw: render a full block in the background color
		// instead of an invisible foreground glyph over a colored background.
		return lipgloss.NewStyle().Foreground(lipgloss.Color(hex(bottom))).Render("▄")
	}
	return style.Render("▀")
}

func hex(c rgba) string {
	const hexDigits = "0123456789abcdef"
	b := [7]byte{'#'}
	for i, v := range [3]uint8{c.r, c.g, c.b} {
		b[1+i*2] = hexDigits[v>>4]
		b[2+i*2] = hexDigits[v&0xF]
	}
	return string(b[:])
}
