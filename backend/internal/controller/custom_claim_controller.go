package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// NewCustomClaimController creates a new controller for custom claim management
func NewCustomClaimController(r *authz.Router, customClaimService *service.CustomClaimService) {
	wkc := &CustomClaimController{customClaimService: customClaimService}

	customClaimsGroup := r.Group("/custom-claims")
	customClaimsGroup.GET("/suggestions", authz.UsersRead, httpserver.JSON(http.StatusOK, wkc.getSuggestionsHandler),
		option.Summary("Get custom claim suggestions"),
		option.Description("Get a list of suggested custom claim names"),
		option.Tags("Custom Claims"),
	)
	customClaimsGroup.PUT("/user/:userId", authz.UsersWrite, httpserver.BodyJSON(http.StatusOK, wkc.UpdateCustomClaimsForUserHandler),
		option.Summary("Update custom claims for a user"),
		option.Description("Update or create custom claims for a specific user"),
		option.Tags("Custom Claims"),
		option.Path("userId", "User ID"),
	)
	customClaimsGroup.PUT("/user-group/:userGroupId", authz.GroupsWrite, httpserver.BodyJSON(http.StatusOK, wkc.UpdateCustomClaimsForUserGroupHandler),
		option.Summary("Update custom claims for a user group"),
		option.Description("Update or create custom claims for a specific user group"),
		option.Tags("Custom Claims"),
		option.Path("userGroupId", "User Group ID"),
	)
}

type CustomClaimController struct {
	customClaimService *service.CustomClaimService
}

func (ccc *CustomClaimController) getSuggestionsHandler(c *gin.Context) ([]string, error) {
	return ccc.customClaimService.GetSuggestions(c.Request.Context())
}

func (ccc *CustomClaimController) UpdateCustomClaimsForUserHandler(c *gin.Context, input []dto.CustomClaimCreateDto) ([]dto.CustomClaimDto, error) {
	userId := c.Param("userId")
	claims, err := ccc.customClaimService.UpdateCustomClaimsForUser(c.Request.Context(), userId, input)
	if err != nil {
		return nil, err
	}

	var customClaimsDto []dto.CustomClaimDto
	if err := dto.MapStructList(claims, &customClaimsDto); err != nil {
		return nil, err
	}

	return customClaimsDto, nil
}

func (ccc *CustomClaimController) UpdateCustomClaimsForUserGroupHandler(c *gin.Context, input []dto.CustomClaimCreateDto) ([]dto.CustomClaimDto, error) {
	userGroupId := c.Param("userGroupId")
	claims, err := ccc.customClaimService.UpdateCustomClaimsForUserGroup(c.Request.Context(), userGroupId, input)
	if err != nil {
		return nil, err
	}

	var customClaimsDto []dto.CustomClaimDto
	if err := dto.MapStructList(claims, &customClaimsDto); err != nil {
		return nil, err
	}

	return customClaimsDto, nil
}
