//go:build !exclude_frontend

package frontend

import (
	"embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed all:dist/*
var frontendFS embed.FS

func RegisterFrontend(router *gin.Engine) error {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return fmt.Errorf("failed to create sub FS: %w", err)
	}

	// Create a file server for the embedded files
	fileServer := http.FileServer(http.FS(distFS))

	router.NoRoute(func(c *gin.Context) {
		// Try to serve the requested file
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if _, err := fs.Stat(distFS, path); os.IsNotExist(err) {
			// File doesn't exist, serve index.html instead
			c.Request.URL.Path = "/"
		}

		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return nil
}
