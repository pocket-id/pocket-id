package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type ApiKeyController struct {
	apiKeyService *service.ApiKeyService
}

func NewApiKeyController(group *gin.RouterGroup, jwtAuthMiddleware *middleware.JwtAuthMiddleware, apiKeyService *service.ApiKeyService) {
	uc := &ApiKeyController{apiKeyService: apiKeyService}

	apiKeyGroup := group.Group("/api-keys")
	apiKeyGroup.Use(jwtAuthMiddleware.Add(false))
	{
		apiKeyGroup.GET("", uc.listApiKeysHandler)
		apiKeyGroup.POST("", uc.createApiKeyHandler)
		apiKeyGroup.DELETE("/:id", uc.revokeApiKeyHandler)
	}
}

func (c *ApiKeyController) listApiKeysHandler(ctx *gin.Context) {
	userID := ctx.GetString("userID")

	apiKeys, err := c.apiKeyService.ListApiKeys(userID)
	if err != nil {
		ctx.Error(err)
		return
	}

	var apiKeysDto []dto.ApiKeyDto
	if err := dto.MapStructList(apiKeys, &apiKeysDto); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, apiKeysDto)
}

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

func (c *ApiKeyController) revokeApiKeyHandler(ctx *gin.Context) {
	userID := ctx.GetString("userID")
	apiKeyID := ctx.Param("id")

	if err := c.apiKeyService.RevokeApiKey(userID, apiKeyID); err != nil {
		ctx.Error(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
