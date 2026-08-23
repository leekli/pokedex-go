package pokemon

import "testing"

func TestResolveQuery(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Query
	}{
		{"simple dex number", "25", Query{DexNumber, "25"}},
		{"dex number with whitespace", "  250  ", Query{DexNumber, "250"}},
		{"leading zeros preserved as digits", "007", Query{DexNumber, "007"}},
		{"simple name", "pikachu", Query{Name, "pikachu"}},
		{"name with trailing digit is still a name", "porygon2", Query{Name, "porygon2"}},
		{"name with space is normalized", "Mr Mime", Query{Name, "mr-mime"}},
		{"mixed alnum with hyphen is a name, not a dex number", "porygon-z", Query{Name, "porygon-z"}},
		{"empty input is a name", "", Query{Name, ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveQuery(tt.input)
			if got != tt.want {
				t.Errorf("ResolveQuery(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}
