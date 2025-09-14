package dto

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/utils"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// [a-zA-Z0-9]      : The username must start with an alphanumeric character
// [a-zA-Z0-9_.@-]* : The rest of the username can contain alphanumeric characters, dots, underscores, hyphens, and "@" symbols
// [a-zA-Z0-9]$     : The username must end with an alphanumeric character
var validateUsernameRegex = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.@-]*[a-zA-Z0-9]$")

var validateClientIDRegex = regexp.MustCompile("^[a-zA-Z0-9._-]+$")

func init() {
	v := binding.Validator.Engine().(*validator.Validate)

	// Maximum allowed value for TTLs
	const maxTTL = 31 * 24 * time.Hour

	if err := v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		return ValidateUsername(fl.Field().String())
	}); err != nil {
		panic("Failed to register custom validation for username: " + err.Error())
	}

	if err := v.RegisterValidation("client_id", func(fl validator.FieldLevel) bool {
		return ValidateClientID(fl.Field().String())
	}); err != nil {
		panic("Failed to register custom validation for client_id: " + err.Error())
	}

	if err := v.RegisterValidation("ttl", func(fl validator.FieldLevel) bool {
		ttl, ok := fl.Field().Interface().(utils.JSONDuration)
		if !ok {
			return false
		}
		// Allow zero, which means the field wasn't set
		return ttl.Duration == 0 || (ttl.Duration > time.Second && ttl.Duration <= maxTTL)
	}); err != nil {
		panic("Failed to register custom validation for ttl: " + err.Error())
	}

	if err := v.RegisterValidation("callback_url", func(fl validator.FieldLevel) bool {
		return ValidateCallbackURL(fl.Field().String())
	}); err != nil {
		panic("Failed to register custom validation for callback_url: " + err.Error())
	}
}

// ValidateUsername validates username inputs
func ValidateUsername(username string) bool {
	return validateUsernameRegex.MatchString(username)
}

// ValidateClientID validates client ID inputs
func ValidateClientID(clientID string) bool {
	return validateClientIDRegex.MatchString(clientID)
}

// ValidateCallbackURL validates callback URLs with support for wildcards
func ValidateCallbackURL(raw string) bool {
	if raw == "*" {
		return true
	}

	// Replace all '*' with 'x' to check if the rest is still a valid URI
	test := strings.ReplaceAll(raw, "*", "x")

	u, err := url.Parse(test)
	if err != nil {
		return false
	}

	if !u.IsAbs() {
		return false
	}

	return true
}
