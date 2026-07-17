package bootstrap

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/pires/go-proxyproto"
)

const proxyProtocolReadHeaderTimeout = 5 * time.Second

func newProxyProtocolListener(listener net.Listener, trustedProxies []string) (net.Listener, error) {
	if len(trustedProxies) == 0 {
		return listener, nil
	}

	// PROXY protocol transports client metadata on TCP connections before any TLS handshake
	network := listener.Addr().Network()
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, fmt.Errorf("PROXY protocol requires a TCP listener, got %q", network)
	}

	// Require headers from configured network proxies and reject every other network peer
	trustedProxyPolicy, err := proxyproto.TrustProxyHeaderFromRanges(trustedProxies)
	if err != nil {
		return nil, fmt.Errorf("failed to configure PROXY protocol trusted proxies: %w", err)
	}

	// We need to add another policy for loopback connections because healthcheck requests are made from the same host
	// loopbackPolicy handles requests from trusted loopback like normal, but it also allows untrusted loopback connections to be accepted
	// untrusted loopback connections aren't allowed to send PROXY headers though
	loopbackPolicy, err := proxyproto.PolicyFromRanges(trustedProxies, proxyproto.USE, proxyproto.SKIP)
	if err != nil {
		return nil, fmt.Errorf("failed to configure PROXY protocol loopback proxies: %w", err)
	}

	// Create a policy which decides which of the two policies to use based on the upstream address
	policy := func(options proxyproto.ConnPolicyOptions) (proxyproto.Policy, error) {
		upstream, ok := options.Upstream.(*net.TCPAddr)
		if ok && upstream.IP.IsLoopback() {
			return loopbackPolicy(options)
		}

		return trustedProxyPolicy(options)
	}

	slog.Info("PROXY protocol enabled")
	return &proxyproto.Listener{
		Listener:          listener,
		ConnPolicy:        policy,
		ReadHeaderTimeout: proxyProtocolReadHeaderTimeout,
	}, nil
}
