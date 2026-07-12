package controller

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type userGroupListInput struct {
	utils.ListRequestOptions
	Search string `query:"search" required:"false"`
}

type userGroupIDInput struct {
	ID string `path:"id"`
}

type userGroupCreateInput struct {
	Body dto.UserGroupCreateDto
}

type userGroupUpdateInput struct {
	ID   string `path:"id"`
	Body dto.UserGroupCreateDto
}

type userGroupUsersInput struct {
	ID   string `path:"id"`
	Body dto.UserGroupUpdateUsersDto
}

type userGroupClientsInput struct {
	ID   string `path:"id"`
	Body dto.UserGroupUpdateAllowedOidcClientsDto
}

// NewUserGroupController registers user group management routes
func NewUserGroupController(api huma.API, authMiddleware *middleware.AuthMiddleware, userGroupService *service.UserGroupService) {
	controller := &UserGroupController{UserGroupService: userGroupService}
	auth := authMiddleware.Huma(api)

	httpapi.Register(api, huma.Operation{
		OperationID: "list-user-groups",
		Method:      http.MethodGet,
		Path:        "/api/user-groups",
		Summary:     "List user groups",
		Tags:        []string{"User Groups"},
	}, controller.list, auth)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-user-group",
		Method:      http.MethodGet,
		Path:        "/api/user-groups/{id}",
		Summary:     "Get user group by ID",
		Tags:        []string{"User Groups"},
	}, controller.get, auth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "create-user-group",
		Method:        http.MethodPost,
		Path:          "/api/user-groups",
		Summary:       "Create user group",
		Tags:          []string{"User Groups"},
		DefaultStatus: http.StatusCreated,
	}, controller.create, auth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-user-group",
		Method:      http.MethodPut,
		Path:        "/api/user-groups/{id}",
		Summary:     "Update user group",
		Tags:        []string{"User Groups"},
	}, controller.update, auth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "delete-user-group",
		Method:        http.MethodDelete,
		Path:          "/api/user-groups/{id}",
		Summary:       "Delete user group",
		Tags:          []string{"User Groups"},
		DefaultStatus: http.StatusNoContent,
	}, controller.delete, auth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-user-group-users",
		Method:      http.MethodPut,
		Path:        "/api/user-groups/{id}/users",
		Summary:     "Update users in a group",
		Tags:        []string{"User Groups"},
	}, controller.updateUsers, auth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-user-group-allowed-oidc-clients",
		Method:      http.MethodPut,
		Path:        "/api/user-groups/{id}/allowed-oidc-clients",
		Summary:     "Update allowed OIDC clients",
		Tags:        []string{"User Groups"},
	}, controller.updateAllowedOIDCClients, auth)
}

type UserGroupController struct {
	UserGroupService *service.UserGroupService
}

func (ugc *UserGroupController) list(ctx context.Context, input *userGroupListInput) (*httpapi.BodyOutput[dto.Paginated[dto.UserGroupMinimalDto]], error) {
	groups, pagination, err := ugc.UserGroupService.List(ctx, input.Search, input.ListRequestOptions)
	if err != nil {
		return nil, err
	}

	groupsDTO := make([]dto.UserGroupMinimalDto, len(groups))
	for i, group := range groups {
		if err := dto.MapStruct(group, &groupsDTO[i]); err != nil {
			return nil, err
		}
		groupsDTO[i].UserCount, err = ugc.UserGroupService.GetUserCountOfGroup(ctx, group.ID)
		if err != nil {
			return nil, err
		}
	}

	return &httpapi.BodyOutput[dto.Paginated[dto.UserGroupMinimalDto]]{Body: dto.Paginated[dto.UserGroupMinimalDto]{Data: groupsDTO, Pagination: pagination}}, nil
}

func (ugc *UserGroupController) get(ctx context.Context, input *userGroupIDInput) (*httpapi.BodyOutput[dto.UserGroupDto], error) {
	group, err := ugc.UserGroupService.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return mapUserGroup(group)
}

func (ugc *UserGroupController) create(ctx context.Context, input *userGroupCreateInput) (*httpapi.BodyOutput[dto.UserGroupDto], error) {
	group, err := ugc.UserGroupService.Create(ctx, input.Body)
	if err != nil {
		return nil, err
	}
	return mapUserGroup(group)
}

func (ugc *UserGroupController) update(ctx context.Context, input *userGroupUpdateInput) (*httpapi.BodyOutput[dto.UserGroupDto], error) {
	group, err := ugc.UserGroupService.Update(ctx, input.ID, input.Body)
	if err != nil {
		return nil, err
	}
	return mapUserGroup(group)
}

func (ugc *UserGroupController) delete(ctx context.Context, input *userGroupIDInput) (*httpapi.EmptyOutput, error) {
	if err := ugc.UserGroupService.Delete(ctx, input.ID); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (ugc *UserGroupController) updateUsers(ctx context.Context, input *userGroupUsersInput) (*httpapi.BodyOutput[dto.UserGroupDto], error) {
	group, err := ugc.UserGroupService.UpdateUsers(ctx, input.ID, input.Body.UserIDs)
	if err != nil {
		return nil, err
	}
	return mapUserGroup(group)
}

func (ugc *UserGroupController) updateAllowedOIDCClients(ctx context.Context, input *userGroupClientsInput) (*httpapi.BodyOutput[dto.UserGroupDto], error) {
	group, err := ugc.UserGroupService.UpdateAllowedOidcClient(ctx, input.ID, input.Body)
	if err != nil {
		return nil, err
	}
	return mapUserGroup(group)
}

func mapUserGroup(group any) (*httpapi.BodyOutput[dto.UserGroupDto], error) {
	var output dto.UserGroupDto
	if err := dto.MapStruct(group, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.UserGroupDto]{Body: output}, nil
}
