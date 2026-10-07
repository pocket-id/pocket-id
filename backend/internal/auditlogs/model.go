package auditlogs

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type AuditLog struct {
	model.Base

	Event     Event   `sortable:"true" filterable:"true"`
	IpAddress *string `sortable:"true"`
	Country   string  `sortable:"true"`
	City      string  `sortable:"true"`
	UserAgent string  `sortable:"true"`
	Username  string  `gorm:"-"`
	Data      Data

	UserID string `filterable:"true"`
	User   model.User
}

type Data map[string]string //nolint:recvcheck

type Event string //nolint:recvcheck

const (
	EventSignIn                     Event = "SIGN_IN"
	EventOneTimeAccessTokenSignIn   Event = "TOKEN_SIGN_IN"
	EventRemoteSignIn               Event = "REMOTE_SIGN_IN"
	EventAccountCreated             Event = "ACCOUNT_CREATED"
	EventClientAuthorization        Event = "CLIENT_AUTHORIZATION"
	EventNewClientAuthorization     Event = "NEW_CLIENT_AUTHORIZATION"
	EventDeviceCodeAuthorization    Event = "DEVICE_CODE_AUTHORIZATION"
	EventNewDeviceCodeAuthorization Event = "NEW_DEVICE_CODE_AUTHORIZATION"
	EventPasskeyAdded               Event = "PASSKEY_ADDED"
	EventPasskeyRemoved             Event = "PASSKEY_REMOVED"
)

// Scan and Value methods for GORM to handle the custom type

func (e *Event) Scan(value any) error {
	*e = Event(value.(string))
	return nil
}

func (e Event) Value() (driver.Value, error) {
	return string(e), nil
}

func (d *Data) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(d, value)
}

func (d Data) Value() (driver.Value, error) {
	return json.Marshal(d)
}
