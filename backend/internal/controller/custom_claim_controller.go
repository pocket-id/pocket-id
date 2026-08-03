package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// NewCustomClaimController creates a new controller for custom claim management
// @Summary Custom claim management controller
// @Description Initializes all custom claim-related API endpoints
// @Tags Custom Claims
func NewCustomClaimController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, customClaimService *service.CustomClaimService) {
	wkc := &CustomClaimController{customClaimService: customClaimService}

	customClaimsGroup := group.Group("/custom-claims")
	customClaimsGroup.Use(authMiddleware.Add())
	{
		customClaimsGroup.GET("/suggestions", httpserver.Handle(wkc.getSuggestionsHandler))
		customClaimsGroup.PUT("/user/:userId", httpserver.Handle(wkc.UpdateCustomClaimsForUserHandler))
		customClaimsGroup.PUT("/user-group/:userGroupId", httpserver.Handle(wkc.UpdateCustomClaimsForUserGroupHandler))
	}
}

type CustomClaimController struct {
	customClaimService *service.CustomClaimService
}

// getSuggestionsHandler godoc
// @Summary Get custom claim suggestions
// @Description Get a list of suggested custom claim names
// @Tags Custom Claims
// @Produce json
// @Success 200 {array} string "List of suggested custom claim names"
// @Router /api/custom-claims/suggestions [get]
func (ccc *CustomClaimController) getSuggestionsHandler(c *gin.Context) error {
	claims, err := ccc.customClaimService.GetSuggestions(c.Request.Context())
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, claims)
	return nil
}

// UpdateCustomClaimsForUserHandler godoc
// @Summary Update custom claims for a user
// @Description Update or create custom claims for a specific user
// @Tags Custom Claims
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param claims body []dto.CustomClaimCreateDto true "List of custom claims to set for the user"
// @Success 200 {array} dto.CustomClaimDto "Updated custom claims"
// @Router /api/custom-claims/user/{userId} [put]
func (ccc *CustomClaimController) UpdateCustomClaimsForUserHandler(c *gin.Context) error {
	var input []dto.CustomClaimCreateDto

	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	userId := c.Param("userId")
	claims, err := ccc.customClaimService.UpdateCustomClaimsForUser(c.Request.Context(), userId, input)
	if err != nil {
		return err
	}

	var customClaimsDto []dto.CustomClaimDto
	if err := dto.MapStructList(claims, &customClaimsDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, customClaimsDto)
	return nil
}

// UpdateCustomClaimsForUserGroupHandler godoc
// @Summary Update custom claims for a user group
// @Description Update or create custom claims for a specific user group
// @Tags Custom Claims
// @Accept json
// @Produce json
// @Param userGroupId path string true "User Group ID"
// @Param claims body []dto.CustomClaimCreateDto true "List of custom claims to set for the user group"
// @Success 200 {array} dto.CustomClaimDto "Updated custom claims"
// @Router /api/custom-claims/user-group/{userGroupId} [put]
func (ccc *CustomClaimController) UpdateCustomClaimsForUserGroupHandler(c *gin.Context) error {
	var input []dto.CustomClaimCreateDto

	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	userGroupId := c.Param("userGroupId")
	claims, err := ccc.customClaimService.UpdateCustomClaimsForUserGroup(c.Request.Context(), userGroupId, input)
	if err != nil {
		return err
	}

	var customClaimsDto []dto.CustomClaimDto
	if err := dto.MapStructList(claims, &customClaimsDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, customClaimsDto)
	return nil
}
