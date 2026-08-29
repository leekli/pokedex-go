package pokemon

import "testing"

// TestDescribeEvolutionCondition covers every condition case the Result
// Screen's Evolution Chain is expected to describe, including the real
// examples that motivated each branch (Pichu -> Pikachu, Pikachu ->
// Raichu, Machoke -> Machamp, Feebas -> Milotic, ...) - see
// docs/adr/0006-scope-evolution-condition-text-to-common-cases.md.
func TestDescribeEvolutionCondition(t *testing.T) {
	tests := []struct {
		name string
		c    EvolutionCondition
		want string
	}{
		{
			name: "plain level (Charmeleon -> Charizard)",
			c:    EvolutionCondition{Trigger: "level-up", Level: 36},
			want: "Lv. 36",
		},
		{
			name: "level with high friendship required at the same time",
			c:    EvolutionCondition{Trigger: "level-up", Level: 25, Happiness: true},
			want: "Lv. 25, high friendship",
		},
		{
			name: "level with a time-of-day restriction (Eevee -> Espeon)",
			c:    EvolutionCondition{Trigger: "level-up", Level: 1, TimeOfDay: "day"},
			want: "Lv. 1 (day)",
		},
		{
			name: "high friendship alone, no level (Pichu -> Pikachu)",
			c:    EvolutionCondition{Trigger: "level-up", Happiness: true},
			want: "high friendship",
		},
		{
			name: "high friendship with a time-of-day restriction (Eevee -> Espeon/Umbreon)",
			c:    EvolutionCondition{Trigger: "level-up", Happiness: true, TimeOfDay: "night"},
			want: "high friendship (night)",
		},
		{
			name: "high beauty (Feebas -> Milotic)",
			c:    EvolutionCondition{Trigger: "level-up", Beauty: true},
			want: "high beauty",
		},
		{
			name: "item use (Pikachu -> Raichu)",
			c:    EvolutionCondition{Trigger: "use-item", Item: "thunder-stone"},
			want: "use Thunder Stone",
		},
		{
			name: "plain trade, no held item or species (Kadabra -> Alakazam)",
			c:    EvolutionCondition{Trigger: "trade"},
			want: "trade",
		},
		{
			name: "trade holding an item (Feebas -> Milotic in Black/White)",
			c:    EvolutionCondition{Trigger: "trade", HeldItem: "prism-scale"},
			want: "trade holding Prism Scale",
		},
		{
			name: "trade for a specific species (Karrablast <-> Shelmet)",
			c:    EvolutionCondition{Trigger: "trade", TradeSpecies: "shelmet"},
			want: "trade for Shelmet",
		},
		{
			name: "trade species takes priority over a held item if both are somehow set",
			c:    EvolutionCondition{Trigger: "trade", TradeSpecies: "shelmet", HeldItem: "prism-scale"},
			want: "trade for Shelmet",
		},
		{
			name: "known move (Piloswine -> Mamoswine)",
			c:    EvolutionCondition{Trigger: "level-up", KnownMove: "ancient-power"},
			want: "knows Ancient Power",
		},
		{
			name: "time of day alone, no level (Rockruff -> Lycanroc, one form)",
			c:    EvolutionCondition{Trigger: "level-up", TimeOfDay: "day"},
			want: "level up (day)",
		},
		{
			name: "shed trigger (Nincada -> Shedinja)",
			c:    EvolutionCondition{Trigger: "shed"},
			want: "spare Poké Ball & party space",
		},
		{
			name: "unrecognized trigger falls back rather than guessing",
			c:    EvolutionCondition{Trigger: "three-critical-hits"},
			want: "special condition",
		},
		{
			name: "zero-value condition (no fields set at all) also falls back",
			c:    EvolutionCondition{},
			want: "special condition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DescribeEvolutionCondition(tt.c); got != tt.want {
				t.Errorf("DescribeEvolutionCondition(%+v) = %q, want %q", tt.c, got, tt.want)
			}
		})
	}
}

// pichuPikachuRaichuChain mirrors the real Pichu -> Pikachu -> Raichu family,
// used by both TestEvolutionChain_PathTo and TestEvolutionChain_DoesNotEvolve.
func pichuPikachuRaichuChain() EvolutionChain {
	return EvolutionChain{
		Root: EvolutionStage{
			DexNumber: 172,
			Name:      "pichu",
			EvolvesTo: []EvolutionStage{
				{
					DexNumber: 25,
					Name:      "pikachu",
					Condition: "high friendship",
					EvolvesTo: []EvolutionStage{
						{DexNumber: 26, Name: "raichu", Condition: "use Thunder Stone"},
					},
				},
			},
		},
	}
}

// TestEvolutionChain_PathTo_MidChain proves looking up a middle stage
// (Pikachu) returns the path down to it (Pichu, Pikachu) plus what it still
// evolves into (Raichu) - not the full tree, and not stages beyond Raichu.
func TestEvolutionChain_PathTo_MidChain(t *testing.T) {
	chain := pichuPikachuRaichuChain()

	path := chain.PathTo(25)

	if len(path.Stages) != 2 || path.Stages[0].Name != "pichu" || path.Stages[1].Name != "pikachu" {
		t.Errorf("PathTo(25).Stages = %+v, want [pichu, pikachu]", path.Stages)
	}
	if len(path.EvolvesTo) != 1 || path.EvolvesTo[0].Name != "raichu" {
		t.Errorf("PathTo(25).EvolvesTo = %+v, want [raichu]", path.EvolvesTo)
	}
}

// TestEvolutionChain_PathTo_UnknownDexNumber proves a dex number absent from
// the chain returns the zero value rather than panicking.
func TestEvolutionChain_PathTo_UnknownDexNumber(t *testing.T) {
	chain := pichuPikachuRaichuChain()

	path := chain.PathTo(999)

	if len(path.Stages) != 0 {
		t.Errorf("PathTo(999).Stages = %+v, want empty for a dex number not in the chain", path.Stages)
	}
}

// TestEvolutionChain_PathTo_Branching proves a species with multiple
// evolutions (Eevee) returns every sibling in EvolvesTo, not just the first.
func TestEvolutionChain_PathTo_Branching(t *testing.T) {
	chain := EvolutionChain{
		Root: EvolutionStage{
			DexNumber: 133,
			Name:      "eevee",
			EvolvesTo: []EvolutionStage{
				{DexNumber: 134, Name: "vaporeon", Condition: "use Water Stone"},
				{DexNumber: 135, Name: "jolteon", Condition: "use Thunder Stone"},
			},
		},
	}

	path := chain.PathTo(133)

	if len(path.EvolvesTo) != 2 {
		t.Errorf("PathTo(133).EvolvesTo = %+v, want both vaporeon and jolteon", path.EvolvesTo)
	}
}

// TestEvolutionPath_DoesNotEvolve proves a single-stage chain with no
// EvolvesTo (e.g. Tauros) reports DoesNotEvolve, while any other path
// (mid-chain, or a leaf reached via PathTo) does not.
func TestEvolutionPath_DoesNotEvolve(t *testing.T) {
	tauros := EvolutionChain{Root: EvolutionStage{DexNumber: 128, Name: "tauros"}}
	if !tauros.PathTo(128).DoesNotEvolve() {
		t.Error("PathTo(128).DoesNotEvolve() = false for a single-stage chain with no evolutions, want true")
	}

	chain := pichuPikachuRaichuChain()
	if chain.PathTo(25).DoesNotEvolve() {
		t.Error("PathTo(25).DoesNotEvolve() = true for Pikachu (still evolves into Raichu), want false")
	}
	if chain.PathTo(26).DoesNotEvolve() {
		t.Error("PathTo(26).DoesNotEvolve() = true for a leaf reached via a multi-stage path, want false (only a single-stage root with no evolutions counts)")
	}
}

func TestHumanizeSlug(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{"thunder-stone", "Thunder Stone"},
		{"prism-scale", "Prism Scale"},
		{"shelmet", "Shelmet"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := humanizeSlug(tt.slug); got != tt.want {
			t.Errorf("humanizeSlug(%q) = %q, want %q", tt.slug, got, tt.want)
		}
	}
}
