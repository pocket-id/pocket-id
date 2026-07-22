package dto

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// [a-zA-Z0-9]      : The username must start with an alphanumeric character
// [a-zA-Z0-9_.@-]* : The rest of the username can contain alphanumeric characters, dots, underscores, hyphens, and "@" symbols
// [a-zA-Z0-9]$     : The username must end with an alphanumeric character
// (...)?           : This allows single-character usernames (just one alphanumeric character)
var validateUsernameRegex = regexp.MustCompile("^[a-zA-Z0-9]([a-zA-Z0-9_.@-]*[a-zA-Z0-9])?$")

var validateClientIDRegex = regexp.MustCompile("^[a-zA-Z0-9._-]+$")

const maxTTL = 31 * 24 * time.Hour

// ValidateUsername validates username inputs
func ValidateUsername(username string) bool {
	return validateUsernameRegex.MatchString(username)
}

// ValidateClientID validates client ID inputs
func ValidateClientID(clientID string) bool {
	return validateClientIDRegex.MatchString(clientID)
}

// ValidateTTL validates optional API durations against the existing bounds
func ValidateTTL(ttl utils.JSONDuration) bool {
	return ttl.Duration == 0 || (ttl.Duration > time.Second && ttl.Duration <= maxTTL)
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
