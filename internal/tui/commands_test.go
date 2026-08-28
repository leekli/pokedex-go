package tui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leekli/pokedex-go/internal/pokeapi"
	"github.com/leekli/pokedex-go/internal/pokemon"
)

func commandsTestFixturePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{R: 255, G: 216, B: 0, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode fixture PNG: %v", err)
	}
	return buf.Bytes()
}

const lookupCmdPikachuJSONTemplate = `{
	"id": 25,
	"name": "pikachu",
	"height": 4,
	"weight": 60,
	"types": [{"type": {"name": "electric"}}],
	"stats": [{"base_stat": 35, "stat": {"name": "hp"}}],
	"sprites": {"front_default": %s}
}`

// newLookupCmdServer builds a fixture PokeAPI server serving pikachu, whose
// front_default sprite either points back at the server's own
// /sprites/pikachu.png route (includeSprite true) or is absent entirely
// (includeSprite false). The sprite route itself responds with
// spriteStatus, so callers can exercise a working sprite fetch, a failing
// one, or no sprite URL at all.
func newLookupCmdServer(t *testing.T, includeSprite bool, spriteStatus int) *pokeapi.Client {
	t.Helper()
	var baseURL string

	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon/pikachu", func(w http.ResponseWriter, r *http.Request) {
		spriteJSON := "null"
		if includeSprite {
			spriteJSON = fmt.Sprintf(`"%s/sprites/pikachu.png"`, baseURL)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, lookupCmdPikachuJSONTemplate, spriteJSON)
	})
	mux.HandleFunc("/sprites/pikachu.png", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(spriteStatus)
		if spriteStatus == http.StatusOK {
			w.Write(commandsTestFixturePNG(t))
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	baseURL = server.URL
	return pokeapi.NewClient(pokeapi.WithBaseURL(server.URL))
}

// runLookupCmd resolves a Name query for "pikachu" against client and
// returns the resulting message.
func runLookupCmd(t *testing.T, client *pokeapi.Client) lookupResultMsg {
	t.Helper()
	cmd := lookupCmd(client, pokemon.Query{Kind: pokemon.Name, Value: "pikachu"})
	msg, ok := cmd().(lookupResultMsg)
	if !ok {
		t.Fatalf("lookupCmd() produced %T, want lookupResultMsg", cmd())
	}
	return msg
}

// TestLookupCmd_Success proves the straightforward path: a successful
// PokeAPI lookup with a working sprite produces a fully populated message.
func TestLookupCmd_Success(t *testing.T) {
	client := newLookupCmdServer(t, true, http.StatusOK)

	msg := runLookupCmd(t, client)

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil", msg.err)
	}
	if msg.stat.Name != "pikachu" || msg.stat.DexNumber != 25 {
		t.Errorf("lookupResultMsg.stat = %+v, want pikachu #25", msg.stat)
	}
	if msg.sprite == nil {
		t.Error("lookupResultMsg.sprite is nil, want a decoded image on a successful sprite fetch")
	}
}

// TestLookupCmd_SpriteFetchFailureStillSucceeds proves the documented
// behavior in commands.go: a failed sprite fetch does not fail the overall
// lookup, it just leaves lookupResultMsg.sprite nil so the Result Screen
// falls back to its "no sprite" message.
func TestLookupCmd_SpriteFetchFailureStillSucceeds(t *testing.T) {
	client := newLookupCmdServer(t, true, http.StatusInternalServerError)

	msg := runLookupCmd(t, client)

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil (sprite failure must not fail the lookup)", msg.err)
	}
	if msg.sprite != nil {
		t.Error("lookupResultMsg.sprite is non-nil, want nil after a failed sprite fetch")
	}
	if msg.stat.Name != "pikachu" {
		t.Errorf("lookupResultMsg.stat.Name = %q, want pikachu", msg.stat.Name)
	}
}

// TestLookupCmd_NoSpriteURL proves a Pokémon with no front_default sprite at
// all skips the sprite fetch entirely and still succeeds.
func TestLookupCmd_NoSpriteURL(t *testing.T) {
	client := newLookupCmdServer(t, false, http.StatusOK)

	msg := runLookupCmd(t, client)

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil", msg.err)
	}
	if msg.sprite != nil {
		t.Error("lookupResultMsg.sprite is non-nil, want nil when PokeAPI returned no sprite URL")
	}
}

// lookupCmdPikachuFrontBackJSONTemplate is lookupCmdPikachuJSONTemplate with
// an added back_default slot, used only by the back-sprite tests below - the
// front-sprite tests above don't need a back_default at all.
const lookupCmdPikachuFrontBackJSONTemplate = `{
	"id": 25,
	"name": "pikachu",
	"height": 4,
	"weight": 60,
	"types": [{"type": {"name": "electric"}}],
	"stats": [{"base_stat": 35, "stat": {"name": "hp"}}],
	"sprites": {"front_default": null, "back_default": %s}
}`

// newLookupCmdBackSpriteServer builds a fixture PokeAPI server serving
// pikachu with a back_default sprite (and no front_default), whose route
// responds with spriteStatus - the back-sprite equivalent of
// newLookupCmdServer.
func newLookupCmdBackSpriteServer(t *testing.T, spriteStatus int) *pokeapi.Client {
	t.Helper()
	var baseURL string

	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon/pikachu", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, lookupCmdPikachuFrontBackJSONTemplate, fmt.Sprintf(`"%s/sprites/pikachu-back.png"`, baseURL))
	})
	mux.HandleFunc("/sprites/pikachu-back.png", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(spriteStatus)
		if spriteStatus == http.StatusOK {
			w.Write(commandsTestFixturePNG(t))
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	baseURL = server.URL
	return pokeapi.NewClient(pokeapi.WithBaseURL(server.URL))
}

// TestLookupCmd_BackSprite_Success proves a working back_default sprite is
// fetched and decoded into lookupResultMsg.spriteBack, the same way the
// front sprite is - see TestLookupCmd_Success.
func TestLookupCmd_BackSprite_Success(t *testing.T) {
	client := newLookupCmdBackSpriteServer(t, http.StatusOK)

	msg := runLookupCmd(t, client)

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil", msg.err)
	}
	if msg.spriteBack == nil {
		t.Error("lookupResultMsg.spriteBack is nil, want a decoded image on a successful back sprite fetch")
	}
	if msg.sprite != nil {
		t.Error("lookupResultMsg.sprite is non-nil, want nil (fixture has no front_default)")
	}
}

// TestLookupCmd_BackSpriteFetchFailureStillSucceeds proves a failed
// back_default fetch doesn't fail the overall lookup, leaving spriteBack nil
// - the back-sprite equivalent of TestLookupCmd_SpriteFetchFailureStillSucceeds.
func TestLookupCmd_BackSpriteFetchFailureStillSucceeds(t *testing.T) {
	client := newLookupCmdBackSpriteServer(t, http.StatusInternalServerError)

	msg := runLookupCmd(t, client)

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil (back sprite failure must not fail the lookup)", msg.err)
	}
	if msg.spriteBack != nil {
		t.Error("lookupResultMsg.spriteBack is non-nil, want nil after a failed back sprite fetch")
	}
}

// TestLookupCmd_NoBackSpriteURL proves a Pokémon with no back_default sprite
// at all skips the back sprite fetch entirely and still succeeds - the
// back-sprite equivalent of TestLookupCmd_NoSpriteURL.
func TestLookupCmd_NoBackSpriteURL(t *testing.T) {
	client := newLookupCmdServer(t, false, http.StatusOK)

	msg := runLookupCmd(t, client)

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil", msg.err)
	}
	if msg.spriteBack != nil {
		t.Error("lookupResultMsg.spriteBack is non-nil, want nil when PokeAPI returned no back_default sprite URL")
	}
}

const lookupCmdPikachuSpeciesJSON = `{
	"id": 25,
	"name": "pikachu",
	"varieties": [{"is_default": true, "pokemon": {"name": "pikachu"}}],
	"flavor_text_entries": [
		{"flavor_text": "A strange melody plays.", "language": {"name": "en"}, "version": {"name": "red"}}
	]
}`

// TestLookupCmd_IncludesPokedexEntry proves a successful lookup also
// populates lookupResultMsg.pokedexEntry, from a second, best-effort
// GetSpecies call keyed by the query value - see docs/adr/0003-prefer-
// generation-1-pokedex-entry-version.md.
func TestLookupCmd_IncludesPokedexEntry(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon/pikachu", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, lookupCmdPikachuJSONTemplate, "null")
	})
	mux.HandleFunc("/pokemon-species/pikachu", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(lookupCmdPikachuSpeciesJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := pokeapi.NewClient(pokeapi.WithBaseURL(server.URL))

	msg := runLookupCmd(t, client)

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil", msg.err)
	}
	if msg.pokedexEntry != "A strange melody plays." {
		t.Errorf("lookupResultMsg.pokedexEntry = %q, want %q", msg.pokedexEntry, "A strange melody plays.")
	}
}

// TestLookupCmd_PokedexEntryFetchFailureStillSucceeds proves a failed
// species fetch doesn't fail the overall lookup, the same way a failed
// sprite fetch doesn't (see TestLookupCmd_SpriteFetchFailureStillSucceeds).
func TestLookupCmd_PokedexEntryFetchFailureStillSucceeds(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon/pikachu", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, lookupCmdPikachuJSONTemplate, "null")
	})
	mux.HandleFunc("/pokemon-species/pikachu", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := pokeapi.NewClient(pokeapi.WithBaseURL(server.URL))

	msg := runLookupCmd(t, client)

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil (a failed Pokédex Entry fetch must not fail the lookup)", msg.err)
	}
	if msg.pokedexEntry != "" {
		t.Errorf("lookupResultMsg.pokedexEntry = %q, want empty after a failed species fetch", msg.pokedexEntry)
	}
}

// TestLookupCmd_DexNumberQueryReusesSpeciesCacheForPokedexEntry proves a
// DexNumber query's Pokédex Entry fetch is served from the cache Lookup
// itself already populated (Lookup resolves a DexNumber query via
// GetSpecies under this same key), rather than hitting
// /pokemon-species/{id} a second time - the caching payoff described in
// lookupCmd's doc comment.
func TestLookupCmd_DexNumberQueryReusesSpeciesCacheForPokedexEntry(t *testing.T) {
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon-species/25", func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(lookupCmdPikachuSpeciesJSON))
	})
	mux.HandleFunc("/pokemon/pikachu", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, lookupCmdPikachuJSONTemplate, "null")
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := pokeapi.NewClient(pokeapi.WithBaseURL(server.URL))

	cmd := lookupCmd(client, pokemon.Query{Kind: pokemon.DexNumber, Value: "25"})
	msg, ok := cmd().(lookupResultMsg)
	if !ok {
		t.Fatalf("lookupCmd() produced %T, want lookupResultMsg", cmd())
	}

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil", msg.err)
	}
	if msg.pokedexEntry != "A strange melody plays." {
		t.Errorf("lookupResultMsg.pokedexEntry = %q, want %q", msg.pokedexEntry, "A strange melody plays.")
	}
	if hits != 1 {
		t.Errorf("/pokemon-species/25 was hit %d times, want 1 (the Pokédex Entry fetch should reuse Lookup's own cached species call)", hits)
	}
}

// lookupCmdElectricTypeJSON is Electric's real type chart (weak to Ground;
// resists Electric, Flying, Steel), used to prove lookupCmd's single-type
// (Pikachu) path.
const lookupCmdElectricTypeJSON = `{
	"pokemon": [],
	"damage_relations": {
		"double_damage_from": [{"name": "ground"}],
		"half_damage_from": [{"name": "electric"}, {"name": "flying"}, {"name": "steel"}],
		"no_damage_from": []
	}
}`

// TestLookupCmd_IncludesTypeEffectiveness proves a successful lookup
// populates lookupResultMsg.typeEffectiveness from Pikachu's single
// (Electric) type.
func TestLookupCmd_IncludesTypeEffectiveness(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon/pikachu", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, lookupCmdPikachuJSONTemplate, "null")
	})
	mux.HandleFunc("/type/electric", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(lookupCmdElectricTypeJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := pokeapi.NewClient(pokeapi.WithBaseURL(server.URL))

	msg := runLookupCmd(t, client)

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil", msg.err)
	}
	if msg.typeEffectiveness == nil {
		t.Fatal("lookupResultMsg.typeEffectiveness = nil, want a populated result")
	}
	if len(msg.typeEffectiveness.Weaknesses) != 1 || msg.typeEffectiveness.Weaknesses[0].Type != "ground" {
		t.Errorf("Weaknesses = %+v, want [{ground 2}]", msg.typeEffectiveness.Weaknesses)
	}
}

// TestLookupCmd_TypeEffectivenessUnavailableOnFetchFailure proves a failed
// damage-relations fetch leaves typeEffectiveness nil - unlike the sprite
// and Pokédex Entry, this is not best-effort: see
// docs/adr/0004-all-or-nothing-type-effectiveness.md.
func TestLookupCmd_TypeEffectivenessUnavailableOnFetchFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon/pikachu", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, lookupCmdPikachuJSONTemplate, "null")
	})
	mux.HandleFunc("/type/electric", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := pokeapi.NewClient(pokeapi.WithBaseURL(server.URL))

	msg := runLookupCmd(t, client)

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil (a failed type effectiveness fetch must not fail the lookup)", msg.err)
	}
	if msg.typeEffectiveness != nil {
		t.Errorf("lookupResultMsg.typeEffectiveness = %+v, want nil after a failed damage relations fetch", msg.typeEffectiveness)
	}
}

// lookupCmdCharizardJSONTemplate is a dual-type (Fire/Flying) fixture, used
// only to prove the all-or-nothing behavior across two types - see
// TestLookupCmd_DualTypePartialFailureOmitsEffectivenessEntirely.
const lookupCmdCharizardJSONTemplate = `{
	"id": 6,
	"name": "charizard",
	"height": 17,
	"weight": 905,
	"types": [{"type": {"name": "fire"}}, {"type": {"name": "flying"}}],
	"stats": [{"base_stat": 78, "stat": {"name": "hp"}}],
	"sprites": {"front_default": null}
}`

// lookupCmdFireTypeJSON is Fire's real type chart, needed alongside
// lookupCmdCharizardJSONTemplate below (this package can't reach the
// pokeapi package's own equivalent test fixture).
const lookupCmdFireTypeJSON = `{
	"pokemon": [],
	"damage_relations": {
		"double_damage_from": [{"name": "water"}, {"name": "ground"}, {"name": "rock"}],
		"half_damage_from": [{"name": "fire"}, {"name": "grass"}, {"name": "ice"}, {"name": "bug"}, {"name": "steel"}, {"name": "fairy"}],
		"no_damage_from": []
	}
}`

// TestLookupCmd_DualTypePartialFailureOmitsEffectivenessEntirely proves
// that when only one of a dual-type Pokémon's two damage-relations fetches
// fails, the whole result is unavailable rather than a partial chart built
// from just the type that succeeded - see
// docs/adr/0004-all-or-nothing-type-effectiveness.md.
func TestLookupCmd_DualTypePartialFailureOmitsEffectivenessEntirely(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon/charizard", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(lookupCmdCharizardJSONTemplate))
	})
	mux.HandleFunc("/type/fire", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(lookupCmdFireTypeJSON))
	})
	mux.HandleFunc("/type/flying", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := pokeapi.NewClient(pokeapi.WithBaseURL(server.URL))

	cmd := lookupCmd(client, pokemon.Query{Kind: pokemon.Name, Value: "charizard"})
	msg, ok := cmd().(lookupResultMsg)
	if !ok {
		t.Fatalf("lookupCmd() produced %T, want lookupResultMsg", cmd())
	}

	if msg.err != nil {
		t.Fatalf("lookupResultMsg.err = %v, want nil", msg.err)
	}
	if msg.typeEffectiveness != nil {
		t.Errorf("lookupResultMsg.typeEffectiveness = %+v, want nil (Fire succeeded but Flying failed, so nothing should be shown)", msg.typeEffectiveness)
	}
}

// TestLookupCmd_LookupError proves a failed PokeAPI lookup is surfaced as
// lookupResultMsg.err, unmodified, with no stat block.
func TestLookupCmd_LookupError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/pokemon/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := pokeapi.NewClient(pokeapi.WithBaseURL(server.URL))

	cmd := lookupCmd(client, pokemon.Query{Kind: pokemon.Name, Value: "notarealpokemon"})
	got, ok := cmd().(lookupResultMsg)
	if !ok {
		t.Fatalf("lookupCmd() produced %T, want lookupResultMsg", cmd())
	}

	var lookupErr *pokeapi.LookupError
	if !errors.As(got.err, &lookupErr) {
		t.Fatalf("lookupResultMsg.err = %v (%T), want *pokeapi.LookupError", got.err, got.err)
	}
}
