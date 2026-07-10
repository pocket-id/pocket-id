package humautils

import "github.com/pocket-id/pocket-id/backend/internal/utils"

// EmptyInput represents an operation without path, query, header, or body input
type EmptyInput struct{}

// EmptyOutput represents an operation without a response body or headers
type EmptyOutput struct{}

// BodyOutput wraps a typed response body for Huma
type BodyOutput[T any] struct {
	Body T
}

// ListInput exposes the shared list query parameters to Huma
type ListInput struct {
	utils.ListRequestOptions
}
