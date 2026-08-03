package apperror

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"time"
)

type Code string

// #nosec G101 -- these are client-visible error codes, not credentials
const (
	CodeInternal                        Code = "internal_error"
	CodeValidationFailed                Code = "validation_failed"
	CodeInvalidRequestBody              Code = "invalid_request_body"
	CodeRequestTimeout                  Code = "request_timeout"
	CodeNotFound                        Code = "not_found"
	CodeAlreadyInUse                    Code = "already_in_use"
	CodeForbidden                       Code = "forbidden"
	CodeInvalidToken                    Code = "invalid_token"
	CodeRateLimited                     Code = "rate_limited"
	CodeFileTooLarge                    Code = "file_too_large"
	CodeInvalidImage                    Code = "invalid_image"
	CodeInvalidWebAuthnResponse         Code = "invalid_webauthn_response"
	CodeWebAuthnAuthenticationFailed    Code = "webauthn_authentication_failed"
	CodePasskeyUserVerificationRequired Code = "passkey_user_verification_required"
	CodeInvalidWebAuthnSession          Code = "invalid_webauthn_session"
	CodeUserNotFound                    Code = "user_not_found"
	CodeUserDisabled                    Code = "user_disabled"
	CodeAPIKeyNotFound                  Code = "api_key_not_found"
	CodeInvalidAPIKey                   Code = "invalid_api_key"
	CodeDeviceLoginExpired              Code = "device_login_expired"
	CodeReauthenticationRequired        Code = "reauthentication_required"
	CodeEmailVerificationTokenInvalid   Code = "invalid_email_verification_token"
	CodeSetupNotAvailable               Code = "setup_not_available"
	CodeSetupAlreadyCompleted           Code = "setup_already_completed"
	CodeTokenInvalidOrExpired           Code = "token_invalid_or_expired"
	CodeDeviceCodeInvalid               Code = "device_code_invalid"
	CodeOidcMissingAuthorization        Code = "oidc_missing_authorization"
	CodeOidcInvalidCallbackURL          Code = "oidc_invalid_callback_url"
	CodeFileTypeNotSupported            Code = "file_type_not_supported"
	CodeNotSignedIn                     Code = "not_signed_in"
	CodeMissingSessionID                Code = "missing_session_id"
	CodeReservedClaim                   Code = "reserved_claim"
	CodeDuplicateClaim                  Code = "duplicate_claim"
	CodeLdapDisabled                    Code = "ldap_disabled"
	CodeLdapUserUpdate                  Code = "ldap_user_update"
	CodeLdapUserGroupUpdate             Code = "ldap_user_group_update"
	CodeOidcAccessDenied                Code = "oidc_access_denied"
	CodeOidcClientIDNotMatching         Code = "oidc_client_id_not_matching"
	CodeUIConfigDisabled                Code = "ui_config_disabled"
	CodeOneTimeAccessDisabled           Code = "one_time_access_disabled"
	CodeDeviceLoginDenied               Code = "device_login_denied"
	CodeNoAPIKeyProvided                Code = "no_api_key_provided"
	CodeAPIKeyNotExpired                Code = "api_key_not_expired"
	CodeInvalidAPIKeyExpiration         Code = "invalid_api_key_expiration"
	CodeAPIKeyAuthNotAllowed            Code = "api_key_auth_not_allowed"
	CodeOidcDeviceCodeExpired           Code = "oidc_device_code_expired"
	CodeOidcInvalidDeviceCode           Code = "oidc_invalid_device_code"
	CodeOpenSignupDisabled              Code = "open_signup_disabled"
	CodeClientIDAlreadyExists           Code = "client_id_already_exists"
	CodeUserEmailNotSet                 Code = "user_email_not_set"
	CodeImageNotFound                   Code = "image_not_found"
	CodeLogoDownloadFailed              Code = "logo_download_failed"
	CodeLogoTypeNotSupported            Code = "logo_type_not_supported"
	CodeLogoTooLarge                    Code = "logo_too_large"
	CodeOidcPARRequired                 Code = "oidc_par_required"
)

// FieldError describes one safe, client-actionable validation failure
type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error is an application error with a stable code, an HTTP status, and an optional internal cause
type Error struct {
	code       Code
	status     int
	message    string
	details    map[string]string
	fields     []FieldError
	retryAfter time.Duration
	cause      error
}

// New creates a client-safe application error
func New(code Code, status int, message string) *Error {
	return &Error{code: code, status: status, message: message}
}

// Wrap creates a client-safe application error that retains an internal cause
func Wrap(cause error, code Code, status int, message string) *Error {
	if cause == nil {
		return New(code, status, message)
	}

	return &Error{code: code, status: status, message: message, cause: cause}
}

// Internal creates a generic server error that retains a diagnostic cause without exposing it to clients
func Internal(cause error) *Error {
	return Wrap(cause, CodeInternal, http.StatusInternalServerError, "Something went wrong")
}

// Validation creates a structured validation error
func Validation(fields []FieldError) *Error {
	return New(CodeValidationFailed, http.StatusBadRequest, "Request validation failed").WithFields(fields)
}

// Error returns diagnostic text including the internal cause when one exists
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.cause == nil {
		return e.message
	}

	return fmt.Sprintf("%s: %v", e.message, e.cause)
}

// Unwrap exposes the internal cause to errors.Is and errors.As
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

// Is matches application errors by stable code while ignoring message and detail differences
func (e *Error) Is(target error) bool {
	targetError, ok := target.(*Error)
	return ok && e != nil && targetError != nil && e.code == targetError.code
}

// IsCode reports whether an error or one of its wrapped causes has the given application code
func IsCode(err error, code Code) bool {
	return errors.Is(err, &Error{code: code})
}

// Code returns the stable application error code
func (e *Error) Code() Code {
	if e == nil {
		return ""
	}

	return e.code
}

// HTTPStatus returns the HTTP status associated with the error
func (e *Error) HTTPStatus() int {
	if e == nil {
		return http.StatusInternalServerError
	}

	return e.status
}

// ClientMessage returns the message that is safe to include in an HTTP response
func (e *Error) ClientMessage() string {
	if e == nil {
		return ""
	}

	return e.message
}

// Details returns a copy of the additional client-safe details
func (e *Error) Details() map[string]string {
	if e == nil || len(e.details) == 0 {
		return nil
	}

	details := make(map[string]string, len(e.details))
	maps.Copy(details, e.details)

	return details
}

// Fields returns a copy of the structured validation details
func (e *Error) Fields() []FieldError {
	if e == nil || len(e.fields) == 0 {
		return nil
	}

	return append([]FieldError(nil), e.fields...)
}

// RetryAfter returns the duration a client should wait before retrying
func (e *Error) RetryAfter() time.Duration {
	if e == nil {
		return 0
	}

	return e.retryAfter
}

// WithFields attaches structured validation details without exposing the cause
func (e *Error) WithFields(fields []FieldError) *Error {
	if e == nil {
		return nil
	}

	e.fields = append([]FieldError(nil), fields...)
	return e
}

// WithDetail attaches one client-safe string detail
func (e *Error) WithDetail(key, value string) *Error {
	if e == nil {
		return nil
	}

	if e.details == nil {
		e.details = make(map[string]string)
	}
	e.details[key] = value
	return e
}

// WithRetryAfter attaches a retry delay to the error
func (e *Error) WithRetryAfter(retryAfter time.Duration) *Error {
	if e == nil {
		return nil
	}

	e.retryAfter = retryAfter
	return e
}
