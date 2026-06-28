package api

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// apiResponseDto is the full representation of an API including its permissions
type apiResponseDto struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Resource    string                     `json:"resource"`
	CreatedAt   datatype.DateTime          `json:"createdAt"`
	Permissions []apiPermissionResponseDto `json:"permissions"`
}

type apiPermissionResponseDto struct {
	ID          string  `json:"id"`
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// apiListItemDto is the lightweight representation used in list responses
type apiListItemDto struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Resource        string            `json:"resource"`
	CreatedAt       datatype.DateTime `json:"createdAt"`
	PermissionCount int               `json:"permissionCount"`
}

// apiCreateDto is the payload for creating an API
// The resource identifier is only accepted here because changing it later would invalidate every token already minted for the API
type apiCreateDto struct {
	Name     string `json:"name" binding:"required,min=1,max=50" unorm:"nfc"`
	Resource string `json:"resource" binding:"required,uri,max=350" unorm:"nfc"`
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

// clientApiAccessDto is the set of API permissions a client is allowed to request
type clientApiAccessDto struct {
	AllowedPermissionIDs []string `json:"allowedPermissionIds"`
}

type clientApiAccessUpdateDto struct {
	AllowedPermissionIDs []string `json:"allowedPermissionIds" binding:"omitempty,dive,required"`
}
