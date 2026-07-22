package apikey

import (
	"context"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type createInput struct {
	Body apiKeyCreateDto
}

type renewInput struct {
	ID   string `path:"id"`
	Body apiKeyRenewDto
}

type idInput struct {
	ID string `path:"id"`
}

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

func (h *handler) list(ctx context.Context, input *httpapi.ListInput) (*httpapi.BodyOutput[dto.Paginated[apiKeyDto]], error) {
	apiKeys, pagination, err := h.service.ListApiKeys(ctx, httpapi.UserID(ctx), input.ListRequestOptions)
	if err != nil {
		return nil, err
	}

	var output []apiKeyDto
	if err := dto.MapStructList(apiKeys, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.Paginated[apiKeyDto]]{Body: dto.Paginated[apiKeyDto]{Data: output, Pagination: pagination}}, nil
}

func (h *handler) create(ctx context.Context, input *createInput) (*httpapi.BodyOutput[apiKeyResponseDto], error) {
	apiKey, token, err := h.service.CreateApiKey(ctx, httpapi.UserID(ctx), input.Body)
	if err != nil {
		return nil, err
	}
	return mapAPIKeyResponse(apiKey, token)
}

func (h *handler) renew(ctx context.Context, input *renewInput) (*httpapi.BodyOutput[apiKeyResponseDto], error) {
	apiKey, token, err := h.service.RenewApiKey(ctx, httpapi.UserID(ctx), input.ID, input.Body.ExpiresAt.ToTime())
	if err != nil {
		return nil, err
	}
	return mapAPIKeyResponse(apiKey, token)
}

func (h *handler) revoke(ctx context.Context, input *idInput) (*httpapi.EmptyOutput, error) {
	if err := h.service.RevokeApiKey(ctx, httpapi.UserID(ctx), input.ID); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func mapAPIKeyResponse(apiKey ApiKey, token string) (*httpapi.BodyOutput[apiKeyResponseDto], error) {
	var output apiKeyDto
	if err := dto.MapStruct(apiKey, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[apiKeyResponseDto]{Body: apiKeyResponseDto{ApiKey: output, Token: token}}, nil
}
