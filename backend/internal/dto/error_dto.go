package dto

import "github.com/pocket-id/pocket-id/backend/internal/apperror"

// ErrorDto is the response body returned for every failed request.
type ErrorDto struct {
	Error     string         `json:"error"`
	Code      apperror.Code  `json:"code"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}
