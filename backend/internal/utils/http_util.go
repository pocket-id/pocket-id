package utils

import (
	"net/http"
	"strings"
)

// BearerAuth returns the value of the bearer token in the Authorization header if present
func BearerAuth(r *http.Request) (string, bool) {
	const prefix = "bearer "

	authHeader := r.Header.Get("Authorization")
	if len(authHeader) >= len(prefix) && strings.ToLower(authHeader[:len(prefix)]) == prefix {
		return authHeader[len(prefix):], true
	}

	return "", false
}
