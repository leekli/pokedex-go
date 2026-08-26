# Pokedex Go

A terminal (TUI) application for looking up a Pokémon by name or National Dex number and viewing its sprite and stats, styled after the Pokédex screens in the Generation 1 Game Boy games. Built in Go against the public PokeAPI (https://pokeapi.co/).

## Language

**Splash Screen**:
The first screen shown on launch: a static ASCII/ANSI-art rendition of the Pokémon logo (hand-crafted, embedded in the binary — not a copy of the real trademarked artwork) with a "Press Enter to search for Pokémon" prompt beneath it. Accepts Enter (advance to Search Screen), or Q / Esc / Ctrl+C (quit).
_Avoid_: Splash page, title screen, home screen, logo screen

**Search Screen**:
The screen where the user types a Pokémon name or National Dex Number to look up. Input is normalized before lookup (case-folded, trimmed, spaces/punctuation converted to match PokeAPI's slug format, e.g. "Mr Mime" → `mr-mime`). A failed lookup shows a Lookup Error inline on this same screen rather than navigating away. Esc returns to the Splash Screen; Q/Ctrl+C quit.
_Avoid_: Input screen, lookup screen

**Result Screen**:
The screen shown after a successful lookup: the Pokémon's Sprite plus its Stat Block. Enter always returns to the Search Screen to look up another Pokémon; Esc returns to whichever screen led here instead — the Search Screen, or the Type Roster Screen if this Pokémon was reached from there. Q/Ctrl+C quit.
_Avoid_: Details screen, info screen

**Lookup Error**:
The inline message shown on the Search Screen when a name or National Dex Number doesn't resolve to a Pokémon (a bad/unknown input — the user's mistake). Distinct from a Service Error.
_Avoid_: Not-found screen, error screen

**Service Error**:
The inline message shown on the Search Screen when PokeAPI itself is unreachable or fails (timeout, 5xx) — not caused by bad input. Shown instead of a Lookup Error so the user knows retrying may help rather than that their input was wrong.
_Avoid_: Network error, API error

**National Dex Number**:
The number a player means when they say "Pokémon #250" — resolved via PokeAPI's `pokemon-species` resource. Distinct from a PokeAPI `pokemon` resource's internal id, which diverges from the National Dex Number for regional/alternate forms (e.g. Alolan Raichu). Numeric input on the Search Screen is always interpreted as a National Dex Number, then resolved to the species' base-form `pokemon` resource for Sprite and Stat Block data.
_Avoid_: Pokédex number, dex id, pokemon id

**Sprite**:
The Pokémon image shown on the Result Screen — specifically PokeAPI's `sprites.front_default` variant (small, Gen-accurate pixel art), chosen to match the Generation 1 Pokédex look rather than PokeAPI's `official-artwork` (modern box-art render), which is out of scope for this screen.
_Avoid_: Artwork, image, portrait

**Pokédex Entry**:
The descriptive text shown on the Result Screen beneath the Type Badges, sourced from PokeAPI's `flavor_text_entries`, in the style of the original games' Pokédex descriptions. Preferred from a Generation I game version (Red, Blue, then Yellow) where one exists, falling back to the first English entry otherwise — see [`docs/adr/0003`](./docs/adr/0003-prefer-generation-1-pokedex-entry-version.md). Distinct from the Stat Block, which is numeric/structured data rather than descriptive text. Fetching it is best-effort, the same as the Sprite: a failure shows a fallback message rather than failing the whole lookup.
_Avoid_: Flavor text, description, blurb, dex entry

**Stat Block**:
The set of fields shown on the Result Screen for a Pokémon: Pokédex #, Name, Type(s) (rendered as Type Badges), Height, Weight, HP, Attack, Defense, Speed, Special Attack, Special Defense. Height/Weight are converted from PokeAPI's decimetre/hectogram units to imperial (ft-in / lbs), matching how the original English Game Boy games displayed them. Uses the modern Sp. Atk / Sp. Def split from PokeAPI rather than Generation 1's single "Special" stat, since PokeAPI has no real Gen-1 Special value to show — only the visual style (labels, layout, palette, imperial units) is Gen-1-inspired, not the underlying data.
_Avoid_: Stats panel, info block, stat sheet

**Type Badge**:
A colored label for one of a Pokémon's types (e.g. Fire, Water, Grass), styled with that type's conventional color from official Pokédex UIs. Up to two appear per Pokémon in the Stat Block.
_Avoid_: Type tag, type pill

**Search by Type button**:
An interactive control on the Search Screen, beneath the name/number input, that leads to the Type Select Screen. Reachable by keyboard (Tab from the input, then Enter/Space) or, where the terminal supports it, a mouse click.
_Avoid_: Type search button, Browse by type

**Type Select Screen**:
The screen reached from the Search by Type button: a list of all 18 Pokémon types, each normalized to a capitalized display form (e.g. "fire" shows as "Fire"). Esc returns to the Search Screen; selecting a type (by keyboard or, where supported, mouse) advances to the Type Roster Screen for that type. Ctrl+C quits.
_Avoid_: Type screen, Types list, Species screen

**Type Roster Screen**:
The screen reached from the Type Select Screen: a table of every Pokémon of the chosen type, in National Dex Number order, showing each one's Dex #, Name, and Generation where known. Esc returns to the Type Select Screen; selecting a Pokémon (by keyboard or, where supported, mouse) advances to the Result Screen for that Pokémon. Ctrl+C quits.
_Avoid_: Type results screen, Type list screen, Pokémon by type screen

**Generation**:
The numbered era a Pokémon species was first introduced in (e.g. "Generation I" for the original 151), shown only on the Type Roster Screen. Distinct from a Pokémon's stats or types — purely a historical/release grouping.
_Avoid_: Gen, era
