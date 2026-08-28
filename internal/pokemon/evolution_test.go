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
