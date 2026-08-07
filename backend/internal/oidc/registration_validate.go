package oidc

import (
	"github.com/ory/fosite"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// ValidateRegistrationRedirectURIs checks a dynamic client registration request's
// redirect URIs against both the spec and this deployment's configuration.
//
// The protocol-level rules (at least one URI, absolute, no fragment, no
// active-content scheme) live in fosite, so they hold for every authorization
// server built on it. Layered on top is the Pocket ID-specific part: the operator
// allowlist, which is fail-closed, so an empty list denies every registration.
func ValidateRegistrationRedirectURIs(redirectURIs []string, allowlist []string) error {
	if err := fosite.ValidateRegistrationRedirectURIs(redirectURIs); err != nil {
		return apperror.InvalidRegistrationRedirectURI(firstOrEmpty(redirectURIs), fositeRedirectURIReason(err))
	}

	for _, uri := range redirectURIs {
		if !utils.MatchesAnyURLPattern(allowlist, uri) {
			return apperror.InvalidRegistrationRedirectURI(uri,
				"The redirect_uri '"+uri+"' is not permitted by the configured allowlist.")
		}
	}

	return nil
}

// fositeRedirectURIReason extracts the human-readable reason from a fosite
// validation error so it can be surfaced in the RFC 7591 error_description.
func fositeRedirectURIReason(err error) string {
	rfcErr := fosite.ErrorToRFC6749Error(err)
	if hint := rfcErr.HintField; hint != "" {
		return hint
	}
	return rfcErr.DescriptionField
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
