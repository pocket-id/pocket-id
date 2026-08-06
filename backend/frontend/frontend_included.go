//go:build !exclude_frontend

package frontend

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
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

func RegisterFrontend(router *gin.Engine) error {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return fmt.Errorf("failed to create sub FS: %w", err)
	}

	// Init the file server, which compresses assets on demand and caches the result in memory
	fileServer := NewFileServerWithCaching(distFS)

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
