package api

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// apiResponseDto is the full representation of an API including its permissions
type apiResponseDto struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	Resource         string                     `json:"resource" copier:"Audience"`
	CreatedAt        datatype.DateTime          `json:"createdAt"`
	Permissions      []apiPermissionResponseDto `json:"permissions"`
	AllowCIMDClients bool                       `json:"allowCimdClients"`
}

type apiPermissionResponseDto struct {
	ID                    string  `json:"id"`
	Key                   string  `json:"key"`
	Name                  string  `json:"name"`
	Description           *string `json:"description,omitempty"`
	AllowedForCIMDClients bool    `json:"allowedForCimdClients"`
}

// apiCreateDto is the payload for creating an API
// The resource identifier is only accepted here because changing it later would invalidate every token already minted for the API
type apiCreateDto struct {
	Name     string `json:"name" binding:"required,min=1,max=50" unorm:"nfc"`
	Resource string `json:"resource" binding:"required,resource_uri,max=350" unorm:"nfc"`
}

// apiUpdateDto is the payload for updating an API
// The resource identifier is intentionally not updatable
type apiUpdateDto struct {
	Name string `json:"name" binding:"required,min=1,max=50" unorm:"nfc"`
}

type apiPermissionInputDto struct {
	Key         string  `json:"key" binding:"required,min=1,max=128" unorm:"nfc"`
	Name        string  `json:"name" binding:"required,min=1,max=50" unorm:"nfc"`
	Description *string `json:"description" binding:"omitempty,max=200"`
}

// apiPermissionsUpdateDto replaces the full permission set of an API
type apiPermissionsUpdateDto struct {
	Permissions []apiPermissionInputDto `json:"permissions" binding:"omitempty,dive"`
}

// apiCimdAccessUpdateDto replaces which permissions of an API every CIMD client may request
// The permission IDs are kept while Enabled is false, so switching access off and on again preserves the selection
type apiCimdAccessUpdateDto struct {
	Enabled       bool     `json:"enabled"`
	PermissionIDs []string `json:"permissionIds" binding:"omitempty,dive,required"`
}

// apiClientGrantDto is what one client may do with a single API
type apiClientGrantDto struct {
	UserDelegatedAccess        bool     `json:"userDelegatedAccess"`
	ClientAccess               bool     `json:"clientAccess"`
	UserDelegatedPermissionIDs []string `json:"userDelegatedPermissionIds"`
	ClientPermissionIDs        []string `json:"clientPermissionIds"`
}

type apiClientGrantUpdateDto struct {
	UserDelegatedAccess        bool     `json:"userDelegatedAccess"`
	ClientAccess               bool     `json:"clientAccess"`
	UserDelegatedPermissionIDs []string `json:"userDelegatedPermissionIds" binding:"omitempty,dive,required"`
	ClientPermissionIDs        []string `json:"clientPermissionIds" binding:"omitempty,dive,required"`
}

// apiClientAccessDto is one client's grants on a single API, as listed on the API's detail page
// The CIMD fields are read-only here: that access is managed on the API, not per client
type apiClientAccessDto struct {
	Client apiClientDto `json:"client"`
	apiClientGrantDto
	CIMDGrantedAccess        bool     `json:"cimdGrantedAccess"`
	CIMDGrantedPermissionIDs []string `json:"cimdGrantedPermissionIds"`
}

// clientApiGrantDto is one API a client may reach, as listed on the client's detail page
type clientApiGrantDto struct {
	API apiResponseDto `json:"api"`
	apiClientGrantDto
	CIMDGrantedAccess        bool     `json:"cimdGrantedAccess"`
	CIMDGrantedPermissionIDs []string `json:"cimdGrantedPermissionIds"`
}

// apiClientDto identifies an OIDC client with the few fields the API detail page needs to render a row
type apiClientDto struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ClientType  string `json:"clientType"`
	IsPublic    bool   `json:"isPublic"`
	HasLogo     bool   `json:"hasLogo"`
	HasDarkLogo bool   `json:"hasDarkLogo"`
}
