package pokeapi

import "strings"

// pokedexEntryVersionPriority orders which game version's flavor text is
// preferred when several are available in English, favoring Generation I
// versions to match the app's Gen-1 Pokédex styling elsewhere (e.g. imperial
// height/weight units) - see
// docs/adr/0003-prefer-generation-1-pokedex-entry-version.md. A Pokémon
// introduced after Generation I (National Dex Number > 151) has no
// red/blue/yellow entry, so SelectPokedexEntry always falls through to the
// first English entry found for those.
var pokedexEntryVersionPriority = []string{"red", "blue", "yellow"} //nolint:goconst // mirrors evolutionDetailVersionGroupPriority at a different granularity; can't share a constant, see evolution.go

// SelectPokedexEntry picks one English-language Pokédex Entry out of a
// species' flavor text entries (there's typically one per game version the
// Pokémon appeared in), preferring a Generation I version, and cleans up
// the original games' embedded line-break artifacts. Returns "" if entries
// has no English entry at all.
func SelectPokedexEntry(entries []FlavorTextEntry) string {
	var english []FlavorTextEntry
	for _, e := range entries {
		if e.Language == "en" {
			english = append(english, e)
		}
	}
	if len(english) == 0 {
		return ""
	}

	for _, preferred := range pokedexEntryVersionPriority {
		for _, e := range english {
			if e.Version == preferred {
				return cleanPokedexEntryText(e.Text)
			}
		}
	}
	return cleanPokedexEntryText(english[0].Text)
}

// cleanPokedexEntryText replaces PokeAPI's embedded line-break artifacts
// (literal \n and \f characters, carried over from the original games'
// fixed-width text boxes) with spaces, and collapses the resulting runs of
// whitespace down to one, so those artifacts never show up mid-sentence in
// the rendered text.
func cleanPokedexEntryText(text string) string {
	replaced := strings.NewReplacer("\n", " ", "\f", " ").Replace(text)
	return strings.Join(strings.Fields(replaced), " ")
}
