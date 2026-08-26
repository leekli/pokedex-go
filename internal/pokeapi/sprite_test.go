package pokeapi

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fixturePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{R: 255, G: 216, B: 0, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode fixture PNG: %v", err)
	}
	return buf.Bytes()
}

// TestFetchSprite_Success proves the happy path decodes a served PNG into a
// usable image.Image, going through FetchSprite's own request rather than
// Client.get (sprites are served from a different host than the API).
func TestFetchSprite_Success(t *testing.T) {
	sprite := fixturePNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(sprite)
	}))
	t.Cleanup(server.Close)

	client := NewClient(WithBaseURL(server.URL))
	img, err := client.FetchSprite(context.Background(), server.URL+"/sprites/pikachu.png")
	if err != nil {
		t.Fatalf("FetchSprite returned error: %v", err)
	}
	if img == nil {
		t.Fatal("FetchSprite returned a nil image on success")
	}
	bounds := img.Bounds()
	if bounds.Dx() != 2 || bounds.Dy() != 2 {
		t.Errorf("FetchSprite decoded image size = %dx%d, want 2x2", bounds.Dx(), bounds.Dy())
	}
}

// TestFetchSprite_NonOKStatus proves a non-200 response (e.g. a broken
// sprite link) is classified as a ServiceError, never the user's mistake -
// see the doc comment on FetchSprite.
func TestFetchSprite_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := NewClient()
	_, err := client.FetchSprite(context.Background(), server.URL+"/sprites/pikachu.png")

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("FetchSprite error = %v (%T), want *ServiceError", err, err)
	}
}

// TestFetchSprite_MalformedImage proves a 200 response whose body isn't a
// decodable image is still a ServiceError, not a panic or a silent nil.
func TestFetchSprite_MalformedImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not an image"))
	}))
	t.Cleanup(server.Close)

	client := NewClient()
	_, err := client.FetchSprite(context.Background(), server.URL+"/sprites/pikachu.png")

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("FetchSprite error = %v (%T), want *ServiceError on undecodable body", err, err)
	}
}

// TestFetchSprite_TransportError proves a request that never reaches a
// server (connection refused) is also a ServiceError.
func TestFetchSprite_TransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := server.URL + "/sprites/pikachu.png"
	server.Close() // close immediately so the URL is now unreachable

	client := NewClient()
	_, err := client.FetchSprite(context.Background(), deadURL)

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("FetchSprite error = %v (%T), want *ServiceError on transport failure", err, err)
	}
}

// TestFetchSprite_CachesAcrossCalls proves a second FetchSprite call for the
// same URL returns the exact decoded image already cached, rather than
// re-downloading and re-decoding it.
func TestFetchSprite_CachesAcrossCalls(t *testing.T) {
	hits := 0
	sprite := fixturePNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(sprite)
	}))
	t.Cleanup(server.Close)

	client := NewClient(WithBaseURL(server.URL))
	url := server.URL + "/sprites/pikachu.png"

	first, err := client.FetchSprite(context.Background(), url)
	if err != nil {
		t.Fatalf("first FetchSprite returned error: %v", err)
	}
	second, err := client.FetchSprite(context.Background(), url)
	if err != nil {
		t.Fatalf("second FetchSprite returned error: %v", err)
	}

	if hits != 1 {
		t.Errorf("sprite server was hit %d times, want 1 (second call should be served from cache)", hits)
	}
	if second != first {
		t.Error("cached FetchSprite returned a different image value, want the exact same decoded image reused")
	}
}

// TestFetchSprite_InvalidURL proves a malformed URL (fails at
// http.NewRequestWithContext, before any network I/O) is still surfaced as
// a ServiceError rather than a raw url.Parse error escaping the client.
func TestFetchSprite_InvalidURL(t *testing.T) {
	client := NewClient()
	_, err := client.FetchSprite(context.Background(), "http://\x7f.invalid/sprite.png")

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("FetchSprite error = %v (%T), want *ServiceError on invalid URL", err, err)
	}
}
