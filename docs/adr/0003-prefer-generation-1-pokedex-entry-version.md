# Prefer a Generation I game version for the Pokédex Entry

PokeAPI's `pokemon-species` resource returns one `flavor_text_entries` item per language *and* per game version a Pokémon appeared in — often a dozen or more English entries for an older Pokémon (one per Game Boy/DS/Switch title it's featured in), rather than a single canonical description. The Result Screen needs to settle on exactly one to show as the Pokédex Entry (see CONTEXT.md).

We chose to prefer whichever of Red, Blue, or Yellow is available, in that order, falling back to the first English entry found only when none of the three exist — which is always the case for Pokémon introduced after Generation I (National Dex Number > 151), since they simply have no Red/Blue/Yellow entry to prefer. This matches the app's existing Gen-1 Game Boy styling elsewhere (imperial height/weight units, the splash screen's Gen-1-era presentation) rather than showing whatever entry happens to sort first in PokeAPI's response, which PokeAPI does not document or guarantee to be stable or game-appropriate.

The three-version priority list is a small, curated set rather than a discovered one — the same tradeoff already made for `generationCount` (see generation.go) and the 18-type list: not worth a discovery request for something fixed and small.

## Considered options

- **Show whichever English entry PokeAPI returns first**: simplest, but PokeAPI doesn't document or guarantee entry ordering, so this could show a modern game's description (different tone, sometimes referencing modern mechanics) on an app styled entirely after Generation I; rejected for being inconsistent with the app's existing visual identity.
- **Let the user pick a version**: most flexible, but adds a UI control for a piece of flavor text nobody is likely to want to configure; rejected as unnecessary complexity for this app's scope.
