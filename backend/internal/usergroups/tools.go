package usergroups

import (
	"context"
	"fmt"
	"slices"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// RegisterTools registers the MCP tools for managing user groups and their members
func (m *Module) RegisterTools(t *mcp.Tools) {
	mcp.AddTool(t, mcp.Tool{
		Name:        "search_groups",
		Description: "Search user groups by name. Without a query, lists all groups.",
		Effect:      mcp.ReadOnly,
	}, authz.GroupsRead, m.handler.searchGroupsTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "get_group",
		Description: "Get a user group by ID, including its members and the OIDC clients it may sign in to.",
		Effect:      mcp.ReadOnly,
	}, authz.GroupsRead, m.handler.getGroupTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "create_group",
		Description: "Create a user group.",
		Effect:      mcp.Write,
	}, authz.GroupsWrite, m.handler.createGroupTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "update_group",
		Description: "Rename a user group. Only the given fields change.",
		Effect:      mcp.Write,
	}, authz.GroupsWrite, m.handler.updateGroupTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "delete_group",
		Description: "Delete a user group permanently. Its members lose access to OIDC clients that are restricted to the group.",
		Effect:      mcp.Destructive,
	}, authz.GroupsWrite, m.handler.deleteGroupTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "add_group_members",
		Description: "Add users to a user group.",
		Effect:      mcp.Write,
	}, authz.GroupsWrite, m.handler.addGroupMembersTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "remove_group_members",
		Description: "Remove users from a user group.",
		Effect:      mcp.Write,
	}, authz.GroupsWrite, m.handler.removeGroupMembersTool)
}

type searchGroupsInput struct {
	Query string `json:"query,omitempty" jsonschema:"Text to match against the group's name and display name"`
	mcp.ListInput
}

type groupIDInput struct {
	ID string `json:"id" jsonschema:"ID of the group"`
}

type createGroupInput struct {
	FriendlyName string `json:"friendlyName" jsonschema:"Name shown in Pocket ID"`
	Name         string `json:"name" jsonschema:"Name apps receive in the groups claim, such as developers"`
}

type updateGroupInput struct {
	ID           string  `json:"id" jsonschema:"ID of the group"`
	FriendlyName *string `json:"friendlyName,omitempty" jsonschema:"Name shown in Pocket ID"`
	Name         *string `json:"name,omitempty" jsonschema:"Name apps receive in the groups claim"`
}

type groupMembersInput struct {
	ID      string   `json:"id" jsonschema:"ID of the group"`
	UserIDs []string `json:"userIds" jsonschema:"IDs of the users"`
}

func (h *handler) searchGroupsTool(ctx context.Context, _ mcp.Call, input searchGroupsInput) (any, error) {
	return h.listGroups(ctx, input.Query, input.Options())
}

func (h *handler) getGroupTool(ctx context.Context, _ mcp.Call, input groupIDInput) (any, error) {
	group, err := h.service.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return toGroupResult(group)
}

func (h *handler) createGroupTool(ctx context.Context, call mcp.Call, input createGroupInput) (any, error) {
	createDto := dto.UserGroupCreateDto{FriendlyName: input.FriendlyName, Name: input.Name}
	if err := httpserver.Validate(&createDto); err != nil {
		return nil, err
	}

	group, err := h.service.Create(ctx, createDto, auditlogs.OriginFromMCP(call))
	if err != nil {
		return nil, err
	}
	return toGroupResult(group)
}

func (h *handler) updateGroupTool(ctx context.Context, _ mcp.Call, input updateGroupInput) (any, error) {
	dbConfig, err := h.appConfig.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	group, err := h.service.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	updateDto := dto.UserGroupCreateDto{FriendlyName: group.FriendlyName, Name: group.Name}
	mcp.SetIfPresent(&updateDto.FriendlyName, input.FriendlyName)
	mcp.SetIfPresent(&updateDto.Name, input.Name)
	if err := httpserver.Validate(&updateDto); err != nil {
		return nil, err
	}

	group, err = h.service.Update(ctx, dbConfig, input.ID, updateDto)
	if err != nil {
		return nil, err
	}
	return toGroupResult(group)
}

func (h *handler) deleteGroupTool(ctx context.Context, call mcp.Call, input groupIDInput) (any, error) {
	dbConfig, err := h.appConfig.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	if err := h.service.Delete(ctx, dbConfig, input.ID, auditlogs.OriginFromMCP(call)); err != nil {
		return nil, err
	}
	return mcp.Deleted, nil
}

func (h *handler) addGroupMembersTool(ctx context.Context, call mcp.Call, input groupMembersInput) (any, error) {
	group, err := h.service.ChangeUsers(ctx, input.ID, func(memberIDs []string) []string {
		for _, userID := range input.UserIDs {
			if !slices.Contains(memberIDs, userID) {
				memberIDs = append(memberIDs, userID)
			}
		}
		return memberIDs
	}, auditlogs.OriginFromMCP(call))
	if err != nil {
		return nil, err
	}
	return toGroupResult(group)
}

func (h *handler) removeGroupMembersTool(ctx context.Context, call mcp.Call, input groupMembersInput) (any, error) {
	group, err := h.service.ChangeUsers(ctx, input.ID, func(memberIDs []string) []string {
		return slices.DeleteFunc(memberIDs, func(userID string) bool {
			return slices.Contains(input.UserIDs, userID)
		})
	}, auditlogs.OriginFromMCP(call))
	if err != nil {
		return nil, err
	}
	return toGroupResult(group)
}

// groupResult lists the members without their claims and groups, which aren't loaded with the group
type groupResult struct {
	ID                 string                      `json:"id"`
	FriendlyName       string                      `json:"friendlyName"`
	Name               string                      `json:"name"`
	CustomClaims       []dto.CustomClaimDto        `json:"customClaims"`
	LdapID             *string                     `json:"ldapId"`
	CreatedAt          datatype.DateTime           `json:"createdAt"`
	Users              []dto.UserSummaryDto        `json:"users"`
	AllowedOidcClients []dto.OidcClientMetaDataDto `json:"allowedOidcClients"`
}

func toGroupResult(group model.UserGroup) (groupResult, error) {
	var result groupResult
	err := dto.MapStruct(group, &result)
	return result, err
}
