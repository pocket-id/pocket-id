package common

// AuthenticationMethodsClaim is the JWT claim ("amr") used to identify how the user
// authenticated. It is shared between the session JWTs and the OIDC tokens.
const AuthenticationMethodsClaim = "amr"

// TokenTypeClaim is the JWT claim ("type") used to identify the type of token.
const TokenTypeClaim = "type"
