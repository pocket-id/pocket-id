package httpserver

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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

func TestBindUpdateJSONAppliesMergePatchToCurrentResource(t *testing.T) {
	type mapping struct {
		Claim   string `json:"claim"`
		IDToken bool   `json:"idToken"`
	}
	type credentials struct {
		Issuer   string `json:"issuer"`
		Audience string `json:"audience"`
	}
	type input struct {
		Name        string      `json:"name" binding:"required"`
		Description string      `json:"description"`
		Email       *string     `json:"email"`
		Mappings    []mapping   `json:"mappings"`
		Credentials credentials `json:"credentials"`
	}
	email := "current@example.com"
	current := input{
		Name:        "current",
		Description: "current",
		Email:       &email,
		Mappings:    []mapping{{Claim: "a", IDToken: true}},
		Credentials: credentials{Issuer: "current", Audience: "current"},
	}

	for _, test := range []struct {
		name   string
		method string
		body   string
		want   input
	}{
		{
			name:   "missing members keep their value",
			method: http.MethodPatch,
			body:   `{"name":"changed"}`,
			want:   input{Name: "changed", Description: "current", Email: &email, Mappings: current.Mappings, Credentials: current.Credentials},
		},
		{
			name:   "null clears pointer and plain fields",
			method: http.MethodPatch,
			body:   `{"email":null,"description":null}`,
			want:   input{Name: "current", Mappings: current.Mappings, Credentials: current.Credentials},
		},
		{
			name:   "nested objects merge",
			method: http.MethodPatch,
			body:   `{"credentials":{"issuer":"changed"}}`,
			want:   input{Name: "current", Description: "current", Email: &email, Mappings: current.Mappings, Credentials: credentials{Issuer: "changed", Audience: "current"}},
		},
		{
			name:   "arrays replace their elements without keeping old fields",
			method: http.MethodPatch,
			body:   `{"mappings":[{"claim":"b"}]}`,
			want:   input{Name: "current", Description: "current", Email: &email, Mappings: []mapping{{Claim: "b"}}, Credentials: current.Credentials},
		},
		{
			name:   "members match regardless of case",
			method: http.MethodPatch,
			body:   `{"Name":"changed","Email":null,"Credentials":{"Issuer":"changed"}}`,
			want:   input{Name: "changed", Description: "current", Mappings: current.Mappings, Credentials: credentials{Issuer: "changed", Audience: "current"}},
		},
		{
			name:   "PUT ignores the current resource",
			method: http.MethodPut,
			body:   `{"name":"changed"}`,
			want:   input{Name: "changed"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequestWithContext(t.Context(), test.method, "/", strings.NewReader(test.body))
			c.Request.Header.Set("Content-Type", "application/json")

			var got input
			err := BindUpdateJSON(c, &got, func() (any, error) { return current, nil })

			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestBindUpdateJSONAcceptsOnlyMergePatchTypes(t *testing.T) {
	for _, test := range []struct {
		contentType string
		accepted    bool
	}{
		{contentType: "application/merge-patch+json", accepted: true},
		{contentType: "application/json; charset=utf-8", accepted: true},
		{contentType: "application/json-patch+json", accepted: false},
		{contentType: "text/plain", accepted: false},
	} {
		t.Run(test.contentType, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/", strings.NewReader(`{"name":"changed"}`))
			c.Request.Header.Set("Content-Type", test.contentType)

			var input struct {
				Name string `json:"name"`
			}
			err := BindUpdateJSON(c, &input, func() (any, error) { return input, nil })

			require.Equal(t, "application/merge-patch+json, application/json", c.Writer.Header().Get("Accept-Patch"))
			if test.accepted {
				require.NoError(t, err)
			} else {
				require.True(t, apperror.IsCode(err, apperror.CodePatchTypeNotSupported))
			}
		})
	}
}

func TestBindUpdateJSONValidatesMergedResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/", strings.NewReader(`{"name":null}`))
	c.Request.Header.Set("Content-Type", "application/json")

	var input struct {
		Name string `json:"name" binding:"required"`
	}
	err := BindUpdateJSON(c, &input, func() (any, error) { return struct{ Name string }{Name: "current"}, nil })

	_, ok := errors.AsType[validator.ValidationErrors](err)
	require.True(t, ok, "expected a validation error, got %v", err)
}

func TestBindOptionalJSONDoesNotRelyOnContentLength(t *testing.T) {
	for _, test := range []struct {
		name          string
		contentLength int64
	}{
		{name: "zero", contentLength: 0},
		{name: "unknown", contentLength: -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"secret":"custom-secret"}`))
			c.Request.ContentLength = test.contentLength
			c.Request.Header.Set("Content-Type", "application/json")

			var input struct {
				Secret string `json:"secret"`
			}
			err := BindOptionalJSON(c, &input)

			require.NoError(t, err)
			require.Equal(t, "custom-secret", input.Secret)
		})
	}
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
