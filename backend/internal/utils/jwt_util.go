package utils

import (
	"fmt"

	"github.com/lestrrat-go/jwx/v3/jwt"
)

func GetClaimsFromToken(token jwt.Token) (map[string]any, error) {
	claims := make(map[string]any)
	for _, key := range token.Keys() {
		var value any
		if err := token.Get(key, &value); err != nil {
			return nil, fmt.Errorf("failed to get claim %s: %w", key, err)
		}
		claims[key] = value
	}
	return claims, nil
}
