package api

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
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

// apiCreateDto is the payload for creating an API
// The resource identifier is only accepted here because changing it later would invalidate every token already minted for the API
type apiCreateDto struct {
	Name     string `json:"name" required:"true" minLength:"1" maxLength:"50" unorm:"nfc"`
	Resource string `json:"resource" required:"true" maxLength:"350" unorm:"nfc"`
}

// apiUpdateDto is the payload for updating an API
// The resource identifier is intentionally not updatable
type apiUpdateDto struct {
	Name string `json:"name" required:"true" minLength:"1" maxLength:"50" unorm:"nfc"`
}

type apiPermissionInputDto struct {
	Key         string  `json:"key" required:"true" minLength:"1" maxLength:"128" unorm:"nfc"`
	Name        string  `json:"name" required:"true" minLength:"1" maxLength:"50" unorm:"nfc"`
	Description *string `json:"description" required:"false" maxLength:"200"`
}

// apiPermissionsUpdateDto replaces the full permission set of an API
type apiPermissionsUpdateDto struct {
	Permissions []apiPermissionInputDto `json:"permissions" required:"false"`
}

// clientApiAccessDto is the set of API permissions a client is allowed to request, split by subject type
// User-delegated permissions may be requested on behalf of a signed-in user, client permissions may be obtained by the client itself through the client credentials grant
type clientApiAccessDto struct {
	UserDelegatedPermissionIDs []string `json:"userDelegatedPermissionIds"`
	ClientPermissionIDs        []string `json:"clientPermissionIds"`
}

type clientApiAccessUpdateDto struct {
	UserDelegatedPermissionIDs []string `json:"userDelegatedPermissionIds" required:"false"`
	ClientPermissionIDs        []string `json:"clientPermissionIds" required:"false"`
}

func (d *apiCreateDto) Resolve(huma.Context) []error {
	if dto.ValidateResourceURI(d.Resource) {
		return nil
	}
	return []error{&huma.ErrorDetail{Location: "body.resource", Message: "Resource must be an absolute URI without whitespace or a fragment"}}
}

func (d *clientApiAccessUpdateDto) Resolve(huma.Context) []error {
	var errs []error
	for _, id := range d.UserDelegatedPermissionIDs {
		if id == "" {
			errs = append(errs, &huma.ErrorDetail{Location: "body.userDelegatedPermissionIds", Message: "Permission IDs cannot be empty"})
		}
	}
	for _, id := range d.ClientPermissionIDs {
		if id == "" {
			errs = append(errs, &huma.ErrorDetail{Location: "body.clientPermissionIds", Message: "Permission IDs cannot be empty"})
		}
	}
	return errs
}
