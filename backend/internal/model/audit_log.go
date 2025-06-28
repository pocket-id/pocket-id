package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type AuditLog struct {
	Base

	Event     AuditLogEvent `sortable:"true"`
	IpAddress *AuditLogIP   `sortable:"true"`
	Country   string        `sortable:"true"`
	City      string        `sortable:"true"`
	UserAgent string        `sortable:"true"`
	Username  string        `gorm:"-"`
	Data      AuditLogData

	UserID string
	User   User
}

type AuditLogData map[string]string //nolint:recvcheck

type AuditLogIP string //nolint:recvcheck

type AuditLogEvent string //nolint:recvcheck

const (
	AuditLogEventSignIn                     AuditLogEvent = "SIGN_IN"
	AuditLogEventOneTimeAccessTokenSignIn   AuditLogEvent = "TOKEN_SIGN_IN"
	AuditLogEventAccountCreated             AuditLogEvent = "ACCOUNT_CREATED"
	AuditLogEventClientAuthorization        AuditLogEvent = "CLIENT_AUTHORIZATION"
	AuditLogEventNewClientAuthorization     AuditLogEvent = "NEW_CLIENT_AUTHORIZATION"
	AuditLogEventDeviceCodeAuthorization    AuditLogEvent = "DEVICE_CODE_AUTHORIZATION"
	AuditLogEventNewDeviceCodeAuthorization AuditLogEvent = "NEW_DEVICE_CODE_AUTHORIZATION"
)

// Scan and Value methods for GORM to handle the custom type

func (e *AuditLogIP) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*e = ""
		return nil
	case []byte:
		if len(v) == 0 {
			*e = ""
		} else {
			*e = AuditLogIP(string(v))
		}
		return nil
	case string:
		if v == "" {
			*e = ""
		} else {
			*e = AuditLogIP(v)
		}
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

func (e AuditLogIP) Value() (driver.Value, error) {
	// ip_address is stored differently in Postgres and SQLite:
	// - Postgres: nullable INET column - empty values must be stored as NULL
	// - SQLite: non-nullable TEXT column (for historic reasons) - empty values are stored as empty strings
	switch common.EnvConfig.DbProvider {
	case common.DbProviderSqlite:
		if e == "" {
			return "", nil
		}
		return string(e), nil
	case common.DbProviderPostgres:
		if e == "" {
			return nil, nil
		}
		return string(e), nil
	default:
		return nil, fmt.Errorf("unsupported DB provider: %v", common.EnvConfig.DbProvider)
	}
}

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
