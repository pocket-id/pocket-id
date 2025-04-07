package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type AuditLog struct {
	Base

	Event     AuditLogEvent `sortable:"true"`
	IpAddress string        `sortable:"true"`
	Country   string        `sortable:"true"`
	City      string        `sortable:"true"`
	UserAgent string        `sortable:"true"`
	Username  string        `gorm:"-"`
	Data      AuditLogData

	UserID string
	User   User
}

type AuditLogData map[string]string //nolint:recvcheck

type AuditLogEvent string //nolint:recvcheck

const (
	AuditLogEventSignIn                   AuditLogEvent = "SIGN_IN"
	AuditLogEventOneTimeAccessTokenSignIn AuditLogEvent = "TOKEN_SIGN_IN"
	AuditLogEventClientAuthorization      AuditLogEvent = "CLIENT_AUTHORIZATION"
	AuditLogEventNewClientAuthorization   AuditLogEvent = "NEW_CLIENT_AUTHORIZATION"
)

// Scan and Value methods for GORM to handle the custom type

func (e *AuditLogEvent) Scan(value any) error {
	*e = AuditLogEvent(value.(string))
	return nil
}

func (e AuditLogEvent) Value() (driver.Value, error) {
	return string(e), nil
}

func (d *AuditLogData) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, d)
	case string:
		return json.Unmarshal([]byte(v), d)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

func (d AuditLogData) Value() (driver.Value, error) {
	return json.Marshal(d)
}
