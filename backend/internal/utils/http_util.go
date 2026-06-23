package utils

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// SetCacheControlHeader sets the Cache-Control header for the response.
func SetCacheControlHeader(ctx *gin.Context, maxAge, staleWhileRevalidate time.Duration) {
	_, ok := ctx.GetQuery("skipCache")
	if !ok {
		maxAgeSeconds := strconv.Itoa(int(maxAge.Seconds()))
		staleWhileRevalidateSeconds := strconv.Itoa(int(staleWhileRevalidate.Seconds()))
		ctx.Header("Cache-Control", "public, max-age="+maxAgeSeconds+", stale-while-revalidate="+staleWhileRevalidateSeconds)
	}

}
