package pokeapi

import (
	"errors"
	"testing"
)

func TestLookupError_Message(t *testing.T) {
	err := &LookupError{Query: "notarealpokemon"}
	want := `no Pokémon found for "notarealpokemon"`
	if got := err.Error(); got != want {
		t.Errorf("LookupError.Error() = %q, want %q", got, want)
	}
}

func TestServiceError_MessageAndUnwrap(t *testing.T) {
	cause := errors.New("connection reset")
	err := &ServiceError{Err: cause}

	if got := err.Error(); got == "" {
		t.Error("ServiceError.Error() returned an empty message")
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is(ServiceError, cause) = false, want true (Unwrap must expose the wrapped cause)")
	}
}
