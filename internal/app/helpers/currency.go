package helpers

import (
	"math"
	"strconv"
)

const (
	CurrencySubunit int    = 100
	RubleSymbol     string = "₽"
)

// Convert major currency to minor (e.g. 220.50 to 22050).
func CurrencyToMinor(majorValue float64) int {
	return int(math.Floor(majorValue * float64(CurrencySubunit)))
}

// Convert minor currency to major (e.g. 22050 to 220.50).
func CurrencyToMajor(minorValue int) float64 {
	value := float64(minorValue) / float64(CurrencySubunit)

	return math.Round(value*100) / 100
}

// Format major currency as string (e.g. 220.50000 to "220.50 ₽").
func CurrencyFormat(majorValue float64) string {
	result := strconv.FormatFloat(majorValue, 'f', -1, 64)

	return ConcatStrings(result, "\u00A0", RubleSymbol)
}
