package common

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError interface {
	error
	HttpStatusCode() int
}

// Custom error types for various conditions

type AlreadyInUseError struct {
	Property string
}

func (e *AlreadyInUseError) Error() string {
	return e.Property + " is already in use"
}
func (e *AlreadyInUseError) HttpStatusCode() int { return 400 }

func (e *AlreadyInUseError) Is(target error) bool {
	// Ignore the field property when checking if an error is of the type AlreadyInUseError
	x := &AlreadyInUseError{}
	return errors.As(target, &x)
}

type SetupAlreadyCompletedError struct{}

func (e *SetupAlreadyCompletedError) Error() string       { return "setup already completed" }
func (e *SetupAlreadyCompletedError) HttpStatusCode() int { return 400 }

type TokenInvalidOrExpiredError struct{}

func (e *TokenInvalidOrExpiredError) Error() string       { return "token is invalid or expired" }
func (e *TokenInvalidOrExpiredError) HttpStatusCode() int { return 400 }

type DeviceCodeInvalid struct{}

func (e *DeviceCodeInvalid) Error() string {
	return "one time access code must be used on the device it was generated for"
}
func (e *DeviceCodeInvalid) HttpStatusCode() int { return 400 }

type TokenInvalidError struct{}

func (e *TokenInvalidError) Error() string {
	return "Token is invalid"
}
func (e *TokenInvalidError) HttpStatusCode() int { return 400 }

type OidcMissingAuthorizationError struct{}

func (e *OidcMissingAuthorizationError) Error() string       { return "missing authorization" }
func (e *OidcMissingAuthorizationError) HttpStatusCode() int { return http.StatusForbidden }

type OidcGrantTypeNotSupportedError struct{}

func (e *OidcGrantTypeNotSupportedError) Error() string       { return "grant type not supported" }
func (e *OidcGrantTypeNotSupportedError) HttpStatusCode() int { return 400 }

type OidcMissingClientCredentialsError struct{}

func (e *OidcMissingClientCredentialsError) Error() string       { return "client id or secret not provided" }
func (e *OidcMissingClientCredentialsError) HttpStatusCode() int { return 400 }

type OidcClientSecretInvalidError struct{}

func (e *OidcClientSecretInvalidError) Error() string       { return "invalid client secret" }
func (e *OidcClientSecretInvalidError) HttpStatusCode() int { return 400 }

type OidcClientAssertionInvalidError struct{}

func (e *OidcClientAssertionInvalidError) Error() string       { return "invalid client assertion" }
func (e *OidcClientAssertionInvalidError) HttpStatusCode() int { return 400 }

type OidcInvalidAuthorizationCodeError struct{}

func (e *OidcInvalidAuthorizationCodeError) Error() string       { return "invalid authorization code" }
func (e *OidcInvalidAuthorizationCodeError) HttpStatusCode() int { return 400 }

type OidcMissingCallbackURLError struct{}

func (e *OidcMissingCallbackURLError) Error() string {
	return "unable to detect callback url, it might be necessary for an admin to fix this"
}
func (e *OidcMissingCallbackURLError) HttpStatusCode() int { return 400 }

type OidcInvalidCallbackURLError struct{}

func (e *OidcInvalidCallbackURLError) Error() string {
	return "invalid callback URL, it might be necessary for an admin to fix this"
}
func (e *OidcInvalidCallbackURLError) HttpStatusCode() int { return 400 }

type FileTypeNotSupportedError struct{}

func (e *FileTypeNotSupportedError) Error() string       { return "file type not supported" }
func (e *FileTypeNotSupportedError) HttpStatusCode() int { return 400 }

type FileTooLargeError struct {
	MaxSize string
}

func (e *FileTooLargeError) Error() string {
	return fmt.Sprintf("The file can't be larger than %s", e.MaxSize)
}
func (e *FileTooLargeError) HttpStatusCode() int { return http.StatusRequestEntityTooLarge }

type NotSignedInError struct{}

func (e *NotSignedInError) Error() string       { return "You are not signed in" }
func (e *NotSignedInError) HttpStatusCode() int { return http.StatusUnauthorized }

type MissingAccessToken struct{}

func (e *MissingAccessToken) Error() string       { return "Missing access token" }
func (e *MissingAccessToken) HttpStatusCode() int { return http.StatusUnauthorized }

type MissingPermissionError struct{}

func (e *MissingPermissionError) Error() string {
	return "You don't have permission to perform this action"
}
func (e *MissingPermissionError) HttpStatusCode() int { return http.StatusForbidden }

type TooManyRequestsError struct{}

func (e *TooManyRequestsError) Error() string {
	return "Too many requests"
}
func (e *TooManyRequestsError) HttpStatusCode() int { return http.StatusTooManyRequests }

type ClientIdOrSecretNotProvidedError struct{}

func (e *ClientIdOrSecretNotProvidedError) Error() string {
	return "Client id or secret not provided"
}
func (e *ClientIdOrSecretNotProvidedError) HttpStatusCode() int { return http.StatusBadRequest }

type WrongFileTypeError struct {
	ExpectedFileType string
}

func (e *WrongFileTypeError) Error() string {
	return fmt.Sprintf("File must be of type %s", e.ExpectedFileType)
}
func (e *WrongFileTypeError) HttpStatusCode() int { return http.StatusBadRequest }

type MissingSessionIdError struct{}

func (e *MissingSessionIdError) Error() string {
	return "Missing session id"
}
func (e *MissingSessionIdError) HttpStatusCode() int { return http.StatusBadRequest }

type ReservedClaimError struct {
	Key string
}

func (e *ReservedClaimError) Error() string {
	return fmt.Sprintf("Claim %s is reserved and can't be used", e.Key)
}
func (e *ReservedClaimError) HttpStatusCode() int { return http.StatusBadRequest }

type DuplicateClaimError struct {
	Key string
}

func (e *DuplicateClaimError) Error() string {
	return fmt.Sprintf("Claim %s is already defined", e.Key)
}
func (e *DuplicateClaimError) HttpStatusCode() int { return http.StatusBadRequest }

type OidcInvalidCodeVerifierError struct{}

func (e *OidcInvalidCodeVerifierError) Error() string {
	return "Invalid code verifier"
}
func (e *OidcInvalidCodeVerifierError) HttpStatusCode() int { return http.StatusBadRequest }

type OidcMissingCodeChallengeError struct{}

func (e *OidcMissingCodeChallengeError) Error() string {
	return "Missing code challenge"
}
func (e *OidcMissingCodeChallengeError) HttpStatusCode() int { return http.StatusBadRequest }

type LdapUserUpdateError struct{}

func (e *LdapUserUpdateError) Error() string {
	return "LDAP users can't be updated"
}
func (e *LdapUserUpdateError) HttpStatusCode() int { return http.StatusForbidden }

type LdapUserGroupUpdateError struct{}

func (e *LdapUserGroupUpdateError) Error() string {
	return "LDAP user groups can't be updated"
}
func (e *LdapUserGroupUpdateError) HttpStatusCode() int { return http.StatusForbidden }

type OidcAccessDeniedError struct{}

func (e *OidcAccessDeniedError) Error() string {
	return "You're not allowed to access this service"
}
func (e *OidcAccessDeniedError) HttpStatusCode() int { return http.StatusForbidden }

type OidcClientIdNotMatchingError struct{}

func (e *OidcClientIdNotMatchingError) Error() string {
	return "Client id in request doesn't match client id in token"
}
func (e *OidcClientIdNotMatchingError) HttpStatusCode() int { return http.StatusBadRequest }

type OidcNoCallbackURLError struct{}

func (e *OidcNoCallbackURLError) Error() string {
	return "No callback URL provided"
}
func (e *OidcNoCallbackURLError) HttpStatusCode() int { return http.StatusBadRequest }

type UiConfigDisabledError struct{}

func (e *UiConfigDisabledError) Error() string {
	return "The configuration can't be changed since the UI configuration is disabled"
}
func (e *UiConfigDisabledError) HttpStatusCode() int { return http.StatusForbidden }

type InvalidUUIDError struct{}

func (e *InvalidUUIDError) Error() string {
	return "Invalid UUID"
}
func (e *InvalidUUIDError) HttpStatusCode() int { return http.StatusBadRequest }

type OneTimeAccessDisabledError struct{}

func (e *OneTimeAccessDisabledError) Error() string {
	return "One-time access is disabled"
}
func (e *OneTimeAccessDisabledError) HttpStatusCode() int { return http.StatusBadRequest }

type InvalidAPIKeyError struct{}

func (e *InvalidAPIKeyError) Error() string {
	return "Invalid Api Key"
}
func (e *InvalidAPIKeyError) HttpStatusCode() int { return http.StatusUnauthorized }

type NoAPIKeyProvidedError struct{}

func (e *NoAPIKeyProvidedError) Error() string {
	return "No API Key Provided"
}
func (e *NoAPIKeyProvidedError) HttpStatusCode() int { return http.StatusUnauthorized }

type APIKeyNotFoundError struct{}

func (e *APIKeyNotFoundError) Error() string {
	return "API Key Not Found"
}
func (e *APIKeyNotFoundError) HttpStatusCode() int { return http.StatusUnauthorized }

type APIKeyNotExpiredError struct{}

func (e *APIKeyNotExpiredError) Error() string {
	return "API Key is not expired yet"
}
func (e *APIKeyNotExpiredError) HttpStatusCode() int { return http.StatusBadRequest }

type APIKeyExpirationDateError struct{}

func (e *APIKeyExpirationDateError) Error() string {
	return "API Key expiration time must be in the future"
}
func (e *APIKeyExpirationDateError) HttpStatusCode() int { return http.StatusBadRequest }

type OidcInvalidRefreshTokenError struct{}

func (e *OidcInvalidRefreshTokenError) Error() string {
	return "refresh token is invalid or expired"
}
func (e *OidcInvalidRefreshTokenError) HttpStatusCode() int {
	return http.StatusBadRequest
}

type OidcMissingRefreshTokenError struct{}

func (e *OidcMissingRefreshTokenError) Error() string {
	return "refresh token is required"
}
func (e *OidcMissingRefreshTokenError) HttpStatusCode() int {
	return http.StatusBadRequest
}

type OidcMissingAuthorizationCodeError struct{}

func (e *OidcMissingAuthorizationCodeError) Error() string {
	return "authorization code is required"
}
func (e *OidcMissingAuthorizationCodeError) HttpStatusCode() int {
	return http.StatusBadRequest
}

type UserDisabledError struct{}

func (e *UserDisabledError) Error() string {
	return "User account is disabled"
}
func (e *UserDisabledError) HttpStatusCode() int {
	return http.StatusForbidden
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func (e *ValidationError) HttpStatusCode() int {
	return http.StatusBadRequest
}

type OidcDeviceCodeExpiredError struct{}

func (e *OidcDeviceCodeExpiredError) Error() string {
	return "device code has expired"
}
func (e *OidcDeviceCodeExpiredError) HttpStatusCode() int {
	return http.StatusBadRequest
}

type OidcInvalidDeviceCodeError struct{}

func (e *OidcInvalidDeviceCodeError) Error() string {
	return "invalid device code"
}
func (e *OidcInvalidDeviceCodeError) HttpStatusCode() int {
	return http.StatusBadRequest
}

type OidcSlowDownError struct{}

func (e *OidcSlowDownError) Error() string {
	return "polling too frequently"
}
func (e *OidcSlowDownError) HttpStatusCode() int {
	return http.StatusTooManyRequests
}

type OidcAuthorizationPendingError struct{}

func (e *OidcAuthorizationPendingError) Error() string {
	return "authorization is still pending"
}
func (e *OidcAuthorizationPendingError) HttpStatusCode() int {
	return http.StatusBadRequest
}

type ReauthenticationRequiredError struct{}

func (e *ReauthenticationRequiredError) Error() string {
	return "reauthentication required"
}
func (e *ReauthenticationRequiredError) HttpStatusCode() int {
	return http.StatusUnauthorized
}

type OpenSignupDisabledError struct{}

func (e *OpenSignupDisabledError) Error() string {
	return "Open user signup is not enabled"
}

func (e *OpenSignupDisabledError) HttpStatusCode() int {
	return http.StatusForbidden
}

type ClientIdAlreadyExistsError struct{}

func (e *ClientIdAlreadyExistsError) Error() string {
	return "Client ID already in use"
}

func (e *ClientIdAlreadyExistsError) HttpStatusCode() int {
	return http.StatusBadRequest
}

type UserEmailNotSetError struct{}

func (e *UserEmailNotSetError) Error() string {
	return "The user does not have an email address set"
}

func (e *UserEmailNotSetError) HttpStatusCode() int {
	return http.StatusBadRequest
}

type ImageNotFoundError struct{}

func (e *ImageNotFoundError) Error() string {
	return "Image not found"
}

func (e *ImageNotFoundError) HttpStatusCode() int {
	return http.StatusNotFound
}

type InvalidEmailVerificationTokenError struct{}

func (e *InvalidEmailVerificationTokenError) Error() string {
	return "Invalid email verification token"
}

func (e *InvalidEmailVerificationTokenError) HttpStatusCode() int {
	return http.StatusBadRequest
}
