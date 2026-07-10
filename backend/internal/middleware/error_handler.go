package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"gorm.io/gorm"
)

type ErrorHandlerMiddleware struct{}

func NewErrorHandlerMiddleware() *ErrorHandlerMiddleware {
	return &ErrorHandlerMiddleware{}
}

func (m *ErrorHandlerMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		for _, err := range c.Errors {
			// Check for record not found errors
			if errors.Is(err, gorm.ErrRecordNotFound) {
				errorResponse(c, http.StatusNotFound, "Record not found")
				return
			}

			// AppError with description
			appDescErr, ok := errors.AsType[common.AppErrorDescription](err)
			if ok {
				errorResponseWithDescription(c, appDescErr.HttpStatusCode(), appDescErr.Error(), appDescErr.Description())
				return
			}

			// AppError (without description)
			appErr, ok := errors.AsType[common.AppError](err)
			if ok {
				errorResponse(c, appErr.HttpStatusCode(), appErr.Error())
				return
			}

			c.JSON(http.StatusInternalServerError, errorResponseBody{
				Error: "Something went wrong",
			})
		}
	}
}

type errorResponseBody struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func errorResponse(c *gin.Context, statusCode int, message string) {
	// Capitalize the first letter of the message
	message = strings.ToUpper(message[:1]) + message[1:]
	c.JSON(statusCode, errorResponseBody{
		Error: message,
	})
}

func errorResponseWithDescription(c *gin.Context, statusCode int, message string, description string) {
	// Capitalize the first letter of the message
	message = strings.ToUpper(message[:1]) + message[1:]
	c.JSON(statusCode, errorResponseBody{
		Error:            message,
		ErrorDescription: description,
	})
}
