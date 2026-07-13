package humautils

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type testInput struct {
	Body struct {
		Name string `json:"name" required:"true" minLength:"3"`
	}
}

type testOutput struct {
	Body map[string]string
}

type testCookieOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

type testStreamOutput struct {
	ContentType string `header:"Content-Type"`
	Body        func(huma.Context)
}

type optionalBodyInput struct {
	Body *json.RawMessage `required:"false"`
}

type testAppError struct{}

func (testAppError) Error() string       { return "test error" }
func (testAppError) Description() string { return "test description" }
func (testAppError) HttpStatusCode() int { return http.StatusConflict }

type trackingReader struct {
	io.Reader
	closed bool
}

func (r *trackingReader) Close() error {
	r.closed = true
	return nil
}

func newTestAPI(t *testing.T) (*gin.Engine, huma.API) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := New(router, router.Group("/"))
	return router, api
}

func TestRequestAndErrorCompatibility(t *testing.T) {
	router, api := newTestAPI(t)
	Register(api, huma.Operation{OperationID: "test-request", Method: http.MethodPost, Path: "/api/test"}, func(_ context.Context, input *testInput) (*testOutput, error) {
		return &testOutput{Body: map[string]string{"name": input.Body.Name}}, nil
	})
	Register(api, huma.Operation{OperationID: "test-app-error", Method: http.MethodGet, Path: "/api/test-error"}, func(context.Context, *struct{}) (*struct{}, error) {
		return nil, testAppError{}
	})
	Register(api, huma.Operation{OperationID: "test-unknown-error", Method: http.MethodGet, Path: "/api/test-unknown-error"}, func(context.Context, *struct{}) (*struct{}, error) {
		return nil, errors.New("private failure")
	})
	Register(api, huma.Operation{OperationID: "test-optional-body", Method: http.MethodPost, Path: "/api/test-optional-body", DefaultStatus: http.StatusNoContent}, func(_ context.Context, input *optionalBodyInput) (*struct{}, error) {
		require.Nil(t, input.Body)
		return &struct{}{}, nil
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/test", strings.NewReader(`{"name":"Pocket ID","unknown":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "application/json", response.Header().Get("Content-Type"))
	require.JSONEq(t, `{"name":"Pocket ID"}`, response.Body.String())
	require.Empty(t, response.Header().Get("Link"))
	require.NotContains(t, response.Body.String(), "$schema")

	request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/test", nil)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
	require.JSONEq(t, `{"error":"Request body is required"}`, response.Body.String())

	request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/test", strings.NewReader(`{"name":"x"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.JSONEq(t, `{"error":"Expected length >= 3"}`, response.Body.String())

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/test-error", nil))
	require.Equal(t, http.StatusConflict, response.Code)
	require.JSONEq(t, `{"error":"Test error","error_description":"test description"}`, response.Body.String())
	require.Less(t, strings.Index(response.Body.String(), `"error"`), strings.Index(response.Body.String(), `"error_description"`))

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/test-unknown-error", nil))
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.JSONEq(t, `{"error":"Something went wrong"}`, response.Body.String())

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/test-optional-body", nil))
	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestCookiesStreamingAndOpenAPI(t *testing.T) {
	router, api := newTestAPI(t)
	Register(api, huma.Operation{OperationID: "test-cookies", Method: http.MethodPost, Path: "/api/test-cookies", DefaultStatus: http.StatusNoContent}, func(context.Context, *struct{}) (*testCookieOutput, error) {
		return &testCookieOutput{SetCookie: []http.Cookie{{Name: "one", Value: "1"}, {Name: "two", Value: "2"}}}, nil
	})

	reader := &trackingReader{Reader: strings.NewReader("streamed")}
	Register(api, huma.Operation{OperationID: "test-stream", Method: http.MethodGet, Path: "/api/test-stream"}, func(context.Context, *struct{}) (*testStreamOutput, error) {
		return &testStreamOutput{ContentType: "text/plain", Body: func(ctx huma.Context) {
			defer reader.Close()
			_, _ = io.Copy(ctx.BodyWriter(), reader)
		}}, nil
	})
	AddRawOperation(api, huma.Operation{
		OperationID: "test-raw",
		Method:      http.MethodPost,
		Path:        "/api/test-raw",
		Summary:     "Raw test",
		Tags:        []string{"Test"},
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/test-cookies", nil))
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, []string{"one=1", "two=2"}, response.Header().Values("Set-Cookie"))

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/test-stream", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "text/plain", response.Header().Get("Content-Type"))
	require.Equal(t, "streamed", response.Body.String())
	require.True(t, reader.closed)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/openapi.json", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"/api/test-raw"`)
	require.NotContains(t, response.Body.String(), `"422"`)
	require.NotContains(t, response.Body.String(), `"$schema"`)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/docs", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), "@scalar/api-reference@1.62.5")
	require.Contains(t, response.Header().Get("Content-Security-Policy"), "worker-src blob:")
	require.NotContains(t, response.Header().Get("Content-Security-Policy"), "script-src 'unsafe-inline'")
}

func TestRegisterAppliesDecoratorsInOrder(t *testing.T) {
	router, api := newTestAPI(t)
	var order []string

	first := func(operation *huma.Operation) {
		operation.Middlewares = append(operation.Middlewares, func(ctx huma.Context, next func(huma.Context)) {
			order = append(order, "first")
			next(ctx)
		})
	}
	second := func(ctx huma.Context, next func(huma.Context)) {
		order = append(order, "second")
		next(ctx)
	}

	Register(api, huma.Operation{
		OperationID:   "test-decorator-order",
		Method:        http.MethodGet,
		Path:          "/api/test-decorator-order",
		DefaultStatus: http.StatusNoContent,
	}, func(context.Context, *struct{}) (*struct{}, error) {
		order = append(order, "handler")
		return &struct{}{}, nil
	}, first, WithMiddleware(second))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/test-decorator-order", nil))
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, []string{"first", "second", "handler"}, order)
}

func TestRegisterPreservesBodyLimitConfiguration(t *testing.T) {
	router, api := newTestAPI(t)

	Register(api, huma.Operation{
		OperationID: "test-default-body-limits",
		Method:      http.MethodPost,
		Path:        "/api/test-default-body-limits",
	}, func(context.Context, *testInput) (*testOutput, error) {
		return &testOutput{}, nil
	})

	defaultOperation := api.OpenAPI().Paths["/api/test-default-body-limits"].Post
	require.Equal(t, int64(1<<20), defaultOperation.MaxBodyBytes)
	require.Equal(t, 5*time.Second, defaultOperation.BodyReadTimeout)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/test-default-body-limits", strings.NewReader(`{"name":"`+strings.Repeat("x", 1<<20)+`"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
	require.JSONEq(t, `{"error":"Request body is too large limit=1048576 bytes"}`, response.Body.String())

	Register(api, huma.Operation{
		OperationID:     "test-unlimited-body",
		Method:          http.MethodPost,
		Path:            "/api/test-unlimited-body",
		MaxBodyBytes:    -1,
		BodyReadTimeout: -1,
	}, func(context.Context, *testInput) (*testOutput, error) {
		return &testOutput{}, nil
	})

	unlimitedOperation := api.OpenAPI().Paths["/api/test-unlimited-body"].Post
	require.Equal(t, int64(-1), unlimitedOperation.MaxBodyBytes)
	require.Equal(t, time.Duration(-1), unlimitedOperation.BodyReadTimeout)
}
