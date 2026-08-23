package pokemon

import "testing"

func TestTypeColor_AllEighteenTypesPresent(t *testing.T) {
	allTypes := []string{
		"normal", "fire", "water", "electric", "grass", "ice",
		"fighting", "poison", "ground", "flying", "psychic", "bug",
		"rock", "ghost", "dragon", "dark", "steel", "fairy",
	}
	if len(allTypes) != 18 {
		t.Fatalf("test setup error: expected 18 types, got %d", len(allTypes))
	}

	seen := make(map[string]bool)
	for _, typ := range allTypes {
		color := TypeColor(typ)
		if color == defaultTypeColor {
			t.Errorf("TypeColor(%q) fell back to defaultTypeColor, want a dedicated color", typ)
		}
		if seen[string(color)] {
			t.Errorf("TypeColor(%q) reuses a color already assigned to another type: %v", typ, color)
		}
		seen[string(color)] = true
	}
}

func TestTypeColor_UnknownTypeFallsBackSafely(t *testing.T) {
	got := TypeColor("not-a-real-type")
	if got != defaultTypeColor {
		t.Errorf("TypeColor(unknown) = %v, want defaultTypeColor %v", got, defaultTypeColor)
	}
}
