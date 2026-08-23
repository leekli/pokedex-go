package pokemon

import "math"

// Conversion factors used to translate PokeAPI's metric-derived raw units
// (decimetres, hectograms) into the imperial units the original English
// Game Boy games displayed to players.
const (
	metersToFeet     = 3.28084
	kilogramsToPound = 2.20462
)

// DecimetresToFeetInches converts a PokeAPI height value (decimetres) into
// whole feet and inches, rounding to the nearest inch and carrying a
// rounded-up 12 into an extra foot.
func DecimetresToFeetInches(decimetres int) (feet, inches int) {
	meters := float64(decimetres) / 10.0
	totalFeet := meters * metersToFeet

	feet = int(math.Trunc(totalFeet))
	remainder := totalFeet - float64(feet)
	inches = int(math.Round(remainder * 12))

	if inches == 12 {
		feet++
		inches = 0
	}
	return feet, inches
}

// HectogramsToPounds converts a PokeAPI weight value (hectograms) into
// pounds, rounded to one decimal place.
func HectogramsToPounds(hectograms int) float64 {
	kilograms := float64(hectograms) / 10.0
	pounds := kilograms * kilogramsToPound
	return math.Round(pounds*10) / 10
}
