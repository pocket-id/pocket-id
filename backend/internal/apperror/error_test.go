package apperror

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapPreservesCauseWithoutChangingClientMessage(t *testing.T) {
	cause := errors.New("database connection details")
	err := Wrap(cause, CodeNotFound, http.StatusNotFound, "User not found")

	require.ErrorIs(t, err, cause)
	require.Equal(t, "User not found", err.ClientMessage())
	require.Contains(t, err.Error(), "database connection details")
}

func TestErrorsMatchByCode(t *testing.T) {
	err := Wrap(errors.New("database connection details"), CodeAlreadyInUse, http.StatusBadRequest, "email is already in use")
	wrapped := Wrap(err, CodeInternal, http.StatusInternalServerError, "request failed")

	require.True(t, IsCode(err, CodeAlreadyInUse))
	require.True(t, IsCode(wrapped, CodeAlreadyInUse))
	require.ErrorIs(t, err, New(CodeAlreadyInUse, http.StatusBadRequest, "username is already in use"))
	require.False(t, IsCode(wrapped, CodeNotFound))
}

func TestErrorCopiesFields(t *testing.T) {
	fields := []FieldError{{Field: "email", Code: "required", Message: "is required"}}
	err := New(CodeValidationFailed, http.StatusBadRequest, "Request validation failed").WithFields(fields)
	fields[0].Message = "changed"

	require.Equal(t, "is required", err.Fields()[0].Message)
}

func TestErrorCopiesDetails(t *testing.T) {
	err := New(CodeAlreadyInUse, http.StatusBadRequest, "Already in use").WithDetail("property", "email")
	details := err.Details()
	details["property"] = "password"

	require.Equal(t, map[string]string{"property": "email"}, err.Details())
}
