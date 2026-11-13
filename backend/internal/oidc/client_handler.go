package oidc

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type clientHandler struct {
	service   *clientService
	appConfig appconfig.AppConfigResolver
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
func (h *clientHandler) getClientHandler(c *gin.Context) error {
	clientId := c.Param("id")
	client, err := h.service.GetClient(c.Request.Context(), clientId)
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
func (h *clientHandler) listClientsHandler(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	clients, pagination, err := h.service.ListClients(c.Request.Context(), searchTerm, listRequestOptions)
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
func (h *clientHandler) createClientHandler(c *gin.Context) error {
	var input dto.OidcClientCreateDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	config, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return err
	}

	client, createdSecret, err := h.service.CreateClient(c.Request.Context(), input, auditlogs.OriginFrom(c), config.AutoCreateOIDCClientSecret.IsTrue())
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
func (h *clientHandler) deleteClientHandler(c *gin.Context) error {
	err := h.service.DeleteClient(c.Request.Context(), c.Param("id"), auditlogs.OriginFrom(c))
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// updateClientHandler godoc
// @Summary Update OIDC client
// @Description Update an existing OIDC client, where PATCH changes only the fields in the body
// @Tags OIDC
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Param client body dto.OidcClientUpdateDto true "Client information"
// @Success 200 {object} dto.OidcClientWithAllowedUserGroupsDto "Updated client"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id} [put]
// @Router /api/oidc/clients/{id} [patch]
func (h *clientHandler) updateClientHandler(c *gin.Context) error {
	update, err := httpserver.BindUpdate[dto.OidcClientUpdateDto](c)
	if err != nil {
		return err
	}

	client, err := h.service.UpdateClient(c.Request.Context(), c.Param("id"), update, auditlogs.OriginFrom(c))
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
func (h *clientHandler) refreshClientMetadataHandler(c *gin.Context) error {
	client, err := h.service.RefreshClientMetadata(c.Request.Context(), c.Param("id"))
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
func (h *clientHandler) listClientSecretsHandler(c *gin.Context) error {
	secrets, err := h.service.ListClientSecrets(c.Request.Context(), c.Param("id"))
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
func (h *clientHandler) createClientSecretHandler(c *gin.Context) error {
	var input dto.OidcClientSecretCreateDto
	err := httpserver.BindOptionalJSON(c, &input)
	if err != nil {
		return err
	}

	created, secret, err := h.service.CreateClientSecret(c.Request.Context(), c.Param("id"), input, auditlogs.OriginFrom(c))
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
func (h *clientHandler) deleteClientSecretHandler(c *gin.Context) error {
	err := h.service.DeleteClientSecret(c.Request.Context(), c.Param("id"), c.Param("secretId"), auditlogs.OriginFrom(c))
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
func (h *clientHandler) getClientLogoHandler(c *gin.Context) error {
	lightLogo, _ := strconv.ParseBool(c.DefaultQuery("light", "true"))

	reader, size, mimeType, err := h.service.GetClientLogo(c.Request.Context(), c.Param("id"), lightLogo)
	if err != nil {
		return err
	}
	defer reader.Close()

	utils.SetCacheControlHeader(c, 15*time.Minute, 12*time.Hour)

	c.Header("Content-Type", mimeType)
	// An uploaded SVG opened directly renders as a document, so a second policy sandboxes it to keep it from running scripts
	c.Writer.Header().Add("Content-Security-Policy", "sandbox")
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
func (h *clientHandler) updateClientLogoHandler(c *gin.Context) error {
	file, err := httpserver.FormFile(c, "file")
	if err != nil {
		return err
	}

	lightLogo, _ := strconv.ParseBool(c.DefaultQuery("light", "true"))

	err = h.service.UpdateClientLogo(c.Request.Context(), c.Param("id"), file, lightLogo)
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
func (h *clientHandler) deleteClientLogoHandler(c *gin.Context) error {
	var err error

	lightLogo, _ := strconv.ParseBool(c.DefaultQuery("light", "true"))
	if lightLogo {
		err = h.service.DeleteClientLogo(c.Request.Context(), c.Param("id"))
	} else {
		err = h.service.DeleteClientDarkLogo(c.Request.Context(), c.Param("id"))
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
func (h *clientHandler) updateAllowedUserGroupsHandler(c *gin.Context) error {
	var input dto.OidcUpdateAllowedUserGroupsDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	oidcClient, err := h.service.UpdateAllowedUserGroups(c.Request.Context(), c.Param("id"), input, auditlogs.OriginFrom(c))
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
func (h *clientHandler) listOwnAuthorizedClientsHandler(c *gin.Context) error {
	userID := authz.PrincipalFrom(c).UserID
	return h.listAuthorizedClients(c, userID)
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
func (h *clientHandler) listAuthorizedClientsHandler(c *gin.Context) error {
	userID := c.Param("id")
	return h.listAuthorizedClients(c, userID)
}

func (h *clientHandler) listAuthorizedClients(c *gin.Context, userID string) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	authorizedClients, pagination, err := h.service.ListAuthorizedClients(c.Request.Context(), userID, searchTerm, listRequestOptions)
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
func (h *clientHandler) revokeOwnClientAuthorizationHandler(c *gin.Context) error {
	origin := auditlogs.OriginFrom(c)
	err := h.service.RevokeAuthorizedClient(c.Request.Context(), origin.Principal.UserID, c.Param("clientId"), origin)
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
func (h *clientHandler) listOwnAccessibleClientsHandler(c *gin.Context) error {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	userID := authz.PrincipalFrom(c).UserID

	clients, pagination, err := h.service.ListAccessibleOidcClients(c.Request.Context(), userID, searchTerm, listRequestOptions)
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
// @Param scopes query string false "Scopes to include in the preview (space-separated)"
// @Success 200 {object} dto.OidcClientPreviewDto "Preview data including ID token, access token, and userinfo payloads"
// @Security BearerAuth
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/oidc/clients/{id}/preview/{userId} [get]
func (h *clientHandler) getClientPreviewHandler(c *gin.Context) error {
	clientID := c.Param("id")
	userID := c.Param("userId")
	scopes := c.Query("scopes")

	if clientID == "" {
		return apperror.MissingField("clientId")
	}

	if userID == "" {
		return apperror.MissingField("userId")
	}

	preview, err := h.service.GetClientPreview(
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
