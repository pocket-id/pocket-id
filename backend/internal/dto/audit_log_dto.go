package dto

import (
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type AuditLogDto struct {
	ID        string            `json:"id"`
	CreatedAt datatype.DateTime `json:"createdAt"`

	Event     model.AuditLogEvent `json:"event"`
	IpAddress string              `json:"ipAddress,omitempty"`
	Country   string              `json:"country,omitempty"`
	City      string              `json:"city,omitempty"`
	Device    string              `json:"device,omitempty"`
	UserID    string              `json:"userID"`
	Username  string              `json:"username"`
	Data      model.AuditLogData  `json:"data,omitempty"`
}

type AuditLogFilterDto struct {
	UserID     string `form:"filters[userId]"`
	Event      string `form:"filters[event]"`
	ClientName string `form:"filters[clientName]"`
	Location   string `form:"filters[location]"`
}
