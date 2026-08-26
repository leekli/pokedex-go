# CLAUDE.md

Operating notes for agents working in this repo. For what the app does, how
to build/run/test it, and its architecture, see [README.md](./README.md).
For the project's vocabulary (Splash Screen, National Dex Number, Lookup
Error vs. Service Error, etc.), see [CONTEXT.md](./CONTEXT.md) — use those
terms exactly and avoid the synonyms each entry lists under `_Avoid_`.

## Before considering a change done

Run these from the repo root; all four must be clean (CI enforces the same
four on every push/PR — see `.github/workflows/ci.yml`):

```sh
gofmt -l .          # must print nothing
go vet ./...
go build ./...
go test ./...
```

## Layering rules

Enforce the package boundaries described in README's
[Architecture](./README.md#architecture) section: only `internal/tui` may
import Bubble Tea, and only `internal/pokeapi` may perform network I/O.
`internal/pokemon` and `internal/spriteart` stay pure (no TUI or network
imports). If a change seems to need crossing one of these, that's a signal
to reshape the change rather than the boundary.

## Caching

`internal/pokeapi.Client` caches every successful response for its own
lifetime (`internal/pokeapi/cache.go`) — PokeAPI's data doesn't meaningfully
change within a session, so re-fetching it is pure waste. When adding a new
`Client` method that fetches from PokeAPI, cache its result the same way:
populate the cache on success only, keyed by whatever string/id the request
was made with. Never cache a `*LookupError` or `*ServiceError` — a failure
may not hold true on retry (a corrected typo, a resolved outage), so only
successes are safe to remember for the rest of the session.

If a new method reads a resource another method already fetches (e.g.
`GetTypeDamageRelations` reads the same `/type/{name}` response
`GetPokemonByType` does), share one cached fetch between them rather than
adding a second cache map for the same underlying resource — see
`getTypeDetails` for the pattern: one private fetch+cache function, two
public methods that each return their own slice of its result.

## Commit messages

This repo follows Conventional Commits: `type(scope): description`, e.g.
`feat(tui): add Search by Type flow`, `docs: document the Search by Type
feature`. Common types: `feat`, `fix`, `docs`, `test`, `chore`, `refactor`.
Scope is usually the package touched (`tui`, `pokeapi`, `pokemon`,
`spriteart`) and is omitted for changes that aren't package-specific
(`docs:`, `chore:`).

## Architecture Decision Records

Write a new ADR under `docs/adr/` (numbered, following the existing files'
format — problem, chosen approach and why, considered-and-rejected
alternatives) when a change makes a hard-to-reverse design decision,
especially one that overrides a simpler obvious approach for a
non-obvious reason. Skip it for routine feature work that doesn't involve
such a tradeoff.

## Domain vocabulary

When a change introduces a new screen, concept, or term, add an entry to
CONTEXT.md in its existing format (bolded term, definition, `_Avoid_` line
listing rejected synonyms) rather than letting the concept go undocumented
or acquire an inconsistent name across the codebase.
