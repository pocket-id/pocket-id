package customclaims

import (
	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

type handler struct {
	service *Service
}

func (h *handler) getSuggestions(c *gin.Context) ([]string, error) {
	return h.service.GetSuggestions(c.Request.Context())
}

func (h *handler) updateForUser(c *gin.Context, input []CustomClaimCreateDto) ([]dto.CustomClaimDto, error) {
	claims, err := h.service.UpdateCustomClaimsForUser(c.Request.Context(), c.Param("userId"), input)
	if err != nil {
		return nil, err
	}

	var customClaimsDto []dto.CustomClaimDto
	if err := dto.MapStructList(claims, &customClaimsDto); err != nil {
		return nil, err
	}

	return customClaimsDto, nil
}

func (h *handler) updateForUserGroup(c *gin.Context, input []CustomClaimCreateDto) ([]dto.CustomClaimDto, error) {
	claims, err := h.service.UpdateCustomClaimsForUserGroup(c.Request.Context(), c.Param("userGroupId"), input)
	if err != nil {
		return nil, err
	}

	var customClaimsDto []dto.CustomClaimDto
	if err := dto.MapStructList(claims, &customClaimsDto); err != nil {
		return nil, err
	}

	return customClaimsDto, nil
}
