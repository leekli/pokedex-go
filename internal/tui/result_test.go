package tui

import (
	"image"
	"image/color"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leekli/pokedex-go/internal/pokemon"
	"github.com/leekli/pokedex-go/internal/spriteart"
)

func TestCapitalize(t *testing.T) {
	tests := map[string]string{
		"pikachu": "Pikachu",
		"":        "",
		"x":       "X",
	}
	for input, want := range tests {
		if got := capitalize(input); got != want {
			t.Errorf("capitalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func testStatBlock() pokemon.StatBlock {
	return pokemon.StatBlock{
		DexNumber:      6,
		Name:           "charizard",
		Types:          []string{"fire", "flying"},
		HeightFeet:     5,
		HeightInches:   7,
		WeightPounds:   199.5,
		HP:             78,
		Attack:         84,
		Defense:        78,
		SpecialAttack:  109,
		SpecialDefense: 85,
		Speed:          100,
	}
}

// TestResultModel_View_ShowsStatBlock is a smoke test for the Trading Card
// redesign's content: CONTEXT.md's Stat Block fields must all still appear,
// unchanged, just restyled into the card/table layout.
func TestResultModel_View_ShowsStatBlock(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{R: 255, G: 100, B: 0, A: 255})
	m := newResultModel(testStatBlock(), img, nil, "", nil, nil)
	view := m.View()

	for _, want := range []string{
		"#006 Charizard",
		"FIRE", "FLYING",
		`5'07"`, "199.5 lbs",
		"STAT", "VALUE",
		"HP", "78",
		"Attack", "84",
		"Defense",
		"Sp. Atk", "109",
		"Sp. Def", "85",
		"Speed", "100",
		"Enter/Esc to search again · Q to quit",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("resultModel.View() missing %q\ngot:\n%s", want, view)
		}
	}
	if strings.Contains(view, "No sprite available") {
		t.Errorf("resultModel.View() with a sprite should not show the fallback message\ngot:\n%s", view)
	}
}

// TestResultModel_View_NoSprite confirms the existing "no sprite available"
// fallback (sprite == nil) still renders inside the new card, with the rest
// of the Stat Block unaffected.
func TestResultModel_View_NoSprite(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, nil, "", nil, nil)
	view := m.View()

	if !strings.Contains(view, "No sprite available") {
		t.Errorf("resultModel.View() with a nil sprite missing fallback message\ngot:\n%s", view)
	}
	if !strings.Contains(view, "#006 Charizard") || !strings.Contains(view, "HP") {
		t.Errorf("resultModel.View() with a nil sprite should still show the rest of the stat block\ngot:\n%s", view)
	}
}

// TestResultModel_View_FrontAndBackSprites proves that when both a front and
// a back sprite are available, both are rendered side by side (front on the
// left, back on the right) rather than only the front sprite showing.
func TestResultModel_View_FrontAndBackSprites(t *testing.T) {
	front := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	front.Set(0, 0, color.NRGBA{R: 255, G: 100, B: 0, A: 255})
	back := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	back.Set(0, 0, color.NRGBA{R: 0, G: 100, B: 255, A: 255})

	m := newResultModel(testStatBlock(), front, back, "", nil, nil)
	view := m.View()

	frontOnly := renderSprites(front, nil)
	backOnly := renderSprites(nil, back)
	if !strings.Contains(view, strings.SplitN(frontOnly, "\n", 2)[0]) {
		t.Errorf("resultModel.View() missing the front sprite's first rendered line\ngot:\n%s", view)
	}
	if !strings.Contains(view, strings.SplitN(backOnly, "\n", 2)[0]) {
		t.Errorf("resultModel.View() missing the back sprite's first rendered line\ngot:\n%s", view)
	}
	if strings.Contains(view, "No sprite available") {
		t.Errorf("resultModel.View() with both sprites should not show the fallback message\ngot:\n%s", view)
	}
}

// TestRenderSprites_LoneSpriteUsesFullWidth proves that with only one of
// front/back available, renderSprites renders it at the full spriteMaxWidth
// (matching spriteart.Render directly), rather than shrinking it to the
// half-width used when both sprites are shown side by side.
func TestRenderSprites_LoneSpriteUsesFullWidth(t *testing.T) {
	front := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	front.Set(0, 0, color.NRGBA{R: 255, G: 100, B: 0, A: 255})

	got := renderSprites(front, nil)
	want := spriteart.Render(front, spriteart.Options{MaxWidth: spriteMaxWidth})
	if got != want {
		t.Errorf("renderSprites(front, nil) = %q, want %q (full spriteMaxWidth)", got, want)
	}
}

// TestRenderSprites_BothSpritesUseSpriteHalfWidth proves that with both
// front and back available, each is capped to spriteHalfWidth - not a naive
// half of spriteMaxWidth, which would double spriteart.Render's downsample
// scale and destroy small sprite detail (a Pikachu's cheek, an eye) - see
// docs/adr/0005-keep-both-sprites-near-full-resolution.md.
func TestRenderSprites_BothSpritesUseSpriteHalfWidth(t *testing.T) {
	front := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	front.Set(0, 0, color.NRGBA{R: 255, G: 100, B: 0, A: 255})
	back := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	back.Set(0, 0, color.NRGBA{R: 0, G: 100, B: 255, A: 255})

	got := renderSprites(front, back)

	frontWant := spriteart.Render(front, spriteart.Options{MaxWidth: spriteHalfWidth})
	backWant := spriteart.Render(back, spriteart.Options{MaxWidth: spriteHalfWidth})
	wantGap := strings.Repeat(" ", spriteGap)
	want := lipgloss.JoinHorizontal(lipgloss.Top, frontWant, wantGap, backWant)

	if got != want {
		t.Errorf("renderSprites(front, back) did not use spriteHalfWidth for each sprite\ngot:\n%s\nwant:\n%s", got, want)
	}

	naiveHalf := (spriteMaxWidth - spriteGap) / 2
	if naiveHalf == spriteHalfWidth {
		t.Fatalf("test no longer distinguishes spriteHalfWidth (%d) from the naive half-of-spriteMaxWidth split (%d) - update the constants or this test", spriteHalfWidth, naiveHalf)
	}
}

// TestResultModel_View_ShowsPokedexEntry proves a non-empty pokedexEntry
// renders on the card - see docs/adr/0003-prefer-generation-1-pokedex-entry-version.md.
func TestResultModel_View_ShowsPokedexEntry(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, nil, "A mysterious, flame-spitting Pokémon.", nil, nil)
	view := m.View()

	if !strings.Contains(view, "A mysterious, flame-spitting Pokémon.") {
		t.Errorf("resultModel.View() missing the Pokédex Entry\ngot:\n%s", view)
	}
	if strings.Contains(view, "No Pokédex entry available.") {
		t.Errorf("resultModel.View() with a Pokédex Entry should not show the fallback message\ngot:\n%s", view)
	}
}

// TestResultModel_View_NoPokedexEntryFallback proves an empty pokedexEntry
// (no English entry existed, or the fetch failed) shows a fallback message
// rather than an empty gap, the same way a nil sprite does.
func TestResultModel_View_NoPokedexEntryFallback(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, nil, "", nil, nil)
	view := m.View()

	if !strings.Contains(view, "No Pokédex entry available.") {
		t.Errorf("resultModel.View() with no Pokédex Entry missing fallback message\ngot:\n%s", view)
	}
}

// charizardTypeEffectiveness is Charizard's (Fire/Flying, testStatBlock's
// Pokémon) real, complete type chart - see
// pokeapi.TestBuildTypeEffectiveness_CharizardFullChart for how it's
// derived from PokeAPI's raw damage relations.
func charizardTypeEffectiveness() *pokemon.TypeEffectiveness {
	return &pokemon.TypeEffectiveness{
		Weaknesses: []pokemon.TypeMatchup{
			{Type: "rock", Multiplier: 4},
			{Type: "water", Multiplier: 2},
			{Type: "electric", Multiplier: 2},
		},
		Resistances: []pokemon.TypeMatchup{
			{Type: "grass", Multiplier: 0.25},
			{Type: "bug", Multiplier: 0.25},
			{Type: "fire", Multiplier: 0.5},
			{Type: "fighting", Multiplier: 0.5},
			{Type: "steel", Multiplier: 0.5},
			{Type: "fairy", Multiplier: 0.5},
		},
		Immunities: []string{"ground"},
	}
}

// TestResultModel_View_ShowsTypeEffectiveness proves the Weaknesses &
// Resistances section renders weaknesses, resistances, and immunities with
// their multipliers - see docs/adr/0004-all-or-nothing-type-effectiveness.md.
func TestResultModel_View_ShowsTypeEffectiveness(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, nil, "", nil, charizardTypeEffectiveness())
	view := m.View()

	for _, want := range []string{
		"Weak to", "Rock 4×", "Water 2×", "Electric 2×",
		"Resists", "Grass ¼×", "Bug ¼×", "Fire ½×",
		"Immune to", "Ground",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("resultModel.View() missing %q\ngot:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Weaknesses & resistances unavailable") {
		t.Errorf("resultModel.View() with type effectiveness should not show the fallback message\ngot:\n%s", view)
	}
}

// TestResultModel_View_TypeEffectivenessUnavailableFallback proves a nil
// typeEffectiveness (a damage relations fetch failed) shows a fallback
// message rather than an empty gap or, worse, a partial/wrong chart.
func TestResultModel_View_TypeEffectivenessUnavailableFallback(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, nil, "", nil, nil)
	view := m.View()

	if !strings.Contains(view, "Weaknesses & resistances unavailable.") {
		t.Errorf("resultModel.View() with nil type effectiveness missing fallback message\ngot:\n%s", view)
	}
}

// TestResultModel_View_TypeEffectivenessNoneShowsExplicitly proves a
// Pokémon with zero weaknesses (or resistances) shows "None" rather than a
// blank line, so it reads as a confirmed fact rather than missing data.
func TestResultModel_View_TypeEffectivenessNoneShowsExplicitly(t *testing.T) {
	te := &pokemon.TypeEffectiveness{}
	m := newResultModel(testStatBlock(), nil, nil, "", nil, te)
	view := m.View()

	if !strings.Contains(view, "Weak to    None") {
		t.Errorf("resultModel.View() with no weaknesses missing explicit \"None\"\ngot:\n%s", view)
	}
	if !strings.Contains(view, "Resists    None") {
		t.Errorf("resultModel.View() with no resistances missing explicit \"None\"\ngot:\n%s", view)
	}
}

// TestFormatMultiplier covers the four multipliers BuildTypeEffectiveness
// can actually produce from at most two combined types, plus the
// default case guarding against that ever changing.
func TestFormatMultiplier(t *testing.T) {
	tests := map[float64]string{
		4:    "4×",
		2:    "2×",
		0.5:  "½×",
		0.25: "¼×",
		8:    "8×", // not reachable with only 2 types today; proves the fallback degrades sanely rather than panicking or omitting the value.
	}
	for m, want := range tests {
		if got := formatMultiplier(m); got != want {
			t.Errorf("formatMultiplier(%v) = %q, want %q", m, got, want)
		}
	}
}

// TestRenderStatBar checks the fill/empty math at its boundaries: a zero
// stat renders no filled cells, and the (theoretical) max stat fills the
// whole bar without overflowing it - the kind of off-by-one that's easy to
// get wrong when scaling an arbitrary int range onto a fixed bar width.
func TestRenderStatBar(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"zero", 0},
		{"typical", 100},
		{"max", maxBaseStat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := renderStatBar(tt.value, "#F08030", "")
			filled := strings.Count(bar, "█")
			empty := strings.Count(bar, "░")
			if filled+empty != statBarWidth {
				t.Errorf("renderStatBar(%d) has %d+%d cells, want %d total", tt.value, filled, empty, statBarWidth)
			}
			if tt.value == 0 && filled != 0 {
				t.Errorf("renderStatBar(0) filled = %d, want 0", filled)
			}
			if tt.value == maxBaseStat && filled != statBarWidth {
				t.Errorf("renderStatBar(maxBaseStat) filled = %d, want %d", filled, statBarWidth)
			}
		})
	}
}

// TestResultModel_Update_NonKeyMsgIgnored proves a message that isn't a
// key press (e.g. a stray tick from another screen) is simply ignored.
func TestResultModel_Update_NonKeyMsgIgnored(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, nil, "", nil, nil)

	got, cmd := m.Update(struct{}{})

	if cmd != nil {
		t.Error("Update(non-KeyMsg) returned a non-nil cmd, want nil")
	}
	if got.View() != m.View() {
		t.Error("Update(non-KeyMsg) changed the model, want it unchanged")
	}
}

// TestResultModel_Update_EscGoesBack proves Esc asks App to go back to
// whichever screen led here, rather than naming a fixed destination itself
// (see docs/adr/0001-navigation-history-for-back-navigation.md and
// result_navigation_test.go for the full-flow e2e equivalent).
func TestResultModel_Update_EscGoesBack(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, nil, "", nil, nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("Update(Esc) returned a nil cmd, want goBack()")
	}
	if _, ok := cmd().(backMsg); !ok {
		t.Errorf("Update(Esc) cmd produced %T, want backMsg", cmd())
	}
}

// TestResultModel_Update_EnterSearchesAgain proves Enter has a single fixed
// meaning - "look up another Pokémon" - regardless of how this screen was
// reached (see docs/adr/0001).
func TestResultModel_Update_EnterSearchesAgain(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, nil, "", nil, nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Update(Enter) returned a nil cmd, want searchAgain()")
	}
	if _, ok := cmd().(searchAgainMsg); !ok {
		t.Errorf("Update(Enter) cmd produced %T, want searchAgainMsg", cmd())
	}
}

func TestResultModel_Update_QQuits(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, nil, "", nil, nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if cmd == nil {
		t.Fatal(`Update("q") returned a nil cmd, want tea.Quit`)
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf(`Update("q") cmd produced %T, want tea.QuitMsg`, cmd())
	}
}

// TestResultModel_Update_UnhandledKeyNoOp proves a key with no binding on
// this screen (there's no free-text input here) is simply ignored.
func TestResultModel_Update_UnhandledKeyNoOp(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, nil, "", nil, nil)

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if cmd != nil {
		t.Errorf(`Update("x") returned a non-nil cmd, want nil`)
	}
	if got.View() != m.View() {
		t.Errorf(`Update("x") changed the model, want it unchanged`)
	}
}

// pichuPikachuRaichuChain is Pikachu's real evolution family, used
// throughout the Evolution Chain tests below as a real, checkable linear
// example - the same family used in internal/pokeapi's own tests.
func pichuPikachuRaichuChain() pokemon.EvolutionChain {
	return pokemon.EvolutionChain{
		Root: pokemon.EvolutionStage{
			DexNumber: 172,
			Name:      "pichu",
			EvolvesTo: []pokemon.EvolutionStage{
				{
					DexNumber: 25,
					Name:      "pikachu",
					Condition: "high friendship",
					EvolvesTo: []pokemon.EvolutionStage{
						{DexNumber: 26, Name: "raichu", Condition: "use Thunder Stone"},
					},
				},
			},
		},
	}
}

func TestRenderEvolutionChain_NilChainShowsFallback(t *testing.T) {
	got := renderEvolutionChain(25, nil)
	if !strings.Contains(got, "Evolution data unavailable") {
		t.Errorf("renderEvolutionChain(_, nil) = %q, want the unavailable fallback", got)
	}
}

// TestRenderEvolutionChain_DoesNotEvolve proves a Pokémon with no
// evolution relations at all (e.g. Tauros) shows "Does not evolve" - real
// information, not the "data unavailable" failure fallback.
func TestRenderEvolutionChain_DoesNotEvolve(t *testing.T) {
	chain := pokemon.EvolutionChain{Root: pokemon.EvolutionStage{DexNumber: 128, Name: "tauros"}}

	got := renderEvolutionChain(128, &chain)

	if got != "Does not evolve" {
		t.Errorf("renderEvolutionChain(single-stage, no evolutions) = %q, want %q", got, "Does not evolve")
	}
}

// TestRenderEvolutionChain_MidChainShowsAncestorsAndDescendants proves
// viewing a middle stage (Pikachu) shows the whole family: its ancestor
// (Pichu), itself, and what it evolves into (Raichu), with both
// transitions' conditions.
func TestRenderEvolutionChain_MidChainShowsAncestorsAndDescendants(t *testing.T) {
	chain := pichuPikachuRaichuChain()

	got := renderEvolutionChain(25, &chain)

	for _, want := range []string{"Pichu", "Pikachu", "Raichu", "high friendship", "use Thunder Stone"} {
		if !strings.Contains(got, want) {
			t.Errorf("renderEvolutionChain(mid-chain) missing %q\ngot:\n%s", want, got)
		}
	}
}

// TestRenderEvolutionChain_RootStageShowsOnlyItsOwnChildren proves viewing
// the chain's root (Pichu) shows what it evolves into (Pikachu) but not
// further descendants (Raichu, two stages beyond the one being viewed).
func TestRenderEvolutionChain_RootStageShowsOnlyItsOwnChildren(t *testing.T) {
	chain := pichuPikachuRaichuChain()

	got := renderEvolutionChain(172, &chain)

	if !strings.Contains(got, "Pichu") || !strings.Contains(got, "Pikachu") {
		t.Errorf("renderEvolutionChain(root) = %q, want it to show its own evolution (Pikachu)", got)
	}
	if strings.Contains(got, "Raichu") {
		t.Errorf("renderEvolutionChain(root) = %q, should not show Raichu (two stages beyond the one being viewed)", got)
	}
}

// TestRenderEvolutionChain_LeafStageShowsFullAncestry proves viewing the
// end of the chain (Raichu) still shows the full path that led there.
func TestRenderEvolutionChain_LeafStageShowsFullAncestry(t *testing.T) {
	chain := pichuPikachuRaichuChain()

	got := renderEvolutionChain(26, &chain)

	for _, want := range []string{"Pichu", "Pikachu", "Raichu"} {
		if !strings.Contains(got, want) {
			t.Errorf("renderEvolutionChain(leaf) missing %q\ngot:\n%s", want, got)
		}
	}
}

// TestRenderEvolutionChain_UnknownDexNumberShowsFallback is defensive: the
// Result Screen's own Pokémon should always be somewhere in its own
// Evolution Chain, but a mismatch must degrade to the fallback rather than
// panicking on an empty path.
func TestRenderEvolutionChain_UnknownDexNumberShowsFallback(t *testing.T) {
	chain := pichuPikachuRaichuChain()

	got := renderEvolutionChain(999, &chain)

	if !strings.Contains(got, "Evolution data unavailable") {
		t.Errorf("renderEvolutionChain(unknown dex number) = %q, want the unavailable fallback", got)
	}
}

// TestRenderEvolutionChain_Branching proves a species with multiple
// evolutions (like Eevee) shows every sibling and its own condition, not
// just the first one.
func TestRenderEvolutionChain_Branching(t *testing.T) {
	chain := pokemon.EvolutionChain{
		Root: pokemon.EvolutionStage{
			DexNumber: 133,
			Name:      "eevee",
			EvolvesTo: []pokemon.EvolutionStage{
				{DexNumber: 134, Name: "vaporeon", Condition: "use Water Stone"},
				{DexNumber: 135, Name: "jolteon", Condition: "use Thunder Stone"},
			},
		},
	}

	got := renderEvolutionChain(133, &chain)

	for _, want := range []string{"Eevee", "Vaporeon", "Jolteon", "use Water Stone", "use Thunder Stone"} {
		if !strings.Contains(got, want) {
			t.Errorf("renderEvolutionChain(branching) missing %q\ngot:\n%s", want, got)
		}
	}
}

// TestRenderEvolutionBreadcrumb_ConditionsDoNotCollide is a regression test:
// an earlier version of this alignment logic sized each column by name
// width alone, so a condition longer than its name ("high friendship" vs
// "Pikachu") ran straight into the next segment's condition with no
// separating space at all. Column width must be driven by whichever of the
// name or condition is wider.
func TestRenderEvolutionBreadcrumb_ConditionsDoNotCollide(t *testing.T) {
	segments := []evolutionSegment{
		{name: "pichu"},
		{name: "pikachu", current: true, condition: "high friendship"},
		{name: "raichu", condition: "use Thunder Stone"},
	}

	got := renderEvolutionBreadcrumb(segments, 2)
	lines := strings.Split(got, "\n")

	if len(lines) != 2 {
		t.Fatalf("renderEvolutionBreadcrumb produced %d lines, want 2 (names, then conditions)", len(lines))
	}
	if strings.Contains(lines[1], "friendshipuse") {
		t.Errorf("condition line = %q, conditions ran together with no separating space", lines[1])
	}
	if !strings.Contains(lines[1], "high friendship") || !strings.Contains(lines[1], "use Thunder Stone") {
		t.Errorf("condition line = %q, missing a condition", lines[1])
	}
}

// TestRenderEvolutionBreadcrumb_FirstChildUsesArrowNotSlash is a
// regression test: the transition from the currently-viewed stage into
// its *first* evolution must still be an arrow (it's the next stage in
// the lineage); only that first child's siblings (Eevee's second, third,
// ... evolution) are slash-separated from each other.
func TestRenderEvolutionBreadcrumb_FirstChildUsesArrowNotSlash(t *testing.T) {
	segments := []evolutionSegment{
		{name: "eevee", current: true},
		{name: "vaporeon", condition: "use Water Stone"},
		{name: "jolteon", condition: "use Thunder Stone"},
	}

	got := renderEvolutionBreadcrumb(segments, 1)
	lines := strings.Split(got, "\n")
	nameLine := lines[0]

	// Segment columns are padded to fit whichever of the name or its
	// condition is wider (see renderEvolutionBreadcrumb's doc comment), so
	// the separator isn't necessarily hard against the name it follows -
	// find each name's own position and check what separator character
	// comes immediately after it, rather than asserting exact adjacency.
	eevee := strings.Index(nameLine, "Eevee")
	vaporeon := strings.Index(nameLine, "Vaporeon")
	jolteon := strings.Index(nameLine, "Jolteon")
	if eevee < 0 || vaporeon < 0 || jolteon < 0 {
		t.Fatalf("name line = %q, missing a stage name", nameLine)
	}

	between := nameLine[eevee+len("Eevee") : vaporeon]
	if strings.Contains(between, evolutionSlash) || !strings.Contains(between, "→") {
		t.Errorf("separator between Eevee and Vaporeon = %q, want an arrow (it's the next stage in the lineage), not a slash", between)
	}

	between = nameLine[vaporeon+len("Vaporeon") : jolteon]
	if strings.Contains(between, "→") || !strings.Contains(between, "/") {
		t.Errorf("separator between Vaporeon and Jolteon = %q, want a slash (siblings), not an arrow", between)
	}
}

// TestResultModel_View_ShowsEvolutionChain proves the Evolution Chain
// renders inside the full card, keyed off the Pokémon actually being
// viewed (Charizard, #6 - matching testStatBlock()).
func TestResultModel_View_ShowsEvolutionChain(t *testing.T) {
	chain := pokemon.EvolutionChain{
		Root: pokemon.EvolutionStage{
			DexNumber: 4,
			Name:      "charmander",
			EvolvesTo: []pokemon.EvolutionStage{
				{
					DexNumber: 5,
					Name:      "charmeleon",
					Condition: "Lv. 16",
					EvolvesTo: []pokemon.EvolutionStage{
						{DexNumber: 6, Name: "charizard", Condition: "Lv. 36"},
					},
				},
			},
		},
	}
	m := newResultModel(testStatBlock(), nil, nil, "", &chain, nil)

	view := m.View()

	for _, want := range []string{"Charmander", "Charmeleon", "Charizard", "Lv. 16", "Lv. 36"} {
		if !strings.Contains(view, want) {
			t.Errorf("resultModel.View() missing %q from the Evolution Chain\ngot:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Evolution data unavailable") {
		t.Errorf("resultModel.View() with an evolution chain should not show the fallback message\ngot:\n%s", view)
	}
}

// TestResultModel_View_NoEvolutionChainFallback proves a nil evolutionChain
// (the fetch failed, or PokeAPI had none) shows a fallback message rather
// than an empty gap, the same way a nil sprite does.
func TestResultModel_View_NoEvolutionChainFallback(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, nil, "", nil, nil)

	view := m.View()

	if !strings.Contains(view, "Evolution data unavailable") {
		t.Errorf("resultModel.View() with a nil evolutionChain missing fallback message\ngot:\n%s", view)
	}
}
