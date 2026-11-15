package oidc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBindEndSessionRequestOnlyDecodesForms(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, tt := range []struct {
		contentType string
		body        string
		wantHint    string
	}{
		{contentType: "application/x-www-form-urlencoded", body: "id_token_hint=hint", wantHint: "hint"},
		{contentType: "application/x-yaml", body: "idTokenHint: hint"},
		{contentType: "application/x-msgpack", body: "\x81\xabidTokenHint\xa4hint"},
	} {
		t.Run(tt.contentType, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/oidc/end-session", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", tt.contentType)

			input, err := bindEndSessionRequest(c)
			require.NoError(t, err)
			require.Equal(t, tt.wantHint, input.IdTokenHint)
		})
	}
}
