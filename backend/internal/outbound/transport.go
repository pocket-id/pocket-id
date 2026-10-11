package outbound

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

type BlockedError struct {
	Purpose Purpose
	Addr    netip.Addr
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("connection to %s is blocked because it is a private or reserved address; allow it with %s", e.Addr, e.Purpose.EnvVar())
}

// guardedTransport routes each request through a dialer that refuses disallowed addresses
type guardedTransport struct {
	purpose Purpose
	policy  Policy
	guarded http.RoundTripper
	trusted http.RoundTripper
}

// NewTransport returns a transport that only connects to addresses the policy allows
func NewTransport(base *http.Transport, purpose Purpose, policy Policy) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport.(*http.Transport) //nolint:forcetypeassert // The default transport is always an *http.Transport
	}

	// Guard the dialer itself so the check sees the IP the connection actually goes to
	guarded := base.Clone()
	// A proxy would hide the final destination from the dialer and bypass the check
	guarded.Proxy = nil
	guarded.DialTLSContext = nil
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			addrPort, err := netip.ParseAddrPort(address)
			if err != nil {
				return fmt.Errorf("could not parse dialed address %q: %w", address, err)
			}
			if !policy.AllowAddr(addrPort.Addr()) {
				return &BlockedError{Purpose: purpose, Addr: addrPort.Addr().WithZone("").Unmap()}
			}
			return nil
		},
	}
	guarded.DialContext = dialer.DialContext

	t := &guardedTransport{
		purpose: purpose,
		policy:  policy,
		guarded: guarded,
	}

	// Hosts allowed by name skip the address check, so they get the base transport unchanged
	if policy.hasTrustedHosts() {
		t.trusted = base.Clone()
	}

	return t
}

func (t *guardedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return nil, fmt.Errorf("unsupported URL scheme %q", req.URL.Scheme)
	}

	// Pick the transport per request, which also covers every redirect hop
	transport := t.guarded
	if t.trusted != nil && t.policy.AllowURL(req.URL) {
		transport = t.trusted
	}

	return transport.RoundTrip(req)
}
