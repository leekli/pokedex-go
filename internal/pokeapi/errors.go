package pokeapi

import "fmt"

// LookupError means the requested name or National Dex Number didn't
// resolve to a Pokémon — a bad or unknown input, the user's mistake. It
// corresponds to an HTTP 404 from PokeAPI.
type LookupError struct {
	Query string
}

func (e *LookupError) Error() string {
	return fmt.Sprintf("no Pokémon found for %q", e.Query)
}

// ServiceError means PokeAPI itself was unreachable or failed — a network
// error, timeout, unexpected HTTP status, or an undecodable response body.
// Not the user's fault; retrying may help.
type ServiceError struct {
	Err error
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("PokeAPI request failed: %v", e.Err)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}
