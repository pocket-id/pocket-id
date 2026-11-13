package oidc

import (
	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
)

type claimMappingsPreviewHandler struct {
	claimsService *ClaimsService
}

func (h *claimMappingsPreviewHandler) previewClaimMappings(c *gin.Context, input claimMappingsPreviewInput) (ClaimMappingsPreview, error) {
	// The route already requires users:read, and a route can only declare one scope, so the policy scope is checked here
	if !authz.PrincipalFrom(c).Scopes.Has(authz.OidcClaimMappingPoliciesRead) {
		return ClaimMappingsPreview{}, apperror.MissingScope(string(authz.OidcClaimMappingPoliciesRead))
	}

	err := claimmappingpolicy.ValidateClaimMappings(input.ClaimMappings)
	if err != nil {
		return ClaimMappingsPreview{}, err
	}

	preview, err := h.claimsService.PreviewClaimMappings(c.Request.Context(), input.UserID, input.Scopes, claimmappingpolicy.ClaimMappingsFromDto(input.ClaimMappings))
	if err != nil {
		return ClaimMappingsPreview{}, err
	}

	return *preview, nil
}
