//go:build !exclude_frontend

package frontend

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
)

//go:embed all:dist/*
var frontendFS embed.FS

// SvelteKit generates both gzip and Brotli sidecars for these extensions when precompress is enabled
var precompressedExtensions = map[string]struct{}{
	".css":  {},
	".html": {},
	".js":   {},
	".json": {},
	".md":   {},
	".mdx":  {},
	".mjs":  {},
	".svg":  {},
	".txt":  {},
	".wasm": {},
	".xml":  {},
}

// This function, created by the init() method, writes to "w" the index.html page, populating the nonce
var writeIndexFn func(w io.Writer, nonce string) error

func init() {
	const scriptTag = "<script>"

	// Read the index.html from the bundle
	index, iErr := fs.ReadFile(frontendFS, "dist/index.html")
	if iErr != nil {
		panic(fmt.Errorf("failed to read index.html: %w", iErr))
	}

	writeIndexFn = func(w io.Writer, nonce string) (err error) {
		// If there's no nonce, write the index as-is
		if nonce == "" {
			_, err = w.Write(index)
			return err
		}

		// Add nonce to all <script> tags
		// We replace "<script" with `<script nonce="..."` everywhere it appears
		modified := bytes.ReplaceAll(
			index,
			[]byte(scriptTag),
			[]byte(`<script nonce="`+nonce+`">`),
		)

		_, err = w.Write(modified)
		return err
	}
}

func RegisterFrontend(router *gin.Engine) error {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return fmt.Errorf("failed to create sub FS: %w", err)
	}

	// Init the file server
	fileServer := NewFileServerWithCaching(http.FS(distFS))

	// Handler for Gin
	handler := func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")

		if strings.HasSuffix(path, "/") {
			c.Redirect(http.StatusMovedPermanently, strings.TrimRight(c.Request.URL.String(), "/"))
			return
		}

		if strings.HasPrefix(path, "api/") || strings.HasPrefix(path, ".well-known/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}

		if isSPARequest(path, distFS) {
			nonce := middleware.GetCSPNonce(c)

			// Do not cache the HTML shell, as it embeds a per-request nonce
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Header("Cache-Control", "no-store")
			c.Status(http.StatusOK)
			if err := writeIndexFn(c.Writer, nonce); err != nil {
				_ = c.Error(fmt.Errorf("failed to write index.html file: %w", err))
			}
			return
		}

		// Serve other static assets with caching
		c.Request.URL.Path = "/" + path
		fileServer.ServeHTTP(c.Writer, c.Request)
	}

	router.NoRoute(handler)

	return nil
}

func isSPARequest(path string, distFS fs.FS) bool {
	if path == "" {
		return true
	}

	if _, err := fs.Stat(distFS, path); err != nil {
		return true
	}

	return false
}

// FileServerWithCaching wraps http.FileServer to add caching headers
type FileServerWithCaching struct {
	root                    http.FileSystem
	lastModified            time.Time
	lastModifiedHeaderValue string
}

func NewFileServerWithCaching(root http.FileSystem) *FileServerWithCaching {
	return &FileServerWithCaching{
		root:                    root,
		lastModified:            time.Now(),
		lastModifiedHeaderValue: time.Now().UTC().Format(http.TimeFormat),
	}
}

func (f *FileServerWithCaching) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// First, set cache headers
	// Check if the request is for an immutable asset
	if isImmutableAsset(r) {
		// Set the cache control header as immutable with a long expiration
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		// Check if the client has a cached version
		ifModifiedSince := r.Header.Get("If-Modified-Since")
		if ifModifiedSince != "" {
			ifModifiedSinceTime, err := time.Parse(http.TimeFormat, ifModifiedSince)
			if err == nil && f.lastModified.Before(ifModifiedSinceTime.Add(1*time.Second)) {
				// Client's cached version is up to date
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		// Cache other assets for up to 24 hours, but set Last-Modified too
		w.Header().Set("Last-Modified", f.lastModifiedHeaderValue)
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}

	// SvelteKit creates both sidecars for every asset with a precompressed extension
	_, ok := precompressedExtensions[path.Ext(r.URL.Path)]
	if ok {
		// Add a "Vary" with "Accept-Encoding" so CDNs are aware that content is pre-compressed
		w.Header().Add("Vary", "Accept-Encoding")

		// Select the encoding if any
		ext, ce := selectEncoding(r)
		if ext != "" {
			// Set the content type explicitly before changing the path
			ct := mime.TypeByExtension(path.Ext(r.URL.Path))
			if ct != "" {
				w.Header().Set("Content-Type", ct)
			}

			// Make the serve return the encoded content
			w.Header().Set("Content-Encoding", ce)
			r.URL.Path += "." + ext
		}
	}

	http.FileServer(f.root).ServeHTTP(w, r)
}

func selectEncoding(r *http.Request) (ext string, contentEnc string) {
	// Check which available encoding the client prefers
	acceptEncoding := r.Header.Get("Accept-Encoding")
	if acceptEncoding == "" {
		return "", ""
	}

	// Header can have multiple encodings with optional quality values, e.g. "gzip;q=1.0, br;q=0.8, *;q=0.5"
	brWeight, gzipWeight, wildcardWeight := -1.0, -1.0, -1.0
	for part := range strings.SplitSeq(acceptEncoding, ",") {
		codingAndParams := strings.Split(part, ";")
		coding := strings.ToLower(strings.TrimSpace(codingAndParams[0]))
		weight := 1.0

		for _, param := range codingAndParams[1:] {
			key, value, found := strings.Cut(param, "=")
			if !found || !strings.EqualFold(strings.TrimSpace(key), "q") {
				continue
			}

			// Parse the quality value
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil || parsed < 0 || parsed > 1 {
				weight = 0
				break
			}
			weight = parsed
		}

		switch coding {
		case "br":
			brWeight = weight
		case "gzip":
			gzipWeight = weight
		case "*":
			wildcardWeight = weight
		}
	}

	// Apply the wildcard only to encodings the client did not name explicitly
	if brWeight < 0 {
		brWeight = wildcardWeight
	}
	if gzipWeight < 0 {
		gzipWeight = wildcardWeight
	}

	// Prefer brotli when both available encodings have the same quality
	if brWeight > 0 && brWeight >= gzipWeight {
		return "br", "br"
	}
	if gzipWeight > 0 {
		return "gz", "gzip"
	}

	return "", ""
}

func isImmutableAsset(r *http.Request) bool {
	switch {
	// Fonts
	case strings.HasPrefix(r.URL.Path, "/fonts/"):
		return true

	// Compiled SvelteKit assets
	case strings.HasPrefix(r.URL.Path, "/_app/immutable/"):
		return true

	default:
		return false
	}
}
