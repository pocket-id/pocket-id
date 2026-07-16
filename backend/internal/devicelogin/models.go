package devicelogin

import (
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type RequestStatus string

const (
	RequestStatusPending  RequestStatus = "pending"
	RequestStatusApproved RequestStatus = "approved"
	RequestStatusDenied   RequestStatus = "denied"
)

type Request struct {
	model.Base

	Code            string
	DeviceTokenHash string
	Status          RequestStatus
	ExpiresAt       datatype.DateTime
	IpAddress       string
	UserAgent       string

	UserID *string
	User   model.User
}

func (Request) TableName() string {
	return "device_login_requests"
}
