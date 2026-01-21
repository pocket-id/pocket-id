package utils

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

// OAuthClientBasicAuth returns the OAuth client ID and secret provided in the request's
// Authorization header, if present. See RFC 6749, Section 2.3.
func OAuthClientBasicAuth(r *http.Request) (clientID, clientSecret string, ok bool) {
	clientID, clientSecret, ok = r.BasicAuth()
	if !ok {
		return "", "", false
	}

	clientID, err := url.QueryUnescape(clientID)
	if err != nil {
		return "", "", false
	}

	clientSecret, err = url.QueryUnescape(clientSecret)
	if err != nil {
		return "", "", false
	}

	return clientID, clientSecret, true
}

// SetCacheControlHeader sets the Cache-Control header for the response.
func SetCacheControlHeader(ctx *gin.Context, maxAge, staleWhileRevalidate time.Duration) {
	_, ok := ctx.GetQuery("skipCache")
	if !ok {
		maxAgeSeconds := strconv.Itoa(int(maxAge.Seconds()))
		staleWhileRevalidateSeconds := strconv.Itoa(int(staleWhileRevalidate.Seconds()))
		ctx.Header("Cache-Control", "public, max-age="+maxAgeSeconds+", stale-while-revalidate="+staleWhileRevalidateSeconds)
	}

}
