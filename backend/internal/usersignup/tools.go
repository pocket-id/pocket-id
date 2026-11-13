package usersignup

import (
	"context"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// RegisterTools registers the MCP tool that invites people to create their own account
func (m *Module) RegisterTools(t *mcp.Tools) {
	mcp.AddTool(t, mcp.Tool{
		Name:        "create_signup_link",
		Description: "Create a link that lets people create their own account, even when open sign-up is disabled. Users who sign up through it join the given groups.",
		Effect:      mcp.Write,
	}, authz.UsersWrite, m.handler.createSignupLinkTool)
}

type createSignupLinkInput struct {
	ExpiresInMinutes int      `json:"expiresInMinutes,omitempty" jsonschema:"Minutes until the link expires, 60 by default and at most 44640 (31 days)"`
	UsageLimit       int      `json:"usageLimit,omitempty" jsonschema:"How many accounts the link can create, 1 by default and at most 100"`
	UserGroupIDs     []string `json:"userGroupIds,omitempty" jsonschema:"IDs of the groups new users join"`
}

type signupLinkResult struct {
	URL        string                `json:"url"`
	ExpiresAt  datatype.DateTime     `json:"expiresAt"`
	UsageLimit int                   `json:"usageLimit"`
	UserGroups []dto.UserGroupRefDto `json:"userGroups"`
}

func (h *handler) createSignupLinkTool(ctx context.Context, _ mcp.Call, input createSignupLinkInput) (any, error) {
	createDto := signupTokenCreateDto{
		TTL:          utils.JSONDuration{Duration: time.Duration(input.ExpiresInMinutes) * time.Minute},
		UsageLimit:   max(input.UsageLimit, 1),
		UserGroupIDs: input.UserGroupIDs,
	}
	if createDto.TTL.Duration <= 0 {
		createDto.TTL.Duration = defaultSignupTokenDuration
	}
	if err := httpserver.Validate(&createDto); err != nil {
		return nil, err
	}

	signupToken, err := h.service.CreateSignupToken(ctx, createDto.TTL.Duration, createDto.UsageLimit, createDto.UserGroupIDs)
	if err != nil {
		return nil, err
	}

	result := signupLinkResult{URL: common.EnvConfig.AppURL + "/st/" + signupToken.Token}
	if err := dto.MapStruct(signupToken, &result); err != nil {
		return nil, err
	}
	return result, nil
}
