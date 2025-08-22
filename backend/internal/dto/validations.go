package dto

import (
	"regexp"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/utils"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	v := binding.Validator.Engine().(*validator.Validate)

	// [a-zA-Z0-9]      : The username must start with an alphanumeric character
	// [a-zA-Z0-9_.@-]* : The rest of the username can contain alphanumeric characters, dots, underscores, hyphens, and "@" symbols
	// [a-zA-Z0-9]$     : The username must end with an alphanumeric character
	var validateUsernameRegex = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.@-]*[a-zA-Z0-9]$")

	var validateClientIDRegex = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

	// Maximum allowed value for TTLs
	const maxTTL = 31 * 24 * time.Hour

	// Errors here are development-time ones
	err := v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		return validateUsernameRegex.MatchString(fl.Field().String())
	})
	if err != nil {
		panic("Failed to register custom validation for username: " + err.Error())
	}
	err = v.RegisterValidation("client_id", func(fl validator.FieldLevel) bool {
		return validateClientIDRegex.MatchString(fl.Field().String())
	})
	err = v.RegisterValidation("ttl", func(fl validator.FieldLevel) bool {
		ttl, ok := fl.Field().Interface().(utils.JSONDuration)
		if !ok {
			return false
		}
		// Allow zero, which means the field wasn't set
		return ttl.Duration == 0 || ttl.Duration > time.Second && ttl.Duration <= maxTTL
	})
	if err != nil {
		panic("Failed to register custom validation for ttl: " + err.Error())
	}
}
