package middleware

import (
	"encoding/base64"
	"strings"

	"github.com/gin-gonic/gin"
)

// clientIDParamPrefix marks a path parameter whose value is a base64url-encoded
// client ID. CIMD client IDs are full https URLs, so they contain slashes and
// colons that cannot be carried in a single path segment. The frontend encodes
// such IDs as "~<base64url>"; this middleware decodes them back before
// handlers read c.Param.
//
// The prefix "~" is unreserved in RFC 3986 (so proxies leave it intact) and never
// appears in raw pocket-id client IDs ([a-zA-Z0-9._-]+) or user UUIDs, making the
// encoding unambiguous and backward compatible: unprefixed params pass through
// untouched, so external API consumers using plain client IDs are unaffected.
const clientIDParamPrefix = "~"

// decodedClientIDParamKeys lists the path parameter names that may carry an
// encoded client ID.
var decodedClientIDParamKeys = map[string]struct{}{
	"id":       {},
	"clientId": {},
}

// ClientIDParamMiddleware decodes "~<base64url>" client ID path parameters in
// place. Values without the prefix, or that fail to decode, are left unchanged.
type ClientIDParamMiddleware struct{}

func NewClientIDParamMiddleware() *ClientIDParamMiddleware {
	return &ClientIDParamMiddleware{}
}

func (m *ClientIDParamMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		for i, p := range c.Params {
			if _, ok := decodedClientIDParamKeys[p.Key]; !ok {
				continue
			}
			encoded, ok := strings.CutPrefix(p.Value, clientIDParamPrefix)
			if !ok {
				continue
			}
			decoded, err := base64.RawURLEncoding.DecodeString(encoded)
			if err != nil {
				continue
			}
			c.Params[i].Value = string(decoded)
		}

		c.Next()
	}
}
