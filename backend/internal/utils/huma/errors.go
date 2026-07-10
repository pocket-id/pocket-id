package humautils

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"gorm.io/gorm"
)

type apiError struct {
	status      int
	Message     string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

func init() {
	huma.NewError = newHumaError
}

func (e *apiError) Error() string  { return e.Message }
func (e *apiError) GetStatus() int { return e.status }

func (e *apiError) ContentType(contentType string) string {
	if contentType == "application/json" {
		return "application/json; charset=utf-8"
	}
	return contentType
}

func newHumaError(status int, message string, errs ...error) huma.StatusError {
	if status == http.StatusUnprocessableEntity {
		status = http.StatusBadRequest
	}

	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		var detailer huma.ErrorDetailer
		if errors.As(err, &detailer) {
			messages = append(messages, detailer.ErrorDetail().Message)
			continue
		}
		messages = append(messages, err.Error())
	}
	if len(messages) > 0 {
		message = strings.Join(messages, ", ")
	}

	return &apiError{status: status, Message: capitalize(message)}
}

func capitalize(message string) string {
	if message == "" {
		return message
	}
	r, size := utf8.DecodeRuneInString(message)
	return string(unicode.ToUpper(r)) + message[size:]
}

func mapError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &apiError{status: http.StatusNotFound, Message: "Record not found"}
	}

	var appDescriptionError common.AppErrorDescription
	if errors.As(err, &appDescriptionError) {
		return &apiError{
			status:      appDescriptionError.HttpStatusCode(),
			Message:     capitalize(appDescriptionError.Error()),
			Description: appDescriptionError.Description(),
		}
	}

	var appError common.AppError
	if errors.As(err, &appError) {
		return &apiError{status: appError.HttpStatusCode(), Message: capitalize(appError.Error())}
	}

	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		return &apiError{status: http.StatusRequestEntityTooLarge, Message: "The request body is too large"}
	}

	slog.ErrorContext(ctx, "Unhandled API error", slog.Any("error", err))
	return &apiError{status: http.StatusInternalServerError, Message: "Something went wrong"}
}
