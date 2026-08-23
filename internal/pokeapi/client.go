// Package pokeapi is an HTTP client for the public PokeAPI
// (https://pokeapi.co/), plus the error types and JSON-to-domain mapping
// pokedex-go needs. It is the only layer in this app that performs network
// I/O for Pokémon data.
package pokeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultBaseURL = "https://pokeapi.co/api/v2"

// Client fetches Pokémon data from PokeAPI. The zero value is not usable;
// construct one with NewClient.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client constructed by NewClient.
type Option func(*Client)

// WithBaseURL overrides the PokeAPI base URL. Tests use this to point the
// Client at a local httptest.Server instead of the real network.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithHTTPClient overrides the *http.Client used for requests, e.g. to set a
// custom timeout.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient constructs a Client pointed at the real PokeAPI by default.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// get performs a GET request against path (relative to the Client's base
// URL) and decodes a 200 JSON response into out. Any failure is returned as
// either a *LookupError (HTTP 404) or a *ServiceError (everything else that
// went wrong: transport error, unexpected status, undecodable body).
func (c *Client) get(ctx context.Context, path, queryDescription string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return &ServiceError{Err: err}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &ServiceError{Err: err}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return &LookupError{Query: queryDescription}
	case resp.StatusCode != http.StatusOK:
		return &ServiceError{Err: fmt.Errorf("unexpected status %d from %s", resp.StatusCode, path)}
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &ServiceError{Err: fmt.Errorf("decoding response from %s: %w", path, err)}
	}
	return nil
}
