package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func NewCustomClaimController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, customClaimService *service.CustomClaimService) {
	wkc := &CustomClaimController{customClaimService: customClaimService}

	customClaimsGroup := group.Group("/custom-claims")
	customClaimsGroup.Use(authMiddleware.Add())
	{
		customClaimsGroup.GET("/suggestions", wkc.getSuggestionsHandler)
		customClaimsGroup.PUT("/user/:userId", wkc.UpdateCustomClaimsForUserHandler)
		customClaimsGroup.PUT("/user-group/:userGroupId", wkc.UpdateCustomClaimsForUserGroupHandler)
	}

}

type CustomClaimController struct {
	customClaimService *service.CustomClaimService
}

func (ccc *CustomClaimController) getSuggestionsHandler(c *gin.Context) {
	claims, err := ccc.customClaimService.GetSuggestions()
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, claims)
}

func (ccc *CustomClaimController) UpdateCustomClaimsForUserHandler(c *gin.Context) {
	var input []dto.CustomClaimCreateDto

	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	userId := c.Param("userId")
	claims, err := ccc.customClaimService.UpdateCustomClaimsForUser(userId, input)
	if err != nil {
		c.Error(err)
		return
	}

	var customClaimsDto []dto.CustomClaimDto
	if err := dto.MapStructList(claims, &customClaimsDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, customClaimsDto)
}

func (ccc *CustomClaimController) UpdateCustomClaimsForUserGroupHandler(c *gin.Context) {
	var input []dto.CustomClaimCreateDto

	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	userId := c.Param("userGroupId")
	claims, err := ccc.customClaimService.UpdateCustomClaimsForUserGroup(userId, input)
	if err != nil {
		c.Error(err)
		return
	}

	var customClaimsDto []dto.CustomClaimDto
	if err := dto.MapStructList(claims, &customClaimsDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, customClaimsDto)
}
