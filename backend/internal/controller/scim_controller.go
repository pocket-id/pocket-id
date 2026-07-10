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

	createOperation := scimOperation("create-scim-service-provider", http.MethodPost, "/api/scim/service-provider", "Create SCIM service provider")
	createOperation.DefaultStatus = http.StatusCreated
	auth(&createOperation)
	httpapi.Register(api, createOperation, controller.createServiceProviderHandler)

	syncOperation := scimOperation("sync-scim-service-provider", http.MethodPost, "/api/scim/service-provider/{id}/sync", "Sync SCIM service provider")
	syncOperation.DefaultStatus = http.StatusOK
	auth(&syncOperation)
	httpapi.Register(api, syncOperation, controller.syncServiceProviderHandler)

	updateOperation := scimOperation("update-scim-service-provider", http.MethodPut, "/api/scim/service-provider/{id}", "Update SCIM service provider")
	auth(&updateOperation)
	httpapi.Register(api, updateOperation, controller.updateServiceProviderHandler)

	deleteOperation := scimOperation("delete-scim-service-provider", http.MethodDelete, "/api/scim/service-provider/{id}", "Delete SCIM service provider")
	deleteOperation.DefaultStatus = http.StatusNoContent
	auth(&deleteOperation)
	httpapi.Register(api, deleteOperation, controller.deleteServiceProviderHandler)
}

func scimOperation(id, method, path, summary string) huma.Operation {
	return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"SCIM"}}
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
