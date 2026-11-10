package auditlogs

import (
	"context"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
)

// RegisterTools registers the MCP tool that searches the audit log of every user
func (m *Module) RegisterTools(t *mcp.Tools) {
	mcp.AddTool(t, mcp.Tool{
		Name:        "search_audit_logs",
		Description: "Search the audit log of all users, newest first. It records sign-ins, app authorizations, new accounts and passkey changes, with the IP address, location and device of each.",
		Effect:      mcp.ReadOnly,
	}, authz.AuditLogsRead, m.handler.searchAuditLogsTool)
}

type searchAuditLogsInput struct {
	UserID     string `json:"userId,omitempty" jsonschema:"Only entries of this user"`
	Event      Event  `json:"event,omitempty" jsonschema:"Only entries of this event: SIGN_IN, TOKEN_SIGN_IN, REMOTE_SIGN_IN, ACCOUNT_CREATED, CLIENT_AUTHORIZATION, NEW_CLIENT_AUTHORIZATION, DEVICE_CODE_AUTHORIZATION, NEW_DEVICE_CODE_AUTHORIZATION, PASSKEY_ADDED or PASSKEY_REMOVED"`
	ClientName string `json:"clientName,omitempty" jsonschema:"Only entries of the OIDC client with this name"`
	From       string `json:"from,omitempty" jsonschema:"Only entries created at or after this RFC 3339 time"`
	To         string `json:"to,omitempty" jsonschema:"Only entries created before this RFC 3339 time"`
	mcp.ListInput
}

func (h *handler) searchAuditLogsTool(ctx context.Context, _ mcp.Call, input searchAuditLogsInput) (any, error) {
	dateRange, err := parseDateRange(input.From, input.To)
	if err != nil {
		return nil, err
	}

	options := input.Options()
	options.Sort.Column = "createdAt"
	options.Sort.Direction = "desc"
	options.Filters = make(map[string][]any)
	if input.UserID != "" {
		options.Filters["userID"] = []any{input.UserID}
	}
	if input.Event != "" {
		options.Filters["event"] = []any{input.Event}
	}
	if input.ClientName != "" {
		options.Filters["clientName"] = []any{input.ClientName}
	}

	return h.listAllAuditLogs(ctx, dateRange, options)
}
