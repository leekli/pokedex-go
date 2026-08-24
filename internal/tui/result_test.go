package tui

import (
	"image"
	"image/color"
	"strings"
	"testing"

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
	m := newResultModel(testStatBlock(), img)
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
	m := newResultModel(testStatBlock(), nil)
	view := m.View()

	if !strings.Contains(view, "No sprite available") {
		t.Errorf("resultModel.View() with a nil sprite missing fallback message\ngot:\n%s", view)
	}
	if !strings.Contains(view, "#006 Charizard") || !strings.Contains(view, "HP") {
		t.Errorf("resultModel.View() with a nil sprite should still show the rest of the stat block\ngot:\n%s", view)
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
