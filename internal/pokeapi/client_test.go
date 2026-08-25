package pokeapi

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient()
	if c.baseURL != defaultBaseURL {
		t.Errorf("default baseURL = %q, want %q", c.baseURL, defaultBaseURL)
	}
	if c.httpClient == nil {
		t.Fatal("default httpClient is nil")
	}
}

func TestWithBaseURL_Overrides(t *testing.T) {
	c := NewClient(WithBaseURL("http://example.invalid"))
	if c.baseURL != "http://example.invalid" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "http://example.invalid")
	}
}

func TestWithHTTPClient_Overrides(t *testing.T) {
	custom := &http.Client{Timeout: 42 * time.Second}
	c := NewClient(WithHTTPClient(custom))
	if c.httpClient != custom {
		t.Error("WithHTTPClient did not set the provided *http.Client on the Client")
	}
}

// TestGet_InvalidURL proves a Client whose baseURL can't form a valid
// request (fails at http.NewRequestWithContext, before any network I/O) is
// still surfaced as a ServiceError rather than a raw url.Parse error
// escaping the client.
func TestGet_InvalidURL(t *testing.T) {
	c := NewClient(WithBaseURL("http://\x7f.invalid"))

	_, err := c.GetPokemon(context.Background(), "pikachu")

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("GetPokemon error = %v (%T), want *ServiceError on invalid base URL", err, err)
	}
}
