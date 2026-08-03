package httpserver

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

func TestBindJSONClassifiesMalformedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"name":`))
	c.Request.Header.Set("Content-Type", "application/json")

	var input struct {
		Name string `json:"name"`
	}
	err := BindJSON(c, &input)

	require.True(t, apperror.IsCode(err, apperror.CodeInvalidRequestBody))
	require.Contains(t, err.Error(), "unexpected EOF")
	var appErr *apperror.Error
	require.ErrorAs(t, err, &appErr)
	require.NotContains(t, appErr.ClientMessage(), "unexpected end")
}

func TestBindJSONNormalizesTaggedFieldsRecursively(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/",
		strings.NewReader(`{"name":"Cafe\u0301","email":"user@cafe\u0301.example","items":[{"label":"Re\u0301sume\u0301"}]}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	type embeddedInput struct {
		Name string `json:"name" unorm:"nfc"`
	}
	type itemInput struct {
		Label string `json:"label" unorm:"nfc"`
	}
	var input struct {
		embeddedInput
		Email *string     `json:"email" unorm:"nfc"`
		Items []itemInput `json:"items"`
	}
	err := BindJSON(c, &input)

	require.NoError(t, err)
	require.Equal(t, norm.NFC.String("Café"), input.Name)
	require.NotNil(t, input.Email)
	require.Equal(t, norm.NFC.String("user@café.example"), *input.Email)
	require.Equal(t, norm.NFC.String("Résumé"), input.Items[0].Label)
}

func TestBindJSONRejectsFormCompatibleContentTypes(t *testing.T) {
	for _, contentType := range []string{"text/plain", "application/x-www-form-urlencoded", "multipart/form-data; boundary=test"} {
		t.Run(contentType, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"name":"admin"}`))
			c.Request.Header.Set("Content-Type", contentType)

			var input struct {
				Name string `json:"name"`
			}
			err := BindJSON(c, &input)

			require.True(t, apperror.IsCode(err, apperror.CodeInvalidRequestBody))
			require.Empty(t, input.Name)
		})
	}
}

func TestBindJSONAcceptsStructuredJSONContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"name":"admin"}`))
	c.Request.Header.Set("Content-Type", "application/scim+json")

	var input struct {
		Name string `json:"name"`
	}
	err := BindJSON(c, &input)

	require.NoError(t, err)
	require.Equal(t, "admin", input.Name)
}

func TestFormFileClassifiesMissingField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.Close())
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := FormFile(c, "file")

	require.True(t, apperror.IsCode(err, apperror.CodeValidationFailed))
	var appErr *apperror.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, []apperror.FieldError{{
		Field:   "file",
		Code:    "required",
		Message: "is required",
	}}, appErr.Fields())
}
