package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type echoInput struct {
	Name string `json:"name"`
}

// connect serves the module behind a router that authenticates every request as the principal, and connects an MCP client to it
func connect(t *testing.T, tools *Tools, principal authz.Principal) *sdk.ClientSession {
	t.Helper()
	gin.SetMode(gin.TestMode)

	module := New(Dependencies{Tools: tools, HTTPClient: http.DefaultClient})
	router := gin.New()
	router.Use(func(c *gin.Context) { authz.SetPrincipal(c, &principal) })
	router.POST("/mcp", module.serve)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: server.URL + "/mcp", DisableStandaloneSSE: true}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestToolsAreListedByScope(t *testing.T) {
	tools := &Tools{}
	echo := func(_ context.Context, call Call, input echoInput) (any, error) {
		return map[string]string{"name": input.Name, "user": call.Principal.UserID}, nil
	}
	AddTool(tools, Tool{Name: "read_users", Effect: ReadOnly}, authz.UsersRead, echo)
	AddTool(tools, Tool{Name: "write_users", Effect: Write}, authz.UsersWrite, echo)

	session := connect(t, tools, authz.Principal{Kind: authz.KindOAuthUser, UserID: "alice", Scopes: authz.NewScopeSet(authz.UsersRead)})

	result, err := session.ListTools(t.Context(), nil)
	require.NoError(t, err)
	names := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		names[i] = tool.Name
	}
	require.ElementsMatch(t, []string{"read_users", "search_docs"}, names)

	called, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "read_users", Arguments: map[string]any{"name": "bob"}})
	require.NoError(t, err)
	require.False(t, called.IsError)
	require.JSONEq(t, `{"name":"bob","user":"alice"}`, called.Content[0].(*sdk.TextContent).Text)

	_, err = session.CallTool(t.Context(), &sdk.CallToolParams{Name: "write_users", Arguments: map[string]any{"name": "bob"}})
	require.Error(t, err, "a tool the caller can't see must not be callable either")
}

func TestToolErrorsReadLikeAPIErrors(t *testing.T) {
	tools := &Tools{}
	AddPublicTool(tools, Tool{Name: "not_found"}, func(context.Context, Call, struct{}) (any, error) {
		return nil, apperror.NotFound("User")
	})
	AddPublicTool(tools, Tool{Name: "broken"}, func(context.Context, Call, struct{}) (any, error) {
		return nil, errors.New("database password is hunter2")
	})

	session := connect(t, tools, authz.Principal{Kind: authz.KindAPIKey, UserID: "alice"})

	for name, message := range map[string]string{"not_found": "User not found", "broken": "Something went wrong"} {
		result, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: name})
		require.NoError(t, err)
		require.True(t, result.IsError)
		require.Equal(t, message, result.Content[0].(*sdk.TextContent).Text)
	}
}

func TestAddToolRejectsUnknownScope(t *testing.T) {
	require.Panics(t, func() {
		AddTool(&Tools{}, Tool{Name: "typo"}, authz.Scope("users:raed"), func(context.Context, Call, struct{}) (any, error) { return nil, nil })
	})
}

func TestProtectedResourceMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origAppURL := common.EnvConfig.AppURL
	t.Cleanup(func() { common.EnvConfig.AppURL = origAppURL })
	common.EnvConfig.AppURL = "https://id.example.com"

	router := gin.New()
	router.GET(authz.MCPResourceMetadataPath, protectedResourceMetadata)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, authz.MCPResourceMetadataPath, http.NoBody))
	require.Equal(t, http.StatusOK, recorder.Code)

	var doc struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
		ScopesSupported      []string `json:"scopes_supported"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &doc))

	// MCP clients such as the Go SDK refuse metadata whose resource isn't the endpoint's URL
	require.Equal(t, "https://id.example.com/mcp", doc.Resource)
	require.Equal(t, []string{"https://id.example.com"}, doc.AuthorizationServers)
	require.Contains(t, doc.ScopesSupported, string(authz.UsersRead))
	require.Contains(t, doc.ScopesSupported, "offline_access")
	require.NotContains(t, doc.ScopesSupported, string(authz.AccountPasskeysEnroll), "session-only scopes can never be granted to an OAuth token")
}
