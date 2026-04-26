//go:build !exclude_frontend

package frontend

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestIsSPARequest(t *testing.T) {
	distFS := fstest.MapFS{
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('test')")},
	}

	t.Run("root path is spa request", func(t *testing.T) {
		assert.True(t, isSPARequest("", distFS))
	})

	t.Run("existing bundled asset is not spa request", func(t *testing.T) {
		assert.False(t, isSPARequest("assets/app.js", distFS))
	})

	t.Run("unknown path is spa request", func(t *testing.T) {
		assert.True(t, isSPARequest("authorize", distFS))
	})
}

func TestRateLimitOnlyForOAuth2AuthorizationPostRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	distFS := fstest.MapFS{
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('test')")},
	}

	t.Run("rate limits spa form_post request", func(t *testing.T) {
		rateLimited := false
		nextCalled := false
		middleware := rateLimitOnlyForOAuth2AuthorizationPostRequest(func(c *gin.Context) {
			rateLimited = true
			c.Abort()
		}, distFS)

		router := gin.New()
		router.NoRoute(
			middleware,
			func(c *gin.Context) {
				nextCalled = true
			},
		)

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/authorize?response_mode=form_post&client_id=test&redirect_uri=https://example.com/callback", nil)
		router.ServeHTTP(recorder, req)

		assert.True(t, rateLimited)
		assert.False(t, nextCalled)
	})

	t.Run("does not rate limit page request with no form_post params", func(t *testing.T) {
		rateLimited := false
		nextCalled := false
		middleware := rateLimitOnlyForOAuth2AuthorizationPostRequest(func(c *gin.Context) {
			rateLimited = true
			c.Abort()
		}, distFS)

		router := gin.New()
		router.NoRoute(
			middleware,
			func(c *gin.Context) {
				nextCalled = true
			},
		)

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/authorize", nil)
		router.ServeHTTP(recorder, req)

		assert.False(t, rateLimited)
		assert.True(t, nextCalled)
	})

	t.Run("does not rate limit static asset request with form_post params", func(t *testing.T) {
		rateLimited := false
		nextCalled := false
		middleware := rateLimitOnlyForOAuth2AuthorizationPostRequest(func(c *gin.Context) {
			rateLimited = true
			c.Abort()
		}, distFS)

		router := gin.New()
		router.NoRoute(
			middleware,
			func(c *gin.Context) {
				nextCalled = true
			},
		)

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/app.js?response_mode=form_post&client_id=test&redirect_uri=https://example.com/callback", nil)
		router.ServeHTTP(recorder, req)

		assert.False(t, rateLimited)
		assert.True(t, nextCalled)
	})
}
