package dto

import "github.com/pocket-id/pocket-id/backend/internal/utils"

type Pagination = utils.PaginationResponse

type Paginated[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}
