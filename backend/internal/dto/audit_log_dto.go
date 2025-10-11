package dto

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type AuditLogDto struct {
	ID        string            `json:"id"`
	CreatedAt datatype.DateTime `json:"createdAt"`

	Event     string            `json:"event"`
	IpAddress string            `json:"ipAddress"`
	Country   string            `json:"country"`
	City      string            `json:"city"`
	Device    string            `json:"device"`
	UserID    string            `json:"userID"`
	Username  string            `json:"username"`
	Data      map[string]string `json:"data"`
}
