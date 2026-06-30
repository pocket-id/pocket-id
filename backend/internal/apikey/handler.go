package apikey

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
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
func (h *handler) list(c *gin.Context) {
	listRequestOptions := utils.ParseListRequestOptions(c)

	userID := c.GetString("userID")

	apiKeys, pagination, err := h.service.ListApiKeys(c.Request.Context(), userID, listRequestOptions)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var apiKeysDto []apiKeyDto
	if err := dto.MapStructList(apiKeys, &apiKeysDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.Paginated[apiKeyDto]{
		Data:       apiKeysDto,
		Pagination: pagination,
	})
}

// create godoc
// @Summary Create API key
// @Description Create a new API key for the current user
// @Tags API Keys
// @Param api_key body apiKeyCreateDto true "API key information"
// @Success 201 {object} apiKeyResponseDto "Created API key with token"
// @Router /api/api-keys [post]
func (h *handler) create(c *gin.Context) {
	userID := c.GetString("userID")

	var input apiKeyCreateDto
	if err := dto.ShouldBindWithNormalizedJSON(c, &input); err != nil {
		_ = c.Error(err)
		return
	}

	apiKey, token, err := h.service.CreateApiKey(c.Request.Context(), userID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var responseDto apiKeyDto
	if err := dto.MapStruct(apiKey, &responseDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, apiKeyResponseDto{
		ApiKey: responseDto,
		Token:  token,
	})
}

// renew godoc
// @Summary Renew API key
// @Description Renew an existing API key by ID
// @Tags API Keys
// @Param id path string true "API Key ID"
// @Success 200 {object} apiKeyResponseDto "Renewed API key with new token"
// @Router /api/api-keys/{id}/renew [post]
func (h *handler) renew(c *gin.Context) {
	userID := c.GetString("userID")
	apiKeyID := c.Param("id")

	var input apiKeyRenewDto
	if err := dto.ShouldBindWithNormalizedJSON(c, &input); err != nil {
		_ = c.Error(err)
		return
	}

	apiKey, token, err := h.service.RenewApiKey(c.Request.Context(), userID, apiKeyID, input.ExpiresAt.ToTime())
	if err != nil {
		_ = c.Error(err)
		return
	}

	var responseDto apiKeyDto
	if err := dto.MapStruct(apiKey, &responseDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, apiKeyResponseDto{
		ApiKey: responseDto,
		Token:  token,
	})
}

// revoke godoc
// @Summary Revoke API key
// @Description Revoke (delete) an existing API key by ID
// @Tags API Keys
// @Param id path string true "API Key ID"
// @Success 204 "No Content"
// @Router /api/api-keys/{id} [delete]
func (h *handler) revoke(c *gin.Context) {
	userID := c.GetString("userID")
	apiKeyID := c.Param("id")

	if err := h.service.RevokeApiKey(c.Request.Context(), userID, apiKeyID); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
