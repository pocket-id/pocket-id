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

	// UserID is the user the entry is about, empty when it concerns no user, such as a client being created
	UserID string
	User   model.User

	ActorType ActorType
	// ActorID is the user or client that acted, empty for the other actor types
	ActorID string
	// ActorName is the actor's name at the time, so the entry keeps it after the actor is renamed or deleted
	ActorName string
}

// ActorType is who performed an action, empty when nobody was signed in, such as when requesting a login code
type ActorType string

const (
	ActorUser         ActorType = "user"
	ActorClient       ActorType = "client"
	ActorStaticAPIKey ActorType = "static_api_key"
	ActorLDAP         ActorType = "ldap"
	ActorCLI          ActorType = "cli"
	ActorSystem       ActorType = "system"
)

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
	EventLoginCodeCreated           Event = "LOGIN_CODE_CREATED"
	EventEmailChanged               Event = "EMAIL_CHANGED"
	EventEmailVerified              Event = "EMAIL_VERIFIED"
	EventAPIKeyCreated              Event = "API_KEY_CREATED" //nolint:gosec // G101 false positive: event names, not credentials
	EventAPIKeyRenewed              Event = "API_KEY_RENEWED" //nolint:gosec // G101 false positive: event names, not credentials
	EventAPIKeyRevoked              Event = "API_KEY_REVOKED" //nolint:gosec // G101 false positive: event names, not credentials
	EventClientSecretCreated        Event = "CLIENT_SECRET_CREATED"
	EventClientSecretDeleted        Event = "CLIENT_SECRET_DELETED"
	EventSessionRevoked             Event = "SESSION_REVOKED"
	EventSignedOut                  Event = "SIGNED_OUT"
	EventClientAuthorizationRevoked Event = "CLIENT_AUTHORIZATION_REVOKED"
	EventAccountDeleted             Event = "ACCOUNT_DELETED"
	EventAccountDisabled            Event = "ACCOUNT_DISABLED"
	EventAccountEnabled             Event = "ACCOUNT_ENABLED"
	EventAdminGranted               Event = "ADMIN_GRANTED"
	EventAdminRevoked               Event = "ADMIN_REVOKED"
	EventClientCreated              Event = "CLIENT_CREATED"
	EventClientUpdated              Event = "CLIENT_UPDATED"
	EventClientDeleted              Event = "CLIENT_DELETED"
	EventClientGroupAdded           Event = "CLIENT_GROUP_ADDED"
	EventClientGroupRemoved         Event = "CLIENT_GROUP_REMOVED"
	EventGroupCreated               Event = "GROUP_CREATED"
	EventGroupDeleted               Event = "GROUP_DELETED"
	EventGroupMemberAdded           Event = "GROUP_MEMBER_ADDED"
	EventGroupMemberRemoved         Event = "GROUP_MEMBER_REMOVED"
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
