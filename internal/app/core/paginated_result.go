package core

import "math"

type PaginatedResult struct {
	Items       []any
	CurrentPage int
	LastPage    int
	PerPage     int
	Total       int
}

func NewPaginatedResult(items []any, currentPage int, perPage int, total int) PaginatedResult {
	result := PaginatedResult{
		Items:       items,
		CurrentPage: currentPage,
		LastPage:    currentPage,
		PerPage:     perPage,
		Total:       total,
	}

	result.LastPage = result.getLastPage()

	return result
}

func (r *PaginatedResult) getLastPage() int {
	if r.Total == 0 {
		return 1
	}

	return int(math.Ceil(float64(r.Total) / float64(r.PerPage)))
}

func (r *PaginatedResult) IsLastPage() bool {
	return r.CurrentPage == r.LastPage
}
