package pokemon

import "testing"

func TestDecimetresToFeetInches(t *testing.T) {
	tests := []struct {
		name       string
		decimetres int
		wantFeet   int
		wantInches int
	}{
		// 0.4m * 3.28084 = 1.312336ft -> 1ft, 0.312336*12 = 3.748 -> rounds to 4in.
		{"4 decimetres", 4, 1, 4},
		// 2.0m * 3.28084 = 6.56168ft -> 6ft, 0.56168*12 = 6.740 -> rounds to 7in.
		{"20 decimetres", 20, 6, 7},
		// 0.3m * 3.28084 = 0.984252ft -> 0ft, 0.984252*12 = 11.811 -> rounds to 12in,
		// which carries into an extra foot.
		{"rounds up into an extra foot", 3, 1, 0},
		{"zero height", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feet, inches := DecimetresToFeetInches(tt.decimetres)
			if feet != tt.wantFeet || inches != tt.wantInches {
				t.Errorf("DecimetresToFeetInches(%d) = %d'%02d\", want %d'%02d\"",
					tt.decimetres, feet, inches, tt.wantFeet, tt.wantInches)
			}
		})
	}
}

func TestHectogramsToPounds(t *testing.T) {
	tests := []struct {
		name       string
		hectograms int
		wantPounds float64
	}{
		// 6.0kg * 2.20462 = 13.22772 -> rounds to 13.2.
		{"60 hectograms", 60, 13.2},
		// 122.0kg * 2.20462 = 268.96364 -> rounds to 269.0.
		{"1220 hectograms", 1220, 269.0},
		{"zero weight", 0, 0},
		// 1.0kg * 2.20462 = 2.20462 -> rounds to 2.2.
		{"10 hectograms", 10, 2.2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HectogramsToPounds(tt.hectograms)
			if got != tt.wantPounds {
				t.Errorf("HectogramsToPounds(%d) = %v, want %v", tt.hectograms, got, tt.wantPounds)
			}
		})
	}
}
