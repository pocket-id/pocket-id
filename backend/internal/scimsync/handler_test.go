package scimsync

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestServiceProviderResponseOmitsToken(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	err := respondWithServiceProvider(c, http.StatusOK, ServiceProvider{Endpoint: "https://scim.example.com", Token: "stored-token"})
	require.NoError(t, err)
	require.NotContains(t, recorder.Body.String(), "stored-token")
	require.Contains(t, recorder.Body.String(), `"hasToken":true`)
}
