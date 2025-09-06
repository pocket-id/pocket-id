package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
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

			// Check for validation errors
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				message := handleValidationError(validationErrors)
				errorResponse(c, http.StatusBadRequest, message)
				return
			}

			// Check for slice validation errors
			var sliceValidationErrors binding.SliceValidationError
			if errors.As(err, &sliceValidationErrors) {
				if errors.As(sliceValidationErrors[0], &validationErrors) {
					message := handleValidationError(validationErrors)
					errorResponse(c, http.StatusBadRequest, message)
					return
				}
			}

			var appErr common.AppError
			if errors.As(err, &appErr) {
				errorResponse(c, appErr.HttpStatusCode(), appErr.Error())
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
		}
	}
}

func errorResponse(c *gin.Context, statusCode int, message string) {
	// Capitalize the first letter of the message
	message = strings.ToUpper(message[:1]) + message[1:]
	c.JSON(statusCode, gin.H{"error": message})
}

func handleValidationError(validationErrors validator.ValidationErrors) string {
	var errorMessages []string

	for _, ve := range validationErrors {
		fieldName := ve.Field()
		var errorMessage string
		switch ve.Tag() {
		case "required":
			errorMessage = fmt.Sprintf("%s is required", fieldName)
		case "email":
			errorMessage = fmt.Sprintf("%s must be a valid email address", fieldName)
		case "username":
			errorMessage = fmt.Sprintf("%s must only contain letters, numbers, underscores, dots, hyphens, and '@' symbols and not start or end with a special character", fieldName)
		case "url":
			errorMessage = fmt.Sprintf("%s must be a valid URL", fieldName)
		case "min":
			errorMessage = fmt.Sprintf("%s must be at least %s characters long", fieldName, ve.Param())
		case "max":
			errorMessage = fmt.Sprintf("%s must be at most %s characters long", fieldName, ve.Param())
		default:
			errorMessage = fmt.Sprintf("%s is invalid", fieldName)
		}

		errorMessages = append(errorMessages, errorMessage)
	}

	// Join all the error messages into a single string
	combinedErrors := strings.Join(errorMessages, ", ")

	return combinedErrors
}
