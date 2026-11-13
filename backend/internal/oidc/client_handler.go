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

func (h *clientHandler) getClientHandler(c *gin.Context) (dto.OidcClientWithAllowedUserGroupsDto, error) {
	clientId := c.Param("id")
	client, err := h.service.GetClient(c.Request.Context(), clientId)
	if err != nil {
		return dto.OidcClientWithAllowedUserGroupsDto{}, err
	}

	clientDto := dto.OidcClientWithAllowedUserGroupsDto{}
	err = dto.MapStruct(client, &clientDto)
	if err != nil {
		return dto.OidcClientWithAllowedUserGroupsDto{}, err
	}

	return clientDto, nil
}

func (h *clientHandler) listClientsHandler(c *gin.Context) (dto.Paginated[dto.OidcClientWithAllowedGroupsDto], error) {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	clients, pagination, err := h.service.ListClients(c.Request.Context(), searchTerm, listRequestOptions)
	if err != nil {
		return dto.Paginated[dto.OidcClientWithAllowedGroupsDto]{}, err
	}

	// Map the user groups to DTOs
	var clientsDto = make([]dto.OidcClientWithAllowedGroupsDto, len(clients))
	for i, client := range clients {
		var clientDto dto.OidcClientWithAllowedGroupsDto
		if err := dto.MapStruct(client, &clientDto); err != nil {
			return dto.Paginated[dto.OidcClientWithAllowedGroupsDto]{}, err
		}
		clientDto.HasDarkLogo = client.HasDarkLogo()
		clientsDto[i] = clientDto
	}

	return dto.Paginated[dto.OidcClientWithAllowedGroupsDto]{
		Data:       clientsDto,
		Pagination: pagination,
	}, nil
}

func (h *clientHandler) createClientHandler(c *gin.Context, input dto.OidcClientCreateDto) (dto.OidcClientCreatedDto, error) {
	config, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return dto.OidcClientCreatedDto{}, err
	}

	client, createdSecret, err := h.service.CreateClient(c.Request.Context(), input, auditlogs.OriginFrom(c), config.AutoCreateOIDCClientSecret.IsTrue())
	if err != nil {
		return dto.OidcClientCreatedDto{}, err
	}

	var clientDto dto.OidcClientCreatedDto
	err = dto.MapStruct(client, &clientDto)
	if err != nil {
		return dto.OidcClientCreatedDto{}, err
	}
	if createdSecret != "" {
		var secretDto dto.OidcClientSecretCreatedDto
		if err := dto.MapStruct(client.Credentials.Secrets[0], &secretDto); err != nil {
			return dto.OidcClientCreatedDto{}, err
		}
		secretDto.Secret = createdSecret
		clientDto.CreatedSecret = &secretDto
	}

	return clientDto, nil
}

func (h *clientHandler) deleteClientHandler(c *gin.Context) error {
	return h.service.DeleteClient(c.Request.Context(), c.Param("id"), auditlogs.OriginFrom(c))
}

func (h *clientHandler) updateClientHandler(c *gin.Context, update dto.Update[dto.OidcClientUpdateDto]) (dto.OidcClientWithAllowedUserGroupsDto, error) {
	client, err := h.service.UpdateClient(c.Request.Context(), c.Param("id"), update, auditlogs.OriginFrom(c))
	if err != nil {
		return dto.OidcClientWithAllowedUserGroupsDto{}, err
	}

	var clientDto dto.OidcClientWithAllowedUserGroupsDto
	err = dto.MapStruct(client, &clientDto)
	if err != nil {
		return dto.OidcClientWithAllowedUserGroupsDto{}, err
	}

	return clientDto, nil
}

func (h *clientHandler) refreshClientMetadataHandler(c *gin.Context) (dto.OidcClientWithAllowedUserGroupsDto, error) {
	client, err := h.service.RefreshClientMetadata(c.Request.Context(), c.Param("id"))
	if err != nil {
		return dto.OidcClientWithAllowedUserGroupsDto{}, err
	}

	var clientDto dto.OidcClientWithAllowedUserGroupsDto
	err = dto.MapStruct(client, &clientDto)
	if err != nil {
		return dto.OidcClientWithAllowedUserGroupsDto{}, err
	}

	clientDto.HasDarkLogo = client.HasDarkLogo()
	return clientDto, nil
}

func (h *clientHandler) listClientSecretsHandler(c *gin.Context) ([]dto.OidcClientSecretDto, error) {
	secrets, err := h.service.ListClientSecrets(c.Request.Context(), c.Param("id"))
	if err != nil {
		return nil, err
	}

	var secretsDto []dto.OidcClientSecretDto
	err = dto.MapStructList(secrets, &secretsDto)
	if err != nil {
		return nil, err
	}

	return secretsDto, nil
}

func (h *clientHandler) createClientSecretHandler(c *gin.Context) (dto.OidcClientSecretCreatedDto, error) {
	// The body is optional, so it is bound here instead of by the endpoint
	var input dto.OidcClientSecretCreateDto
	err := httpserver.BindOptionalJSON(c, &input)
	if err != nil {
		return dto.OidcClientSecretCreatedDto{}, err
	}

	created, secret, err := h.service.CreateClientSecret(c.Request.Context(), c.Param("id"), input, auditlogs.OriginFrom(c))
	if err != nil {
		return dto.OidcClientSecretCreatedDto{}, err
	}

	var secretDto dto.OidcClientSecretCreatedDto
	err = dto.MapStruct(created, &secretDto)
	secretDto.Secret = secret
	return secretDto, err
}

func (h *clientHandler) deleteClientSecretHandler(c *gin.Context) error {
	return h.service.DeleteClientSecret(c.Request.Context(), c.Param("id"), c.Param("secretId"), auditlogs.OriginFrom(c))
}

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

func (h *clientHandler) updateClientLogoHandler(c *gin.Context) error {
	file, err := httpserver.FormFile(c, "file")
	if err != nil {
		return err
	}

	lightLogo, _ := strconv.ParseBool(c.DefaultQuery("light", "true"))

	return h.service.UpdateClientLogo(c.Request.Context(), c.Param("id"), file, lightLogo)
}

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

	return nil
}

func (h *clientHandler) updateAllowedUserGroupsHandler(c *gin.Context, input dto.OidcUpdateAllowedUserGroupsDto) (dto.OidcClientDto, error) {
	oidcClient, err := h.service.UpdateAllowedUserGroups(c.Request.Context(), c.Param("id"), input, auditlogs.OriginFrom(c))
	if err != nil {
		return dto.OidcClientDto{}, err
	}

	var oidcClientDto dto.OidcClientDto
	if err := dto.MapStruct(oidcClient, &oidcClientDto); err != nil {
		return dto.OidcClientDto{}, err
	}
	oidcClientDto.HasDarkLogo = oidcClient.HasDarkLogo()

	return oidcClientDto, nil
}

func (h *clientHandler) listOwnAuthorizedClientsHandler(c *gin.Context) (dto.Paginated[dto.AuthorizedOidcClientDto], error) {
	return h.listAuthorizedClients(c, authz.PrincipalFrom(c).UserID)
}

func (h *clientHandler) listAuthorizedClientsHandler(c *gin.Context) (dto.Paginated[dto.AuthorizedOidcClientDto], error) {
	return h.listAuthorizedClients(c, c.Param("id"))
}

func (h *clientHandler) listAuthorizedClients(c *gin.Context, userID string) (dto.Paginated[dto.AuthorizedOidcClientDto], error) {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	authorizedClients, pagination, err := h.service.ListAuthorizedClients(c.Request.Context(), userID, searchTerm, listRequestOptions)
	if err != nil {
		return dto.Paginated[dto.AuthorizedOidcClientDto]{}, err
	}

	// Map the clients to DTOs
	var authorizedClientsDto []dto.AuthorizedOidcClientDto
	if err := dto.MapStructList(authorizedClients, &authorizedClientsDto); err != nil {
		return dto.Paginated[dto.AuthorizedOidcClientDto]{}, err
	}

	return dto.Paginated[dto.AuthorizedOidcClientDto]{Data: authorizedClientsDto, Pagination: pagination}, nil
}

func (h *clientHandler) revokeOwnClientAuthorizationHandler(c *gin.Context) error {
	origin := auditlogs.OriginFrom(c)
	return h.service.RevokeAuthorizedClient(c.Request.Context(), origin.Principal.UserID, c.Param("clientId"), origin)
}

func (h *clientHandler) listOwnAccessibleClientsHandler(c *gin.Context) (dto.Paginated[dto.AccessibleOidcClientDto], error) {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	userID := authz.PrincipalFrom(c).UserID

	clients, pagination, err := h.service.ListAccessibleOidcClients(c.Request.Context(), userID, searchTerm, listRequestOptions)
	if err != nil {
		return dto.Paginated[dto.AccessibleOidcClientDto]{}, err
	}

	return dto.Paginated[dto.AccessibleOidcClientDto]{
		Data:       clients,
		Pagination: pagination,
	}, nil
}

func (h *clientHandler) getClientPreviewHandler(c *gin.Context) (dto.OidcClientPreviewDto, error) {
	clientID := c.Param("id")
	userID := c.Param("userId")
	scopes := c.Query("scopes")

	if clientID == "" {
		return dto.OidcClientPreviewDto{}, apperror.MissingField("clientId")
	}

	if userID == "" {
		return dto.OidcClientPreviewDto{}, apperror.MissingField("userId")
	}

	preview, err := h.service.GetClientPreview(
		c.Request.Context(),
		clientID,
		userID,
		strings.Split(scopes, " "),
		authz.PrincipalFrom(c).AuthenticationMethod)

	if err != nil {
		return dto.OidcClientPreviewDto{}, err
	}

	return *preview, nil
}
