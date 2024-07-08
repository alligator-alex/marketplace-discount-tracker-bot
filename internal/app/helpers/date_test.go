package helpers_test

import (
	"bot/internal/app/helpers"
	"testing"
	"time"
)

func TestTimeToHuman(t *testing.T) {
	time := time.Now()
	target := time.Format(helpers.DateHuman)

	result := helpers.TimeToHuman(time)
	if result != target {
		t.Errorf("Invalid result, got: %s, instead of: %s.", result, target)
	}
}

func TestTimeToDatabase(t *testing.T) {
	time := time.Now()
	target := time.Format(helpers.DateDatabase)

	result := helpers.TimeToDatabase(time)
	if result != target {
		t.Errorf("Invalid result, got: %s, instead of: %s.", result, target)
	}
}
