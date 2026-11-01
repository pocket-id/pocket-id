package common

// AuthenticationMethodsClaim is the JWT claim ("amr") used to identify how the user
// authenticated. It is shared between the session JWTs and the OIDC tokens.
const AuthenticationMethodsClaim = "amr"

// TokenTypeClaim is the JWT claim ("type") used to identify the type of token.
const TokenTypeClaim = "type"

// IsProtocolClaim reports whether Pocket ID or fosite sets the claim itself, so neither custom claims nor claim mapping policies may set it.
func IsProtocolClaim(name string) bool {
	switch name {
	case "sub",
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
		"sid",
		"scope",
		"scp",
		"client_id",
		TokenTypeClaim:
		return true
	default:
		return false
	}
}
