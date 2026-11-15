package scimsync

import (
	"time"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// ScimServiceProviderDTO leaves out the token because it lets whoever holds it manage the users of the downstream app
type ScimServiceProviderDTO struct {
	ID           string             `json:"id"`
	Endpoint     string             `json:"endpoint"`
	HasToken     bool               `json:"hasToken"`
	LastSyncedAt *datatype.DateTime `json:"lastSyncedAt"`
	CreatedAt    datatype.DateTime  `json:"createdAt"`
}

type ScimServiceProviderCreateDTO struct {
	Endpoint string `json:"endpoint" binding:"required,url"`
	// Token is a pointer so an update can keep the stored token when it's omitted and remove it when it's empty
	Token        *string `json:"token"`
	OidcClientID string  `json:"oidcClientId" binding:"required"`
}

type ScimUser struct {
	ScimResourceData
	UserName string      `json:"userName"`
	Name     *ScimName   `json:"name,omitempty"`
	Display  string      `json:"displayName,omitempty"`
	Active   bool        `json:"active"`
	Emails   []ScimEmail `json:"emails,omitempty"`
}

type ScimName struct {
	GivenName  string `json:"givenName,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
}

type ScimEmail struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary,omitempty"`
}

type ScimGroup struct {
	ScimResourceData
	Display string            `json:"displayName"`
	Members []ScimGroupMember `json:"members,omitempty"`
}

type ScimGroupMember struct {
	Value string `json:"value"`
}

type ScimListResponse[T any] struct {
	Resources    []T `json:"Resources"`
	TotalResults int `json:"totalResults"`
	StartIndex   int `json:"startIndex"`
	ItemsPerPage int `json:"itemsPerPage"`
}

type ScimResourceData struct {
	ID         string            `json:"id,omitempty"`
	ExternalID string            `json:"externalId,omitempty"`
	Schemas    []string          `json:"schemas"`
	Meta       *ScimResourceMeta `json:"meta,omitempty"`
}

type ScimResourceMeta struct {
	Location     string    `json:"location,omitempty"`
	ResourceType string    `json:"resourceType,omitempty"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified,omitempty"`
	Version      string    `json:"version,omitempty"`
}

func (r ScimResourceData) GetID() string {
	return r.ID
}

func (r ScimResourceData) GetExternalID() string {
	return r.ExternalID
}

func (r ScimResourceData) GetSchemas() []string {
	return r.Schemas
}

type ScimResource interface {
	GetID() string
	GetExternalID() string
	GetSchemas() []string
}
