package api

import (
	"context"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type listInput struct {
	httpapi.ListInput
	Search string `query:"search" required:"false"`
}

type idInput struct {
	ID string `path:"id"`
}

type createInput struct {
	Body apiCreateDto
}

type updateInput struct {
	ID   string `path:"id"`
	Body apiUpdateDto
}

type permissionsInput struct {
	ID   string `path:"id"`
	Body apiPermissionsUpdateDto
}

type clientInput struct {
	ClientID string `path:"clientId"`
}

type clientUpdateInput struct {
	ClientID string `path:"clientId"`
	Body     clientApiAccessUpdateDto
}

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

func (h *handler) list(ctx context.Context, input *listInput) (*httpapi.BodyOutput[dto.Paginated[apiResponseDto]], error) {
	apis, pagination, err := h.service.List(ctx, input.Search, input.ListRequestOptions)
	if err != nil {
		return nil, err
	}

	items := make([]apiResponseDto, len(apis))
	for i := range apis {
		if err := dto.MapStruct(apis[i], &items[i]); err != nil {
			return nil, err
		}
		items[i].Resource = apis[i].Audience
	}
	return &httpapi.BodyOutput[dto.Paginated[apiResponseDto]]{Body: dto.Paginated[apiResponseDto]{Data: items, Pagination: pagination}}, nil
}

func (h *handler) get(ctx context.Context, input *idInput) (*httpapi.BodyOutput[apiResponseDto], error) {
	item, err := h.service.Get(ctx, nil, input.ID)
	if err != nil {
		return nil, err
	}
	return mapAPI(item)
}

func (h *handler) create(ctx context.Context, input *createInput) (*httpapi.BodyOutput[apiResponseDto], error) {
	item, err := h.service.Create(ctx, input.Body)
	if err != nil {
		return nil, err
	}
	return mapAPI(item)
}

func (h *handler) update(ctx context.Context, input *updateInput) (*httpapi.BodyOutput[apiResponseDto], error) {
	item, err := h.service.Update(ctx, input.ID, input.Body)
	if err != nil {
		return nil, err
	}
	return mapAPI(item)
}

func (h *handler) delete(ctx context.Context, input *idInput) (*httpapi.EmptyOutput, error) {
	if err := h.service.Delete(ctx, input.ID); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (h *handler) updatePermissions(ctx context.Context, input *permissionsInput) (*httpapi.BodyOutput[apiResponseDto], error) {
	item, err := h.service.UpdatePermissions(ctx, input.ID, input.Body)
	if err != nil {
		return nil, err
	}
	return mapAPI(item)
}

func (h *handler) getClientAccess(ctx context.Context, input *clientInput) (*httpapi.BodyOutput[clientApiAccessDto], error) {
	access, err := h.service.GetClientAPIAccess(ctx, input.ClientID)
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[clientApiAccessDto]{Body: newClientAPIAccessDTO(access)}, nil
}

func (h *handler) updateClientAccess(ctx context.Context, input *clientUpdateInput) (*httpapi.BodyOutput[clientApiAccessDto], error) {
	applied, err := h.service.SetClientAPIAccess(ctx, input.ClientID, ClientAPIAccess(input.Body))
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[clientApiAccessDto]{Body: newClientAPIAccessDTO(applied)}, nil
}

func newClientAPIAccessDTO(access ClientAPIAccess) clientApiAccessDto {
	output := clientApiAccessDto(access)
	if output.UserDelegatedPermissionIDs == nil {
		output.UserDelegatedPermissionIDs = []string{}
	}
	if output.ClientPermissionIDs == nil {
		output.ClientPermissionIDs = []string{}
	}
	return output
}

func mapAPI(item API) (*httpapi.BodyOutput[apiResponseDto], error) {
	var output apiResponseDto
	if err := dto.MapStruct(item, &output); err != nil {
		return nil, err
	}
	output.Resource = item.Audience
	return &httpapi.BodyOutput[apiResponseDto]{Body: output}, nil
}
