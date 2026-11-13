package apikey

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
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
// @Summary List API keys
// @Description Get a paginated list of API keys belonging to the current user
// @Tags API Keys
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[apiKeyDto]
// @Router /api/api-keys [get]
func (h *handler) list(c *gin.Context) error {
	listRequestOptions := utils.ParseListRequestOptions(c)

	userID := authz.PrincipalFrom(c).UserID

	apiKeys, pagination, err := h.service.ListApiKeys(c.Request.Context(), userID, listRequestOptions)
	if err != nil {
		return err
	}

	var apiKeysDto []apiKeyDto
	err = dto.MapStructList(apiKeys, &apiKeysDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[apiKeyDto]{
		Data:       apiKeysDto,
		Pagination: pagination,
	})
	return nil
}

// create godoc
// @Summary Create API key
// @Description Create a new API key for the current user
// @Tags API Keys
// @Param api_key body apiKeyCreateDto true "API key information"
// @Success 201 {object} apiKeyResponseDto "Created API key with token"
// @Router /api/api-keys [post]
func (h *handler) create(c *gin.Context) error {
	origin := auditlogs.OriginFrom(c)

	var input apiKeyCreateDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	apiKey, token, err := h.service.CreateApiKey(c.Request.Context(), origin.Principal.UserID, grantableScopes(origin.Principal.Scopes), input, origin)
	if err != nil {
		return err
	}

	var responseDto apiKeyDto
	err = dto.MapStruct(apiKey, &responseDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusCreated, apiKeyResponseDto{
		ApiKey: responseDto,
		Token:  token,
	})
	return nil
}

// update godoc
// @Summary Update API key
// @Description Change the name, description and scopes of an existing API key, where PATCH changes only the fields in the body; its token stays the same
// @Tags API Keys
// @Param id path string true "API Key ID"
// @Param api_key body apiKeyUpdateDto true "API key information"
// @Success 200 {object} apiKeyDto
// @Router /api/api-keys/{id} [put]
// @Router /api/api-keys/{id} [patch]
func (h *handler) update(c *gin.Context) error {
	principal := authz.PrincipalFrom(c)

	var input apiKeyUpdateDto
	err := httpserver.BindUpdateJSON(c, &input, func() (any, error) { return h.service.GetApiKey(c.Request.Context(), principal.UserID, c.Param("id")) })
	if err != nil {
		return err
	}

	apiKey, err := h.service.UpdateApiKey(c.Request.Context(), principal.UserID, c.Param("id"), grantableScopes(principal.Scopes), input)
	if err != nil {
		return err
	}

	var responseDto apiKeyDto
	err = dto.MapStruct(apiKey, &responseDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, responseDto)
	return nil
}

// grantableScopes returns the held scopes that an API key can carry
// Deriving them from the caller's scopes keeps the role rule in authz.UserScopes and keeps session-only scopes off API keys
func grantableScopes(held authz.ScopeSet) authz.ScopeSet {
	set := make(authz.ScopeSet, len(held))
	for scope := range held {
		if scope.GrantableTo(authz.KindAPIKey) {
			set[scope] = struct{}{}
		}
	}
	return set
}

// renew godoc
// @Summary Renew API key
// @Description Replace the token of an API key, which also rotates a key that has not expired yet
// @Description While a key has not expired, its old token keeps working for the given grace period, but never past the key's previous expiration
// @Tags API Keys
// @Param id path string true "API Key ID"
// @Param api_key body apiKeyRenewDto true "New expiration and grace period"
// @Success 200 {object} apiKeyResponseDto "Renewed API key with new token"
// @Router /api/api-keys/{id}/renew [post]
func (h *handler) renew(c *gin.Context) error {
	origin := auditlogs.OriginFrom(c)

	var input apiKeyRenewDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	apiKey, token, err := h.service.RenewApiKey(c.Request.Context(), origin.Principal.UserID, c.Param("id"), input.ExpiresAt.ToTime(), time.Duration(input.GracePeriodHours)*time.Hour, origin)
	if err != nil {
		return err
	}

	var responseDto apiKeyDto
	err = dto.MapStruct(apiKey, &responseDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, apiKeyResponseDto{
		ApiKey: responseDto,
		Token:  token,
	})
	return nil
}

// revoke godoc
// @Summary Revoke API key
// @Description Revoke (delete) an existing API key by ID
// @Tags API Keys
// @Param id path string true "API Key ID"
// @Success 204 "No Content"
// @Router /api/api-keys/{id} [delete]
func (h *handler) revoke(c *gin.Context) error {
	origin := auditlogs.OriginFrom(c)
	err := h.service.RevokeApiKey(c.Request.Context(), origin.Principal.UserID, c.Param("id"), origin)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}
