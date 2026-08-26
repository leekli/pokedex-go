package pokeapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
)

// generationFixtureJSON renders a minimal /generation/{id} response naming
// one species per generation, keyed by dex number so tests can assert
// precisely which generation each id landed in.
func generationFixtureJSON(name string, speciesID int, speciesName string) string {
	return `{"name": "` + name + `", "pokemon_species": [{"name": "` + speciesName + `", "url": "https://pokeapi.co/api/v2/pokemon-species/` + strconv.Itoa(speciesID) + `/"}]}`
}

func TestGetGenerationIndex_BuildsFullIndex(t *testing.T) {
	mux := http.NewServeMux()
	for i := 1; i <= generationCount; i++ {
		i := i
		mux.HandleFunc("/generation/"+strconv.Itoa(i), func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			// dex id = generation number, so each generation's one entry is
			// trivially distinguishable in the assertions below.
			w.Write([]byte(generationFixtureJSON("generation-"+strconv.Itoa(i), i, "species-"+strconv.Itoa(i))))
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	got := client.GetGenerationIndex(context.Background())

	if len(got) != generationCount {
		t.Fatalf("GetGenerationIndex returned %d entries, want %d: %+v", len(got), generationCount, got)
	}
	for i := 1; i <= generationCount; i++ {
		want := "generation-" + strconv.Itoa(i)
		if got[i] != want {
			t.Errorf("index[%d] = %q, want %q", i, got[i], want)
		}
	}
}

func TestGetGenerationIndex_PartialFailureStillReturnsWhatSucceeded(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/generation/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	for i := 2; i <= generationCount; i++ {
		i := i
		mux.HandleFunc("/generation/"+strconv.Itoa(i), func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(generationFixtureJSON("generation-"+strconv.Itoa(i), i, "species-"+strconv.Itoa(i))))
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	got := client.GetGenerationIndex(context.Background())

	if _, ok := got[1]; ok {
		t.Error("index contains dex id 1, whose generation fetch failed; want it simply absent")
	}
	if len(got) != generationCount-1 {
		t.Errorf("GetGenerationIndex returned %d entries, want %d (one generation failed, rest should still be present)", len(got), generationCount-1)
	}
}

// TestGetGenerationIndex_SkipsUnparsableSpeciesURL proves a malformed
// species URL within an otherwise-successful generation response is simply
// skipped, rather than corrupting the index or panicking.
func TestGetGenerationIndex_SkipsUnparsableSpeciesURL(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/generation/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "generation-i", "pokemon_species": [
			{"name": "bad", "url": "https://pokeapi.co/api/v2/pokemon-species/not-a-number/"},
			{"name": "bulbasaur", "url": "https://pokeapi.co/api/v2/pokemon-species/1/"}
		]}`))
	})
	for i := 2; i <= generationCount; i++ {
		mux.HandleFunc("/generation/"+strconv.Itoa(i), func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	got := client.GetGenerationIndex(context.Background())

	if len(got) != 1 {
		t.Fatalf("GetGenerationIndex = %+v, want exactly one entry (the unparsable URL should be skipped)", got)
	}
	if got[1] != "generation-i" {
		t.Errorf("index[1] = %q, want %q", got[1], "generation-i")
	}
}

// TestGetGenerationIndex_CachesAcrossCalls proves a second call is served
// from the Client's cache: none of the 9 generation resources get
// re-fetched, matching GetGenerationIndex's doc comment on being safe to
// call on every Type Roster load.
func TestGetGenerationIndex_CachesAcrossCalls(t *testing.T) {
	var hits atomic.Int32
	mux := http.NewServeMux()
	for i := 1; i <= generationCount; i++ {
		i := i
		mux.HandleFunc("/generation/"+strconv.Itoa(i), func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(generationFixtureJSON("generation-"+strconv.Itoa(i), i, "species-"+strconv.Itoa(i))))
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	first := client.GetGenerationIndex(context.Background())
	second := client.GetGenerationIndex(context.Background())

	if got := hits.Load(); got != generationCount {
		t.Errorf("PokeAPI was hit %d times, want %d (second call should be served entirely from cache)", got, generationCount)
	}
	if len(second) != len(first) {
		t.Errorf("cached GetGenerationIndex returned %d entries, want %d", len(second), len(first))
	}
}

func TestGetGenerationIndex_TotalFailureReturnsEmptyMapNotNilPanic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	client := NewClient(WithBaseURL(server.URL))

	got := client.GetGenerationIndex(context.Background())

	if got == nil {
		t.Error("GetGenerationIndex returned a nil map on total failure, want a non-nil empty map")
	}
	if len(got) != 0 {
		t.Errorf("GetGenerationIndex on total failure returned %d entries, want 0", len(got))
	}
}
