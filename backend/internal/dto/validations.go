package dto

import (
	"log"
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var validateUsername validator.Func = func(fl validator.FieldLevel) bool {
	// [a-zA-Z0-9]      : The username must start with an alphanumeric character
	// [a-zA-Z0-9_.@-]* : The rest of the username can contain alphanumeric characters, dots, underscores, hyphens, and "@" symbols
	// [a-zA-Z0-9]$     : The username must end with an alphanumeric character
	regex := "^[a-zA-Z0-9][a-zA-Z0-9_.@-]*[a-zA-Z0-9]$"
	matched, _ := regexp.MatchString(regex, fl.Field().String())
	return matched
}

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("username", validateUsername); err != nil {
			log.Fatalf("Failed to register custom validation: %v", err)
		}
	}
}
