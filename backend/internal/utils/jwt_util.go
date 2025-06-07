package utils

import (
	"fmt"

	"github.com/lestrrat-go/jwx/v3/jwt"
)

func ParseTokenPayload(tokenString string, isIdToken bool) (map[string]interface{}, error) {
	token, err := jwt.ParseInsecure([]byte(tokenString))
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	payload := make(map[string]interface{})

	extractStandardClaims(token, payload)

	if isIdToken {
		extractIdTokenClaims(token, payload)
	} else {
		extractAccessTokenClaims(token, payload)
	}

	extractProfileClaims(token, payload)
	extractCustomClaims(token, payload)

	return payload, nil
}

func extractStandardClaims(token jwt.Token, payload map[string]interface{}) {
	if sub, ok := token.Subject(); ok {
		payload["sub"] = sub
	}
	if iss, ok := token.Issuer(); ok {
		payload["iss"] = iss
	}
	if aud, ok := token.Audience(); ok {
		payload["aud"] = aud
	}
	if exp, ok := token.Expiration(); ok {
		payload["exp"] = exp.Unix()
	}
	if iat, ok := token.IssuedAt(); ok {
		payload["iat"] = iat.Unix()
	}
	if nbf, ok := token.NotBefore(); ok {
		payload["nbf"] = nbf.Unix()
	}
	if jti, ok := token.JwtID(); ok {
		payload["jti"] = jti
	}
}

func extractIdTokenClaims(token jwt.Token, payload map[string]interface{}) {
	extractClaimsByNames(token, payload, []string{"nonce"})
}

func extractAccessTokenClaims(token jwt.Token, payload map[string]interface{}) {
	extractClaimsByNames(token, payload, []string{"type", "isAdmin"})
}

func extractProfileClaims(token jwt.Token, payload map[string]interface{}) {
	profileClaims := []string{
		"given_name", "family_name", "name", "preferred_username",
		"picture", "email", "email_verified", "groups",
	}
	extractClaimsByNames(token, payload, profileClaims)
}

func extractCustomClaims(token jwt.Token, payload map[string]interface{}) {
	customClaims := []string{
		"scope", "client_id", "auth_time", "acr", "amr",
	}
	extractClaimsByNames(token, payload, customClaims)
}

func extractClaimsByNames(token jwt.Token, payload map[string]interface{}, claimNames []string) {
	for _, claim := range claimNames {
		if token.Has(claim) {
			var value interface{}
			if err := token.Get(claim, &value); err == nil {
				payload[claim] = value
			}
		}
	}
}
