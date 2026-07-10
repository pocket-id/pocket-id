package utils

import (
	"strconv"
	"time"
)

func CacheControlValue(maxAge, staleWhileRevalidate time.Duration) string {
	maxAgeSeconds := strconv.Itoa(int(maxAge.Seconds()))
	staleWhileRevalidateSeconds := strconv.Itoa(int(staleWhileRevalidate.Seconds()))
	return "public, max-age=" + maxAgeSeconds + ", stale-while-revalidate=" + staleWhileRevalidateSeconds
}
