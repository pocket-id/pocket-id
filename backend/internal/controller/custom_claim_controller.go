package controller

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type customClaimUserInput struct {
	UserID string                     `path:"userId"`
	Body   []dto.CustomClaimCreateDto `required:"true"`
}

type customClaimUserGroupInput struct {
	UserGroupID string                     `path:"userGroupId"`
	Body        []dto.CustomClaimCreateDto `required:"true"`
}

// NewCustomClaimController registers custom claim management routes
func NewCustomClaimController(api huma.API, authMiddleware *middleware.AuthMiddleware, customClaimService *service.CustomClaimService) {
	controller := &CustomClaimController{customClaimService: customClaimService}
	auth := authMiddleware.Huma(api)

	httpapi.Register(api, huma.Operation{
		OperationID: "list-custom-claim-suggestions",
		Method:      http.MethodGet,
		Path:        "/api/custom-claims/suggestions",
		Summary:     "Get custom claim suggestions",
		Tags:        []string{"Custom Claims"},
	}, controller.getSuggestionsHandler, auth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-user-custom-claims",
		Method:      http.MethodPut,
		Path:        "/api/custom-claims/user/{userId}",
		Summary:     "Update custom claims for a user",
		Tags:        []string{"Custom Claims"},
	}, controller.updateCustomClaimsForUserHandler, auth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-user-group-custom-claims",
		Method:      http.MethodPut,
		Path:        "/api/custom-claims/user-group/{userGroupId}",
		Summary:     "Update custom claims for a user group",
		Tags:        []string{"Custom Claims"},
	}, controller.updateCustomClaimsForUserGroupHandler, auth)
}

type CustomClaimController struct {
	customClaimService *service.CustomClaimService
}

func (ccc *CustomClaimController) getSuggestionsHandler(ctx context.Context, _ *httpapi.EmptyInput) (*httpapi.BodyOutput[[]string], error) {
	claims, err := ccc.customClaimService.GetSuggestions(ctx)
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[[]string]{Body: claims}, nil
}

func (ccc *CustomClaimController) updateCustomClaimsForUserHandler(ctx context.Context, input *customClaimUserInput) (*httpapi.BodyOutput[[]dto.CustomClaimDto], error) {
	claims, err := ccc.customClaimService.UpdateCustomClaimsForUser(ctx, input.UserID, input.Body)
	if err != nil {
		return nil, err
	}

	var output []dto.CustomClaimDto
	if err := dto.MapStructList(claims, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[[]dto.CustomClaimDto]{Body: output}, nil
}

func (ccc *CustomClaimController) updateCustomClaimsForUserGroupHandler(ctx context.Context, input *customClaimUserGroupInput) (*httpapi.BodyOutput[[]dto.CustomClaimDto], error) {
	claims, err := ccc.customClaimService.UpdateCustomClaimsForUserGroup(ctx, input.UserGroupID, input.Body)
	if err != nil {
		return nil, err
	}

	var output []dto.CustomClaimDto
	if err := dto.MapStructList(claims, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[[]dto.CustomClaimDto]{Body: output}, nil
}
