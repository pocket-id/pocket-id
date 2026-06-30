package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// NewOidcController creates a new controller for OIDC related endpoints
// @Summary OIDC controller
// @Description Initializes all OIDC-related API endpoints for authentication and client management
// @Tags OIDC
func NewOidcController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, fileSizeLimitMiddleware *middleware.FileSizeLimitMiddleware, oidcService *service.OidcService) {
	oc := &OidcController{
		oidcService: oidcService,
	}

	group.GET("/oidc/clients", authMiddleware.Add(), oc.listClientsHandler)
	group.POST("/oidc/clients", authMiddleware.Add(), oc.createClientHandler)
	group.GET("/oidc/clients/:id", authMiddleware.Add(), oc.getClientHandler)
	group.GET("/oidc/clients/:id/meta", oc.getClientMetaDataHandler)
	group.PUT("/oidc/clients/:id", authMiddleware.Add(), oc.updateClientHandler)
	group.DELETE("/oidc/clients/:id", authMiddleware.Add(), oc.deleteClientHandler)

	group.PUT("/oidc/clients/:id/allowed-user-groups", authMiddleware.Add(), oc.updateAllowedUserGroupsHandler)
	group.POST("/oidc/clients/:id/secret", authMiddleware.Add(), oc.createClientSecretHandler)

	group.GET("/oidc/clients/:id/logo", oc.getClientLogoHandler)
	group.DELETE("/oidc/clients/:id/logo", authMiddleware.Add(), oc.deleteClientLogoHandler)
	group.POST("/oidc/clients/:id/logo", authMiddleware.Add(), fileSizeLimitMiddleware.Add(2<<20), oc.updateClientLogoHandler)

	group.GET("/oidc/redirect-uri/registered", authMiddleware.WithAdminNotRequired().Add(), oc.getRegisteredRedirectURIHandler)

	group.GET("/oidc/clients/:id/preview/:userId", authMiddleware.Add(), oc.getClientPreviewHandler)

	group.GET("/oidc/users/me/authorized-clients", authMiddleware.WithAdminNotRequired().Add(), oc.listOwnAuthorizedClientsHandler)
	group.GET("/oidc/users/:id/authorized-clients", authMiddleware.Add(), oc.listAuthorizedClientsHandler)

	group.DELETE("/oidc/users/me/authorized-clients/:clientId", authMiddleware.WithAdminNotRequired().Add(), oc.revokeOwnClientAuthorizationHandler)

	group.GET("/oidc/users/me/clients", authMiddleware.WithAdminNotRequired().Add(), oc.listOwnAccessibleClientsHandler)

	group.GET("/oidc/clients/:id/scim-service-provider", authMiddleware.Add(), oc.getClientScimServiceProviderHandler)

}

type OidcController struct {
	oidcService *service.OidcService
}

// getClientMetaDataHandler godoc
// @Summary Get client metadata
// @Description Get OIDC client metadata for discovery and configuration
// @Tags OIDC
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} dto.OidcClientMetaDataDto "Client metadata"
// @Router /api/oidc/clients/{id}/meta [get]
func (oc *OidcController) getClientMetaDataHandler(c *gin.Context) {
	clientId := c.Param("id")
	client, err := oc.oidcService.GetClient(c.Request.Context(), clientId)
	if err != nil {
		_ = c.Error(err)
		return
	}

	clientDto := dto.OidcClientMetaDataDto{}
	err = dto.MapStruct(client, &clientDto)
	if err == nil {
		clientDto.HasDarkLogo = client.HasDarkLogo()
		c.JSON(http.StatusOK, clientDto)
		return
	}

	_ = c.Error(err)
}

// getClientHandler godoc
// @Summary Get OIDC client
// @Description Get detailed information about an OIDC client
// @Tags OIDC
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} dto.OidcClientWithAllowedUserGroupsDto "Client information"
// @Router /api/oidc/clients/{id} [get]
func (oc *OidcController) getClientHandler(c *gin.Context) {
	clientId := c.Param("id")
	client, err := oc.oidcService.GetClient(c.Request.Context(), clientId)
	if err != nil {
		_ = c.Error(err)
		return
	}

	clientDto := dto.OidcClientWithAllowedUserGroupsDto{}
	err = dto.MapStruct(client, &clientDto)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, clientDto)
}

// listClientsHandler godoc
// @Summary List OIDC clients
// @Description Get a paginated list of OIDC clients with optional search and sorting
// @Tags OIDC
// @Param search query string false "Search term to filter clients by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.OidcClientWithAllowedGroupsCountDto]
// @Router /api/oidc/clients [get]
func (oc *OidcController) listClientsHandler(c *gin.Context) {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	clients, pagination, err := oc.oidcService.ListClients(c.Request.Context(), searchTerm, listRequestOptions)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// Map the user groups to DTOs
	var clientsDto = make([]dto.OidcClientWithAllowedGroupsCountDto, len(clients))
	for i, client := range clients {
		var clientDto dto.OidcClientWithAllowedGroupsCountDto
		if err := dto.MapStruct(client, &clientDto); err != nil {
			_ = c.Error(err)
			return
		}
		clientDto.HasDarkLogo = client.HasDarkLogo()
		clientDto.AllowedUserGroupsCount, err = oc.oidcService.GetAllowedGroupsCountOfClient(c, client.ID)
		if err != nil {
			_ = c.Error(err)
			return
		}
		clientsDto[i] = clientDto
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.OidcClientWithAllowedGroupsCountDto]{
		Data:       clientsDto,
		Pagination: pagination,
	})
}

func (oc *OidcController) getRegisteredRedirectURIHandler(c *gin.Context) {
	redirectURI, err := oc.oidcService.GetRegisteredCallbackURL(c.Request.Context(), c.Query("redirect_uri"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.OidcRegisteredCallbackURLDto{
		Registered:  redirectURI != "",
		RedirectURI: redirectURI,
	})
}

// createClientHandler godoc
// @Summary Create OIDC client
// @Description Create a new OIDC client
// @Tags OIDC
// @Accept json
// @Produce json
// @Param client body dto.OidcClientCreateDto true "Client information"
// @Success 201 {object} dto.OidcClientWithAllowedUserGroupsDto "Created client"
// @Router /api/oidc/clients [post]
func (oc *OidcController) createClientHandler(c *gin.Context) {
	var input dto.OidcClientCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	client, err := oc.oidcService.CreateClient(c.Request.Context(), input, c.GetString("userID"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	var clientDto dto.OidcClientWithAllowedUserGroupsDto
	if err := dto.MapStruct(client, &clientDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, clientDto)
}

// deleteClientHandler godoc
// @Summary Delete OIDC client
// @Description Delete an OIDC client by ID
// @Tags OIDC
// @Param id path string true "Client ID"
// @Success 204 "No Content"
// @Router /api/oidc/clients/{id} [delete]
func (oc *OidcController) deleteClientHandler(c *gin.Context) {
	err := oc.oidcService.DeleteClient(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// updateClientHandler godoc
// @Summary Update OIDC client
// @Description Update an existing OIDC client
// @Tags OIDC
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Param client body dto.OidcClientUpdateDto true "Client information"
// @Success 200 {object} dto.OidcClientWithAllowedUserGroupsDto "Updated client"
// @Router /api/oidc/clients/{id} [put]
func (oc *OidcController) updateClientHandler(c *gin.Context) {
	var input dto.OidcClientUpdateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	client, err := oc.oidcService.UpdateClient(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var clientDto dto.OidcClientWithAllowedUserGroupsDto
	if err := dto.MapStruct(client, &clientDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, clientDto)
}

// createClientSecretHandler godoc
// @Summary Create client secret
// @Description Generate a new secret for an OIDC client
// @Tags OIDC
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} object "{ \"secret\": \"string\" }"
// @Router /api/oidc/clients/{id}/secret [post]
func (oc *OidcController) createClientSecretHandler(c *gin.Context) {
	secret, err := oc.oidcService.CreateClientSecret(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"secret": secret})
}

// getClientLogoHandler godoc
// @Summary Get client logo
// @Description Get the logo image for an OIDC client
// @Tags OIDC
// @Produce image/png
// @Produce image/jpeg
// @Produce image/svg+xml
// @Param id path string true "Client ID"
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Success 200 {file} binary "Logo image"
// @Router /api/oidc/clients/{id}/logo [get]
func (oc *OidcController) getClientLogoHandler(c *gin.Context) {
	lightLogo, _ := strconv.ParseBool(c.DefaultQuery("light", "true"))

	reader, size, mimeType, err := oc.oidcService.GetClientLogo(c.Request.Context(), c.Param("id"), lightLogo)
	if err != nil {
		_ = c.Error(err)
		return
	}
	defer reader.Close()

	utils.SetCacheControlHeader(c, 15*time.Minute, 12*time.Hour)

	c.Header("Content-Type", mimeType)
	c.DataFromReader(http.StatusOK, size, mimeType, reader, nil)
}

// updateClientLogoHandler godoc
// @Summary Update client logo
// @Description Upload or update the logo for an OIDC client
// @Tags OIDC
// @Accept multipart/form-data
// @Param id path string true "Client ID"
// @Param file formData file true "Logo image file (PNG, JPG, or SVG)"
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Success 204 "No Content"
// @Router /api/oidc/clients/{id}/logo [post]
func (oc *OidcController) updateClientLogoHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		_ = c.Error(err)
		return
	}

	lightLogo, _ := strconv.ParseBool(c.DefaultQuery("light", "true"))

	err = oc.oidcService.UpdateClientLogo(c.Request.Context(), c.Param("id"), file, lightLogo)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// deleteClientLogoHandler godoc
// @Summary Delete client logo
// @Description Delete the logo for an OIDC client
// @Tags OIDC
// @Param id path string true "Client ID"
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Success 204 "No Content"
// @Router /api/oidc/clients/{id}/logo [delete]
func (oc *OidcController) deleteClientLogoHandler(c *gin.Context) {
	var err error

	lightLogo, _ := strconv.ParseBool(c.DefaultQuery("light", "true"))
	if lightLogo {
		err = oc.oidcService.DeleteClientLogo(c.Request.Context(), c.Param("id"))
	} else {
		err = oc.oidcService.DeleteClientDarkLogo(c.Request.Context(), c.Param("id"))
	}

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// updateAllowedUserGroupsHandler godoc
// @Summary Update allowed user groups
// @Description Update the user groups allowed to access an OIDC client
// @Tags OIDC
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Param groups body dto.OidcUpdateAllowedUserGroupsDto true "User group IDs"
// @Success 200 {object} dto.OidcClientDto "Updated client"
// @Router /api/oidc/clients/{id}/allowed-user-groups [put]
func (oc *OidcController) updateAllowedUserGroupsHandler(c *gin.Context) {
	var input dto.OidcUpdateAllowedUserGroupsDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	oidcClient, err := oc.oidcService.UpdateAllowedUserGroups(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var oidcClientDto dto.OidcClientDto
	if err := dto.MapStruct(oidcClient, &oidcClientDto); err != nil {
		_ = c.Error(err)
		return
	}
	oidcClientDto.HasDarkLogo = oidcClient.HasDarkLogo()

	c.JSON(http.StatusOK, oidcClientDto)
}

// listOwnAuthorizedClientsHandler godoc
// @Summary List authorized clients for current user
// @Description Get a paginated list of OIDC clients that the current user has authorized
// @Tags OIDC
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.AuthorizedOidcClientDto]
// @Router /api/oidc/users/me/authorized-clients [get]
func (oc *OidcController) listOwnAuthorizedClientsHandler(c *gin.Context) {
	userID := c.GetString("userID")
	oc.listAuthorizedClients(c, userID)
}

// listAuthorizedClientsHandler godoc
// @Summary List authorized clients for a user
// @Description Get a paginated list of OIDC clients that a specific user has authorized
// @Tags OIDC
// @Param id path string true "User ID"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.AuthorizedOidcClientDto]
// @Router /api/oidc/users/{id}/authorized-clients [get]
func (oc *OidcController) listAuthorizedClientsHandler(c *gin.Context) {
	userID := c.Param("id")
	oc.listAuthorizedClients(c, userID)
}

func (oc *OidcController) listAuthorizedClients(c *gin.Context, userID string) {
	listRequestOptions := utils.ParseListRequestOptions(c)

	authorizedClients, pagination, err := oc.oidcService.ListAuthorizedClients(c.Request.Context(), userID, listRequestOptions)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// Map the clients to DTOs
	var authorizedClientsDto []dto.AuthorizedOidcClientDto
	if err := dto.MapStructList(authorizedClients, &authorizedClientsDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.AuthorizedOidcClientDto]{
		Data:       authorizedClientsDto,
		Pagination: pagination,
	})
}

// revokeOwnClientAuthorizationHandler godoc
// @Summary Revoke authorization for an OIDC client
// @Description Revoke the authorization for a specific OIDC client for the current user
// @Tags OIDC
// @Param clientId path string true "Client ID to revoke authorization for"
// @Success 204 "No Content"
// @Router /api/oidc/users/me/authorized-clients/{clientId} [delete]
func (oc *OidcController) revokeOwnClientAuthorizationHandler(c *gin.Context) {
	clientID := c.Param("clientId")

	userID := c.GetString("userID")

	err := oc.oidcService.RevokeAuthorizedClient(c.Request.Context(), userID, clientID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// listOwnAccessibleClientsHandler godoc
// @Summary List accessible OIDC clients for current user
// @Description Get a list of OIDC clients that the current user can access
// @Tags OIDC
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.AccessibleOidcClientDto]
// @Router /api/oidc/users/me/clients [get]
func (oc *OidcController) listOwnAccessibleClientsHandler(c *gin.Context) {
	listRequestOptions := utils.ParseListRequestOptions(c)

	userID := c.GetString("userID")

	clients, pagination, err := oc.oidcService.ListAccessibleOidcClients(c.Request.Context(), userID, listRequestOptions)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.AccessibleOidcClientDto]{
		Data:       clients,
		Pagination: pagination,
	})
}

// getClientPreviewHandler godoc
// @Summary Preview OIDC client data for user
// @Description Get a preview of the OIDC data (ID token, access token, userinfo) that would be sent to the client for a specific user
// @Tags OIDC
// @Produce json
// @Param id path string true "Client ID"
// @Param userId path string true "User ID to preview data for"
// @Param scopes query string false "Scopes to include in the preview (comma-separated)"
// @Success 200 {object} dto.OidcClientPreviewDto "Preview data including ID token, access token, and userinfo payloads"
// @Security BearerAuth
// @Router /api/oidc/clients/{id}/preview/{userId} [get]
func (oc *OidcController) getClientPreviewHandler(c *gin.Context) {
	clientID := c.Param("id")
	userID := c.Param("userId")
	scopes := c.Query("scopes")

	if clientID == "" {
		_ = c.Error(&common.ValidationError{Message: "client ID is required"})
		return
	}

	if userID == "" {
		_ = c.Error(&common.ValidationError{Message: "user ID is required"})
		return
	}

	if scopes == "" {
		_ = c.Error(&common.ValidationError{Message: "scopes are required"})
		return
	}

	preview, err := oc.oidcService.GetClientPreview(
		c.Request.Context(),
		clientID,
		userID,
		strings.Split(scopes, " "),
		c.GetString("authenticationMethod"))

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, preview)
}

// getClientScimServiceProviderHandler godoc
// @Summary Get SCIM service provider
// @Description Get the SCIM service provider configuration for an OIDC client
// @Tags OIDC
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} dto.ScimServiceProviderDTO "SCIM service provider configuration"
// @Router /api/oidc/clients/{id}/scim-service-provider [get]
func (oc *OidcController) getClientScimServiceProviderHandler(c *gin.Context) {
	clientID := c.Param("id")

	provider, err := oc.oidcService.GetClientScimServiceProvider(c.Request.Context(), clientID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var providerDto dto.ScimServiceProviderDTO
	if err := dto.MapStruct(provider, &providerDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, providerDto)
}
