package apperror

import (
	"fmt"
	"net/http"
)

// The constructors in this file keep public messages, statuses, and details in one place

func AlreadyInUse(property string) *Error {
	return New(CodeAlreadyInUse, http.StatusConflict, property+" is already in use").WithDetail("property", property)
}

func NotFound(resource string) *Error {
	return New(CodeNotFound, http.StatusNotFound, resource+" not found").WithDetail("resource", resource)
}

func InvalidRequestBody(cause error) *Error {
	return Wrap(cause, CodeInvalidRequestBody, http.StatusBadRequest, "Request body is invalid")
}

func MissingField(field string) *Error {
	return InvalidField(field, "required", "is required")
}

func InvalidField(field, code, message string) *Error {
	return New(CodeValidationFailed, http.StatusBadRequest, fmt.Sprintf("%s %s", field, message)).WithFields([]FieldError{{
		Field:   field,
		Code:    code,
		Message: message,
	}})
}

func SetupNotAvailable() *Error {
	return New(CodeSetupNotAvailable, http.StatusNotFound, "Not found")
}

func SetupAlreadyCompleted() *Error {
	return New(CodeSetupAlreadyCompleted, http.StatusConflict, "Initial setup has already been completed")
}

func TokenInvalidOrExpired() *Error {
	return New(CodeTokenInvalidOrExpired, http.StatusUnauthorized, "Token is invalid or expired")
}

func DeviceCodeInvalid() *Error {
	return New(CodeDeviceCodeInvalid, http.StatusUnauthorized, "One-time access code must be used on the device it was generated for")
}

func TokenInvalid() *Error {
	return New(CodeInvalidToken, http.StatusUnauthorized, "Token is invalid")
}

func OidcMissingAuthorization() *Error {
	return New(CodeOidcMissingAuthorization, http.StatusForbidden, "Authorization is missing")
}

func OidcInvalidCallbackURL() *Error {
	return New(CodeOidcInvalidCallbackURL, http.StatusBadRequest, "Callback URL is invalid and may need to be corrected by an administrator")
}

func InvalidCIMDURLPattern(pattern string) *Error {
	return New(CodeValidationFailed, http.StatusBadRequest, "Metadata document URL pattern is invalid").
		WithDetail("pattern", pattern).
		WithFields([]FieldError{{
			Field:   "cimdUrlAllowlist",
			Code:    "invalid_value",
			Message: "contains an invalid URL pattern",
		}})
}

func UnsupportedFileType(expected string) *Error {
	if expected == "" {
		return New(CodeFileTypeNotSupported, http.StatusUnsupportedMediaType, "File type is not supported")
	}

	return New(CodeFileTypeNotSupported, http.StatusUnsupportedMediaType, "File must be of type "+expected).
		WithDetail("expected_file_type", expected)
}

func FileTooLarge(maxSize string) *Error {
	return New(CodeFileTooLarge, http.StatusRequestEntityTooLarge, fmt.Sprintf("File must not exceed %s", maxSize)).WithDetail("max_size", maxSize)
}

func NotSignedIn() *Error {
	return New(CodeNotSignedIn, http.StatusUnauthorized, "You are not signed in")
}

func MissingPermission() *Error {
	return New(CodeForbidden, http.StatusForbidden, "You don't have permission to perform this action")
}

func AuthenticationPathChangeBlocked() *Error {
	return New(CodeAuthenticationPathChangeBlocked, http.StatusConflict, "Authentication path can't change while active credentials exist")
}

func AuthenticationPathMismatch() *Error {
	return New(CodeAuthenticationPathMismatch, http.StatusConflict, "This credential isn't available for the configured authentication path")
}

func RuntimeCredentialInvalid() *Error {
	return New(CodeRuntimeCredentialInvalid, http.StatusUnauthorized, "Runtime credential request is invalid or expired")
}

func RuntimeCredentialExists() *Error {
	return New(CodeRuntimeCredentialExists, http.StatusConflict, "An active runtime credential already exists")
}

func TooManyRequests() *Error {
	return New(CodeRateLimited, http.StatusTooManyRequests, "Too many requests")
}

func UserNotFound() *Error {
	return New(CodeUserNotFound, http.StatusNotFound, "User not found")
}

func InvalidImage(cause error) *Error {
	return Wrap(cause, CodeInvalidImage, http.StatusBadRequest, "File is not a valid image")
}

func MissingSessionID() *Error {
	return New(CodeMissingSessionID, http.StatusBadRequest, "Session ID is missing")
}

func InvalidWebAuthnSession() *Error {
	return New(CodeInvalidWebAuthnSession, http.StatusBadRequest, "Your passkey request has expired")
}

func InvalidWebAuthnResponse(cause error) *Error {
	return Wrap(cause, CodeInvalidWebAuthnResponse, http.StatusBadRequest, "We couldn't process the response from your passkey")
}

func WebAuthnAuthenticationFailed(cause error) *Error {
	return Wrap(cause, CodeWebAuthnAuthenticationFailed, http.StatusUnauthorized, "We couldn't verify your passkey")
}

func PasskeyUserVerificationRequired(cause error) *Error {
	return Wrap(cause, CodePasskeyUserVerificationRequired, http.StatusBadRequest, "Your passkey couldn't verify you. If you're using a security key, configure a FIDO2 PIN and try again")
}

func SyncedPasskeyNotAllowed() *Error {
	return New(CodeSyncedPasskeyNotAllowed, http.StatusBadRequest, "Synced passkeys are not allowed")
}

func ReservedClaim(key string) *Error {
	return New(CodeReservedClaim, http.StatusBadRequest, fmt.Sprintf("Claim %s is reserved and can't be used", key)).
		WithDetail("key", key).
		WithFields([]FieldError{{
			Field:   "key",
			Code:    "reserved",
			Message: "is reserved",
		}})
}

func DuplicateClaim(key string) *Error {
	return New(CodeDuplicateClaim, http.StatusBadRequest, fmt.Sprintf("Claim %s is already defined", key)).
		WithDetail("key", key).
		WithFields([]FieldError{{
			Field:   "key",
			Code:    "duplicate",
			Message: "is listed more than once",
		}})
}

func LdapDisabled() *Error {
	return New(CodeLdapDisabled, http.StatusConflict, "LDAP is not enabled")
}

func LdapUserUpdate() *Error {
	return New(CodeLdapUserUpdate, http.StatusForbidden, "LDAP users can't be updated")
}

func LdapUserGroupUpdate() *Error {
	return New(CodeLdapUserGroupUpdate, http.StatusForbidden, "LDAP user groups can't be updated")
}

func OidcAccessDenied() *Error {
	return New(CodeOidcAccessDenied, http.StatusForbidden, "You're not allowed to access this service")
}

func OidcInteractionNotFound() *Error {
	return New(CodeNotFound, http.StatusNotFound, "OIDC interaction not found or expired").
		WithDetail("resource", "OIDC interaction")
}

func OidcClientIDNotMatching() *Error {
	return New(CodeOidcClientIDNotMatching, http.StatusBadRequest, "Client ID in request doesn't match client ID in token")
}

func UIConfigDisabled() *Error {
	return New(CodeUIConfigDisabled, http.StatusForbidden, "The configuration can't be changed since the UI configuration is disabled")
}

func InvalidUserID() *Error {
	return Validation([]FieldError{{
		Field:   "userId",
		Code:    "invalid_format",
		Message: "must be a valid UUID",
	}})
}

func OneTimeAccessDisabled() *Error {
	return New(CodeOneTimeAccessDisabled, http.StatusForbidden, "One-time access is disabled")
}

func DeviceLoginRequestInvalidOrExpired() *Error {
	return New(CodeDeviceLoginExpired, http.StatusUnauthorized, "Device login request is invalid or expired")
}

func DeviceLoginDenied() *Error {
	return New(CodeDeviceLoginDenied, http.StatusForbidden, "Device login request was denied")
}

func InvalidAPIKey() *Error {
	return New(CodeInvalidAPIKey, http.StatusUnauthorized, "API key is invalid")
}

func NoAPIKeyProvided() *Error {
	return New(CodeNoAPIKeyProvided, http.StatusUnauthorized, "API key is missing")
}

func APIKeyNotFound() *Error {
	return New(CodeAPIKeyNotFound, http.StatusNotFound, "API key not found")
}

func APIKeyNotExpired() *Error {
	return New(CodeAPIKeyNotExpired, http.StatusConflict, "API key is not expired yet")
}

func InvalidAPIKeyExpiration() *Error {
	return New(CodeInvalidAPIKeyExpiration, http.StatusBadRequest, "API key expiration time must be in the future").WithFields([]FieldError{{
		Field:   "expiresAt",
		Code:    "invalid_value",
		Message: "must be in the future",
	}})
}

func APIKeyAuthNotAllowed() *Error {
	return New(CodeAPIKeyAuthNotAllowed, http.StatusForbidden, "API key authentication is not allowed for this endpoint")
}

func UserDisabled() *Error {
	return New(CodeUserDisabled, http.StatusForbidden, "User account is disabled")
}

func ValidationMessage(message string) *Error {
	return New(CodeValidationFailed, http.StatusBadRequest, message)
}

func OidcDeviceCodeExpired() *Error {
	return New(CodeOidcDeviceCodeExpired, http.StatusBadRequest, "Device code has expired")
}

func OidcInvalidDeviceCode() *Error {
	return New(CodeOidcInvalidDeviceCode, http.StatusBadRequest, "Device code is invalid")
}

func ReauthenticationRequired() *Error {
	return New(CodeReauthenticationRequired, http.StatusUnauthorized, "Reauthentication is required")
}

func ReauthenticationRequiredWithCause(cause error) *Error {
	return Wrap(cause, CodeReauthenticationRequired, http.StatusUnauthorized, "Reauthentication is required")
}

func OpenSignupDisabled() *Error {
	return New(CodeOpenSignupDisabled, http.StatusForbidden, "Open user signup is not enabled")
}

func ClientIDAlreadyExists() *Error {
	return New(CodeClientIDAlreadyExists, http.StatusConflict, "Client ID is already in use")
}

func UserEmailNotSet() *Error {
	return New(CodeUserEmailNotSet, http.StatusConflict, "The user does not have an email address set")
}

func ImageNotFound() *Error {
	return New(CodeImageNotFound, http.StatusNotFound, "Image not found")
}

func LogoDownloadFailed(cause error) *Error {
	return Wrap(cause, CodeLogoDownloadFailed, http.StatusUnprocessableEntity, "Logo could not be downloaded")
}

func LogoTypeNotSupported() *Error {
	return New(CodeLogoTypeNotSupported, http.StatusUnprocessableEntity, "Downloaded logo has an unsupported file type")
}

func LogoTooLarge(maxSize string) *Error {
	return New(CodeLogoTooLarge, http.StatusUnprocessableEntity, fmt.Sprintf("Downloaded logo must not exceed %s", maxSize)).
		WithDetail("max_size", maxSize)
}

func InvalidLogoURL(cause error) *Error {
	return Wrap(cause, CodeValidationFailed, http.StatusBadRequest, "Logo URL is not allowed").WithFields([]FieldError{{
		Field:   "logoUrl",
		Code:    "invalid_value",
		Message: "must be a public HTTP or HTTPS URL",
	}})
}

func OidcPARRequired() *Error {
	return New(CodeOidcPARRequired, http.StatusBadRequest, "This client requires pushed authorization requests")
}

func InvalidEmailVerificationToken() *Error {
	return New(CodeEmailVerificationTokenInvalid, http.StatusBadRequest, "Email verification token is invalid")
}
