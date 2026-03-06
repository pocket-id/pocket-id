//go:build !exclude_frontend

package frontend

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

//go:embed all:dist/*
var frontendFS embed.FS

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

// validateRedirectURI validates that the redirect_uri is in the client's allowed callback URLs
func validateRedirectURI(ctx any, oidcService *service.OidcService, clientID, redirectURI string) (bool, error) {
	client, err := oidcService.GetClient(ctx.(context.Context), clientID)
	if err != nil {
		return false, err
	}

	// If the client has no callback URLs configured, reject the redirect URI
	if len(client.CallbackURLs) == 0 {
		return false, errors.New("client has no callback URLs configured")
	}

	// Validate the redirect URI against the client's callback URLs
	_, err = utils.GetCallbackURLFromList(client.CallbackURLs, redirectURI)
	if err != nil {
		return false, err
	}

	return true, nil
}

func RegisterFrontend(router *gin.Engine, rateLimitMiddleware gin.HandlerFunc, oidcService *service.OidcService) error {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return fmt.Errorf("failed to create sub FS: %w", err)
	}

	// Load a map of all files to see which ones are available pre-compressed
	preCompressed, err := listPreCompressedAssets(distFS)
	if err != nil {
		return fmt.Errorf("failed to index pre-compressed frontend assets: %w", err)
	}

	// Init the file server
	fileServer := NewFileServerWithCaching(http.FS(distFS), preCompressed)

	// Handler for Gin
	handler := func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")

		if strings.HasSuffix(path, "/") {
			c.Redirect(http.StatusMovedPermanently, strings.TrimRight(c.Request.URL.String(), "/"))
			return
		}

		if strings.HasPrefix(path, "api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}

		// If path is / or does not exist, serve index.html
		if path == "" {
			path = "index.html"
		} else if _, err := fs.Stat(distFS, path); os.IsNotExist(err) {
			path = "index.html"
		}

		if path == "index.html" {
			// Check if this is an OAuth2 authorization request with response_mode=form_post
			// In that case, we need to validate and allow form submissions to the redirect_uri
			responseMode := c.Query("response_mode")
			redirectURI := c.Query("redirect_uri")
			clientID := c.Query("client_id")
			if responseMode == "form_post" && redirectURI != "" && clientID != "" {
				// Validate the redirect_uri against the client's allowlist
				isValid, err := validateRedirectURI(c.Request.Context(), oidcService, clientID, redirectURI)
				if err == nil && isValid {
					// Set the allowed form-action in CSP to include the redirect URI
					middleware.SetAllowedFormAction(c, redirectURI)
				}
			}

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

	router.NoRoute(rateLimitMiddleware, handler)

	return nil
}

// FileServerWithCaching wraps http.FileServer to add caching headers
type FileServerWithCaching struct {
	root                    http.FileSystem
	lastModified            time.Time
	lastModifiedHeaderValue string
	preCompressed           preCompressedMap
}

func NewFileServerWithCaching(root http.FileSystem, preCompressed preCompressedMap) *FileServerWithCaching {
	return &FileServerWithCaching{
		root:                    root,
		lastModified:            time.Now(),
		lastModifiedHeaderValue: time.Now().UTC().Format(http.TimeFormat),
		preCompressed:           preCompressed,
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

	// Check if the asset is available pre-compressed
	_, ok := f.preCompressed[r.URL.Path]
	if ok {
		// Add a "Vary" with "Accept-Encoding" so CDNs are aware that content is pre-compressed
		w.Header().Add("Vary", "Accept-Encoding")

		// Select the encoding if any
		ext, ce := f.selectEncoding(r)
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

func (f *FileServerWithCaching) selectEncoding(r *http.Request) (ext string, contentEnc string) {
	available, ok := f.preCompressed[r.URL.Path]
	if !ok {
		return "", ""
	}

	// Check if the client accepts compressed files
	acceptEncoding := strings.TrimSpace(strings.ToLower(r.Header.Get("Accept-Encoding")))
	if acceptEncoding == "" {
		return "", ""
	}

	// Prefer brotli over gzip when both are accepted.
	if available.br && (acceptEncoding == "*" || acceptEncoding == "br" || strings.Contains(acceptEncoding, "br")) {
		return "br", "br"
	}
	if available.gz && (acceptEncoding == "gzip" || strings.Contains(acceptEncoding, "gzip")) {
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

type preCompressedMap map[string]struct {
	br bool
	gz bool
}

func listPreCompressedAssets(distFS fs.FS) (preCompressedMap, error) {
	preCompressed := make(preCompressedMap, 0)
	err := fs.WalkDir(distFS, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		switch {
		case strings.HasSuffix(path, ".br"):
			originalPath := "/" + strings.TrimSuffix(path, ".br")
			entry := preCompressed[originalPath]
			entry.br = true
			preCompressed[originalPath] = entry
		case strings.HasSuffix(path, ".gz"):
			originalPath := "/" + strings.TrimSuffix(path, ".gz")
			entry := preCompressed[originalPath]
			entry.gz = true
			preCompressed[originalPath] = entry
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return preCompressed, nil
}
