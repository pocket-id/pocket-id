package cookie

import (
	"net/http"
	"time"
)

func NewAccessTokenCookie(maxAgeInSeconds int, token string) *http.Cookie {
	return newCookie(AccessTokenCookieName, token, maxAgeInSeconds, "/")
}

func NewSessionIDCookie(maxAgeInSeconds int, sessionID string) *http.Cookie {
	return newCookie(SessionIdCookieName, sessionID, maxAgeInSeconds, "/")
}

func NewDeviceTokenCookie(deviceToken string) *http.Cookie {
	return newCookie(DeviceTokenCookieName, deviceToken, int(15*time.Minute.Seconds()), "/api/one-time-access-token")
}

func NewReauthenticationTokenCookie(reauthenticationToken string) *http.Cookie {
	return newCookie(ReauthenticationTokenCookieName, reauthenticationToken, int(3*time.Minute.Seconds()), "/")
}

func newCookie(name, value string, maxAge int, path string) *http.Cookie {
	// SameSite remains unset to preserve the cookies emitted by the existing Gin helpers
	//nolint:gosec
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		Secure:   true,
		HttpOnly: true,
	}
}
