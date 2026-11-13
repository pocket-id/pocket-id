package oidc

import (
	"context"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// RegisterTools registers the MCP tools for managing OIDC clients
func (m *Module) RegisterTools(t *mcp.Tools) {
	mcp.AddTool(t, mcp.Tool{
		Name:        "search_oidc_clients",
		Description: "Search OIDC clients, the apps that sign in with Pocket ID, by name. Without a query, lists all clients. Returns a summary of each client, get_oidc_client returns its full configuration.",
		Effect:      mcp.ReadOnly,
	}, authz.OidcClientsRead, m.clientHandler.searchClientsTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "get_oidc_client",
		Description: "Get an OIDC client by ID, including its callback URLs and the groups allowed to sign in to it.",
		Effect:      mcp.ReadOnly,
	}, authz.OidcClientsRead, m.clientHandler.getClientTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "create_oidc_client",
		Description: "Create an OIDC client for an app that should sign in with Pocket ID. The result contains what the app needs: the client ID (id), the issuer and, for a confidential client, the client secret, which can't be retrieved again, so pass them on to the user right away. Use search_docs to find the callback URLs an app needs.",
		Effect:      mcp.Write,
	}, authz.OidcClientsWrite, m.clientHandler.createClientTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "update_oidc_client",
		Description: "Update an OIDC client. Only the given fields change.",
		Effect:      mcp.Write,
	}, authz.OidcClientsWrite, m.clientHandler.updateClientTool)
	mcp.AddTool(t, mcp.Tool{
		Name:        "delete_oidc_client",
		Description: "Delete an OIDC client permanently. The app can no longer sign users in until it is set up again with a new client ID and secret.",
		Effect:      mcp.Destructive,
	}, authz.OidcClientsWrite, m.clientHandler.deleteClientTool)
}

type searchClientsInput struct {
	Query string `json:"query,omitempty" jsonschema:"Text to match against the client name"`
	mcp.ListInput
}

type clientIDInput struct {
	ID string `json:"id" jsonschema:"ID of the OIDC client"`
}

type createClientInput struct {
	Name                string   `json:"name" jsonschema:"Name of the app"`
	Description         string   `json:"description,omitempty"`
	CallbackURLs        []string `json:"callbackURLs,omitempty" jsonschema:"URLs the app may redirect to after sign-in, where * matches any characters"`
	LogoutCallbackURLs  []string `json:"logoutCallbackURLs,omitempty" jsonschema:"URLs the app may redirect to after sign-out"`
	IsPublic            bool     `json:"isPublic,omitempty" jsonschema:"Whether the app can't keep a secret, such as a single-page or mobile app. Public clients have no secret and must use PKCE"`
	PkceEnabled         bool     `json:"pkceEnabled,omitempty" jsonschema:"Whether the app must use PKCE"`
	LaunchURL           *string  `json:"launchURL,omitempty" jsonschema:"URL that opens the app from the user's app list"`
	AllowedUserGroupIDs []string `json:"allowedUserGroupIds,omitempty" jsonschema:"IDs of the only groups whose members may sign in. Left empty, every user may sign in"`
}

type updateClientInput struct {
	ID                  string    `json:"id" jsonschema:"ID of the OIDC client"`
	Name                *string   `json:"name,omitempty"`
	Description         *string   `json:"description,omitempty"`
	CallbackURLs        *[]string `json:"callbackURLs,omitempty" jsonschema:"URLs the app may redirect to after sign-in, replacing the current ones"`
	LogoutCallbackURLs  *[]string `json:"logoutCallbackURLs,omitempty" jsonschema:"URLs the app may redirect to after sign-out, replacing the current ones"`
	IsPublic            *bool     `json:"isPublic,omitempty"`
	PkceEnabled         *bool     `json:"pkceEnabled,omitempty"`
	LaunchURL           *string   `json:"launchURL,omitempty"`
	SkipConsent         *bool     `json:"skipConsent,omitempty" jsonschema:"Whether users sign in without being asked to approve the app"`
	AllowedUserGroupIDs *[]string `json:"allowedUserGroupIds,omitempty" jsonschema:"IDs of the only groups whose members may sign in, replacing the current ones. An empty list lets every user sign in"`
}

// clientSummary is a client in a list, without the configuration that get_oidc_client returns
type clientSummary struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	LaunchURL         *string `json:"launchURL"`
	IsPublic          bool    `json:"isPublic"`
	IsGroupRestricted bool    `json:"isGroupRestricted"`
}

// clientResult refers to the allowed groups without their claims and member counts, which aren't loaded with the client
type clientResult struct {
	dto.OidcClientDto
	AllowedUserGroups []dto.UserGroupRefDto `json:"allowedUserGroups"`
}

type createdClientResult struct {
	clientResult
	Issuer       string `json:"issuer"`
	ClientSecret string `json:"clientSecret,omitempty"`
}

func (h *clientHandler) searchClientsTool(ctx context.Context, _ mcp.Call, input searchClientsInput) (any, error) {
	clients, pagination, err := h.service.ListClients(ctx, input.Query, input.Options())
	if err != nil {
		return nil, err
	}

	var summaries []clientSummary
	if err := dto.MapStructList(clients, &summaries); err != nil {
		return nil, err
	}
	return dto.Paginated[clientSummary]{Data: summaries, Pagination: pagination}, nil
}

func (h *clientHandler) getClientTool(ctx context.Context, _ mcp.Call, input clientIDInput) (any, error) {
	client, err := h.service.GetClient(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return toClientResult(client)
}

func (h *clientHandler) createClientTool(ctx context.Context, call mcp.Call, input createClientInput) (any, error) {
	createDto := dto.OidcClientCreateDto{OidcClientUpdateDto: dto.OidcClientUpdateDto{
		Name:               input.Name,
		Description:        input.Description,
		CallbackURLs:       input.CallbackURLs,
		LogoutCallbackURLs: input.LogoutCallbackURLs,
		IsPublic:           input.IsPublic,
		PkceEnabled:        input.PkceEnabled,
		LaunchURL:          input.LaunchURL,
		IsGroupRestricted:  len(input.AllowedUserGroupIDs) > 0,
	}}
	if err := httpserver.Validate(&createDto); err != nil {
		return nil, err
	}

	// The secret is created regardless of the auto-create setting, because the assistant can't open the client page to create one
	client, createdSecret, err := h.service.CreateClient(ctx, createDto, auditlogs.OriginFromMCP(call), true)
	if err != nil {
		return nil, err
	}

	if createDto.IsGroupRestricted {
		client, err = h.service.UpdateAllowedUserGroups(ctx, client.ID, dto.OidcUpdateAllowedUserGroupsDto{UserGroupIDs: input.AllowedUserGroupIDs}, auditlogs.OriginFromMCP(call))
		if err != nil {
			return nil, err
		}
	}

	result := createdClientResult{Issuer: common.EnvConfig.AppURL, ClientSecret: createdSecret}
	if err := dto.MapStruct(client, &result.clientResult); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *clientHandler) updateClientTool(ctx context.Context, call mcp.Call, input updateClientInput) (any, error) {
	// Assign the groups before the restriction takes effect, otherwise turning it on would log their members out of the client
	if input.AllowedUserGroupIDs != nil && len(*input.AllowedUserGroupIDs) > 0 {
		_, err := h.service.UpdateAllowedUserGroups(ctx, input.ID, dto.OidcUpdateAllowedUserGroupsDto{UserGroupIDs: *input.AllowedUserGroupIDs}, auditlogs.OriginFromMCP(call))
		if err != nil {
			return nil, err
		}
	}

	// The service replaces every field, so start from the locked client and apply only the given fields
	client, err := h.service.UpdateClient(ctx, input.ID, func(current any) (dto.OidcClientUpdateDto, error) {
		var updateDto dto.OidcClientUpdateDto
		if err := dto.MapStruct(current, &updateDto); err != nil {
			return updateDto, err
		}
		mcp.SetIfPresent(&updateDto.Name, input.Name)
		mcp.SetIfPresent(&updateDto.Description, input.Description)
		mcp.SetIfPresent(&updateDto.CallbackURLs, input.CallbackURLs)
		mcp.SetIfPresent(&updateDto.LogoutCallbackURLs, input.LogoutCallbackURLs)
		mcp.SetIfPresent(&updateDto.IsPublic, input.IsPublic)
		mcp.SetIfPresent(&updateDto.PkceEnabled, input.PkceEnabled)
		mcp.SetIfPresent(&updateDto.SkipConsent, input.SkipConsent)
		if input.LaunchURL != nil {
			updateDto.LaunchURL = input.LaunchURL
		}
		if input.AllowedUserGroupIDs != nil {
			updateDto.IsGroupRestricted = len(*input.AllowedUserGroupIDs) > 0
		}
		return updateDto, httpserver.Validate(&updateDto)
	}, auditlogs.OriginFromMCP(call))
	if err != nil {
		return nil, err
	}
	return toClientResult(client)
}

func (h *clientHandler) deleteClientTool(ctx context.Context, call mcp.Call, input clientIDInput) (any, error) {
	if err := h.service.DeleteClient(ctx, input.ID, auditlogs.OriginFromMCP(call)); err != nil {
		return nil, err
	}
	return mcp.Deleted, nil
}

func toClientResult(client model.OidcClient) (clientResult, error) {
	var result clientResult
	err := dto.MapStruct(client, &result)
	return result, err
}
