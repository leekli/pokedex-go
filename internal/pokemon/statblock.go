package pokemon

// StatBlock is the full set of fields shown on the Result Screen for a
// looked-up Pokémon. Height and weight are already converted to imperial
// units; base stats use the modern Special Attack / Special Defense split.
//
// StatBlock itself is a plain data holder — the mapping from raw PokeAPI
// responses into a StatBlock lives in the pokeapi package (which already
// depends on this package for Query, so building the mapping here would
// create an import cycle).
type StatBlock struct {
	DexNumber      int
	Name           string
	Types          []string
	HeightFeet     int
	HeightInches   int
	WeightPounds   float64
	HP             int
	Attack         int
	Defense        int
	Speed          int
	SpecialAttack  int
	SpecialDefense int
}
