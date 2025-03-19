package tokenutils

import (
	"github.com/lestrrat-go/jwx/v3/jwt"
)

const (
	// Boolean value used in access tokens for admin users
	// This may be omitted on non-admin tokens
	IsAdminClaim = "isAdmin"
)

// GetIsAdmin returns the value of the "isAdmin" claim in the token
func GetIsAdmin(token jwt.Token) (bool, error) {
	if !token.Has(IsAdminClaim) {
		return false, nil
	}
	var isAdmin bool
	err := token.Get(IsAdminClaim, &isAdmin)
	return isAdmin, err
}

// SetIsAdmin sets the "isAdmin" claim in the token
func SetIsAdmin(token jwt.Token, isAdmin bool) error {
	// Only set if true
	if !isAdmin {
		return nil
	}
	return token.Set(IsAdminClaim, isAdmin)
}
