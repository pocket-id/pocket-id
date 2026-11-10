package users

import (
	"context"
	"fmt"
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// RegisterTools registers the MCP tools for the signed-in user's account and for managing users
func (m *Module) RegisterTools(t *mcp.Tools) {
	mcp.AddTool(t, mcp.Tool{
		Name:        "get_my_account",
		Description: "Get the account of the signed-in user, including the groups they belong to.",
		Effect:      mcp.ReadOnly,
	}, authz.AccountRead, m.handler.getMyAccountTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "search_users",
		Description: "Search users by username, email, first or last name. Without a query, lists all users. Returns a summary of each user, get_user returns their groups and custom claims.",
		Effect:      mcp.ReadOnly,
	}, authz.UsersRead, m.handler.searchUsersTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "get_user",
		Description: "Get a user by ID, including their groups and custom claims.",
		Effect:      mcp.ReadOnly,
	}, authz.UsersRead, m.handler.getUserTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "create_user",
		Description: "Create a user. The user signs in with a passkey, so send them a login link with create_login_link afterwards to let them add one.",
		Effect:      mcp.Write,
	}, authz.UsersWrite, m.handler.createUserTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "update_user",
		Description: "Update a user. Only the given fields change. Set disabled to block a user from signing in without deleting them.",
		Effect:      mcp.Write,
	}, authz.UsersWrite, m.handler.updateUserTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "delete_user",
		Description: "Delete a user permanently, including their passkeys and sessions. Prefer update_user with disabled set to block a user reversibly.",
		Effect:      mcp.Destructive,
	}, authz.UsersWrite, m.handler.deleteUserTool)
}

type searchUsersInput struct {
	Query string `json:"query,omitempty" jsonschema:"Text to match against the username, email, first and last name"`
	mcp.ListInput
}

type userIDInput struct {
	ID string `json:"id" jsonschema:"ID of the user"`
}

type createUserInput struct {
	Username      string   `json:"username" jsonschema:"Username, which may contain letters, digits, underscores, dots, hyphens and @"`
	Email         *string  `json:"email,omitempty"`
	EmailVerified bool     `json:"emailVerified,omitempty" jsonschema:"Whether the email is known to belong to the user"`
	FirstName     string   `json:"firstName,omitempty"`
	LastName      string   `json:"lastName,omitempty"`
	DisplayName   string   `json:"displayName,omitempty" jsonschema:"Name shown to apps, defaults to the first and last name"`
	IsAdmin       bool     `json:"isAdmin,omitempty" jsonschema:"Whether the user can administrate Pocket ID"`
	UserGroupIDs  []string `json:"userGroupIds,omitempty" jsonschema:"IDs of the groups to add the user to"`
}

type updateUserInput struct {
	ID            string  `json:"id" jsonschema:"ID of the user"`
	Username      *string `json:"username,omitempty"`
	Email         *string `json:"email,omitempty"`
	EmailVerified *bool   `json:"emailVerified,omitempty"`
	FirstName     *string `json:"firstName,omitempty"`
	LastName      *string `json:"lastName,omitempty"`
	DisplayName   *string `json:"displayName,omitempty"`
	IsAdmin       *bool   `json:"isAdmin,omitempty"`
	Disabled      *bool   `json:"disabled,omitempty" jsonschema:"Whether the user is blocked from signing in"`
}

// userResult refers to the user's groups without their claims and member counts, which aren't loaded with the user
type userResult struct {
	ID            string                `json:"id"`
	Username      string                `json:"username"`
	Email         *string               `json:"email"`
	EmailVerified bool                  `json:"emailVerified"`
	FirstName     string                `json:"firstName"`
	LastName      *string               `json:"lastName"`
	DisplayName   string                `json:"displayName"`
	IsAdmin       bool                  `json:"isAdmin"`
	Disabled      bool                  `json:"disabled"`
	Locale        *string               `json:"locale"`
	LdapID        *string               `json:"ldapId"`
	CustomClaims  []dto.CustomClaimDto  `json:"customClaims"`
	UserGroups    []dto.UserGroupRefDto `json:"userGroups"`

	EmailVerificationDeadline *datatype.DateTime `json:"emailVerificationDeadline,omitempty"`
}

func (h *handler) getMyAccountTool(ctx context.Context, call mcp.Call, _ struct{}) (any, error) {
	user, err := h.service.GetUser(ctx, call.Principal.UserID)
	if err != nil {
		return nil, err
	}
	return toUserResult(user)
}

func (h *handler) searchUsersTool(ctx context.Context, _ mcp.Call, input searchUsersInput) (any, error) {
	users, pagination, err := h.service.ListUsers(ctx, input.Query, input.Options())
	if err != nil {
		return nil, err
	}

	var summaries []dto.UserSummaryDto
	if err := dto.MapStructList(users, &summaries); err != nil {
		return nil, err
	}
	return dto.Paginated[dto.UserSummaryDto]{Data: summaries, Pagination: pagination}, nil
}

func (h *handler) getUserTool(ctx context.Context, _ mcp.Call, input userIDInput) (any, error) {
	user, err := h.service.GetUser(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return toUserResult(user)
}

func (h *handler) createUserTool(ctx context.Context, call mcp.Call, input createUserInput) (any, error) {
	dbConfig, err := h.appConfig.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	displayName := input.DisplayName
	if displayName == "" {
		displayName = strings.TrimSpace(input.FirstName + " " + input.LastName)
	}

	createDto := dto.UserCreateDto{
		Username:      input.Username,
		Email:         input.Email,
		EmailVerified: input.EmailVerified,
		FirstName:     input.FirstName,
		LastName:      input.LastName,
		DisplayName:   displayName,
		IsAdmin:       input.IsAdmin,
		UserGroupIds:  input.UserGroupIDs,
	}
	if err := httpserver.Validate(&createDto); err != nil {
		return nil, err
	}

	user, err := h.service.CreateUser(ctx, dbConfig, createDto, auditlogs.OriginFromMCP(call))
	if err != nil {
		return nil, err
	}
	return toUserResult(user)
}

func (h *handler) updateUserTool(ctx context.Context, call mcp.Call, input updateUserInput) (any, error) {
	dbConfig, err := h.appConfig.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	// The service replaces every field, so start from the current user and apply only the given fields
	user, err := h.service.GetUser(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	var updateDto dto.UserCreateDto
	if err := dto.MapStruct(user, &updateDto); err != nil {
		return nil, err
	}
	mcp.SetIfPresent(&updateDto.Username, input.Username)
	mcp.SetIfPresent(&updateDto.EmailVerified, input.EmailVerified)
	mcp.SetIfPresent(&updateDto.FirstName, input.FirstName)
	mcp.SetIfPresent(&updateDto.LastName, input.LastName)
	mcp.SetIfPresent(&updateDto.DisplayName, input.DisplayName)
	mcp.SetIfPresent(&updateDto.IsAdmin, input.IsAdmin)
	mcp.SetIfPresent(&updateDto.Disabled, input.Disabled)
	if input.Email != nil {
		updateDto.Email = input.Email
	}
	if err := httpserver.Validate(&updateDto); err != nil {
		return nil, err
	}

	user, err = h.service.UpdateUser(ctx, dbConfig, input.ID, updateDto, false, false, auditlogs.OriginFromMCP(call))
	if err != nil {
		return nil, err
	}
	return toUserResult(user)
}

func (h *handler) deleteUserTool(ctx context.Context, call mcp.Call, input userIDInput) (any, error) {
	dbConfig, err := h.appConfig.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading app configuration: %w", err)
	}

	if err := h.service.DeleteUser(ctx, dbConfig, input.ID, false, auditlogs.OriginFromMCP(call)); err != nil {
		return nil, err
	}
	return mcp.Deleted, nil
}

func toUserResult(user model.User) (userResult, error) {
	var result userResult
	err := dto.MapStruct(user, &result)
	return result, err
}
