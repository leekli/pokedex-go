# Navigation history replaces hardcoded screen destinations

Every screen's Esc/Enter used to hardcode its destination (e.g. the Result Screen always returned to the Search Screen), which worked while the app's screen graph was a single linear path. Adding the Type Select and Type Roster Screens creates a second path into the Result Screen (Search → Result, and Search → Type Select → Type Roster → Result), so "Esc goes back one screen" stopped having a single fixed answer per screen.

We chose to give `App` a real navigation history (a stack of previously-visited screens) that Esc pops, rather than keep hardcoded per-screen destinations and special-case the Result Screen for its second entry point. This keeps "Esc goes back one screen" true everywhere, including for screens added later, instead of requiring a new hardcoded destination (and a new special case) every time the screen graph grows. Enter is unaffected — it keeps each screen's existing fixed meaning (e.g. the Result Screen's Enter always means "search again," landing on the Search Screen regardless of history).

## Considered options

- **Keep hardcoded destinations, special-case the Result Screen**: least code short-term, but the special-casing would only grow with every future screen; rejected for not generalizing.
