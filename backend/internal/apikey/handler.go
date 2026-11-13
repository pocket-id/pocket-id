package apikey

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

func (h *handler) list(c *gin.Context) (dto.Paginated[apiKeyDto], error) {
	listRequestOptions := utils.ParseListRequestOptions(c)

	userID := authz.PrincipalFrom(c).UserID

	apiKeys, pagination, err := h.service.ListApiKeys(c.Request.Context(), userID, listRequestOptions)
	if err != nil {
		return dto.Paginated[apiKeyDto]{}, err
	}

	var apiKeysDto []apiKeyDto
	err = dto.MapStructList(apiKeys, &apiKeysDto)
	if err != nil {
		return dto.Paginated[apiKeyDto]{}, err
	}

	return dto.Paginated[apiKeyDto]{
		Data:       apiKeysDto,
		Pagination: pagination,
	}, nil
}

func (h *handler) create(c *gin.Context, input apiKeyCreateDto) (apiKeyResponseDto, error) {
	origin := auditlogs.OriginFrom(c)

	apiKey, token, err := h.service.CreateApiKey(c.Request.Context(), origin.Principal.UserID, grantableScopes(origin.Principal.Scopes), input, origin)
	if err != nil {
		return apiKeyResponseDto{}, err
	}

	var responseDto apiKeyDto
	err = dto.MapStruct(apiKey, &responseDto)
	if err != nil {
		return apiKeyResponseDto{}, err
	}

	return apiKeyResponseDto{
		ApiKey: responseDto,
		Token:  token,
	}, nil
}

func (h *handler) update(c *gin.Context, update dto.Update[apiKeyUpdateDto]) (apiKeyDto, error) {
	principal := authz.PrincipalFrom(c)

	apiKey, err := h.service.UpdateApiKey(c.Request.Context(), principal.UserID, c.Param("id"), grantableScopes(principal.Scopes), update)
	if err != nil {
		return apiKeyDto{}, err
	}

	var responseDto apiKeyDto
	err = dto.MapStruct(apiKey, &responseDto)
	if err != nil {
		return apiKeyDto{}, err
	}

	return responseDto, nil
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

func (h *handler) renew(c *gin.Context, input apiKeyRenewDto) (apiKeyResponseDto, error) {
	origin := auditlogs.OriginFrom(c)

	apiKey, token, err := h.service.RenewApiKey(c.Request.Context(), origin.Principal.UserID, c.Param("id"), input.ExpiresAt.ToTime(), time.Duration(input.GracePeriodHours)*time.Hour, origin)
	if err != nil {
		return apiKeyResponseDto{}, err
	}

	var responseDto apiKeyDto
	err = dto.MapStruct(apiKey, &responseDto)
	if err != nil {
		return apiKeyResponseDto{}, err
	}

	return apiKeyResponseDto{
		ApiKey: responseDto,
		Token:  token,
	}, nil
}

func (h *handler) revoke(c *gin.Context) error {
	origin := auditlogs.OriginFrom(c)
	return h.service.RevokeApiKey(c.Request.Context(), origin.Principal.UserID, c.Param("id"), origin)
}
