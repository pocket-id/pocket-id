package httpserver

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type greeting struct {
	Name string `json:"name" binding:"required"`
}

func serve(t *testing.T, endpoint Endpoint, body string) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	endpoint.Handler(c)
	return recorder, c
}

func TestBodyJSONBindsTheBodyAndWritesTheStatus(t *testing.T) {
	recorder, _ := serve(t, BodyJSON(http.StatusCreated, func(_ *gin.Context, input greeting) (greeting, error) {
		return greeting{Name: "hello " + input.Name}, nil
	}), `{"name":"pocket"}`)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.JSONEq(t, `{"name":"hello pocket"}`, recorder.Body.String())
}

func TestBodyJSONRejectsAnInvalidBodyWithoutCallingTheHandler(t *testing.T) {
	_, c := serve(t, BodyJSON(http.StatusCreated, func(*gin.Context, greeting) (greeting, error) {
		t.Fatal("handler called with an invalid body")
		return greeting{}, nil
	}), `{}`)

	require.True(t, c.IsAborted())
	require.Len(t, c.Errors, 1)
}

func TestJSONLeavesErrorsToTheErrorMiddleware(t *testing.T) {
	expected := errors.New("lookup failed")
	recorder, c := serve(t, JSON(http.StatusOK, func(*gin.Context) (greeting, error) {
		return greeting{Name: "partial"}, expected
	}), "")

	require.ErrorIs(t, c.Errors.Last(), expected)
	require.Empty(t, recorder.Body.String())
}
