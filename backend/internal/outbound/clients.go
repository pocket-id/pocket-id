package outbound

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ory/fosite"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

const envVarPrefix = "OUTBOUND_ALLOWED_HOSTS_"

// Purpose identifies a feature that sends requests to URLs that admins or third parties control
type Purpose string

const (
	PurposeClientLogo        Purpose = "client_logo"
	PurposeClientMetadata    Purpose = "client_metadata"
	PurposeSCIM              Purpose = "scim"
	PurposeBackchannelLogout Purpose = "backchannel_logout"
	PurposeFederatedJWKS     Purpose = "federated_jwks"
	PurposeLDAPPicture       Purpose = "ldap_picture"
)

var AllPurposes = []Purpose{
	PurposeClientLogo,
	PurposeClientMetadata,
	PurposeSCIM,
	PurposeBackchannelLogout,
	PurposeFederatedJWKS,
	PurposeLDAPPicture,
}

func (p Purpose) EnvVar() string {
	return envVarPrefix + strings.ToUpper(string(p))
}

func (p Purpose) allowlist(config *common.EnvConfigSchema) string {
	switch p {
	case PurposeClientLogo:
		return config.OutboundAllowedHostsClientLogo
	case PurposeClientMetadata:
		return config.OutboundAllowedHostsClientMetadata
	case PurposeSCIM:
		return config.OutboundAllowedHostsSCIM
	case PurposeBackchannelLogout:
		return config.OutboundAllowedHostsBackchannelLogout
	case PurposeFederatedJWKS:
		return config.OutboundAllowedHostsFederatedJWKS
	case PurposeLDAPPicture:
		return config.OutboundAllowedHostsLDAPPicture
	default:
		return ""
	}
}

// Clients holds one guarded transport and client per purpose
type Clients struct {
	transports map[Purpose]http.RoundTripper
	clients    map[Purpose]*http.Client
}

// New builds the guarded clients from the OUTBOUND_ALLOWED_HOSTS_* environment variables
func New(config *common.EnvConfigSchema) (*Clients, error) {
	// LOCAL_IPV6_RANGES are blocked like private ranges and opened up by the "private" keyword
	localIPv6, err := ParseLocalIPv6Ranges(config.LocalIPv6Ranges)
	if err != nil {
		return nil, fmt.Errorf("invalid LOCAL_IPV6_RANGES: %w", err)
	}

	c := &Clients{
		transports: make(map[Purpose]http.RoundTripper, len(AllPurposes)),
		clients:    make(map[Purpose]*http.Client, len(AllPurposes)),
	}
	for _, purpose := range AllPurposes {
		policy, err := ParsePolicy(purpose.allowlist(config), localIPv6)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", purpose.EnvVar(), err)
		}

		// The operator configured the icon library, which lets a self-hosted mirror live on the local network
		if purpose == PurposeClientLogo && config.IconLibraryEnabled() {
			if u, err := url.Parse(config.IconLibraryURL); err == nil && u.Hostname() != "" {
				policy.allowedHosts = append(policy.allowedHosts, normalizeHostname(u.Hostname()))
			}
		}

		// Preserve Fosite's header limit before wrapping the transport so it applies to trusted hosts too
		var base *http.Transport
		if purpose == PurposeClientMetadata {
			base = http.DefaultTransport.(*http.Transport).Clone() //nolint:forcetypeassert // The default transport is always an *http.Transport
			base.MaxResponseHeaderBytes = fosite.DefaultCIMDMaxSize * 8
		}

		transport := NewTransport(base, purpose, policy)
		c.transports[purpose] = transport
		c.clients[purpose] = &http.Client{Transport: otelhttp.NewTransport(transport)}
	}

	return c, nil
}

// Client returns the guarded HTTP client for the purpose, instrumented with OpenTelemetry
func (c *Clients) Client(purpose Purpose) *http.Client {
	return c.clients[purpose]
}

// Transport returns the guarded transport for the purpose without instrumentation, for callers that add their own
func (c *Clients) Transport(purpose Purpose) http.RoundTripper {
	return c.transports[purpose]
}
