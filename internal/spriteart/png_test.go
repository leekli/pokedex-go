package spriteart

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// TestRender_DecodedPNGRoundTrip exercises the full pipeline a real PokeAPI
// sprite goes through: PNG bytes -> image.Decode -> Render. The PNG is
// generated in-memory rather than fetched, so this test needs no network
// access and no checked-in binary fixture, while still proving PNG decoding
// (not just constructing image.NRGBA directly, as the other tests do)
// produces output Render can handle correctly.
func TestRender_DecodedPNGRoundTrip(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 1, 2))
	src.Set(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255}) // opaque red
	src.Set(0, 1, color.NRGBA{A: 0})                       // transparent

	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatalf("failed to encode fixture PNG: %v", err)
	}

	decoded, _, err := image.Decode(&buf)
	if err != nil {
		t.Fatalf("failed to decode fixture PNG: %v", err)
	}

	got := Render(decoded, Options{})

	if got == "" || got == " " {
		t.Errorf("Render(decoded PNG) = %q, want non-blank output for the visible red pixel", got)
	}
}
