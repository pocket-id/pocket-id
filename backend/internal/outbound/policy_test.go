package outbound

import (
	"net/netip"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePolicy(t *testing.T) {
	t.Run("accepts every entry type", func(t *testing.T) {
		_, err := ParsePolicy(" private, LOOPBACK ,none,192.168.1.5,fd00::/8,10.0.0.0/8,scim-app,*.internal,my_host.example.com", nil)
		require.NoError(t, err)
	})

	invalid := []string{
		"10.0.0.0/33",
		"http://example.com",
		"example.com:443",
		"*.",
		"exa mple.com",
		".example.com",
		"example..com",
	}
	for _, value := range invalid {
		t.Run("rejects "+value, func(t *testing.T) {
			_, err := ParsePolicy(value, nil)
			require.Error(t, err)
		})
	}
}

func TestPolicyAllowAddr(t *testing.T) {
	localIPv6 := []netip.Prefix{netip.MustParsePrefix("2a01:abcd::/32")}

	mustPolicy := func(value string) Policy {
		policy, err := ParsePolicy(value, localIPv6)
		require.NoError(t, err)
		return policy
	}
	strict := mustPolicy("")
	lan := mustPolicy("private,loopback")
	explicit := mustPolicy("169.254.169.254,fe80::/10,64:ff9b::/96")

	tests := []struct {
		addr     string
		strict   bool
		lan      bool
		explicit bool
	}{
		{addr: "8.8.8.8", strict: true, lan: true, explicit: true},
		{addr: "2606:4700:4700::1111", strict: true, lan: true, explicit: true},
		{addr: "127.0.0.1", strict: false, lan: true, explicit: false},
		{addr: "::1", strict: false, lan: true, explicit: false},
		{addr: "10.1.2.3", strict: false, lan: true, explicit: false},
		{addr: "172.20.0.5", strict: false, lan: true, explicit: false},
		{addr: "192.168.1.10", strict: false, lan: true, explicit: false},
		{addr: "100.100.1.1", strict: false, lan: true, explicit: false},
		{addr: "fd12::1", strict: false, lan: true, explicit: false},
		{addr: "2a01:abcd::1", strict: false, lan: true, explicit: false},
		{addr: "::ffff:10.0.0.1", strict: false, lan: true, explicit: false},
		// Cloud metadata and other link-local addresses need an explicit entry even when the LAN is allowed
		{addr: "169.254.169.254", strict: false, lan: false, explicit: true},
		{addr: "fe80::1%eth0", strict: false, lan: false, explicit: true},
		// NAT64 can embed a private IPv4 address, so it is never covered by the keywords
		{addr: "64:ff9b::a00:1", strict: false, lan: false, explicit: true},
		{addr: "0.0.0.0", strict: false, lan: false, explicit: false},
		{addr: "::", strict: false, lan: false, explicit: false},
		{addr: "224.0.0.1", strict: false, lan: false, explicit: false},
		{addr: "255.255.255.255", strict: false, lan: false, explicit: false},
		{addr: "198.18.0.1", strict: false, lan: false, explicit: false},
		{addr: "2002:a00:1::1", strict: false, lan: false, explicit: false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			addr := netip.MustParseAddr(tt.addr)
			assert.Equal(t, tt.strict, strict.AllowAddr(addr), "strict policy")
			assert.Equal(t, tt.lan, lan.AllowAddr(addr), "private,loopback policy")
			assert.Equal(t, tt.explicit, explicit.AllowAddr(addr), "explicit policy")
		})
	}
}

func TestPolicyAllowURL(t *testing.T) {
	policy, err := ParsePolicy("scim-app,*.internal", nil)
	require.NoError(t, err)

	tests := []struct {
		url     string
		allowed bool
	}{
		{url: "http://scim-app:8080/scim/v2", allowed: true},
		{url: "http://SCIM-APP./scim/v2", allowed: true},
		{url: "https://idp.internal/jwks", allowed: true},
		{url: "https://a.b.internal/jwks", allowed: true},
		{url: "https://internal/jwks", allowed: false},
		{url: "https://evilinternal/jwks", allowed: false},
		{url: "https://scim-app.example.com/", allowed: false},
		{url: "http://10.0.0.1/", allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			u, err := url.Parse(tt.url)
			require.NoError(t, err)
			assert.Equal(t, tt.allowed, policy.AllowURL(u))
		})
	}
}

func TestParseLocalIPv6Ranges(t *testing.T) {
	prefixes, err := ParseLocalIPv6Ranges("2a01:abcd::/32, 2001:db8:1::/48")
	require.NoError(t, err)
	assert.Len(t, prefixes, 2)

	_, err = ParseLocalIPv6Ranges("10.0.0.0/8")
	require.Error(t, err)
}
