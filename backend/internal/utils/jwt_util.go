package utils

import (
	"fmt"

	"github.com/lestrrat-go/jwx/v3/jwt"
)

func GetClaimsFromToken(token jwt.Token) (map[string]any, error) {
	keys := token.Keys()
	claims := make(map[string]any, len(keys))
	for _, key := range keys {
		var value any
		if err := token.Get(key, &value); err != nil {
			return nil, fmt.Errorf("failed to get claim %s: %w", key, err)
		}
		claims[key] = value
	}
	return claims, nil
}
