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

	listOperation := userGroupOperation("list-user-groups", http.MethodGet, "/api/user-groups", "List user groups")
	auth(&listOperation)
	httpapi.Register(api, listOperation, controller.list)

	getOperation := userGroupOperation("get-user-group", http.MethodGet, "/api/user-groups/{id}", "Get user group by ID")
	auth(&getOperation)
	httpapi.Register(api, getOperation, controller.get)

	createOperation := userGroupOperation("create-user-group", http.MethodPost, "/api/user-groups", "Create user group")
	createOperation.DefaultStatus = http.StatusCreated
	auth(&createOperation)
	httpapi.Register(api, createOperation, controller.create)

	updateOperation := userGroupOperation("update-user-group", http.MethodPut, "/api/user-groups/{id}", "Update user group")
	auth(&updateOperation)
	httpapi.Register(api, updateOperation, controller.update)

	deleteOperation := userGroupOperation("delete-user-group", http.MethodDelete, "/api/user-groups/{id}", "Delete user group")
	deleteOperation.DefaultStatus = http.StatusNoContent
	auth(&deleteOperation)
	httpapi.Register(api, deleteOperation, controller.delete)

	usersOperation := userGroupOperation("update-user-group-users", http.MethodPut, "/api/user-groups/{id}/users", "Update users in a group")
	auth(&usersOperation)
	httpapi.Register(api, usersOperation, controller.updateUsers)

	clientsOperation := userGroupOperation("update-user-group-allowed-oidc-clients", http.MethodPut, "/api/user-groups/{id}/allowed-oidc-clients", "Update allowed OIDC clients")
	auth(&clientsOperation)
	httpapi.Register(api, clientsOperation, controller.updateAllowedOIDCClients)
}

func userGroupOperation(id, method, path, summary string) huma.Operation {
	return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"User Groups"}}
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
