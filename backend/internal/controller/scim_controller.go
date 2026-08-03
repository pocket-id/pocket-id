package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func NewScimController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, scimService *service.ScimService) {
	ugc := ScimController{
		scimService: scimService,
	}

	group.POST("/scim/service-provider", authMiddleware.Add(), httpserver.Handle(ugc.createServiceProviderHandler))
	group.POST("/scim/service-provider/:id/sync", authMiddleware.Add(), httpserver.Handle(ugc.syncServiceProviderHandler))
	group.PUT("/scim/service-provider/:id", authMiddleware.Add(), httpserver.Handle(ugc.updateServiceProviderHandler))
	group.DELETE("/scim/service-provider/:id", authMiddleware.Add(), httpserver.Handle(ugc.deleteServiceProviderHandler))
}

type ScimController struct {
	scimService *service.ScimService
}

// syncServiceProviderHandler godoc
// @Summary Sync SCIM service provider
// @Description Trigger synchronization for a SCIM service provider
// @Tags SCIM
// @Param id path string true "Service Provider ID"
// @Success 200 "OK"
// @Router /api/scim/service-provider/{id}/sync [post]
func (c *ScimController) syncServiceProviderHandler(ctx *gin.Context) error {
	err := c.scimService.SyncServiceProvider(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		return err
	}

	ctx.Status(http.StatusOK)
	return nil
}

// createServiceProviderHandler godoc
// @Summary Create SCIM service provider
// @Description Create a new SCIM service provider
// @Tags SCIM
// @Accept json
// @Produce json
// @Param serviceProvider body dto.ScimServiceProviderCreateDTO true "SCIM service provider information"
// @Success 201 {object} dto.ScimServiceProviderDTO "Created SCIM service provider"
// @Router /api/scim/service-provider [post]
func (c *ScimController) createServiceProviderHandler(ctx *gin.Context) error {
	var input dto.ScimServiceProviderCreateDTO
	if err := httpserver.BindJSON(ctx, &input); err != nil {
		return err
	}

	provider, err := c.scimService.CreateServiceProvider(ctx.Request.Context(), &input)
	if err != nil {
		return err
	}

	var providerDTO dto.ScimServiceProviderDTO
	if err := dto.MapStruct(provider, &providerDTO); err != nil {
		return err
	}

	ctx.JSON(http.StatusCreated, providerDTO)
	return nil
}

// updateServiceProviderHandler godoc
// @Summary Update SCIM service provider
// @Description Update an existing SCIM service provider
// @Tags SCIM
// @Accept json
// @Produce json
// @Param id path string true "Service Provider ID"
// @Param serviceProvider body dto.ScimServiceProviderCreateDTO true "SCIM service provider information"
// @Success 200 {object} dto.ScimServiceProviderDTO "Updated SCIM service provider"
// @Router /api/scim/service-provider/{id} [put]
func (c *ScimController) updateServiceProviderHandler(ctx *gin.Context) error {
	var input dto.ScimServiceProviderCreateDTO
	if err := httpserver.BindJSON(ctx, &input); err != nil {
		return err
	}

	provider, err := c.scimService.UpdateServiceProvider(ctx.Request.Context(), ctx.Param("id"), &input)
	if err != nil {
		return err
	}

	var providerDTO dto.ScimServiceProviderDTO
	if err := dto.MapStruct(provider, &providerDTO); err != nil {
		return err
	}

	ctx.JSON(http.StatusOK, providerDTO)
	return nil
}

// deleteServiceProviderHandler godoc
// @Summary Delete SCIM service provider
// @Description Delete a SCIM service provider by ID
// @Tags SCIM
// @Param id path string true "Service Provider ID"
// @Success 204 "No Content"
// @Router /api/scim/service-provider/{id} [delete]
func (c *ScimController) deleteServiceProviderHandler(ctx *gin.Context) error {
	err := c.scimService.DeleteServiceProvider(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		return err
	}

	ctx.Status(http.StatusNoContent)
	return nil
}
