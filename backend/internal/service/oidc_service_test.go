package service

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestListAuthorizedClientsRejectsMissingUser(t *testing.T) {
	service := &OidcService{db: testutils.NewDatabaseForTest(t)}

	_, _, err := service.ListAuthorizedClients(t.Context(), "missing-user", utils.ListRequestOptions{})

	require.True(t, apperror.IsCode(err, apperror.CodeUserNotFound))
}

func TestOidcService_DeleteClientDeletesOAuth2Sessions(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)

	client := model.OidcClient{Base: model.Base{ID: "deleted-client"}, Name: "Deleted Client"}
	otherClient := model.OidcClient{Base: model.Base{ID: "other-client"}, Name: "Other Client"}
	require.NoError(t, db.Create(&client).Error)
	require.NoError(t, db.Create(&otherClient).Error)

	for i, kind := range []string{"authorize_code", "access_token", "refresh_token", "par", "device_code"} {
		session := oidc.OAuth2Session{
			Base:        model.Base{ID: "deleted-client-session-" + strconv.Itoa(i)},
			Kind:        kind,
			Key:         "deleted-client-key-" + strconv.Itoa(i),
			RequestID:   "deleted-client-request",
			ClientID:    client.ID,
			Active:      true,
			RequestData: `{"client_id":"deleted-client","session":{"subject":"test-user","id_token_claims":{"jti":"test-jti"}}}`,
		}
		require.NoError(t, db.Create(&session).Error)
	}
	require.NoError(t, db.Create(&oidc.OAuth2Session{
		Base:        model.Base{ID: "other-client-session"},
		Kind:        "refresh_token",
		Key:         "other-client-key",
		RequestID:   "other-client-request",
		ClientID:    otherClient.ID,
		Active:      true,
		RequestData: `{"client_id":"other-client","session":{"subject":"test-user"}}`,
	}).Error)

	service := &OidcService{db: db}
	require.NoError(t, service.DeleteClient(t.Context(), client.ID))

	var deletedClientSessionCount int64
	require.NoError(t, db.Model(&oidc.OAuth2Session{}).Where("client_id = ?", client.ID).Count(&deletedClientSessionCount).Error)
	assert.Zero(t, deletedClientSessionCount)

	var otherClientSessionCount int64
	require.NoError(t, db.Model(&oidc.OAuth2Session{}).Where("client_id = ?", otherClient.ID).Count(&otherClientSessionCount).Error)
	assert.Equal(t, int64(1), otherClientSessionCount)
}

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
		CallbackURLs: datatype.StringList{"https://example.com/callback"},
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
		require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	})
}

func TestOidcClientImagePath(t *testing.T) {
	const metadataClientID = "https://app.example.com/oauth/client"

	assert.Equal(t, "oidc-client-images/client-id.png", oidcClientImagePath("client-id", "", "png"))
	assert.Equal(
		t,
		"oidc-client-images/cimd-"+utils.CreateSha256Hash(metadataClientID)+"-dark.webp",
		oidcClientImagePath(metadataClientID, "-dark", "webp"),
	)
	assert.NotContains(t, oidcClientImagePath(metadataClientID, "", "png"), "app.example.com")
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
		CallbackURLs: datatype.StringList{"https://example.com/callback"},
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
		require.True(t, apperror.IsCode(err, apperror.CodeValidationFailed))
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
		require.True(t, apperror.IsCode(err, apperror.CodeLogoDownloadFailed))
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
		require.True(t, apperror.IsCode(err, apperror.CodeLogoTooLarge))
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
		require.True(t, apperror.IsCode(err, apperror.CodeLogoTypeNotSupported))
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
		require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	})
}

func TestOidcService_CreateClient_withDescription(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	s, err := NewOidcService(db, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	description := "A test client description"
	input := dto.OidcClientCreateDto{
		OidcClientUpdateDto: dto.OidcClientUpdateDto{
			Name:                        "Test Client",
			Description:                 description,
			CallbackURLs:                []string{"https://example.com/callback"},
			AccessTokenDurationSeconds:  model.DefaultAccessTokenDurationSeconds,
			RefreshTokenDurationSeconds: model.DefaultRefreshTokenDurationSeconds,
		},
	}

	client, err := s.CreateClient(t.Context(), input, "user-id")
	require.NoError(t, err)

	var fetched model.OidcClient
	err = db.First(&fetched, "id = ?", client.ID).Error
	require.NoError(t, err)
	require.NotEmpty(t, fetched.Description)
	assert.Equal(t, description, fetched.Description)
}

func TestOidcService_CreateClient_withoutDescription(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	s, err := NewOidcService(db, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	input := dto.OidcClientCreateDto{
		OidcClientUpdateDto: dto.OidcClientUpdateDto{
			Name:                        "Test Client",
			CallbackURLs:                []string{"https://example.com/callback"},
			AccessTokenDurationSeconds:  model.DefaultAccessTokenDurationSeconds,
			RefreshTokenDurationSeconds: model.DefaultRefreshTokenDurationSeconds,
		},
	}

	client, err := s.CreateClient(t.Context(), input, "user-id")
	require.NoError(t, err)

	var fetched model.OidcClient
	err = db.First(&fetched, "id = ?", client.ID).Error
	require.NoError(t, err)
	assert.Empty(t, fetched.Description)
}

func TestOidcService_CreateClientSecret_withCustomSecret(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	s, err := NewOidcService(db, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	client := model.OidcClient{Name: "Test Client"}
	err = db.Create(&client).Error
	require.NoError(t, err)

	customSecret := "custom-client-secret-with-a-minimum-length"
	input := dto.OidcClientSecretDto{Secret: customSecret}

	secret, err := s.CreateClientSecret(t.Context(), client.ID, input)
	require.NoError(t, err)
	assert.Equal(t, customSecret, secret)

	var fetched model.OidcClient
	err = db.First(&fetched, "id = ?", client.ID).Error
	require.NoError(t, err)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(fetched.Secret), []byte(customSecret)))
}

func TestOidcService_UpdateClient_description(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	s, err := NewOidcService(db, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	// Create a client without a description
	client := model.OidcClient{
		Name:         "Test Client",
		CallbackURLs: datatype.StringList{"https://example.com/callback"},
	}
	err = db.Create(&client).Error
	require.NoError(t, err)

	// Update with a description
	description := "Updated description"
	input := dto.OidcClientUpdateDto{
		Name:                        "Test Client",
		Description:                 description,
		CallbackURLs:                []string{"https://example.com/callback"},
		AccessTokenDurationSeconds:  model.DefaultAccessTokenDurationSeconds,
		RefreshTokenDurationSeconds: model.DefaultRefreshTokenDurationSeconds,
	}

	_, err = s.UpdateClient(t.Context(), client.ID, input)
	require.NoError(t, err)

	var fetched model.OidcClient
	err = db.First(&fetched, "id = ?", client.ID).Error
	require.NoError(t, err)
	require.NotEmpty(t, fetched.Description)
	assert.Equal(t, description, fetched.Description)

	// Update to clear the description
	input.Description = ""
	_, err = s.UpdateClient(t.Context(), client.ID, input)
	require.NoError(t, err)

	err = db.First(&fetched, "id = ?", client.ID).Error
	require.NoError(t, err)
	assert.Empty(t, fetched.Description)
}

func TestOidcService_UpdateClient_CIMDPreservesMetadataFields(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	s, err := NewOidcService(db, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	client := model.OidcClient{
		Name:               "Metadata Client",
		CallbackURLs:       datatype.StringList{"https://metadata.example.com/callback"},
		LogoutCallbackURLs: datatype.StringList{"https://metadata.example.com/logout"},
		IsPublic:           true,
		PkceEnabled:        true,
		Credentials: model.OidcClientCredentials{
			FederatedIdentities: []model.OidcClientFederatedIdentity{{
				Issuer:  "https://metadata.example.com/client.json",
				Subject: "https://metadata.example.com/client.json",
				JWKS:    "https://metadata.example.com/jwks.json",
			}},
		},
		ClientType: model.OidcClientTypeCIMD,
	}
	require.NoError(t, db.Create(&client).Error)

	launchURL := "https://app.example.com"
	accessDuration := int64(2 * 60 * 60)
	refreshDuration := int64(7 * 24 * 60 * 60)
	input := dto.OidcClientUpdateDto{
		Name:                                "Overridden Client",
		Description:                         "Locally managed description",
		CallbackURLs:                        []string{"https://override.example.com/callback"},
		LogoutCallbackURLs:                  []string{"https://override.example.com/logout"},
		IsPublic:                            false,
		PkceEnabled:                         false,
		RequiresReauthentication:            true,
		RequiresPushedAuthorizationRequests: true,
		SkipConsent:                         true,
		LaunchURL:                           &launchURL,
		IsGroupRestricted:                   true,
		AccessTokenDurationSeconds:          accessDuration,
		RefreshTokenDurationSeconds:         refreshDuration,
		Credentials: dto.OidcClientCredentialsDto{
			FederatedIdentities: []dto.OidcClientFederatedIdentityDto{{
				Issuer: "https://override.example.com",
				JWKS:   "https://override.example.com/jwks.json",
			}},
		},
	}

	_, err = s.UpdateClient(t.Context(), client.ID, input)
	require.NoError(t, err)

	var fetched model.OidcClient
	require.NoError(t, db.First(&fetched, "id = ?", client.ID).Error)
	assert.Equal(t, client.Name, fetched.Name)
	assert.Equal(t, client.CallbackURLs, fetched.CallbackURLs)
	assert.Equal(t, client.LogoutCallbackURLs, fetched.LogoutCallbackURLs)
	assert.Equal(t, client.IsPublic, fetched.IsPublic)
	assert.Equal(t, client.PkceEnabled, fetched.PkceEnabled)
	assert.Equal(t, client.Credentials, fetched.Credentials)
	assert.Equal(t, input.Description, fetched.Description)
	assert.Equal(t, input.RequiresReauthentication, fetched.RequiresReauthentication)
	assert.Equal(t, input.RequiresPushedAuthorizationRequests, fetched.RequiresPushedAuthorizationRequests)
	assert.Equal(t, input.SkipConsent, fetched.SkipConsent)
	assert.Equal(t, input.LaunchURL, fetched.LaunchURL)
	assert.Equal(t, input.IsGroupRestricted, fetched.IsGroupRestricted)
	assert.Equal(t, accessDuration, fetched.AccessTokenDurationSeconds)
	assert.Equal(t, refreshDuration, fetched.RefreshTokenDurationSeconds)
}

func TestOidcService_UpdateClient_CIMDDoesNotOverwriteConcurrentMetadataRefresh(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	s, err := NewOidcService(db, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	client := model.OidcClient{
		Name:         "Original metadata name",
		CallbackURLs: datatype.StringList{"https://metadata.example.com/callback"},
		ClientType:   model.OidcClientTypeCIMD,
	}
	require.NoError(t, db.Create(&client).Error)

	// Simulate metadata refresh changing a document-owned column after the admin request read its snapshot
	require.NoError(t, db.Exec(`
		CREATE TRIGGER refresh_metadata_before_admin_update
		BEFORE UPDATE OF description ON oidc_clients
		BEGIN
			UPDATE oidc_clients SET name = 'Refreshed metadata name' WHERE id = OLD.id;
		END;
	`).Error)

	input := dto.OidcClientUpdateDto{
		Description:                 "Locally managed description",
		AccessTokenDurationSeconds:  model.DefaultAccessTokenDurationSeconds,
		RefreshTokenDurationSeconds: model.DefaultRefreshTokenDurationSeconds,
	}
	_, err = s.UpdateClient(t.Context(), client.ID, input)
	require.NoError(t, err)

	var fetched model.OidcClient
	require.NoError(t, db.First(&fetched, "id = ?", client.ID).Error)
	assert.Equal(t, "Refreshed metadata name", fetched.Name)
	assert.Equal(t, input.Description, fetched.Description)
}

func TestOidcService_ListAccessibleOidcClients_requiresExplicitGroupPermission(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	s, err := NewOidcService(db, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	allowedGroup := model.UserGroup{Name: "allowed", FriendlyName: "Allowed"}
	otherGroup := model.UserGroup{Name: "other", FriendlyName: "Other"}
	require.NoError(t, db.Create(&allowedGroup).Error)
	require.NoError(t, db.Create(&otherGroup).Error)

	userWithGroup := model.User{Username: "with-group", UserGroups: []model.UserGroup{allowedGroup}}
	userWithoutGroup := model.User{Username: "without-group"}
	require.NoError(t, db.Create(&userWithGroup).Error)
	require.NoError(t, db.Create(&userWithoutGroup).Error)

	clients := []model.OidcClient{
		{Name: "Unrestricted", CallbackURLs: datatype.StringList{"https://unrestricted.example.com/callback"}},
		{Name: "Restricted without groups", CallbackURLs: datatype.StringList{"https://empty.example.com/callback"}, IsGroupRestricted: true},
		{Name: "Restricted to user group", CallbackURLs: datatype.StringList{"https://allowed.example.com/callback"}, IsGroupRestricted: true, AllowedUserGroups: []model.UserGroup{allowedGroup}},
		{Name: "Restricted to other group", CallbackURLs: datatype.StringList{"https://other.example.com/callback"}, IsGroupRestricted: true, AllowedUserGroups: []model.UserGroup{otherGroup}},
	}
	for i := range clients {
		require.NoError(t, db.Create(&clients[i]).Error)
	}

	groupClients, _, err := s.ListAccessibleOidcClients(t.Context(), userWithGroup.ID, utils.ListRequestOptions{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Unrestricted", "Restricted to user group"}, accessibleClientNames(groupClients))

	noGroupClients, _, err := s.ListAccessibleOidcClients(t.Context(), userWithoutGroup.ID, utils.ListRequestOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"Unrestricted"}, accessibleClientNames(noGroupClients))
}

func accessibleClientNames(clients []dto.AccessibleOidcClientDto) []string {
	names := make([]string, len(clients))
	for i := range clients {
		names[i] = clients[i].Name
	}
	return names
}
