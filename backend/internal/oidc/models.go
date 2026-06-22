package oidc

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// OAuth2Session is a fosite requester persisted by the Store. A single table holds
// all session kinds (authorization codes, tokens, PKCE, PAR, device codes, ...).
type OAuth2Session struct {
	model.Base

	Kind                 string
	Key                  string
	RequestID            string
	AccessTokenSignature string
	Active               bool
	RequestData          string
	ExpiresAt            *datatype.DateTime
}

func (OAuth2Session) TableName() string {
	return "oauth2_sessions"
}

// clientAssertionJTI stores consumed client assertion JWT IDs for replay protection.
type clientAssertionJTI struct {
	model.Base

	JTI       string            `gorm:"not null;uniqueIndex"`
	ExpiresAt datatype.DateTime `gorm:"not null;index"`
}

func (clientAssertionJTI) TableName() string {
	return "oauth2_jtis"
}

// InteractionSession tracks the user-facing steps (authentication, account selection,
// reauthentication, consent) that must be completed before an authorization request
// can be granted.
type InteractionSession struct {
	model.Base

	Scopes datatype.StringList

	ClientID string
	Client   model.OidcClient

	UserID *string
	User   model.User

	ConsentRequired          bool
	ReauthenticationRequired bool
	AuthenticationRequired   bool
	AccountSelectionRequired bool

	RequestedAt       datatype.DateTime
	ReauthenticatedAt *datatype.DateTime

	Parameters InteractionSessionParameters
}

type InteractionSessionParameters map[string]string //nolint:recvcheck

func (p *InteractionSessionParameters) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(p, value)
}

func (p InteractionSessionParameters) Value() (driver.Value, error) {
	return json.Marshal(p)
}
