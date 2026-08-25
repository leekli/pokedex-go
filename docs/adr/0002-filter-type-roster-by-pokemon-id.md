# Filter Type Roster entries by pokémon resource id, not `is_default`

`GET /type/{name}` returns every Pokémon of that type, including non-default forms (mega evolutions, regional forms, formes) — e.g. Alolan Raichu appears as pokémon resource id `10100`, not its National Dex Number `#26` (see CONTEXT.md's National Dex Number entry, which already warns about this divergence). The Type Roster Screen must show only real, dex-numbered Pokémon, in Dex order.

The "correct" filter is each entry's `is_default` flag, but that field only exists on the individual `/pokemon/{name}` resource, not in the type list response — checking it would mean one extra HTTP call per entry (50–150+ per type). Instead we filter by `pokemon.id < 10000`: verified against the live API that every default-form Pokémon we sampled has `id == species id == National Dex Number` and is `< 10000`, while every alt-form/mega/regional entry sampled is `≥ 10000`. This also means the filtered list is already Dex-ascending with no client-side sort needed, since PokeAPI returns type rosters pre-sorted by id.

This trades a small risk — PokeAPI reusing the sub-10000 range differently in the future — for avoiding an unnecessary large fan-out of requests on every type selection. If PokeAPI's id convention ever changes, this filter would need revisiting.

## Considered options

- **Fetch `/pokemon/{name}` per entry and check `is_default`**: robust to PokeAPI's id-numbering convention changing, but reintroduces the N+1 fan-out this decision exists to avoid; rejected on cost.
