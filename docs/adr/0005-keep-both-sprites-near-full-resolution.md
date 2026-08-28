# Widen the Result Screen card rather than shrink both sprites to fit it

Adding the back sprite next to the front one (see the Sprite entry in
CONTEXT.md) initially split the existing single-sprite column budget
(`spriteMaxWidth`, 40) in half between the two. That doubled
`spriteart.Render`'s downsample scale for each sprite (3 → 6), and at
scale 6 the renderer's nearest-neighbor sampling — a single point sample
per output cell, by design (see `spriteart/render.go`'s doc comment) —
throws away most of a typical 96×96 sprite's pixels. On a Pokémon with
large flat color regions (Charizard) the shape was still just about
readable; on one whose identity depends on a couple of small marks
(Pikachu's cheeks, its eyes) it wasn't — the sprite stopped being
recognizable as that Pokémon at all.

A candidate fix was tried first: replace the single point sample with a
majority-vote ("mode") color per downsample block, so noisy/aliased edge
pixels don't get picked at random. That measurably helped Charizard's
silhouette, but made Pikachu *worse*: a one-pixel cheek or eye highlight
always loses a majority vote to the surrounding fill color and gets erased
outright, every time — nearest-neighbor's raw randomness at least has a
chance of landing on it. No resampling algorithm can recover detail that's
already been discarded by halving the column budget; the only reliable fix
is to not discard it in the first place.

So instead, each sprite keeps a width close to its original budget
(`spriteHalfWidth`, 32, vs. the original 40) rather than half of it, and
`resultCardWidth` grows from 52 to 70 to fit the resulting sprite row
(`2*spriteHalfWidth + spriteGap` = 66) with a little margin. This trades
screen real estate for fidelity: the Result Screen's card is now
noticeably wider, and may not fit comfortably in an 80-column terminal, a
narrow split pane, or a narrow SSH session — a real cost, accepted
deliberately over shipping a feature that made the app's core sprite
display harder to read.

## Considered options

- **Majority-vote ("mode") color per downsample block, same narrow width**:
  helps large flat-color sprites, but systematically erases small
  minority-color features that are often exactly what makes a Pokémon
  recognizable (see Pikachu above); rejected as unreliable across the
  full Pokédex.
- **Color-average per downsample block, same narrow width**: keeps a
  blended hint of minority colors rather than erasing them outright, but
  turns the app's deliberately flat, saturated pixel-art look (see
  `spriteart/render.go`) to a muddy blur; rejected on aesthetic grounds as
  well as still being lossy.
- **Stack front above back instead of side by side**: keeps each sprite at
  its full original resolution and the card narrow, but changes the
  requested side-by-side layout and makes the card taller instead of
  wider; not chosen, but the fallback if a wider card proves impractical
  in practice.
