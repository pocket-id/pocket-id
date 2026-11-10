package onetimeaccess

import (
	"context"
	"fmt"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// RegisterTools registers the MCP tool that lets an admin sign a user in with a one-time link
func (m *Module) RegisterTools(t *mcp.Tools) {
	mcp.AddTool(t, mcp.Tool{
		Name:        "create_login_link",
		Description: "Create a one-time login link for a user, for example so a new user can sign in and add a passkey, or a user who lost their passkey can add a new one. With sendEmail, the link is emailed to the user instead of returned.",
		Effect:      mcp.Write,
	}, authz.UsersWrite, m.handler.createLoginLinkTool)
}

type createLoginLinkInput struct {
	UserID           string `json:"userId" jsonschema:"ID of the user"`
	ExpiresInMinutes int    `json:"expiresInMinutes,omitempty" jsonschema:"Minutes until the link expires, 15 by default and at most 44640 (31 days)"`
	SendEmail        bool   `json:"sendEmail,omitempty" jsonschema:"Whether to email the link to the user instead of returning it"`
}

type loginLinkResult struct {
	URL       string `json:"url,omitempty"`
	EmailSent bool   `json:"emailSent,omitempty"`
}

func (h *handler) createLoginLinkTool(ctx context.Context, _ mcp.Call, input createLoginLinkInput) (any, error) {
	ttlInput := tokenCreateDto{TTL: utils.JSONDuration{Duration: time.Duration(input.ExpiresInMinutes) * time.Minute}}
	if err := httpserver.Validate(&ttlInput); err != nil {
		return nil, err
	}
	ttl := ttlInput.TTL.Duration
	if ttl <= 0 {
		ttl = defaultTokenDuration
	}

	if input.SendEmail {
		dbConfig, err := h.appConfig.GetConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("error loading app configuration: %w", err)
		}
		if err := h.service.RequestOneTimeAccessEmailAsAdmin(ctx, dbConfig, input.UserID, ttl); err != nil {
			return nil, err
		}
		return loginLinkResult{EmailSent: true}, nil
	}

	token, err := h.service.CreateToken(ctx, input.UserID, ttl)
	if err != nil {
		return nil, err
	}
	return loginLinkResult{URL: common.EnvConfig.AppURL + "/lc/" + token}, nil
}
