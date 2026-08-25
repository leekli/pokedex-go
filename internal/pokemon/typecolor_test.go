package pokemon

import "testing"

func TestTypeColor_AllEighteenTypesPresent(t *testing.T) {
	allTypes := AllTypes()
	if len(allTypes) != 18 {
		t.Fatalf("AllTypes() returned %d types, want 18", len(allTypes))
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

// TestAllTypes_ReturnsIndependentCopy proves a caller mutating the returned
// slice can't corrupt AllTypes' own backing data for later callers.
func TestAllTypes_ReturnsIndependentCopy(t *testing.T) {
	got := AllTypes()
	got[0] = "corrupted"

	if again := AllTypes(); again[0] == "corrupted" {
		t.Error("mutating a previous AllTypes() result affected a later call; want an independent copy each time")
	}
}
