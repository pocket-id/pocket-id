package utils

import (
	"net/http"
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

// SetCacheControlHeader sets the Cache-Control header for the response.
func SetCacheControlHeader(ctx *gin.Context, maxAge, staleWhileRevalidate time.Duration) {
	_, ok := ctx.GetQuery("skipCache")
	if !ok {
		maxAgeSeconds := strconv.Itoa(int(maxAge.Seconds()))
		staleWhileRevalidateSeconds := strconv.Itoa(int(staleWhileRevalidate.Seconds()))
		ctx.Header("Cache-Control", "public, max-age="+maxAgeSeconds+", stale-while-revalidate="+staleWhileRevalidateSeconds)
	}

}
