package service

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestOidcService_updateClientLogoType(t *testing.T) {
	// Create a test database
	db := testutils.NewDatabaseForTest(t)

	// Create database storage
	dbStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	// Init the OidcService
	s := &OidcService{
		db:          db,
		fileStorage: dbStorage,
	}

	// Create a test client
	client := model.OidcClient{
		Name:         "Test Client",
		CallbackURLs: model.UrlList{"https://example.com/callback"},
	}
	err = db.Create(&client).Error
	require.NoError(t, err)

	// Helper function to check if a file exists in storage
	fileExists := func(t *testing.T, path string) bool {
		t.Helper()
		_, _, err := dbStorage.Open(t.Context(), path)
		return err == nil
	}

	// Helper function to create a dummy file in storage
	createDummyFile := func(t *testing.T, path string) {
		t.Helper()
		err := dbStorage.Save(t.Context(), path, strings.NewReader("dummy content"))
		require.NoError(t, err)
	}

	t.Run("Updates light logo type for client without previous logo", func(t *testing.T) {
		// Update the logo type
		err := s.updateClientLogoType(t.Context(), client.ID, "png", true)
		require.NoError(t, err)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.ImageType)
		assert.Equal(t, "png", *updatedClient.ImageType)
	})

	t.Run("Updates dark logo type for client without previous dark logo", func(t *testing.T) {
		// Update the dark logo type
		err := s.updateClientLogoType(t.Context(), client.ID, "jpg", false)
		require.NoError(t, err)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.DarkImageType)
		assert.Equal(t, "jpg", *updatedClient.DarkImageType)
	})

	t.Run("Updates light logo type and deletes old file when type changes", func(t *testing.T) {
		// Create the old PNG file in storage
		oldPath := "oidc-client-images/" + client.ID + ".png"
		createDummyFile(t, oldPath)
		require.True(t, fileExists(t, oldPath), "Old file should exist before update")

		// Client currently has a PNG logo, update to WEBP
		err := s.updateClientLogoType(t.Context(), client.ID, "webp", true)
		require.NoError(t, err)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.ImageType)
		assert.Equal(t, "webp", *updatedClient.ImageType)

		// Old PNG file should be deleted
		assert.False(t, fileExists(t, oldPath), "Old PNG file should have been deleted")
	})

	t.Run("Updates dark logo type and deletes old file when type changes", func(t *testing.T) {
		// Create the old JPG dark file in storage
		oldPath := "oidc-client-images/" + client.ID + "-dark.jpg"
		createDummyFile(t, oldPath)
		require.True(t, fileExists(t, oldPath), "Old dark file should exist before update")

		// Client currently has a JPG dark logo, update to WEBP
		err := s.updateClientLogoType(t.Context(), client.ID, "webp", false)
		require.NoError(t, err)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.DarkImageType)
		assert.Equal(t, "webp", *updatedClient.DarkImageType)

		// Old JPG dark file should be deleted
		assert.False(t, fileExists(t, oldPath), "Old JPG dark file should have been deleted")
	})

	t.Run("Does not delete file when type remains the same", func(t *testing.T) {
		// Create the WEBP file in storage
		webpPath := "oidc-client-images/" + client.ID + ".webp"
		createDummyFile(t, webpPath)
		require.True(t, fileExists(t, webpPath), "WEBP file should exist before update")

		// Update to the same type (WEBP)
		err := s.updateClientLogoType(t.Context(), client.ID, "webp", true)
		require.NoError(t, err)

		// Verify the client still has WEBP
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.ImageType)
		assert.Equal(t, "webp", *updatedClient.ImageType)

		// WEBP file should still exist since type didn't change
		assert.True(t, fileExists(t, webpPath), "WEBP file should still exist")
	})

	t.Run("Returns error for non-existent client", func(t *testing.T) {
		err := s.updateClientLogoType(t.Context(), "non-existent-client-id", "png", true)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to look up client")
	})
}

func TestOidcService_GetRegisteredCallbackURL(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	s := &OidcService{db: db}

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-one"},
		Name: "Client One",
		CallbackURLs: model.UrlList{
			"https://example.com/callback",
			"https://*.example.org/callback",
		},
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:         model.Base{ID: "client-two"},
		Name:         "Client Two",
		CallbackURLs: model.UrlList{"https://client.example.net/oauth/callback"},
	}).Error)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "exact callback URL is returned",
			input:    "https://example.com/callback",
			expected: "https://example.com/callback",
		},
		{
			name:     "wildcard matching delegates to the shared callback URL matcher",
			input:    "https://tenant.example.org/callback",
			expected: "https://tenant.example.org/callback",
		},
		{
			name:  "unknown external URL is rejected",
			input: "https://evil.example/steal",
		},
		{
			name:  "callback URL with fragment is rejected before matcher ignores it",
			input: "https://example.com/callback#fragment",
		},
		{
			name:  "missing URL is rejected",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.GetRegisteredCallbackURL(t.Context(), tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}

	t.Run("relative URL is rejected before a global wildcard can match it", func(t *testing.T) {
		require.NoError(t, db.Create(&model.OidcClient{
			Base:         model.Base{ID: "wildcard-client"},
			Name:         "Wildcard Client",
			CallbackURLs: model.UrlList{"*"},
		}).Error)

		got, err := s.GetRegisteredCallbackURL(t.Context(), "/settings/account")
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestOidcService_downloadAndSaveLogoFromURL(t *testing.T) {
	const publicLogoHost = "https://8.8.8.8"

	// Create a test database
	db := testutils.NewDatabaseForTest(t)

	// Create database storage
	dbStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	// Create a test client
	client := model.OidcClient{
		Name:         "Test Client",
		CallbackURLs: model.UrlList{"https://example.com/callback"},
	}
	err = db.Create(&client).Error
	require.NoError(t, err)

	// Helper function to check if a file exists in storage
	fileExists := func(t *testing.T, path string) bool {
		t.Helper()
		_, _, err := dbStorage.Open(t.Context(), path)
		return err == nil
	}

	// Helper function to get file content from storage
	getFileContent := func(t *testing.T, path string) []byte {
		t.Helper()
		reader, _, err := dbStorage.Open(t.Context(), path)
		require.NoError(t, err)
		defer reader.Close()
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		return content
	}

	t.Run("Successfully downloads and saves PNG logo from URL", func(t *testing.T) {
		// Create mock PNG content
		pngContent := []byte("fake-png-content")

		// Create a mock HTTP response with headers
		//nolint:bodyclose
		pngResponse := testutils.NewMockResponse(http.StatusOK, string(pngContent))
		pngResponse.Header.Set("Content-Type", "image/png")

		// Create a mock HTTP client with responses
		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			publicLogoHost + "/logo.png": pngResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		// Init the OidcService with mock HTTP client
		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		// Download and save the logo
		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, publicLogoHost+"/logo.png", true)
		require.NoError(t, err)

		// Verify the file was saved
		logoPath := "oidc-client-images/" + client.ID + ".png"
		require.True(t, fileExists(t, logoPath), "Logo file should exist in storage")

		// Verify the content
		savedContent := getFileContent(t, logoPath)
		assert.Equal(t, pngContent, savedContent)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.ImageType)
		assert.Equal(t, "png", *updatedClient.ImageType)
	})

	t.Run("Successfully downloads and saves dark logo", func(t *testing.T) {
		// Create mock WEBP content
		webpContent := []byte("fake-webp-content")

		//nolint:bodyclose
		webpResponse := testutils.NewMockResponse(http.StatusOK, string(webpContent))
		webpResponse.Header.Set("Content-Type", "image/webp")

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			publicLogoHost + "/dark-logo.webp": webpResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		// Download and save the dark logo
		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, publicLogoHost+"/dark-logo.webp", false)
		require.NoError(t, err)

		// Verify the dark logo file was saved
		darkLogoPath := "oidc-client-images/" + client.ID + "-dark.webp"
		require.True(t, fileExists(t, darkLogoPath), "Dark logo file should exist in storage")

		// Verify the content
		savedContent := getFileContent(t, darkLogoPath)
		assert.Equal(t, webpContent, savedContent)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.DarkImageType)
		assert.Equal(t, "webp", *updatedClient.DarkImageType)
	})

	t.Run("Detects extension from URL path", func(t *testing.T) {
		svgContent := []byte("<svg></svg>")

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			publicLogoHost + "/icon.svg": testutils.NewMockResponse(http.StatusOK, string(svgContent)),
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, publicLogoHost+"/icon.svg", true)
		require.NoError(t, err)

		// Verify SVG file was saved
		logoPath := "oidc-client-images/" + client.ID + ".svg"
		require.True(t, fileExists(t, logoPath), "SVG logo should exist")
	})

	t.Run("Detects extension from Content-Type when path has no extension", func(t *testing.T) {
		jpgContent := []byte("fake-jpg-content")

		//nolint:bodyclose
		jpgResponse := testutils.NewMockResponse(http.StatusOK, string(jpgContent))
		jpgResponse.Header.Set("Content-Type", "image/jpeg")

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			publicLogoHost + "/logo": jpgResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, publicLogoHost+"/logo", true)
		require.NoError(t, err)

		// Verify JPG file was saved (jpeg extension is normalized to jpg)
		logoPath := "oidc-client-images/" + client.ID + ".jpg"
		require.True(t, fileExists(t, logoPath), "JPG logo should exist")
	})

	t.Run("Returns error for invalid URL", func(t *testing.T) {
		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  &http.Client{},
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, "://invalid-url", true)
		require.Error(t, err)
	})

	t.Run("Returns error for non-200 status code", func(t *testing.T) {
		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			publicLogoHost + "/not-found.png": testutils.NewMockResponse(http.StatusNotFound, "Not Found"),
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, publicLogoHost+"/not-found.png", true)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to fetch logo")
	})

	t.Run("Returns error for too large content", func(t *testing.T) {
		// Create content larger than 2MB (maxLogoSize)
		largeContent := strings.Repeat("x", 2<<20+100) // 2.1MB

		//nolint:bodyclose
		largeResponse := testutils.NewMockResponse(http.StatusOK, largeContent)
		largeResponse.Header.Set("Content-Type", "image/png")
		largeResponse.Header.Set("Content-Length", strconv.Itoa(len(largeContent)))

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			publicLogoHost + "/large.png": largeResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, publicLogoHost+"/large.png", true)
		require.Error(t, err)
		require.ErrorIs(t, err, errLogoTooLarge)
	})

	t.Run("Returns error for unsupported file type", func(t *testing.T) {
		//nolint:bodyclose
		textResponse := testutils.NewMockResponse(http.StatusOK, "text content")
		textResponse.Header.Set("Content-Type", "text/plain")

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			publicLogoHost + "/file.txt": textResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, publicLogoHost+"/file.txt", true)
		require.Error(t, err)
		var fileTypeErr *common.FileTypeNotSupportedError
		require.ErrorAs(t, err, &fileTypeErr)
	})

	t.Run("Returns error for non-existent client", func(t *testing.T) {
		//nolint:bodyclose
		pngResponse := testutils.NewMockResponse(http.StatusOK, "content")
		pngResponse.Header.Set("Content-Type", "image/png")

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			publicLogoHost + "/logo.png": pngResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), "non-existent-client-id", publicLogoHost+"/logo.png", true)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to look up client")
	})
}
