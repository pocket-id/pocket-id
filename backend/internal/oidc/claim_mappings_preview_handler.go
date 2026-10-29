package oidc

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type claimMappingsPreviewHandler struct {
	claimsService *ClaimsService
}

type claimMappingsPreviewInput struct {
	UserID        string                                   `json:"userId" binding:"required"`
	Scopes        []string                                 `json:"scopes"`
	ClaimMappings []claimmappingpolicy.OidcClaimMappingDto `json:"claimMappings" binding:"dive"`
}

// previewClaimMappings godoc
// @Summary Preview claim mappings
// @Description Get the claims that claim mappings, saved or not, release for a user and set of scopes
// @Tags OIDC Claim OidcClaimMapping OidcClaimMappingPolicy
// @Accept json
// @Produce json
// @Param preview body claimMappingsPreviewInput true "User, scopes and claim mappings to preview"
// @Success 200 {object} ClaimMappingsPreview "Mapped claims per token type"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/preview [post]
func (h *claimMappingsPreviewHandler) previewClaimMappings(c *gin.Context) error {
	var input claimMappingsPreviewInput
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	preview, err := h.claimsService.PreviewClaimMappings(c.Request.Context(), input.UserID, input.Scopes, claimmappingpolicy.ClaimMappingsFromDto(input.ClaimMappings))
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, preview)
	return nil
}
