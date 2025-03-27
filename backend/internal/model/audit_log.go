package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type AuditLog struct {
	Base

	Event     AuditLogEvent `sortable:"true"`
	IpAddress string        `sortable:"true"`
	Country   string        `sortable:"true"`
	City      string        `sortable:"true"`
	UserAgent string        `sortable:"true"`
	UserID    string
	Data      AuditLogData
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

func (e *AuditLogEvent) Scan(value interface{}) error {
	*e = AuditLogEvent(value.(string))
	return nil
}

func (e AuditLogEvent) Value() (driver.Value, error) {
	return string(e), nil
}

func (d *AuditLogData) Scan(value interface{}) error {
	if v, ok := value.([]byte); ok {
		return json.Unmarshal(v, d)
	} else {
		return errors.New("type assertion to []byte failed")
	}
}

func (d *AuditLogData) Value() (driver.Value, error) {
	return json.Marshal(d)
}
