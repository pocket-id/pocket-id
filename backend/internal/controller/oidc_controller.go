package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// NewOidcController creates a new controller for OIDC related endpoints
// @Summary OIDC controller
// @Description Initializes all OIDC-related API endpoints for authentication and client management
// @Tags OIDC
func NewOidcController(r *authz.Router, fileSizeLimitMiddleware *middleware.FileSizeLimitMiddleware, oidcService *service.OidcService, appConfigService appconfig.AppConfigResolver) {
	oc := &OidcController{
		oidcService:      oidcService,
		appConfigService: appConfigService,
	}

	r.GET("/oidc/clients", authz.OidcClientsRead, httpserver.Handle(oc.listClientsHandler))
	r.POST("/oidc/clients", authz.OidcClientsWrite, httpserver.Handle(oc.createClientHandler))
	r.GET("/oidc/clients/:id", authz.OidcClientsRead, httpserver.Handle(oc.getClientHandler))
	r.PUT("/oidc/clients/:id", authz.OidcClientsWrite, httpserver.Handle(oc.updateClientHandler))
	r.POST("/oidc/clients/:id/refresh", authz.OidcClientsWrite, httpserver.Handle(oc.refreshClientMetadataHandler))
	r.DELETE("/oidc/clients/:id", authz.OidcClientsWrite, httpserver.Handle(oc.deleteClientHandler))

	r.PUT("/oidc/clients/:id/allowed-user-groups", authz.OidcClientsWrite, httpserver.Handle(oc.updateAllowedUserGroupsHandler))
	r.GET("/oidc/clients/:id/secrets", authz.OidcClientsRead, httpserver.Handle(oc.listClientSecretsHandler))
	r.POST("/oidc/clients/:id/secrets", authz.OidcClientsWrite, httpserver.Handle(oc.createClientSecretHandler))
	r.DELETE("/oidc/clients/:id/secrets/:secretId", authz.OidcClientsWrite, httpserver.Handle(oc.deleteClientSecretHandler))

	r.Public().GET("/oidc/clients/:id/logo", httpserver.Handle(oc.getClientLogoHandler))
	r.DELETE("/oidc/clients/:id/logo", authz.OidcClientsWrite, httpserver.Handle(oc.deleteClientLogoHandler))
	r.POST("/oidc/clients/:id/logo", authz.OidcClientsWrite, fileSizeLimitMiddleware.Add(2<<20), httpserver.Handle(oc.updateClientLogoHandler))

	// The preview renders a user's claims, so it is guarded by the user scope rather than the client scope
	r.GET("/oidc/clients/:id/preview/:userId", authz.UsersRead, httpserver.Handle(oc.getClientPreviewHandler))

	r.GET("/oidc/users/me/authorized-clients", authz.AccountApps, httpserver.Handle(oc.listOwnAuthorizedClientsHandler))
	r.GET("/oidc/users/:id/authorized-clients", authz.UsersRead, httpserver.Handle(oc.listAuthorizedClientsHandler))

	r.DELETE("/oidc/users/me/authorized-clients/:clientId", authz.AccountApps, httpserver.Handle(oc.revokeOwnClientAuthorizationHandler))

	r.GET("/oidc/users/me/clients", authz.AccountApps, httpserver.Handle(oc.listOwnAccessibleClientsHandler))
}

type OidcController struct {
	oidcService      *service.OidcService
	appConfigService appconfig.AppConfigResolver
}

// getClientHandler godoc
// @Summary Get OIDC client
// @Description Get detailed information about an OIDC client
// @Tags OIDC
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} dto.OidcClientWithAllowedUserGroupsDto "Client information"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id} [get]
func (oc *OidcController) getClientHandler(c *gin.Context) error {
	clientId := c.Param("id")
	client, err := oc.oidcService.GetClient(c.Request.Context(), clientId)
	if err != nil {
		return err
	}

	clientDto := dto.OidcClientWithAllowedUserGroupsDto{}
	err = dto.MapStruct(client, &clientDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, clientDto)
	return nil
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
// @Success 200 {object} dto.Paginated[dto.OidcClientWithAllowedGroupsDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients [get]
func (oc *OidcController) listClientsHandler(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	clients, pagination, err := oc.oidcService.ListClients(c.Request.Context(), searchTerm, listRequestOptions)
	if err != nil {
		return err
	}

	// Map the user groups to DTOs
	var clientsDto = make([]dto.OidcClientWithAllowedGroupsDto, len(clients))
	for i, client := range clients {
		var clientDto dto.OidcClientWithAllowedGroupsDto
		if err := dto.MapStruct(client, &clientDto); err != nil {
			return err
		}
		clientDto.HasDarkLogo = client.HasDarkLogo()
		clientsDto[i] = clientDto
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.OidcClientWithAllowedGroupsDto]{
		Data:       clientsDto,
		Pagination: pagination,
	})
	return nil
}

// createClientHandler godoc
// @Summary Create OIDC client
// @Description Create a new OIDC client
// @Tags OIDC
// @Accept json
// @Produce json
// @Param client body dto.OidcClientCreateDto true "Client information"
// @Success 201 {object} dto.OidcClientCreatedDto "Created client"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients [post]
func (oc *OidcController) createClientHandler(c *gin.Context) error {
	var input dto.OidcClientCreateDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	config, err := oc.appConfigService.GetConfig(c.Request.Context())
	if err != nil {
		return err
	}

	client, createdSecret, err := oc.oidcService.CreateClient(c.Request.Context(), input, authz.PrincipalFrom(c).UserID, config.AutoCreateOIDCClientSecret.IsTrue())
	if err != nil {
		return err
	}

	var clientDto dto.OidcClientCreatedDto
	err = dto.MapStruct(client, &clientDto)
	if err != nil {
		return err
	}
	if createdSecret != "" {
		var secretDto dto.OidcClientSecretCreatedDto
		if err := dto.MapStruct(client.Credentials.Secrets[0], &secretDto); err != nil {
			return err
		}
		secretDto.Secret = createdSecret
		clientDto.CreatedSecret = &secretDto
	}

	c.JSON(http.StatusCreated, clientDto)
	return nil
}

// deleteClientHandler godoc
// @Summary Delete OIDC client
// @Description Delete an OIDC client by ID
// @Tags OIDC
// @Param id path string true "Client ID"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id} [delete]
func (oc *OidcController) deleteClientHandler(c *gin.Context) error {
	err := oc.oidcService.DeleteClient(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id} [put]
func (oc *OidcController) updateClientHandler(c *gin.Context) error {
	var input dto.OidcClientUpdateDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	client, err := oc.oidcService.UpdateClient(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return err
	}

	var clientDto dto.OidcClientWithAllowedUserGroupsDto
	err = dto.MapStruct(client, &clientDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, clientDto)
	return nil
}

// refreshClientMetadataHandler godoc
// @Summary Refresh client metadata document
// @Description Force a re-fetch of the OAuth Client ID Metadata Document for a CIMD client
// @Tags OIDC
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} dto.OidcClientWithAllowedUserGroupsDto "Refreshed client"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id}/refresh [post]
func (oc *OidcController) refreshClientMetadataHandler(c *gin.Context) error {
	client, err := oc.oidcService.RefreshClientMetadata(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	var clientDto dto.OidcClientWithAllowedUserGroupsDto
	err = dto.MapStruct(client, &clientDto)
	if err != nil {
		return err
	}

	clientDto.HasDarkLogo = client.HasDarkLogo()
	c.JSON(http.StatusOK, clientDto)
	return nil
}

// listClientSecretsHandler godoc
// @Summary List client secrets
// @Description List the secrets of an OIDC client, without disclosing their values
// @Tags OIDC
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {array} dto.OidcClientSecretDto "Client secrets"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id}/secrets [get]
func (oc *OidcController) listClientSecretsHandler(c *gin.Context) error {
	secrets, err := oc.oidcService.ListClientSecrets(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	var secretsDto []dto.OidcClientSecretDto
	err = dto.MapStructList(secrets, &secretsDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, secretsDto)
	return nil
}

// createClientSecretHandler godoc
// @Summary Create client secret
// @Description Add a new secret to an OIDC client, leaving the existing ones usable. The value is only returned by this endpoint and cannot be retrieved later.
// @Tags OIDC
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Param payload body dto.OidcClientSecretCreateDto false "Client secret"
// @Success 201 {object} dto.OidcClientSecretCreatedDto "Created client secret"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id}/secrets [post]
func (oc *OidcController) createClientSecretHandler(c *gin.Context) error {
	var input dto.OidcClientSecretCreateDto
	err := httpserver.BindOptionalJSON(c, &input)
	if err != nil {
		return err
	}

	created, secret, err := oc.oidcService.CreateClientSecret(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return err
	}

	var secretDto dto.OidcClientSecretCreatedDto
	err = dto.MapStruct(created, &secretDto)
	if err != nil {
		return err
	}
	secretDto.Secret = secret

	c.JSON(http.StatusCreated, secretDto)
	return nil
}

// deleteClientSecretHandler godoc
// @Summary Delete client secret
// @Description Delete a single secret of an OIDC client, making it immediately unusable
// @Tags OIDC
// @Param id path string true "Client ID"
// @Param secretId path string true "Client secret ID"
// @Success 204 "No content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id}/secrets/{secretId} [delete]
func (oc *OidcController) deleteClientSecretHandler(c *gin.Context) error {
	err := oc.oidcService.DeleteClientSecret(c.Request.Context(), c.Param("id"), c.Param("secretId"))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id}/logo [get]
func (oc *OidcController) getClientLogoHandler(c *gin.Context) error {
	lightLogo, _ := strconv.ParseBool(c.DefaultQuery("light", "true"))

	reader, size, mimeType, err := oc.oidcService.GetClientLogo(c.Request.Context(), c.Param("id"), lightLogo)
	if err != nil {
		return err
	}
	defer reader.Close()

	utils.SetCacheControlHeader(c, 15*time.Minute, 12*time.Hour)

	c.Header("Content-Type", mimeType)
	c.DataFromReader(http.StatusOK, size, mimeType, reader, nil)
	return nil
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id}/logo [post]
func (oc *OidcController) updateClientLogoHandler(c *gin.Context) error {
	file, err := httpserver.FormFile(c, "file")
	if err != nil {
		return err
	}

	lightLogo, _ := strconv.ParseBool(c.DefaultQuery("light", "true"))

	err = oc.oidcService.UpdateClientLogo(c.Request.Context(), c.Param("id"), file, lightLogo)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// deleteClientLogoHandler godoc
// @Summary Delete client logo
// @Description Delete the logo for an OIDC client
// @Tags OIDC
// @Param id path string true "Client ID"
// @Param light query boolean false "Light mode logo (true) or dark mode logo (false)"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id}/logo [delete]
func (oc *OidcController) deleteClientLogoHandler(c *gin.Context) error {
	var err error

	lightLogo, _ := strconv.ParseBool(c.DefaultQuery("light", "true"))
	if lightLogo {
		err = oc.oidcService.DeleteClientLogo(c.Request.Context(), c.Param("id"))
	} else {
		err = oc.oidcService.DeleteClientDarkLogo(c.Request.Context(), c.Param("id"))
	}

	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id}/allowed-user-groups [put]
func (oc *OidcController) updateAllowedUserGroupsHandler(c *gin.Context) error {
	var input dto.OidcUpdateAllowedUserGroupsDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	oidcClient, err := oc.oidcService.UpdateAllowedUserGroups(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		return err
	}

	var oidcClientDto dto.OidcClientDto
	if err := dto.MapStruct(oidcClient, &oidcClientDto); err != nil {
		return err
	}
	oidcClientDto.HasDarkLogo = oidcClient.HasDarkLogo()

	c.JSON(http.StatusOK, oidcClientDto)
	return nil
}

// listOwnAuthorizedClientsHandler godoc
// @Summary List authorized clients for current user
// @Description Get a paginated list of OIDC clients that the current user has authorized
// @Tags OIDC
// @Param search query string false "Search term to filter clients by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by (name or lastUsedAt)"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Param filters[hasLaunchURL] query bool false "Filter clients by whether a launch URL is configured"
// @Success 200 {object} dto.Paginated[dto.AuthorizedOidcClientDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/users/me/authorized-clients [get]
func (oc *OidcController) listOwnAuthorizedClientsHandler(c *gin.Context) error {
	userID := authz.PrincipalFrom(c).UserID
	return oc.listAuthorizedClients(c, userID)
}

// listAuthorizedClientsHandler godoc
// @Summary List authorized clients for a user
// @Description Get a paginated list of OIDC clients that a specific user has authorized
// @Tags OIDC
// @Param id path string true "User ID"
// @Param search query string false "Search term to filter clients by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by (name or lastUsedAt)"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Param filters[hasLaunchURL] query bool false "Filter clients by whether a launch URL is configured"
// @Success 200 {object} dto.Paginated[dto.AuthorizedOidcClientDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/users/{id}/authorized-clients [get]
func (oc *OidcController) listAuthorizedClientsHandler(c *gin.Context) error {
	userID := c.Param("id")
	return oc.listAuthorizedClients(c, userID)
}

func (oc *OidcController) listAuthorizedClients(c *gin.Context, userID string) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	authorizedClients, pagination, err := oc.oidcService.ListAuthorizedClients(c.Request.Context(), userID, searchTerm, listRequestOptions)
	if err != nil {
		return err
	}

	// Map the clients to DTOs
	var authorizedClientsDto []dto.AuthorizedOidcClientDto
	if err := dto.MapStructList(authorizedClients, &authorizedClientsDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.AuthorizedOidcClientDto]{
		Data:       authorizedClientsDto,
		Pagination: pagination,
	})
	return nil
}

// revokeOwnClientAuthorizationHandler godoc
// @Summary Revoke authorization for an OIDC client
// @Description Revoke the authorization for a specific OIDC client for the current user
// @Tags OIDC
// @Param clientId path string true "Client ID to revoke authorization for"
// @Success 204 "No Content"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/users/me/authorized-clients/{clientId} [delete]
func (oc *OidcController) revokeOwnClientAuthorizationHandler(c *gin.Context) error {
	clientID := c.Param("clientId")

	userID := authz.PrincipalFrom(c).UserID

	err := oc.oidcService.RevokeAuthorizedClient(c.Request.Context(), userID, clientID)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// listOwnAccessibleClientsHandler godoc
// @Summary List accessible OIDC clients for current user
// @Description Get a list of OIDC clients that the current user can access
// @Tags OIDC
// @Param search query string false "Search term to filter clients by name"
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by (name or lastUsedAt)"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Param filters[hasLaunchURL] query bool false "Filter clients by whether a launch URL is configured"
// @Success 200 {object} dto.Paginated[dto.AccessibleOidcClientDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/users/me/clients [get]
func (oc *OidcController) listOwnAccessibleClientsHandler(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	userID := authz.PrincipalFrom(c).UserID

	clients, pagination, err := oc.oidcService.ListAccessibleOidcClients(c.Request.Context(), userID, searchTerm, listRequestOptions)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.AccessibleOidcClientDto]{
		Data:       clients,
		Pagination: pagination,
	})
	return nil
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
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id}/preview/{userId} [get]
func (oc *OidcController) getClientPreviewHandler(c *gin.Context) error {
	clientID := c.Param("id")
	userID := c.Param("userId")
	scopes := c.Query("scopes")

	if clientID == "" {
		return apperror.MissingField("clientId")
	}

	if userID == "" {
		return apperror.MissingField("userId")
	}

	if scopes == "" {
		return apperror.MissingField("scopes")
	}

	preview, err := oc.oidcService.GetClientPreview(
		c.Request.Context(),
		clientID,
		userID,
		strings.Split(scopes, " "),
		authz.PrincipalFrom(c).AuthenticationMethod)

	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, preview)
	return nil
}
