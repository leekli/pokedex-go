package pokeapi

import (
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
