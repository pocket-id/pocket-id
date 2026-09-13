package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// NewOidcClaimMappingPolicyController creates a new controller for OIDC claim mapping policy management
// @Summary OIDC claim mapping policy controller
// @Description Initializes all OIDC claim mapping policy-related API endpoints
// @Tags OIDC Claim Mapping Policy
func NewOidcClaimMappingPolicyController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, oidcClaimMappingPolicyService *service.OidcClaimMappingPolicyService) {
	ocmc := &OidcClaimMappingPolicyController{oidcClaimMappingService: oidcClaimMappingPolicyService}

	group.GET("/oidc/claim-mapping-policies", authMiddleware.Add(), httpserver.Handle(ocmc.listClaimMappingPolicyHandler))
	group.POST("/oidc/claim-mapping-policies", authMiddleware.Add(), httpserver.Handle(ocmc.createClaimMappingPolicyHandler))
	group.GET("/oidc/claim-mapping-policies/:id", authMiddleware.Add(), httpserver.Handle(ocmc.getClaimMappingPolicyHandler))
	group.PUT("/oidc/claim-mapping-policies/:id", authMiddleware.Add(), httpserver.Handle(ocmc.updateClaimMappingPolicyHandler))
	group.DELETE("/oidc/claim-mapping-policies/:id", authMiddleware.Add(), httpserver.Handle(ocmc.deleteClaimMappingPolicyHandler))
	group.GET("/oidc/claim-mapping-policies/:id/clients", authMiddleware.Add(), httpserver.Handle(ocmc.listClientsByClaimMappingPolicyHandler))
	group.GET("/oidc/claim-mapping-policies/:id/assignable-clients", authMiddleware.Add(), httpserver.Handle(ocmc.listAssignableClientsHandler))
	group.POST("/oidc/claim-mapping-policies/:id/clients/:clientId", authMiddleware.Add(), httpserver.Handle(ocmc.assignClientHandler))
	group.DELETE("/oidc/claim-mapping-policies/:id/clients/:clientId", authMiddleware.Add(), httpserver.Handle(ocmc.removeClientHandler))

}

type OidcClaimMappingPolicyController struct {
	oidcClaimMappingService *service.OidcClaimMappingPolicyService
}

// listClaimMappingPolicyHandler godoc
// @Summary List OIDC claim mapping policies
// @Description Get a paginated list of OIDC claim mapping policies with optional search and sorting
// @Tags OIDC Claim Mapping Policy
// @Param search query string false "Search term to filter claim mapping policies by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.OidcClaimMappingPolicyMetadataDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies [get]
func (ocmc *OidcClaimMappingPolicyController) listClaimMappingPolicyHandler(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	claimMappingPolicys, pagination, err := ocmc.oidcClaimMappingService.ListClaimMappingPolicy(c.Request.Context(), searchTerm, listRequestOptions)
	if err != nil {
		return err
	}

	dtos := make([]dto.OidcClaimMappingPolicyMetadataDto, len(claimMappingPolicys))
	for i, claimMappingPolicy := range claimMappingPolicys {
		dtos[i] = dto.OidcClaimMappingPolicyMetadataDto{
			ID:        claimMappingPolicy.ID,
			Name:      claimMappingPolicy.Name,
			IsDefault: claimMappingPolicy.IsDefault,
		}
	}
	c.JSON(http.StatusOK, dto.Paginated[dto.OidcClaimMappingPolicyMetadataDto]{
		Data:       dtos,
		Pagination: pagination,
	})
	return nil
}

// createClaimMappingPolicyHandler godoc
// @Summary Create OIDC claim mapping policy
// @Description Create a new OIDC claim mapping policy
// @Tags OIDC Claim Mapping Policy
// @Accept json
// @Produce json
// @Param claimMappingPolicy body dto.OidcClaimMappingPolicyDto true "Claim mapping policy information"
// @Success 201 {object} dto.OidcClaimMappingPolicyDto "Created claim mapping policy"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies [post]
func (ocmc *OidcClaimMappingPolicyController) createClaimMappingPolicyHandler(c *gin.Context) error {
	var input dto.OidcClaimMappingPolicyDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	claimMappingPolicy, err := ocmc.oidcClaimMappingService.CreateClaimMappingPolicy(c.Request.Context(), input)
	if err != nil {
		return err
	}

	var claimMappingPolicyDto dto.OidcClaimMappingPolicyDto
	err = dto.MapStruct(claimMappingPolicy, &claimMappingPolicyDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusCreated, claimMappingPolicyDto)
	return nil
}

// getClaimMappingPolicyHandler godoc
// @Summary Get an OIDC claim mapping policy
// @Description Retrieve an existing OIDC claim mapping policy
// @Tags OIDC Claim Mapping Policy
// @Accept json
// @Produce json
// @Param id path string true "Claim mapping policy ID"
// @Success 200 {object} dto.OidcClaimMappingPolicyDto "Retrieved claim mapping policy"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id} [get]
func (ocmc *OidcClaimMappingPolicyController) getClaimMappingPolicyHandler(c *gin.Context) error {

	claimMappingPolicy, err := ocmc.oidcClaimMappingService.GetClaimMappingPolicy(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	var claimMappingPolicyDto dto.OidcClaimMappingPolicyDto
	err = dto.MapStruct(claimMappingPolicy, &claimMappingPolicyDto)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, claimMappingPolicyDto)
	return nil
}

// deleteClaimMappingPolicyHandler godoc
// @Summary Delete an OIDC claim mapping policy
// @Description Delete an OIDC claim mapping policy by ID
// @Tags OIDC Claim Mapping Policy
// @Param id path string true "Claim mapping policy ID"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id} [delete]
func (ocmc *OidcClaimMappingPolicyController) deleteClaimMappingPolicyHandler(c *gin.Context) error {
	err := ocmc.oidcClaimMappingService.DeleteClaimMappingPolicy(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// updateClaimMappingPolicyHandler godoc
// @Summary Update an OIDC claim mapping policy
// @Description Update an existing OIDC claim mapping policy
// @Tags OIDC Claim Mapping Policy
// @Accept json
// @Produce json
// @Param id path string true "Claim mapping policy ID"
// @Param claimMappingPolicy body dto.OidcClaimMappingPolicyDto true "Claim mapping policy information"
// @Success 200 {object} dto.OidcClaimMappingPolicyDto "Updated claim mapping policy"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id} [put]
func (ocmc *OidcClaimMappingPolicyController) updateClaimMappingPolicyHandler(c *gin.Context) error {
	var input dto.OidcClaimMappingPolicyDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	claimMappingPolicy, err := ocmc.oidcClaimMappingService.UpdateClaimMappingPolicy(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return err
	}

	var claimMappingPolicyDto dto.OidcClaimMappingPolicyDto
	err = dto.MapStruct(claimMappingPolicy, &claimMappingPolicyDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, claimMappingPolicyDto)
	return nil
}

// listClientsByClaimMappingPolicyHandler godoc
// @Summary List OIDC clients using a claim mapping policy
// @Description Get a paginated list of the OIDC clients assigned to a claim mapping policy
// @Tags OIDC Claim Mapping Policy
// @Param id path string true "Claim mapping policy ID"
// @Param search query string false "Search term to filter clients by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.OidcClientMetaDataDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id}/clients [get]
func (ocmc *OidcClaimMappingPolicyController) listClientsByClaimMappingPolicyHandler(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	clients, pagination, err := ocmc.oidcClaimMappingService.ListClientsByClaimMappingPolicy(c.Request.Context(), c.Param("id"), searchTerm, listRequestOptions)
	if err != nil {
		return err
	}

	dtos, err := clientMetaDataDtos(clients)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.OidcClientMetaDataDto]{
		Data:       dtos,
		Pagination: pagination,
	})
	return nil
}

// listAssignableClientsHandler godoc
// @Summary List OIDC clients that can still be assigned to a claim mapping policy
// @Description Get a paginated list of the OIDC clients that are not on this claim mapping policy yet
// @Tags OIDC Claim Mapping Policy
// @Param id path string true "Claim mapping policy ID"
// @Param search query string false "Search term to filter clients by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.OidcClientMetaDataDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id}/assignable-clients [get]
func (ocmc *OidcClaimMappingPolicyController) listAssignableClientsHandler(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	clients, pagination, err := ocmc.oidcClaimMappingService.ListAssignableClients(c.Request.Context(), c.Param("id"), searchTerm, listRequestOptions)
	if err != nil {
		return err
	}

	dtos, err := clientMetaDataDtos(clients)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.OidcClientMetaDataDto]{
		Data:       dtos,
		Pagination: pagination,
	})
	return nil
}

// assignClientHandler godoc
// @Summary Assign an OIDC client to a claim mapping policy
// @Description Point an OIDC client at this claim mapping policy, replacing the policy it was on
// @Tags OIDC Claim Mapping Policy
// @Param id path string true "Claim mapping policy ID"
// @Param clientId path string true "OIDC Client ID"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id}/clients/{clientId} [post]
func (ocmc *OidcClaimMappingPolicyController) assignClientHandler(c *gin.Context) error {
	err := ocmc.oidcClaimMappingService.AssignClientToClaimMappingPolicy(c.Request.Context(), c.Param("id"), c.Param("clientId"))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// removeClientHandler godoc
// @Summary Detach an OIDC client from a claim mapping policy
// @Description Remove an OIDC client from this claim mapping policy, so it falls back to the default one
// @Tags OIDC Claim Mapping Policy
// @Param id path string true "Claim mapping policy ID"
// @Param clientId path string true "OIDC Client ID"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id}/clients/{clientId} [delete]
func (ocmc *OidcClaimMappingPolicyController) removeClientHandler(c *gin.Context) error {
	err := ocmc.oidcClaimMappingService.RemoveClientFromClaimMappingPolicy(c.Request.Context(), c.Param("id"), c.Param("clientId"))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// clientMetaDataDtos maps OIDC clients to their metadata DTO, deriving the logo flags the model
// exposes as methods rather than fields
func clientMetaDataDtos(clients []model.OidcClient) ([]dto.OidcClientMetaDataDto, error) {
	dtos := make([]dto.OidcClientMetaDataDto, len(clients))
	for i, client := range clients {
		var clientDto dto.OidcClientMetaDataDto
		if err := dto.MapStruct(client, &clientDto); err != nil {
			return nil, err
		}
		clientDto.HasDarkLogo = client.HasDarkLogo()
		dtos[i] = clientDto
	}
	return dtos, nil
}
