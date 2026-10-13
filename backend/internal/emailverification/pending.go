package emailverification

import (
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// IsPending reports whether the user must verify their email before they can use their account
func IsPending(cfg *appconfig.AppConfigModel, user *model.User) bool {
	return cfg.EmailVerification.String() == appconfig.EmailVerificationRequired && CanBePending(user)
}

// CanBePending reports whether the user would have to verify their email if verification were required
func CanBePending(user *model.User) bool {
	return !user.IsAdmin && !user.EmailVerified
}

// DefersEmailChange reports whether the user's own email change has to be confirmed before it replaces their current email
func DefersEmailChange(cfg *appconfig.AppConfigModel, user *model.User) bool {
	return cfg.EmailVerification.String() == appconfig.EmailVerificationRequired && !user.IsAdmin && user.EmailVerified && user.Email != nil
}
