package webauthn

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// WebauthnSession holds the transient challenge state for an in-progress registration or login
type WebauthnSession struct {
	model.Base

	Challenge        string
	ExpiresAt        datatype.DateTime
	UserVerification string
	CredentialParams CredentialParameters
}

// ReauthenticationToken is a short-lived token proving a user recently re-verified themselves
type ReauthenticationToken struct {
	model.Base
	Token     string
	ExpiresAt datatype.DateTime

	UserID string
	User   model.User
}

// PublicKeyCredentialCreationOptions is the registration challenge returned to the browser
type PublicKeyCredentialCreationOptions struct {
	Response  protocol.PublicKeyCredentialCreationOptions
	SessionID string
	Timeout   time.Duration
}

// PublicKeyCredentialRequestOptions is the login challenge returned to the browser
type PublicKeyCredentialRequestOptions struct {
	Response  protocol.PublicKeyCredentialRequestOptions
	SessionID string
	Timeout   time.Duration
}

type CredentialParameters []protocol.CredentialParameter //nolint:recvcheck

// Scan and Value methods for GORM to handle the custom type
func (cp *CredentialParameters) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(cp, value)
}

func (cp CredentialParameters) Value() (driver.Value, error) {
	return json.Marshal(cp)
}
