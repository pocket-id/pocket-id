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
// (...)?           : This allows single-character usernames (just one alphanumeric character)
var validateUsernameRegex = regexp.MustCompile("^[a-zA-Z0-9]([a-zA-Z0-9_.@-]*[a-zA-Z0-9])?$")

var validateClientIDRegex = regexp.MustCompile("^[a-zA-Z0-9._-]+$")

func init() {
	engine := binding.Validator.Engine().(*validator.Validate)

	// Maximum allowed value for TTLs
	const maxTTL = 31 * 24 * time.Hour

	validators := map[string]validator.Func{
		"username": func(fl validator.FieldLevel) bool {
			return ValidateUsername(fl.Field().String())
		},
		"client_id": func(fl validator.FieldLevel) bool {
			return ValidateClientID(fl.Field().String())
		},
		"ttl": func(fl validator.FieldLevel) bool {
			ttl, ok := fl.Field().Interface().(utils.JSONDuration)
			if !ok {
				return false
			}
			// Allow zero, which means the field wasn't set
			return ttl.Duration == 0 || (ttl.Duration > time.Second && ttl.Duration <= maxTTL)
		},
		"callback_url": func(fl validator.FieldLevel) bool {
			return ValidateCallbackURL(fl.Field().String())
		},
		"callback_url_pattern": func(fl validator.FieldLevel) bool {
			return ValidateCallbackURLPattern(fl.Field().String())
		},
	}
	for k, v := range validators {
		err := engine.RegisterValidation(k, v)
		if err != nil {
			panic("Failed to register custom validation for " + k + ": " + err.Error())
		}
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

// ValidateCallbackURL validates the input callback URL
func ValidateCallbackURL(str string) bool {
	// Ensure the URL is a valid one and that the protocol is not "javascript:" or "data:"
	u, err := url.Parse(str)
	if err != nil {
		return false
	}

	switch strings.ToLower(u.Scheme) {
	case "javascript", "data":
		return false
	default:
		return true
	}
}

// ValidateCallbackURLPattern validates callback URL patterns, with support for wildcards
func ValidateCallbackURLPattern(raw string) bool {
	err := utils.ValidateCallbackURLPattern(raw)
	return err == nil
}
