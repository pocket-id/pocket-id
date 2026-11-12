package common

import (
	"slices"
	"strings"
)

// AuthenticationMethodsClaim is the JWT claim ("amr") used to identify how the user
// authenticated. It is shared between the session JWTs and the OIDC tokens.
const AuthenticationMethodsClaim = "amr"

// TokenTypeClaim is the JWT claim ("type") used to identify the type of token.
const TokenTypeClaim = "type"

// SessionIDClaim is the JWT claim ("sid") that identifies the Pocket ID session an OIDC token or logout token belongs to
const SessionIDClaim = "sid"

var protocolClaims = []string{
	"sub",
	"iss",
	"aud",
	"exp",
	"iat",
	"nbf",
	"jti",
	"auth_time",
	"nonce",
	"acr",
	AuthenticationMethodsClaim,
	"azp",
	"at_hash",
	"c_hash",
	"rat",
	SessionIDClaim,
	"scope",
	"scp",
	"client_id",
	TokenTypeClaim,
}

// IsProtocolClaim reports whether Pocket ID or fosite sets the claim itself, so neither custom claims nor claim mapping policies may set it.
func IsProtocolClaim(name string) bool {
	return ContainsClaimName(protocolClaims, name)
}

// ContainsClaimName reports whether names contains name, ignoring case with Unicode folding.
// Go's encoding/json matches keys the same way, so a claim like "ſub" would otherwise override "sub" at Go relying parties.
func ContainsClaimName(names []string, name string) bool {
	return slices.ContainsFunc(names, func(n string) bool {
		return strings.EqualFold(n, name)
	})
}
