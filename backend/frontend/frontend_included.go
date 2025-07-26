//go:build !exclude_frontend

package frontend

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

//go:embed all:dist/*
var frontendFS embed.FS

const appManifestFile = "app.webmanifest"

func RegisterFrontend(router *gin.Engine, appConfigService *service.AppConfigService) error {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return fmt.Errorf("failed to create sub FS: %w", err)
	}

	cacheMaxAge := time.Hour * 24
	fileServer := NewFileServerWithCaching(http.FS(distFS), int(cacheMaxAge.Seconds()))

	// The app.webmanifest file needs special handling, as we need to set the app's name in the body
	// Read the file and parse it as JSON
	appManifestData, err := fs.ReadFile(distFS, appManifestFile)
	if err != nil {
		return fmt.Errorf("failed to read app manifest file '%s' in bundle: %w", appManifestFile, err)
	}
	var appManifest map[string]any
	err = json.Unmarshal(appManifestData, &appManifest)
	if err != nil {
		return fmt.Errorf("failed to parse app manifest file '%s' as JSON: %w", appManifestFile, err)
	}

	// Handle the route for the manifest
	router.GET("/"+appManifestFile, func(c *gin.Context) {
		// Replace the name in the manifest
		dbConfig := appConfigService.GetDbConfig()
		appManifest["name"] = dbConfig.AppName.Value

		c.Header("Content-Type", "application/manifest+json")
		c.JSON(http.StatusOK, appManifest)
	})

	// Register the fallback handler that serves the SvelteKit app for all routes that we can't match
	router.NoRoute(func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")

		// Block requests to certain files that don't exist, otherwise the SvelteKit frontend will render a page and that breaks certain behaviors (like loading the icon in the app's manifest in Safari)
		if c.Request.Method == http.MethodGet {
			switch path {
			case "apple-touch-icon.png", "apple-touch-icon-precomposed.png", "favicon.ico", "apple-touch-icon-120x120.png", "apple-touch-icon-120x120-precomposed.png":
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
		}

		// Try to serve the requested file
		if _, err := fs.Stat(distFS, path); os.IsNotExist(err) {
			// File doesn't exist, serve index.html instead
			c.Request.URL.Path = "/"
		}

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
