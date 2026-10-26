// Package models holds request/response types shared across HTTP handlers.
package models

var defaultLimit = 20

type CursorPaging struct {
	NextCursor string `query:"next_cursor" json:"next_cursor"`
	Limit      int    `query:"limit" json:"limit"`
}

func (p *CursorPaging) GetLimit() int {
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}

	return p.Limit
}

func (p *CursorPaging) GetCursor() string {
	return p.NextCursor
}

type Filter struct {
	Search string `query:"search" json:"search,omitempty"`
}

type PaginatedResponse[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
	Total      int    `json:"total,omitempty"`
}
