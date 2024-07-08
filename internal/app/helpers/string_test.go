package helpers_test

import (
	"bot/internal/app/helpers"
	"testing"
)

func TestConcatStrings(t *testing.T) {
	target := "Dr. Isaac Kleiner"

	result := helpers.ConcatStrings("Dr.", " ", "Isaac", " ", "Kleiner")
	if result != target {
		t.Errorf("Invalid result, got: %s, instead of: %s.", result, target)
	}
}

func TestGetRandomString(t *testing.T) {
	stringA := helpers.GetRandomString()
	stringB := helpers.GetRandomString()

	if stringA == stringB {
		t.Errorf("Invalid result, strings are equal")
	}
}
