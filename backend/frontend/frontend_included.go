//go:build !exclude_frontend

package frontend

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

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

	cacheMaxAge := time.Hour * 24
	fileServer := NewFileServerWithCaching(http.FS(distFS), int(cacheMaxAge.Seconds()))

	router.NoRoute(func(c *gin.Context) {
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
	})

	return nil
}

// FileServerWithCaching wraps http.FileServer to add caching headers
type FileServerWithCaching struct {
	root                    http.FileSystem
	lastModified            time.Time
	cacheMaxAge             int
	lastModifiedHeaderValue string
	cacheControlHeaderValue string
}

func NewFileServerWithCaching(root http.FileSystem, maxAge int) *FileServerWithCaching {
	return &FileServerWithCaching{
		root:                    root,
		lastModified:            time.Now(),
		cacheMaxAge:             maxAge,
		lastModifiedHeaderValue: time.Now().UTC().Format(http.TimeFormat),
		cacheControlHeaderValue: fmt.Sprintf("public, max-age=%d", maxAge),
	}
}

func (f *FileServerWithCaching) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if the client has a cached version
	if ifModifiedSince := r.Header.Get("If-Modified-Since"); ifModifiedSince != "" {
		ifModifiedSinceTime, err := time.Parse(http.TimeFormat, ifModifiedSince)
		if err == nil && f.lastModified.Before(ifModifiedSinceTime.Add(1*time.Second)) {
			// Client's cached version is up to date
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	w.Header().Set("Last-Modified", f.lastModifiedHeaderValue)
	w.Header().Set("Cache-Control", f.cacheControlHeaderValue)

	http.FileServer(f.root).ServeHTTP(w, r)
}
