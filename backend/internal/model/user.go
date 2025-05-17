package model

import (
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type User struct {
	Base

	Username  string `sortable:"true"`
	Email     string `sortable:"true"`
	FirstName string `sortable:"true"`
	LastName  string `sortable:"true"`
	IsAdmin   bool   `sortable:"true"`
	Locale    *string
	LdapID    *string
	Disabled  bool `sortable:"true"`

	CustomClaims []CustomClaim
	UserGroups   []UserGroup `gorm:"many2many:user_groups_users;"`
	Credentials  []WebauthnCredential
}

func (u User) WebAuthnID() []byte { return []byte(u.ID) }

func (u User) WebAuthnName() string { return u.Username }

func (u User) WebAuthnDisplayName() string { return u.FirstName + " " + u.LastName }

func (u User) WebAuthnIcon() string { return "" }

func (u User) WebAuthnCredentials() []webauthn.Credential {
	credentials := make([]webauthn.Credential, len(u.Credentials))

	for i, credential := range u.Credentials {
		credentials[i] = webauthn.Credential{
			ID:              credential.CredentialID,
			AttestationType: credential.AttestationType,
			PublicKey:       credential.PublicKey,
			Transport:       credential.Transport,
			Flags: webauthn.CredentialFlags{
				BackupState:    credential.BackupState,
				BackupEligible: credential.BackupEligible,
			},
		}

	}
	return credentials
}

func (u User) WebAuthnCredentialDescriptors() (descriptors []protocol.CredentialDescriptor) {
	credentials := u.WebAuthnCredentials()

	descriptors = make([]protocol.CredentialDescriptor, len(credentials))

	for i, credential := range credentials {
		descriptors[i] = credential.Descriptor()
	}

	return descriptors
}

func (u User) FullName() string { return u.FirstName + " " + u.LastName }

func (u User) Initials() string {
	first := utils.GetFirstCharacter(u.FirstName)
	last := utils.GetFirstCharacter(u.LastName)
	if first == "" && last == "" && len(u.Username) >= 2 {
		return strings.ToUpper(u.Username[:2])
	}
	return strings.ToUpper(first + last)
}

type OneTimeAccessToken struct {
	Base
	Token     string
	ExpiresAt datatype.DateTime

	UserID string
	User   User
}

func NewOneTimeAccessToken(userID string, expiresAt time.Time) (*OneTimeAccessToken, error) {
	// If expires at is less than 15 minutes, use a 6-character token instead of 16
	tokenLength := 16
	if time.Until(expiresAt) <= 15*time.Minute {
		tokenLength = 6
	}

	randomString, err := utils.GenerateRandomAlphanumericString(tokenLength)
	if err != nil {
		return nil, err
	}

	o := &OneTimeAccessToken{
		UserID:    userID,
		ExpiresAt: datatype.DateTime(expiresAt),
		Token:     randomString,
	}

	return o, nil
}
