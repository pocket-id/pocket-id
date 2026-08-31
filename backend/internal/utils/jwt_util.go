package utils

import (
	"fmt"

	"github.com/lestrrat-go/jwx/v4/jwt"
)

func GetClaimsFromToken(token jwt.Token) (map[string]any, error) {
	keys := token.Keys()
	claims := make(map[string]any, len(keys))
	for _, key := range keys {
		value, err := jwt.Get[any](token, key)
		if err != nil {
			return nil, fmt.Errorf("failed to get claim %s: %w", key, err)
		}
		claims[key] = value
	}
	return claims, nil
}
