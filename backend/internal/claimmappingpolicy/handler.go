package claimmappingpolicy

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

// list godoc
// @Summary List OIDC claim mapping policies
// @Description Get a paginated list of OIDC claim mapping policies with optional search and sorting
// @Tags OIDC Claim OidcClaimMapping OidcClaimMappingPolicy
// @Param search query string false "Search term to filter claim mapping policies by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[OidcClaimMappingPolicyMetadataDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies [get]
func (h *handler) list(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	claimMappingPolicys, pagination, err := h.service.ListClaimMappingPolicy(c.Request.Context(), searchTerm, listRequestOptions)
	if err != nil {
		return err
	}

	clientCounts, err := h.service.ClientCounts(c.Request.Context(), claimMappingPolicys)
	if err != nil {
		return err
	}

	dtos := make([]OidcClaimMappingPolicyMetadataDto, len(claimMappingPolicys))
	for i, claimMappingPolicy := range claimMappingPolicys {
		dtos[i] = OidcClaimMappingPolicyMetadataDto{
			ID:          claimMappingPolicy.ID,
			Name:        claimMappingPolicy.Name,
			IsDefault:   claimMappingPolicy.IsDefault,
			CreatedAt:   claimMappingPolicy.CreatedAt,
			ClaimCount:  len(claimMappingPolicy.ClaimMappings),
			ClientCount: clientCounts[claimMappingPolicy.ID],
		}
	}
	c.JSON(http.StatusOK, dto.Paginated[OidcClaimMappingPolicyMetadataDto]{
		Data:       dtos,
		Pagination: pagination,
	})
	return nil
}

// create godoc
// @Summary Create OIDC claim mapping policy
// @Description Create a new OIDC claim mapping policy
// @Tags OIDC Claim OidcClaimMapping OidcClaimMappingPolicy
// @Accept json
// @Produce json
// @Param claimMappingPolicy body OidcClaimMappingPolicyDto true "Claim mapping policy information"
// @Success 201 {object} OidcClaimMappingPolicyDto "Created claim mapping policy"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies [post]
func (h *handler) create(c *gin.Context) error {
	var input OidcClaimMappingPolicyDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	claimMappingPolicy, err := h.service.CreateClaimMappingPolicy(c.Request.Context(), input)
	if err != nil {
		return err
	}

	var claimMappingPolicyDto OidcClaimMappingPolicyDto
	err = dto.MapStruct(claimMappingPolicy, &claimMappingPolicyDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusCreated, claimMappingPolicyDto)
	return nil
}

// get godoc
// @Summary Get an OIDC claim mapping policy
// @Description Retrieve an existing OIDC claim mapping policy
// @Tags OIDC Claim OidcClaimMapping OidcClaimMappingPolicy
// @Accept json
// @Produce json
// @Param id path string true "Claim mapping policy ID"
// @Success 200 {object} OidcClaimMappingPolicyDto "Retrieved claim mapping policy"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id} [get]
func (h *handler) get(c *gin.Context) error {
	claimMappingPolicy, err := h.service.GetClaimMappingPolicy(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	clientCounts, err := h.service.ClientCounts(c.Request.Context(), []OidcClaimMappingPolicy{claimMappingPolicy})
	if err != nil {
		return err
	}

	var claimMappingPolicyDto OidcClaimMappingPolicyDto
	err = dto.MapStruct(claimMappingPolicy, &claimMappingPolicyDto)
	if err != nil {
		return err
	}
	claimMappingPolicyDto.ClaimCount = len(claimMappingPolicy.ClaimMappings)
	claimMappingPolicyDto.ClientCount = clientCounts[claimMappingPolicy.ID]
	c.JSON(http.StatusOK, claimMappingPolicyDto)
	return nil
}

// delete godoc
// @Summary Delete an OIDC claim mapping policy
// @Description Delete an OIDC claim mapping policy by ID
// @Tags OIDC Claim OidcClaimMapping OidcClaimMappingPolicy
// @Param id path string true "Claim mapping policy ID"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id} [delete]
func (h *handler) delete(c *gin.Context) error {
	err := h.service.DeleteClaimMappingPolicy(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// update godoc
// @Summary Update an OIDC claim mapping policy
// @Description Update an existing OIDC claim mapping policy
// @Tags OIDC Claim OidcClaimMapping OidcClaimMappingPolicy
// @Accept json
// @Produce json
// @Param id path string true "Claim mapping policy ID"
// @Param claimMappingPolicy body OidcClaimMappingPolicyDto true "Claim mapping policy information"
// @Success 200 {object} OidcClaimMappingPolicyDto "Updated claim mapping policy"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id} [put]
func (h *handler) update(c *gin.Context) error {
	var input OidcClaimMappingPolicyDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	claimMappingPolicy, err := h.service.UpdateClaimMappingPolicy(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return err
	}

	var claimMappingPolicyDto OidcClaimMappingPolicyDto
	err = dto.MapStruct(claimMappingPolicy, &claimMappingPolicyDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, claimMappingPolicyDto)
	return nil
}

// listClients godoc
// @Summary List OIDC clients using a claim mapping policy
// @Description Get a paginated list of the OIDC clients assigned to a claim mapping policy
// @Tags OIDC Claim OidcClaimMapping OidcClaimMappingPolicy
// @Param id path string true "Claim mapping policy ID"
// @Param search query string false "Search term to filter clients by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.OidcClientMetaDataDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id}/clients [get]
func (h *handler) listClients(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	clients, pagination, err := h.service.ListClientsByClaimMappingPolicy(
		c.Request.Context(), c.Param("id"), searchTerm, listRequestOptions,
	)
	if err != nil {
		return err
	}

	var dtos []dto.OidcClientMetaDataDto
	err = dto.MapStructList(clients, &dtos)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.OidcClientMetaDataDto]{
		Data:       dtos,
		Pagination: pagination,
	})
	return nil
}

// listAssignableClients godoc
// @Summary List OIDC clients that can still be assigned to a claim mapping policy
// @Description Get a paginated list of the OIDC clients that are not on this claim mapping policy yet
// @Tags OIDC Claim OidcClaimMapping OidcClaimMappingPolicy
// @Param id path string true "Claim mapping policy ID"
// @Param search query string false "Search term to filter clients by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.OidcClientMetaDataDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id}/assignable-clients [get]
func (h *handler) listAssignableClients(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	clients, pagination, err := h.service.ListAssignableClients(
		c.Request.Context(), c.Param("id"), searchTerm, listRequestOptions,
	)
	if err != nil {
		return err
	}

	var dtos []dto.OidcClientMetaDataDto
	err = dto.MapStructList(clients, &dtos)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.OidcClientMetaDataDto]{
		Data:       dtos,
		Pagination: pagination,
	})
	return nil
}

// assignClient godoc
// @Summary Assign an OIDC client to a claim mapping policy
// @Description Point an OIDC client at this claim mapping policy, replacing the policy it was on
// @Tags OIDC Claim OidcClaimMapping OidcClaimMappingPolicy
// @Param id path string true "Claim mapping policy ID"
// @Param clientId path string true "OIDC Client ID"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id}/clients/{clientId} [post]
func (h *handler) assignClient(c *gin.Context) error {
	err := h.service.AssignClientToClaimMappingPolicy(
		c.Request.Context(), c.Param("id"), c.Param("clientId"),
	)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// removeClient godoc
// @Summary Detach an OIDC client from a claim mapping policy
// @Description Remove an OIDC client from this claim mapping policy, so it falls back to the default one
// @Tags OIDC Claim OidcClaimMapping OidcClaimMappingPolicy
// @Param id path string true "Claim mapping policy ID"
// @Param clientId path string true "OIDC Client ID"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/claim-mapping-policies/{id}/clients/{clientId} [delete]
func (h *handler) removeClient(c *gin.Context) error {
	err := h.service.RemoveClientFromClaimMappingPolicy(
		c.Request.Context(), c.Param("id"), c.Param("clientId"),
	)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}
