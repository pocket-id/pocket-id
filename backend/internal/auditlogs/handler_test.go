package auditlogs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestAuditLogRoutesPreservePermissionsAndResponses(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	module, err := New(Dependencies{DB: db, CleanupDisabled: true})
	require.NoError(t, err)

	// Different owners make accidental loss of the current-user filter visible
	for _, id := range []string{"alice", "bob"} {
		require.NoError(t, db.Create(&model.User{Base: model.Base{ID: id}, Username: id}).Error)
		require.NoError(t, db.Create(&AuditLog{
			Base: model.Base{ID: id + "-log"}, UserID: id, Event: EventSignIn,
			IpAddress: new("192.0.2.1"), UserAgent: "Firefox", Country: "Switzerland", City: "Zurich",
			Data: Data{"clientName": id + "-client", "actorUsername": "administrator"},
		}).Error)
	}

	// Keep authentication lightweight while exercising which middleware each route receives
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": c.Errors.String()})
		}
	})
	module.RegisterRoutes(router.Group("/api"), auditLogTestAuth(true), auditLogTestAuth(false))

	for _, path := range []string{"/audit-logs", "/audit-logs/all", "/audit-logs/filters/client-names", "/audit-logs/filters/users"} {
		for _, role := range []string{"", "user", "admin"} {
			t.Run(path+"/"+role, func(t *testing.T) {
				request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api"+path, nil)
				request.Header.Set("X-Test-Role", role)
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				status := http.StatusOK
				if role == "" {
					status = http.StatusUnauthorized
				} else if role != "admin" && path != "/audit-logs" {
					status = http.StatusForbidden
				}
				require.Equal(t, status, response.Code, response.Body.String())
				if status != http.StatusOK {
					return
				}

				assertAuditLogRouteResponse(t, path, response)
			})
		}
	}
}

func auditLogTestAuth(adminRequired bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetHeader("X-Test-Role")
		if role == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if adminRequired && role != "admin" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Set("userID", "alice")
	}
}

func assertAuditLogRouteResponse(t *testing.T, path string, response *httptest.ResponseRecorder) {
	t.Helper()
	switch path {
	case "/audit-logs", "/audit-logs/all":
		var body struct {
			Data       []map[string]any `json:"data"`
			Pagination map[string]any   `json:"pagination"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.NotEmpty(t, body.Pagination)
		count := 1
		if path == "/audit-logs/all" {
			count = 2
		}
		require.Len(t, body.Data, count)
		for _, entry := range body.Data {
			require.Equal(t, "SIGN_IN", entry["event"])
			require.Equal(t, "192.0.2.1", entry["ipAddress"])
			require.Equal(t, "administrator", entry["actorUsername"])
			require.Contains(t, entry, "device")
			require.Contains(t, entry, "createdAt")
			require.NotContains(t, entry, "userAgent")
			if path == "/audit-logs" {
				require.Equal(t, "alice", entry["userID"])
			} else {
				require.Equal(t, entry["userID"], entry["username"])
			}
		}
	case "/audit-logs/filters/client-names":
		var names []string
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &names))
		require.ElementsMatch(t, []string{"alice-client", "bob-client"}, names)
	case "/audit-logs/filters/users":
		var users map[string]string
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &users))
		require.Equal(t, map[string]string{"alice": "alice", "bob": "bob"}, users)
	}
}
