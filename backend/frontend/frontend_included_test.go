//go:build !exclude_frontend

package frontend

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

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

func TestFileServerWithCachingServesPrecompressedAssets(t *testing.T) {
	distFS := fstest.MapFS{
		"assets/app.js":    &fstest.MapFile{Data: []byte("original")},
		"assets/app.js.br": &fstest.MapFile{Data: []byte("brotli")},
		"assets/app.js.gz": &fstest.MapFile{Data: []byte("gzip")},
	}

	fileServer := NewFileServerWithCaching(http.FS(distFS))

	tests := []struct {
		name            string
		acceptEncoding  string
		expectedBody    string
		contentEncoding string
	}{
		{
			name:            "serves brotli when accepted",
			acceptEncoding:  "gzip, deflate, br",
			expectedBody:    "brotli",
			contentEncoding: "br",
		},
		{
			name:            "serves gzip when brotli is not accepted",
			acceptEncoding:  "gzip",
			expectedBody:    "gzip",
			contentEncoding: "gzip",
		},
		{
			name:            "honors encoding quality",
			acceptEncoding:  "br;q=0.5, gzip;q=1",
			expectedBody:    "gzip",
			contentEncoding: "gzip",
		},
		{
			name:            "does not serve a rejected encoding",
			acceptEncoding:  "br;q=0, gzip;q=0",
			expectedBody:    "original",
			contentEncoding: "",
		},
		{
			name:            "serves the original without accept encoding",
			acceptEncoding:  "",
			expectedBody:    "original",
			contentEncoding: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/assets/app.js", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			res := httptest.NewRecorder()

			fileServer.ServeHTTP(res, req)

			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, tt.expectedBody, res.Body.String())
			assert.Equal(t, tt.contentEncoding, res.Header().Get("Content-Encoding"))
			assert.Equal(t, "Accept-Encoding", res.Header().Get("Vary"))
			assert.Equal(t, "text/javascript; charset=utf-8", res.Header().Get("Content-Type"))
		})
	}
}

func TestFileServerWithCachingDoesNotCompressUnsupportedExtensions(t *testing.T) {
	distFS := fstest.MapFS{
		"assets/font.woff2":    &fstest.MapFile{Data: []byte("font")},
		"assets/font.woff2.br": &fstest.MapFile{Data: []byte("unused")},
	}
	fileServer := NewFileServerWithCaching(http.FS(distFS))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/assets/font.woff2", nil)
	req.Header.Set("Accept-Encoding", "br")
	res := httptest.NewRecorder()

	fileServer.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "font", res.Body.String())
	assert.Empty(t, res.Header().Get("Content-Encoding"))
	assert.Empty(t, res.Header().Get("Vary"))
}
