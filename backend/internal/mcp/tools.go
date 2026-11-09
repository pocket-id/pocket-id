package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// Call is the authenticated request a tool runs for
type Call struct {
	Principal authz.Principal
	IPAddress string
	UserAgent string
}

// Effect tells MCP clients what a tool changes, so they can ask the user before running it
type Effect int

const (
	ReadOnly Effect = iota
	Write
	Destructive
)

var effectAnnotations = map[Effect]*sdk.ToolAnnotations{
	ReadOnly:    {ReadOnlyHint: true},
	Write:       {DestructiveHint: new(false)},
	Destructive: {DestructiveHint: new(true)},
}

// Tool describes a tool to MCP clients
type Tool struct {
	Name        string
	Description string
	Effect      Effect
}

// Handler runs a tool and returns its result, which MCP clients receive as JSON
type Handler[In any] func(ctx context.Context, call Call, input In) (any, error)

// Tools collects the tools of every package, the way an authz.Router collects routes
type Tools struct {
	tools []registeredTool
}

type registeredTool struct {
	// scope is empty for tools that every caller may use
	scope authz.Scope
	add   func(server *sdk.Server, call Call)
}

// AddTool registers a tool that only callers holding the scope can see and call
func AddTool[In any](t *Tools, tool Tool, scope authz.Scope, handler Handler[In]) {
	// An unknown scope can never be granted, so it is a programming error just like on a route
	if !scope.Known() {
		panic(fmt.Sprintf("tool %s requires unknown scope %q", tool.Name, scope))
	}
	addTool(t, tool, scope, handler)
}

// AddPublicTool registers a tool that every authenticated caller can use, because it reveals nothing about the instance
func AddPublicTool[In any](t *Tools, tool Tool, handler Handler[In]) {
	addTool(t, tool, "", handler)
}

func addTool[In any](t *Tools, tool Tool, scope authz.Scope, handler Handler[In]) {
	definition := &sdk.Tool{
		Name:        tool.Name,
		Description: tool.Description,
		Annotations: effectAnnotations[tool.Effect],
	}

	t.tools = append(t.tools, registeredTool{
		scope: scope,
		add: func(server *sdk.Server, call Call) {
			sdk.AddTool(server, definition, func(ctx context.Context, _ *sdk.CallToolRequest, input In) (*sdk.CallToolResult, any, error) {
				output, err := handler(ctx, call, input)
				if err != nil {
					return nil, nil, toolError(ctx, tool.Name, err)
				}
				return nil, output, nil
			})
		},
	})
}

// addAllowed adds the tools the caller's scopes allow to the server, so clients never see tools they can't call
func (t *Tools) addAllowed(server *sdk.Server, call Call) {
	for _, tool := range t.tools {
		if tool.scope == "" || call.Principal.Scopes.Has(tool.scope) {
			tool.add(server, call)
		}
	}
}

// toolError turns an error into the message the model sees, hiding internal errors the same way the REST API does
func toolError(ctx context.Context, toolName string, err error) error {
	classified := httpserver.ClassifyError(err)
	if classified.Code == apperror.CodeInternal {
		slog.ErrorContext(ctx, "MCP tool failed", slog.String("tool", toolName), slog.Any("error", err))
	}
	return errors.New(classified.Message)
}

// Deleted is the result of every delete tool, since an MCP tool result can't be empty
var Deleted = struct {
	Deleted bool `json:"deleted"`
}{Deleted: true}

// SetIfPresent applies an optional tool argument to the field it updates
func SetIfPresent[T any](field *T, value *T) {
	if value != nil {
		*field = *value
	}
}

// ListInput is the paging input of every search tool
type ListInput struct {
	Page  int `json:"page,omitempty" jsonschema:"Page to return, starting at 1"`
	Limit int `json:"limit,omitempty" jsonschema:"Results per page, at most 100, 20 by default"`
}

// Options converts the paging input into the options of the list services
func (in ListInput) Options() utils.ListRequestOptions {
	var options utils.ListRequestOptions
	options.Pagination.Page = in.Page
	options.Pagination.Limit = in.Limit
	return options
}
