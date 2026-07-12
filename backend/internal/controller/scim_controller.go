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

type scimIDInput struct {
	ID string `path:"id"`
}

type scimCreateInput struct {
	Body dto.ScimServiceProviderCreateDTO
}

type scimUpdateInput struct {
	ID   string `path:"id"`
	Body dto.ScimServiceProviderCreateDTO
}

func NewScimController(api huma.API, authMiddleware *middleware.AuthMiddleware, scimService *service.ScimService) {
	controller := &ScimController{scimService: scimService}
	auth := authMiddleware.Huma(api)

	httpapi.Register(api, huma.Operation{
		OperationID:   "create-scim-service-provider",
		Method:        http.MethodPost,
		Path:          "/api/scim/service-provider",
		Summary:       "Create SCIM service provider",
		Tags:          []string{"SCIM"},
		DefaultStatus: http.StatusCreated,
	}, controller.createServiceProviderHandler, auth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "sync-scim-service-provider",
		Method:        http.MethodPost,
		Path:          "/api/scim/service-provider/{id}/sync",
		Summary:       "Sync SCIM service provider",
		Tags:          []string{"SCIM"},
		DefaultStatus: http.StatusOK,
	}, controller.syncServiceProviderHandler, auth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-scim-service-provider",
		Method:      http.MethodPut,
		Path:        "/api/scim/service-provider/{id}",
		Summary:     "Update SCIM service provider",
		Tags:        []string{"SCIM"},
	}, controller.updateServiceProviderHandler, auth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "delete-scim-service-provider",
		Method:        http.MethodDelete,
		Path:          "/api/scim/service-provider/{id}",
		Summary:       "Delete SCIM service provider",
		Tags:          []string{"SCIM"},
		DefaultStatus: http.StatusNoContent,
	}, controller.deleteServiceProviderHandler, auth)
}

type ScimController struct {
	scimService *service.ScimService
}

func (c *ScimController) syncServiceProviderHandler(ctx context.Context, input *scimIDInput) (*httpapi.EmptyOutput, error) {
	if err := c.scimService.SyncServiceProvider(ctx, input.ID); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (c *ScimController) createServiceProviderHandler(ctx context.Context, input *scimCreateInput) (*httpapi.BodyOutput[dto.ScimServiceProviderDTO], error) {
	provider, err := c.scimService.CreateServiceProvider(ctx, &input.Body)
	if err != nil {
		return nil, err
	}
	return mapSCIMProvider(provider)
}

func (c *ScimController) updateServiceProviderHandler(ctx context.Context, input *scimUpdateInput) (*httpapi.BodyOutput[dto.ScimServiceProviderDTO], error) {
	provider, err := c.scimService.UpdateServiceProvider(ctx, input.ID, &input.Body)
	if err != nil {
		return nil, err
	}
	return mapSCIMProvider(provider)
}

func (c *ScimController) deleteServiceProviderHandler(ctx context.Context, input *scimIDInput) (*httpapi.EmptyOutput, error) {
	if err := c.scimService.DeleteServiceProvider(ctx, input.ID); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func mapSCIMProvider(provider any) (*httpapi.BodyOutput[dto.ScimServiceProviderDTO], error) {
	var output dto.ScimServiceProviderDTO
	if err := dto.MapStruct(provider, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.ScimServiceProviderDTO]{Body: output}, nil
}
