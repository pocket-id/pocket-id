package oidc

import (
	"github.com/ory/fosite"
)

// Interface assertions
var (
	_ fosite.Client             = (*Client)(nil)
	_ fosite.ResponseModeClient = (*Client)(nil)
)
