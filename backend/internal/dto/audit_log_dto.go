package dto

import (
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type AuditLogDto struct {
	ID        string            `json:"id"`
	CreatedAt datatype.DateTime `json:"createdAt"`

	Event     model.AuditLogEvent `json:"event"`
	IpAddress string              `json:"ipAddress"`
	Country   string              `json:"country"`
	City      string              `json:"city"`
	Device    string              `json:"device"`
	UserID    string              `json:"userID"`
	Username  string              `json:"username"`
	Data      model.AuditLogData  `json:"data"`
}

type AuditLogFilterDto struct {
	UserID     string `form:"filters[userId]"`
	Event      string `form:"filters[event]"`
	ClientName string `form:"filters[clientName]"`
}
