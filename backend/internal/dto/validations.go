package dto

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/model"
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

	// Use JSON tags to keep client-visible validation field names stable
	engine.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})

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
		"resource_uri": func(fl validator.FieldLevel) bool {
			return ValidateResourceURI(fl.Field().String())
		},
		"token_duration": func(fl validator.FieldLevel) bool {
			return model.IsValidTokenDurationMinutes(fl.Field().Int())
		},
		"json_string_array": func(fl validator.FieldLevel) bool {
			return validateJSONStringArray(fl.Field().String())
		},
		"json_custom_claims": func(fl validator.FieldLevel) bool {
			return validateJSONCustomClaims(fl.Field().String())
		},
		"cimd_url_allowlist": func(fl validator.FieldLevel) bool {
			return validateCIMDURLAllowlist(fl.Field().String())
		},
		"boolean_string": func(fl validator.FieldLevel) bool {
			return validateBooleanString(fl.Field().String())
		},
		"integer_string": func(fl validator.FieldLevel) bool {
			return validateIntegerString(fl.Field().String())
		},
	}
	for k, v := range validators {
		err := engine.RegisterValidation(k, v)
		if err != nil {
			panic("Failed to register custom validation for " + k + ": " + err.Error())
		}
	}
}

// validateJSONStringArray requires an array so downstream consumers never receive another valid JSON type
func validateJSONStringArray(value string) bool {
	var items []string
	return json.Unmarshal([]byte(value), &items) == nil && items != nil
}

// validateJSONCustomClaims requires every claim to contain string key and value properties
func validateJSONCustomClaims(value string) bool {
	var claims []*struct {
		Key   *string `json:"key"`
		Value *string `json:"value"`
	}
	if err := json.Unmarshal([]byte(value), &claims); err != nil || claims == nil {
		return false
	}

	for _, claim := range claims {
		if claim == nil || claim.Key == nil || claim.Value == nil {
			return false
		}
	}

	return true
}

// validateCIMDURLAllowlist requires an array of safe callback URL patterns
func validateCIMDURLAllowlist(value string) bool {
	var patterns []string
	if err := json.Unmarshal([]byte(value), &patterns); err != nil || patterns == nil {
		return false
	}

	for _, pattern := range patterns {
		if !ValidateCallbackURLPattern(pattern) {
			return false
		}
	}

	return true
}

// validateBooleanString accepts the exact values understood consistently by the backend and frontend
func validateBooleanString(value string) bool {
	return value == "true" || value == "false"
}

// validateIntegerString accepts the same integer representation used by AppConfigValue.AsDurationMinutes
func validateIntegerString(value string) bool {
	_, err := strconv.Atoi(value)
	return err == nil
}

// ValidationErrorDetails returns the stable code and human-readable message for a validation failure
func ValidationErrorDetails(validationError validator.FieldError) (string, string) {
	switch validationError.Tag() {
	case "required":
		return "required", "is required"
	case "email":
		return "invalid_format", "must be a valid email address"
	case "username":
		return "invalid_format", "must only contain letters, numbers, underscores, dots, hyphens, and '@' symbols and not start or end with a special character"
	case "url":
		return "invalid_format", "must be a valid URL"
	case "resource_uri":
		return "invalid_format", "must be an absolute URI without whitespace or a fragment"
	case "min":
		return "too_short", fmt.Sprintf("must be at least %s characters long", validationError.Param())
	case "max":
		return "too_long", fmt.Sprintf("must be at most %s characters long", validationError.Param())
	case "json_string_array":
		return "invalid_format", "must be a JSON array of strings"
	case "json_custom_claims":
		return "invalid_format", `must be a JSON array of objects with string "key" and "value" properties`
	case "cimd_url_allowlist":
		return "invalid_format", "must be a JSON array of valid callback URL patterns"
	case "boolean_string":
		return "invalid_format", "must be either true or false"
	case "integer_string":
		return "invalid_format", "must be an integer"
	default:
		return validationError.Tag(), "is invalid"
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

// isActiveContentScheme reports whether the URL scheme can carry executable content, so it must never be accepted where a URL might later be rendered as a link
func isActiveContentScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "javascript", "data":
		return true
	default:
		return false
	}
}

// ValidateResourceURI validates RFC 8707 resource identifiers
func ValidateResourceURI(str string) bool {
	if !fosite.IsValidResourceIndicatorURI(str) {
		return false
	}

	// Reject active-content schemes so a resource identifier can never carry executable content if it is ever surfaced as a link
	u, _ := url.Parse(str)
	return !isActiveContentScheme(u.Scheme)
}

// ValidateCallbackURL validates the input callback URL
func ValidateCallbackURL(str string) bool {
	// Ensure the URL is a valid one and that the protocol is not "javascript:" or "data:"
	u, err := url.Parse(str)
	if err != nil {
		return false
	}

	return !isActiveContentScheme(u.Scheme)
}

// ValidateCallbackURLPattern validates callback URL patterns, with support for wildcards.
func ValidateCallbackURLPattern(raw string) bool {
	return utils.ValidateCallbackURLPattern(raw) == nil
}
