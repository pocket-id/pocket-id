package controller

import (
	"net/http"

	"github.com/pocket-id/pocket-id/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// swag init -g cmd/main.go -o ./docs/swagger --parseDependency

// ApiKeyController manages API keys for authenticated users
type ApiKeyController struct {
	apiKeyService *service.ApiKeyService
}

// NewApiKeyController creates a new controller for API key management
// @Summary API key management controller
// @Description Initializes API endpoints for managing API keys
// @Tags API Keys
func NewApiKeyController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, apiKeyService *service.ApiKeyService) {
	uc := &ApiKeyController{apiKeyService: apiKeyService}

	apiKeyGroup := group.Group("/api-keys")
	apiKeyGroup.Use(authMiddleware.WithAdminNotRequired().Add())
	{
		apiKeyGroup.GET("", uc.listApiKeysHandler)
		apiKeyGroup.POST("", uc.createApiKeyHandler)
		apiKeyGroup.DELETE("/:id", uc.revokeApiKeyHandler)
	}
}

// listApiKeysHandler godoc
// @Summary List API keys
// @Description Get a paginated list of API keys belonging to the current user
// @Tags API Keys
// @Param page query int false "Page number, starting from 1" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param sort_column query string false "Column to sort by" default("created_at")
// @Param sort_direction query string false "Sort direction (asc or desc)" default("desc")
// @Success 200 {object} dto.Paginated[dto.ApiKeyDto]
// @Router /api-keys [get]
func (c *ApiKeyController) listApiKeysHandler(ctx *gin.Context) {
	userID := ctx.GetString("userID")

	var sortedPaginationRequest utils.SortedPaginationRequest
	if err := ctx.ShouldBindQuery(&sortedPaginationRequest); err != nil {
		ctx.Error(err)
		return
	}

	apiKeys, pagination, err := c.apiKeyService.ListApiKeys(userID, sortedPaginationRequest)
	if err != nil {
		ctx.Error(err)
		return
	}

	var apiKeysDto []dto.ApiKeyDto
	if err := dto.MapStructList(apiKeys, &apiKeysDto); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Paginated[dto.ApiKeyDto]{
		Data:       apiKeysDto,
		Pagination: pagination,
	})
}

// createApiKeyHandler godoc
// @Summary Create API key
// @Description Create a new API key for the current user
// @Tags API Keys
// @Param api_key body dto.ApiKeyCreateDto true "API key information"
// @Success 201 {object} dto.ApiKeyResponseDto "Created API key with token"
// @Router /api-keys [post]
func (c *ApiKeyController) createApiKeyHandler(ctx *gin.Context) {
	userID := ctx.GetString("userID")

	var input dto.ApiKeyCreateDto
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.Error(err)
		return
	}

	apiKey, token, err := c.apiKeyService.CreateApiKey(userID, input)
	if err != nil {
		ctx.Error(err)
		return
	}

	var apiKeyDto dto.ApiKeyDto
	if err := dto.MapStruct(apiKey, &apiKeyDto); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, dto.ApiKeyResponseDto{
		ApiKey: apiKeyDto,
		Token:  token,
	})
}

// revokeApiKeyHandler godoc
// @Summary Revoke API key
// @Description Revoke (delete) an existing API key by ID
// @Tags API Keys
// @Param id path string true "API Key ID"
// @Success 204 "No Content"
// @Router /api-keys/{id} [delete]
func (c *ApiKeyController) revokeApiKeyHandler(ctx *gin.Context) {
	userID := ctx.GetString("userID")
	apiKeyID := ctx.Param("id")

	if err := c.apiKeyService.RevokeApiKey(userID, apiKeyID); err != nil {
		ctx.Error(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
