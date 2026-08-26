package pokeapi

import "testing"

func TestSelectPokedexEntry_PrefersRedOverBlueAndYellow(t *testing.T) {
	entries := []FlavorTextEntry{
		{Text: "yellow entry", Language: "en", Version: "yellow"},
		{Text: "blue entry", Language: "en", Version: "blue"},
		{Text: "red entry", Language: "en", Version: "red"},
	}
	if got := SelectPokedexEntry(entries); got != "red entry" {
		t.Errorf("SelectPokedexEntry = %q, want the red entry preferred", got)
	}
}

func TestSelectPokedexEntry_FallsBackToBlueWhenNoRed(t *testing.T) {
	entries := []FlavorTextEntry{
		{Text: "yellow entry", Language: "en", Version: "yellow"},
		{Text: "blue entry", Language: "en", Version: "blue"},
	}
	if got := SelectPokedexEntry(entries); got != "blue entry" {
		t.Errorf("SelectPokedexEntry = %q, want blue entry when red is absent", got)
	}
}

// TestSelectPokedexEntry_FallsBackToFirstEnglishEntry proves a Pokémon with
// no Generation I entry at all (introduced after Dex #151) still gets a
// description, from whichever English entry PokeAPI has.
func TestSelectPokedexEntry_FallsBackToFirstEnglishEntry(t *testing.T) {
	entries := []FlavorTextEntry{
		{Text: "modern entry", Language: "en", Version: "scarlet"},
		{Text: "another modern entry", Language: "en", Version: "violet"},
	}
	if got := SelectPokedexEntry(entries); got != "modern entry" {
		t.Errorf("SelectPokedexEntry = %q, want the first English entry when no Gen I version exists", got)
	}
}

func TestSelectPokedexEntry_IgnoresNonEnglishEntries(t *testing.T) {
	entries := []FlavorTextEntry{
		{Text: "texte français", Language: "fr", Version: "red"},
	}
	if got := SelectPokedexEntry(entries); got != "" {
		t.Errorf("SelectPokedexEntry = %q, want empty when no English entry exists", got)
	}
}

func TestSelectPokedexEntry_NoEntriesReturnsEmpty(t *testing.T) {
	if got := SelectPokedexEntry(nil); got != "" {
		t.Errorf("SelectPokedexEntry(nil) = %q, want empty", got)
	}
}

// TestSelectPokedexEntry_CleansEmbeddedLineBreakArtifacts proves the
// original games' literal \n / \f line-break characters (PokeAPI passes
// these through verbatim) are replaced with spaces rather than showing up
// mid-sentence in the rendered text.
func TestSelectPokedexEntry_CleansEmbeddedLineBreakArtifacts(t *testing.T) {
	entries := []FlavorTextEntry{
		{
			Text:     "When several of\nthese Pokémon gather,\ntheir electricity could\nbuild and cause lightning\nstorms.\f",
			Language: "en",
			Version:  "red",
		},
	}
	want := "When several of these Pokémon gather, their electricity could build and cause lightning storms."
	if got := SelectPokedexEntry(entries); got != want {
		t.Errorf("SelectPokedexEntry = %q, want %q", got, want)
	}
}

func TestSelectPokedexEntry_CollapsesRepeatedWhitespace(t *testing.T) {
	entries := []FlavorTextEntry{
		{Text: "Extra   spaces\n\nhere.", Language: "en", Version: "red"},
	}
	want := "Extra spaces here."
	if got := SelectPokedexEntry(entries); got != want {
		t.Errorf("SelectPokedexEntry = %q, want %q", got, want)
	}
}
