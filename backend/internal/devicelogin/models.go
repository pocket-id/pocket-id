package devicelogin

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type RequestStatus string

const (
	RequestStatusPending  RequestStatus = "pending"
	RequestStatusApproved RequestStatus = "approved"
	RequestStatusDenied   RequestStatus = "denied"
)

type Request struct {
	ID        string
	Code      string
	Status    RequestStatus
	ExpiresAt datatype.DateTime
}
