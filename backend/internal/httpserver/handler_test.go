package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandleAttachesAndAbortsOnError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	expected := errors.New("request failed")

	Handle(func(*gin.Context) error {
		return expected
	})(c)

	require.True(t, c.IsAborted())
	require.Len(t, c.Errors, 1)
	require.ErrorIs(t, c.Errors[0], expected)
}

func TestHandleIgnoresCanceledRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	Handle(func(*gin.Context) error {
		return context.Canceled
	})(c)

	require.True(t, c.IsAborted())
	require.Empty(t, c.Errors)
}

func TestHandleRecoversPanicsWithStack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	Handle(func(*gin.Context) error {
		panic("private panic details")
	})(c)

	require.True(t, c.IsAborted())
	require.Len(t, c.Errors, 1)
	require.ErrorContains(t, c.Errors[0], "panic in HTTP handler (string): private panic details")
	require.ErrorContains(t, c.Errors[0], "TestHandleRecoversPanicsWithStack")
}

func TestHandlePreservesAbortHandlerPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	require.PanicsWithValue(t, http.ErrAbortHandler, func() {
		Handle(func(*gin.Context) error {
			panic(http.ErrAbortHandler)
		})(c)
	})
	require.Empty(t, c.Errors)
}
