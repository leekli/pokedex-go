package tui

import (
	"image"
	"image/color"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leekli/pokedex-go/internal/pokemon"
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
	m := newResultModel(testStatBlock(), img, "", nil)
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
	m := newResultModel(testStatBlock(), nil, "", nil)
	view := m.View()

	if !strings.Contains(view, "No sprite available") {
		t.Errorf("resultModel.View() with a nil sprite missing fallback message\ngot:\n%s", view)
	}
	if !strings.Contains(view, "#006 Charizard") || !strings.Contains(view, "HP") {
		t.Errorf("resultModel.View() with a nil sprite should still show the rest of the stat block\ngot:\n%s", view)
	}
}

// TestResultModel_View_ShowsPokedexEntry proves a non-empty pokedexEntry
// renders on the card - see docs/adr/0003-prefer-generation-1-pokedex-entry-version.md.
func TestResultModel_View_ShowsPokedexEntry(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, "A mysterious, flame-spitting Pokémon.", nil)
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
	m := newResultModel(testStatBlock(), nil, "", nil)
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
	m := newResultModel(testStatBlock(), nil, "", charizardTypeEffectiveness())
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
	if strings.Contains(view, "unavailable") {
		t.Errorf("resultModel.View() with type effectiveness should not show the fallback message\ngot:\n%s", view)
	}
}

// TestResultModel_View_TypeEffectivenessUnavailableFallback proves a nil
// typeEffectiveness (a damage relations fetch failed) shows a fallback
// message rather than an empty gap or, worse, a partial/wrong chart.
func TestResultModel_View_TypeEffectivenessUnavailableFallback(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, "", nil)
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
	m := newResultModel(testStatBlock(), nil, "", te)
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
	m := newResultModel(testStatBlock(), nil, "", nil)

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
	m := newResultModel(testStatBlock(), nil, "", nil)

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
	m := newResultModel(testStatBlock(), nil, "", nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Update(Enter) returned a nil cmd, want searchAgain()")
	}
	if _, ok := cmd().(searchAgainMsg); !ok {
		t.Errorf("Update(Enter) cmd produced %T, want searchAgainMsg", cmd())
	}
}

func TestResultModel_Update_QQuits(t *testing.T) {
	m := newResultModel(testStatBlock(), nil, "", nil)

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
	m := newResultModel(testStatBlock(), nil, "", nil)

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if cmd != nil {
		t.Errorf(`Update("x") returned a non-nil cmd, want nil`)
	}
	if got.View() != m.View() {
		t.Errorf(`Update("x") changed the model, want it unchanged`)
	}
}
