# ![Pokemon Pokeball](https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/items/poke-ball.png) pokedex-go ![Pokemon Pokeball](https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/items/poke-ball.png)

A terminal Pokédex. Launch it, press Enter, type a Pokémon's name or its
National Dex Number, and see its sprite and stats rendered in your terminal,
styled after the Pokédex screens from the Generation 1 Game Boy games.

Built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea),
backed by the public [PokeAPI](https://pokeapi.co/).

For the project's vocabulary (Splash Screen, Search Screen, Result Screen,
National Dex Number, Lookup Error vs. Service Error, etc.), see
[CONTEXT.md](./CONTEXT.md).

![Pokemon Bulbasaur](https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/1.png)
![Pokemon Charmander](https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/4.png)
![Pokemon Squirtle](https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/7.png)
![Pokemon Pikachu](https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/25.png)
![Pokemon Mewtwo](https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/150.png)
![Pokemon Mew](https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/151.png)

## Features

- **Splash Screen** — a hand-crafted ASCII/ANSI "POKEDEX" logo, embedded in
  the binary.
- **Search by name or number** — type a Pokémon's name (`pikachu`, `Mr Mime`,
  `farfetch'd`, `nidoran♀`) or its National Dex Number (`25`). Input is
  normalized automatically to match PokeAPI's expected format.
- **Result Screen** — the Pokémon's sprite, rendered as colored terminal
  block art, plus a Gen-1-styled stat block: Pokédex #, name, type badges
  (color-coded per type), height and weight in imperial units (matching the
  original English games), and base stats (HP, Attack, Defense, Sp. Atk,
  Sp. Def, Speed).
- **Distinct error handling** — a bad or unknown name/number shows a
  _Lookup Error_ inline; a PokeAPI outage or timeout shows a distinguishable
  _Service Error_ instead, so you know whether retrying will help.
- **Graceful color degradation** — colors automatically adapt to your
  terminal's capabilities (truecolor, 256-color, or none).

## Controls

| Screen | Keys                                                  |
| ------ | ----------------------------------------------------- |
| Splash | `Enter` search · `Q` / `Esc` / `Ctrl+C` quit          |
| Search | `Enter` search · `Esc` back to Splash · `Ctrl+C` quit |
| Result | `Enter` / `Esc` search again · `Q` / `Ctrl+C` quit    |

(On the Search Screen, only `Ctrl+C` quits — a bare `Q` stays typeable, since
plenty of Pokémon names contain it, e.g. Squirtle.)

## Requirements

- Go 1.26 or later (matches the `.devcontainer` image)
- Network access to `pokeapi.co` to look up Pokémon at runtime

## Development setup

```sh
git clone https://github.com/leekli/pokedex-go.git
cd pokedex-go
go build ./...
```

No further setup is needed — there are no API keys, config files, or
external services beyond PokeAPI itself. The `.devcontainer/` config
provisions a matching Go environment automatically if you're using VS Code
Dev Containers / GitHub Codespaces.

### Running the app

```sh
go run ./cmd/pokedex-go
```

### Building the app

```sh
go build ./cmd/pokedex-go
```

## Testing

The project has three layers of automated tests, plus one opt-in live check
against the real PokeAPI. None of the default tests touch the network.

```sh
# Everything (unit + integration + e2e):
go test ./...

# With coverage:
go test ./... -cover

# Just the pure domain logic (internal/pokemon) and sprite renderer
# (internal/spriteart) — fast, no HTTP involved at all:
go test ./internal/pokemon/... ./internal/spriteart/...

# Just the PokeAPI client, against a local httptest mock server:
go test ./internal/pokeapi/...

# Just the Bubble Tea models in isolation (Update/View for Splash, Search,
# Result, and App) plus the pure helpers behind them (the startup sweep's
# color math, lookupCmd's PokeAPI/sprite orchestration):
go test ./internal/tui/...

# Just the full-flow TUI tests (splash → search → result, quitting,
# error paths), driven via Bubble Tea's teatest against a local mock:
go test ./test/e2e/...
```

### Coverage

`internal/pokemon`, `internal/spriteart`, and `internal/pokeapi` sit at
100% statement coverage on their own. `internal/tui` reports ~96% from its
own unit tests (`go test ./internal/tui/... -cover`); the remaining lines —
mostly `App`/`searchModel`/`resultModel` `Update` branches that only fire
mid-animation or mid-lookup — are exercised by `test/e2e`'s full-flow runs
instead of being duplicated at the unit level. Combined
(`go test -coverpkg=./internal/... ./...`), the suite reaches 100% of
`internal/...` statements.

Every file under `internal/tui/` (including `splash.go` and
`splash_legend.go`) has direct unit-test coverage of its pure
logic — `sweepProgress`, `Update`'s key/tick handling, `blendHex`,
`sweepColor`, `hexRGB`, `lerpByte`, `widestLine`. What's *not* re-tested at
the unit level is pixel-exact ANSI output (`renderSplashArtSweep`'s styled
string, lipgloss table borders, etc.) — those are covered by `test/e2e`
asserting on the substrings that matter (e.g. `"POKÉDEX SEARCH"`, a
Pokémon's stat block) against a real terminal-sized render, which is a
better tool for "does the screen show the right thing" than pinning exact
ANSI escape sequences in a unit test would be.

`cmd/pokedex-go/main.go` (0% coverage) is intentionally excluded: it's a
five-line entrypoint that wires an already-tested `pokeapi.Client` into an
already-tested `tui.App` and hands both to Bubble Tea's `Program.Run()` —
there's no branching logic of its own to test, and exercising it would mean
driving a real OS terminal program rather than testing this app's code.

### Live smoke test (opt-in, not part of the default suite)

One additional test hits the _real_ PokeAPI once, to catch drift between the
mocked fixtures used everywhere else and PokeAPI's actual response shape.
It's excluded from normal builds and test runs by a `live` build tag and a
runtime environment-variable check, so it never runs by accident:

```sh
POKEDEX_LIVE_TEST=1 go test -tags=live ./test/live/...
```

## Architecture

pokedex-go follows Bubble Tea's [Elm Architecture](https://github.com/charmbracelet/bubbletea#tutorial):
a single root `Model` (`tui.App`) receives each event as a `Msg`, its
`Update` returns a new `Model` plus an optional `Cmd` (an async side
effect, e.g. a PokeAPI call), and `View` renders the current `Model` to a
string every cycle. There's no shared mutable state — every screen
transition and network result flows through this `Msg → Update → Cmd → Msg`
loop, driven from `cmd/pokedex-go/main.go`, which just wires a
`pokeapi.Client` into a `tui.App` and hands both to Bubble Tea.

Layering is strict and one-directional: `internal/tui` is the only package
that imports Bubble Tea, and `internal/pokeapi` is the only package that
performs network I/O. Both build on `internal/pokemon` — pure query/name
parsing, unit conversion, type colors, and the `StatBlock` type, with no
knowledge of the TUI or the network. `internal/spriteart` (image → ANSI
block art) is likewise pure, used only by `tui` to render a fetched sprite
— see [Project layout](#project-layout) below.

A Pokémon lookup — the app's one real workflow — moves through these
layers like this:

```mermaid
flowchart LR
    Splash -->|Enter| Search
    Search -->|pokemon.ResolveQuery| Client["pokeapi.Client.Lookup"]
    Client -->|HTTPS| PokeAPI[(pokeapi.co)]
    Client -->|BuildStatBlock| Stat[pokemon.StatBlock]
    Client -->|FetchSprite| Sprite[spriteart.Render]
    Stat --> Msg[lookupResultMsg]
    Sprite --> Msg
    Msg -->|showResultMsg| Result
    Result -->|Esc / Enter| Search
```

`pokeapi.Client.Lookup` resolves a National Dex Number via
`GetSpecies` → `GetPokemon`, or a name via `GetPokemon` directly, and
classifies any failure as a `*LookupError` (bad input) or `*ServiceError`
(PokeAPI's fault) — see CONTEXT.md. A failed sprite fetch doesn't fail the
whole lookup; the Result Screen just falls back to a "no sprite" message.

## Project layout

```
cmd/pokedex-go/       entrypoint - wires everything together and runs the program
internal/pokemon/     pure domain logic: input normalization, unit conversion, type colors
internal/pokeapi/     PokeAPI HTTP client, JSON decoding, Lookup/Service error classification
internal/spriteart/   renders a decoded image as colored terminal block art
internal/tui/         Bubble Tea layer: Splash, Search, and Result screens
test/e2e/              full-flow tests driven through the TUI via teatest
test/live/             opt-in live smoke test against the real PokeAPI
CONTEXT.md            domain glossary — the project's vocabulary
```
