package utils

import (
	"github.com/mileusna/useragent"
)

// TruncateUserAgent cuts a client-chosen User-Agent to a length no real browser reaches, so it stays cheap to store and to parse
func TruncateUserAgent(userAgent string) string {
	return Truncate(userAgent, 512)
}

// ParseUserAgent parses only the start of a User-Agent, because the parser takes time quadratic in the length of some inputs
func ParseUserAgent(userAgent string) useragent.UserAgent {
	return useragent.Parse(TruncateUserAgent(userAgent))
}
