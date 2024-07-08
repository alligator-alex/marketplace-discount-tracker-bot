package core_test

import (
	"bot/internal/app/core"
	"testing"
)

func TestPaginatedResult(t *testing.T) {
	items := []any{
		"one",
		"two",
		"three",
	}

	var result core.PaginatedResult

	result = core.NewPaginatedResult(items, 1, 10, 800)
	if result.PerPage != 10 {
		t.Errorf("Invalid per page, got: %d, instead of: %d.", result.PerPage, 10)
	}

	if result.Total != 800 {
		t.Errorf("Invalid total, got: %d, instead of: %d.", result.Total, 800)
	}

	if result.LastPage != 80 {
		t.Errorf("Invalid last page, got: %d, instead of: %d.", result.LastPage, 80)
	}

	result = core.NewPaginatedResult(items, 80, 10, 800)
	if !result.IsLastPage() {
		t.Errorf("Not last page")
	}
}
