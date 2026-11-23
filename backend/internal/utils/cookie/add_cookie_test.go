package cookie

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddSignInSessionTokenCookie(t *testing.T) {
	tests := []struct {
		name   string
		target string
		maxAge int
	}{
		{name: "remembered by default", target: "/", maxAge: 3600},
		{name: "browser session when not remembered", target: "/?remember=false", maxAge: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequestWithContext(t.Context(), "POST", tt.target, nil)

			AddSignInSessionTokenCookie(c, time.Hour, RememberSignIn(c), "token")

			cookies := w.Result().Cookies()
			require.Len(t, cookies, 1)
			assert.Equal(t, "token", cookies[0].Value)
			assert.Equal(t, tt.maxAge, cookies[0].MaxAge)
		})
	}
}
