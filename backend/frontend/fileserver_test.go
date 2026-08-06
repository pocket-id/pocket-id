package frontend

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compressible payload that is comfortably above minCompressibleSize
var testAssetData = []byte(strings.Repeat("console.log('pocket-id');\n", 200))

func testDistFS() fstest.MapFS {
	return fstest.MapFS{
		"_app/immutable/app.js":  &fstest.MapFile{Data: testAssetData},
		"_app/immutable/app.css": &fstest.MapFile{Data: []byte(strings.Repeat(".foo{color:red}\n", 200))},
		"small.js":               &fstest.MapFile{Data: []byte("console.log(1)")},
		"fonts/font.woff2":       &fstest.MapFile{Data: bytes.Repeat([]byte{0xde, 0xad, 0xbe, 0xef}, 500)},
		"robots.txt":             &fstest.MapFile{Data: []byte(strings.Repeat("User-agent: *\nDisallow:\n", 100))},
	}
}

func serveAsset(t *testing.T, f *FileServerWithCaching, method string, path string, acceptEncoding string) *http.Response {
	t.Helper()

	r := httptest.NewRequest(method, path, nil)
	if acceptEncoding != "" {
		r.Header.Set("Accept-Encoding", acceptEncoding)
	}

	w := httptest.NewRecorder()
	f.ServeHTTP(w, r)

	return w.Result()
}

func TestFileServerCompression(t *testing.T) {
	f := NewFileServerWithCaching(testDistFS())

	t.Run("serves brotli when accepted", func(t *testing.T) {
		res := serveAsset(t, f, http.MethodGet, "/_app/immutable/app.js", "gzip, deflate, br, zstd")
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "br", res.Header.Get("Content-Encoding"))
		assert.Equal(t, "text/javascript; charset=utf-8", res.Header.Get("Content-Type"))
		assert.Equal(t, "Accept-Encoding", res.Header.Get("Vary"))

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Less(t, len(body), len(testAssetData))

		decoded, err := io.ReadAll(brotli.NewReader(bytes.NewReader(body)))
		require.NoError(t, err)
		assert.Equal(t, testAssetData, decoded)
	})

	t.Run("serves gzip when brotli is not accepted", func(t *testing.T) {
		res := serveAsset(t, f, http.MethodGet, "/_app/immutable/app.js", "gzip, deflate")
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
		assert.Equal(t, "text/javascript; charset=utf-8", res.Header.Get("Content-Type"))
		assert.Equal(t, "Accept-Encoding", res.Header.Get("Vary"))

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		gzr, err := gzip.NewReader(bytes.NewReader(body))
		require.NoError(t, err)
		defer gzr.Close()

		decoded, err := io.ReadAll(gzr)
		require.NoError(t, err)
		assert.Equal(t, testAssetData, decoded)
	})

	t.Run("serves uncompressed when no encoding is accepted", func(t *testing.T) {
		res := serveAsset(t, f, http.MethodGet, "/_app/immutable/app.js", "")
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Empty(t, res.Header.Get("Content-Encoding"))
		assert.Equal(t, "Accept-Encoding", res.Header.Get("Vary"))

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Equal(t, testAssetData, body)
	})

	t.Run("does not compress assets that are already compressed", func(t *testing.T) {
		res := serveAsset(t, f, http.MethodGet, "/fonts/font.woff2", "br, gzip")
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Empty(t, res.Header.Get("Content-Encoding"))
		assert.Empty(t, res.Header.Get("Vary"))
	})

	t.Run("does not compress assets below the minimum size", func(t *testing.T) {
		res := serveAsset(t, f, http.MethodGet, "/small.js", "br, gzip")
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Empty(t, res.Header.Get("Content-Encoding"))
		assert.Empty(t, res.Header.Get("Vary"))
	})

	t.Run("compresses non-immutable assets too", func(t *testing.T) {
		res := serveAsset(t, f, http.MethodGet, "/robots.txt", "br")
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "br", res.Header.Get("Content-Encoding"))
		assert.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-Type"))
	})

	t.Run("serves headers only for HEAD requests", func(t *testing.T) {
		res := serveAsset(t, f, http.MethodHead, "/_app/immutable/app.js", "br")
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "br", res.Header.Get("Content-Encoding"))
		assert.NotEmpty(t, res.Header.Get("Content-Length"))
	})

	t.Run("returns 404 for unknown assets", func(t *testing.T) {
		res := serveAsset(t, f, http.MethodGet, "/does-not-exist.js", "br")
		defer res.Body.Close()

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})
}

func TestFileServerCacheHeaders(t *testing.T) {
	f := NewFileServerWithCaching(testDistFS())

	t.Run("immutable assets get a long-lived cache header", func(t *testing.T) {
		res := serveAsset(t, f, http.MethodGet, "/_app/immutable/app.js", "br")
		defer res.Body.Close()

		assert.Equal(t, "public, max-age=31536000, immutable", res.Header.Get("Cache-Control"))
		assert.Empty(t, res.Header.Get("Last-Modified"))
	})

	t.Run("other assets are revalidated with Last-Modified", func(t *testing.T) {
		res := serveAsset(t, f, http.MethodGet, "/robots.txt", "br")
		defer res.Body.Close()

		assert.Equal(t, "public, max-age=86400", res.Header.Get("Cache-Control"))
		assert.NotEmpty(t, res.Header.Get("Last-Modified"))
	})

	t.Run("If-Modified-Since returns 304", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
		r.Header.Set("Accept-Encoding", "br")
		r.Header.Set("If-Modified-Since", f.lastModified.UTC().Add(time.Minute).Format(http.TimeFormat))

		w := httptest.NewRecorder()
		f.ServeHTTP(w, r)

		assert.Equal(t, http.StatusNotModified, w.Code)
		assert.Empty(t, w.Body.Bytes())
	})
}

func TestAssetCompressorCaching(t *testing.T) {
	t.Run("compresses each asset only once", func(t *testing.T) {
		distFS := testDistFS()
		c := newAssetCompressor(distFS)

		first, err := c.Get("_app/immutable/app.js", encodingBrotli)
		require.NoError(t, err)
		require.NotEmpty(t, first)

		// Removing the file from the FS proves the second call is served from the cache
		delete(distFS, "_app/immutable/app.js")

		second, err := c.Get("_app/immutable/app.js", encodingBrotli)
		require.NoError(t, err)
		assert.Equal(t, first, second)
	})

	t.Run("caches each encoding separately", func(t *testing.T) {
		c := newAssetCompressor(testDistFS())

		br, err := c.Get("_app/immutable/app.js", encodingBrotli)
		require.NoError(t, err)

		gz, err := c.Get("_app/immutable/app.js", encodingGzip)
		require.NoError(t, err)

		assert.NotEqual(t, br, gz)
	})

	t.Run("returns nil when compression does not help", func(t *testing.T) {
		// Random data doesn't compress, so there's nothing to gain from serving it compressed
		incompressible := make([]byte, 4096)
		_, err := rand.Read(incompressible)
		require.NoError(t, err)

		distFS := fstest.MapFS{"noise.bin": &fstest.MapFile{Data: incompressible}}
		c := newAssetCompressor(distFS)

		data, err := c.Get("noise.bin", encodingGzip)
		require.NoError(t, err)
		assert.Nil(t, data)

		// The decision not to compress is cached like any other, so the asset isn't compressed again on every request
		// Removing the file from the FS makes a second compression attempt fail loudly instead of silently costing CPU time
		delete(distFS, "noise.bin")

		data, err = c.Get("noise.bin", encodingGzip)
		require.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("returns an error for missing assets", func(t *testing.T) {
		c := newAssetCompressor(testDistFS())

		_, err := c.Get("missing.js", encodingBrotli)
		require.Error(t, err)
	})

	t.Run("is safe for concurrent use", func(t *testing.T) {
		c := newAssetCompressor(testDistFS())

		const concurrency = 20
		results := make([][]byte, concurrency)

		wg := sync.WaitGroup{}
		wg.Add(concurrency)
		for i := range concurrency {
			go func() {
				defer wg.Done()
				data, err := c.Get("_app/immutable/app.js", encodingBrotli)
				assert.NoError(t, err)
				results[i] = data
			}()
		}
		wg.Wait()

		for i := range concurrency {
			assert.Equal(t, results[0], results[i])
		}
	})
}

func TestNegotiateEncoding(t *testing.T) {
	tests := []struct {
		name           string
		acceptEncoding string
		expect         string
	}{
		{name: "empty header", acceptEncoding: "", expect: ""},
		{name: "brotli only", acceptEncoding: "br", expect: encodingBrotli},
		{name: "gzip only", acceptEncoding: "gzip", expect: encodingGzip},
		{name: "brotli preferred over gzip", acceptEncoding: "gzip, deflate, br", expect: encodingBrotli},
		{name: "uppercase codings", acceptEncoding: "GZIP, BR", expect: encodingBrotli},
		{name: "unsupported codings only", acceptEncoding: "deflate, zstd", expect: ""},
		{name: "wildcard", acceptEncoding: "*", expect: encodingBrotli},
		{name: "wildcard with brotli disabled", acceptEncoding: "*, br;q=0", expect: encodingGzip},
		{name: "brotli rejected", acceptEncoding: "gzip, br;q=0", expect: encodingGzip},
		{name: "everything rejected", acceptEncoding: "gzip;q=0, br;q=0", expect: ""},
		{name: "gzip preferred by weight", acceptEncoding: "br;q=0.5, gzip;q=1.0", expect: encodingGzip},
		{name: "brotli preferred by weight", acceptEncoding: "br;q=1.0, gzip;q=0.5", expect: encodingBrotli},
		{name: "equal weights prefer brotli", acceptEncoding: "br;q=0.5, gzip;q=0.5", expect: encodingBrotli},
		{name: "extra whitespace", acceptEncoding: "  gzip ;  q=0.8 ,  br ; q=0.9 ", expect: encodingBrotli},
		{name: "identity only", acceptEncoding: "identity", expect: ""},
		{name: "malformed weight is ignored", acceptEncoding: "br;q=abc", expect: encodingBrotli},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, negotiateEncoding(tt.acceptEncoding))
		})
	}
}
