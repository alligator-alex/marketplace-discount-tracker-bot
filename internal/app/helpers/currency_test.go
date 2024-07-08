package helpers_test

import (
	"bot/internal/app/helpers"
	"testing"
)

func TestCurrencyToMinor(t *testing.T) {
	price := 220.50
	target := 22050

	result := helpers.CurrencyToMinor(price)
	if result != target {
		t.Errorf("Invalid result, got: %d, instead of: %d.", result, target)
	}
}

func TestCurrencyToMajor(t *testing.T) {
	price := 22050
	target := 220.50

	result := helpers.CurrencyToMajor(price)
	if result != target {
		t.Errorf("Invalid result, got: %.2f, instead of: %.2f.", result, target)
	}
}

func TestCurrencyFormat(t *testing.T) {
	price := 220.50000
	target := "220.50\u00A0" + helpers.RubleSymbol

	result := helpers.CurrencyFormat(price)
	if result != target {
		t.Errorf("Invalid result, got: %s, instead of: %s.", result, target)
	}
}
