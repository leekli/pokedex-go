# Wrap screens in one App-level viewport with a hand-restricted keymap, not `bubbles/viewport`'s default one

On a terminal shorter than a screen's rendered content (e.g. 80x25 for the
Result Screen's sprites, evolution chain, and stat table), Bubble Tea's
alt-screen mode gives no scrollback and the app had no way to scroll its
own content — the top of the Result Screen, including both sprites, could
be pushed off-screen with no way to see it again.

`bubbles/viewport` (already vendored via the `bubbles` dependency used for
`textinput`/`spinner`/`table`) is the natural fit for vertical pager-style
scrolling. Two decisions in how it's wired up are non-obvious enough to
record here.

## One shared viewport on `App`, not one embedded per screen model

`App` (the root model) owns a single `viewport.Model` and wraps whichever
screen's rendered `View()` output is currently active, rather than each of
`resultModel`/`searchModel`/`splashModel`/`typeSelectModel` owning its own.
Only one screen is ever visible at a time, so a single shared instance is
enough, and it keeps every screen model's own `View()` method exactly as
it already was — full, unwrapped content, with no width/height parameters
threaded through it. That matters concretely: `result_test.go`'s
`TestResultModel_View_ShowsStatBlock` and its siblings across
`search_test.go`/`splash_test.go` call `.View()` directly, with no
terminal size in play at all, and needed no changes.

## A hand-restricted `viewport.KeyMap`, not `viewport.DefaultKeyMap()`

`viewport.DefaultKeyMap()` binds vim-style letters (`j`/`k`/`f`/`b`/`u`/`d`),
space, and left/right in addition to the arrow/page keys. Those letters
are real substrings of Pokémon names the Search Screen's text input must
stay free to receive — "pikachu" alone contains both "u" and "k". Using
the default keymap as-is would silently corrupt typed input the moment a
query contained one of those letters. Instead, `app.go` defines
`viewportPagerKeyMap` (Up/Down/PageUp/PageDown only — literal arrow/page
keys, nothing else) for the Splash, Search, and Result Screens, and
`viewportPagingOnlyKeyMap` (PageUp/PageDown only, no Up/Down) for the Type
Select Screen, where Up/Down/j/k must stay dedicated to its own cursor
navigation rather than compete with the viewport for the same keys — see
`followCursor` in `typeselect.go` for how that screen's selection still
gets kept on screen without those keys. Left/Right and horizontal
scrolling generally are left out of both keymaps entirely: horizontal
responsiveness is a deliberate non-goal of this change.

### A landmine this relies on not being reintroduced

`viewport.Model.Update` resets `.KeyMap` back to `viewport.DefaultKeyMap()`
the first time it's used, unless the model was constructed via
`viewport.New(...)` (which sets an unexported `initialized` flag that
skips that reset on every later call). `App`'s viewport is built with
`viewport.New(0, 0)` in `NewApp`, and its `.KeyMap` is overwritten
immediately after — and `a.viewport` must never be reassigned or rebuilt
from a bare `viewport.Model{}` literal anywhere else in the package, or
this override would silently and intermittently be lost, letting vim
letters and space reach the viewport instead of whatever screen's text
input the user is typing into.
`TestApp_Update_SearchScreen_RestrictedKeymapDoesNotSwallowLetters` in
`app_test.go` exists specifically to catch a regression here.

## Considered options

- **Per-screen embedded viewport**: each screen model owns and sizes its
  own `viewport.Model`. Rejected — it would require threading terminal
  width/height into every screen constructor and `View()` call, breaking
  the existing screen-level `View()` tests' no-argument call convention for
  no benefit, since only one screen is ever visible at once anyway.
- **`viewport.DefaultKeyMap()` as-is**: simplest to wire up, but collides
  with real Pokémon-name substrings on the Search Screen, as above.
  Rejected outright, not just for the Search Screen specifically, since a
  restricted keymap costs nothing on the screens where it wasn't strictly
  necessary either.
- **Give the Type Select Screen no scrolling and defer it**: the simplest
  scope cut, since 18 types plus its own chrome (~22 lines) only overflows
  a genuinely short terminal. Rejected because the alternative isn't "no
  regression," it's "cursor navigation can silently move the selection off
  screen with no feedback" — a real, if narrower, version of the same bug
  this change exists to fix, and Phase 1's shared-viewport plumbing was
  already in place to support it (see `followCursor`) at low marginal cost.
