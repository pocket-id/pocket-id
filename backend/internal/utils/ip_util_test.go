package utils

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

func TestIsLocalhostIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"127.255.255.255", true},
		{"::1", true},
		{"192.168.1.1", false},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		got := IsLocalhostIP(ip)
		assert.Equal(t, tt.expected, got)
	}
}

func TestIsPrivateLanIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"10.0.0.1", true},
		{"172.16.5.4", true},
		{"192.168.100.200", true},
		{"8.8.8.8", false},
		{"::1", false}, // IPv6 should return false
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		got := IsPrivateLanIP(ip)
		assert.Equal(t, tt.expected, got)
	}
}

func TestIsTailscaleIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"100.64.0.1", true},
		{"100.127.255.254", true},
		{"8.8.8.8", false},
		{"::1", false}, // IPv6 should return false
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)

		got := IsTailscaleIP(ip)
		assert.Equal(t, tt.expected, got)
	}
}

func TestIsLocalIPv6(t *testing.T) {
	// Save and restore env config
	origRanges := common.EnvConfig.LocalIPv6Ranges
	defer func() { common.EnvConfig.LocalIPv6Ranges = origRanges }()

	common.EnvConfig.LocalIPv6Ranges = "fd00::/8,fc00::/7"
	localIPv6Ranges = nil // reset
	loadLocalIPv6Ranges()

	tests := []struct {
		ip       string
		expected bool
	}{
		{"fd00::1", true},
		{"fc00::abcd", true},
		{"::1", false},         // loopback handled separately
		{"192.168.1.1", false}, // IPv4 should return false
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		got := IsLocalIPv6(ip)
		assert.Equal(t, tt.expected, got)
	}
}

func TestIsPrivateIP(t *testing.T) {
	// Save and restore env config
	origRanges := common.EnvConfig.LocalIPv6Ranges
	t.Cleanup(func() {
		common.EnvConfig.LocalIPv6Ranges = origRanges
	})

	common.EnvConfig.LocalIPv6Ranges = "fd00::/8"
	localIPv6Ranges = nil // reset
	loadLocalIPv6Ranges()

	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},             // localhost
		{"192.168.1.1", true},           // private LAN
		{"100.64.0.1", true},            // Tailscale
		{"fd00::1", true},               // local IPv6
		{"8.8.8.8", false},              // public IPv4
		{"2001:4860:4860::8888", false}, // public IPv6
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		got := IsPrivateIP(ip)
		assert.Equal(t, tt.expected, got)
	}
}

func TestListContainsIP(t *testing.T) {
	_, ipNet1, _ := net.ParseCIDR("10.0.0.0/8")
	_, ipNet2, _ := net.ParseCIDR("192.168.0.0/16")

	list := []*net.IPNet{ipNet1, ipNet2}

	tests := []struct {
		ip       string
		expected bool
	}{
		{"10.1.1.1", true},
		{"192.168.5.5", true},
		{"172.16.0.1", false},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		got := listContainsIP(list, ip)
		assert.Equal(t, tt.expected, got)
	}
}

func TestInit_LocalIPv6Ranges(t *testing.T) {
	// Save and restore env config
	origRanges := common.EnvConfig.LocalIPv6Ranges
	t.Cleanup(func() {
		common.EnvConfig.LocalIPv6Ranges = origRanges
	})

	common.EnvConfig.LocalIPv6Ranges = "fd00::/8, invalidCIDR ,fc00::/7"
	localIPv6Ranges = nil
	loadLocalIPv6Ranges()

	assert.Len(t, localIPv6Ranges, 2)
}

func TestIsURLPrivate(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	tests := []struct {
		name        string
		urlStr      string
		expectPriv  bool
		expectError bool
	}{
		{
			name:        "localhost by name",
			urlStr:      "http://localhost",
			expectPriv:  true,
			expectError: false,
		},
		{
			name:        "localhost with port",
			urlStr:      "http://localhost:8080",
			expectPriv:  true,
			expectError: false,
		},
		{
			name:        "127.0.0.1 IP",
			urlStr:      "http://127.0.0.1",
			expectPriv:  true,
			expectError: false,
		},
		{
			name:        "127.0.0.1 with port",
			urlStr:      "http://127.0.0.1:3000",
			expectPriv:  true,
			expectError: false,
		},
		{
			name:        "IPv6 loopback",
			urlStr:      "http://[::1]",
			expectPriv:  true,
			expectError: false,
		},
		{
			name:        "IPv6 loopback with port",
			urlStr:      "http://[::1]:8080",
			expectPriv:  true,
			expectError: false,
		},
		{
			name:        "private IP 10.x.x.x",
			urlStr:      "http://10.0.0.1",
			expectPriv:  true,
			expectError: false,
		},
		{
			name:        "private IP 192.168.x.x",
			urlStr:      "http://192.168.1.1",
			expectPriv:  true,
			expectError: false,
		},
		{
			name:        "private IP 172.16.x.x",
			urlStr:      "http://172.16.0.1",
			expectPriv:  true,
			expectError: false,
		},
		{
			name:        "Tailscale IP",
			urlStr:      "http://100.64.0.1",
			expectPriv:  true,
			expectError: false,
		},
		{
			name:        "public IP - Google DNS",
			urlStr:      "http://8.8.8.8",
			expectPriv:  false,
			expectError: false,
		},
		{
			name:        "public IP - Cloudflare DNS",
			urlStr:      "http://1.1.1.1",
			expectPriv:  false,
			expectError: false,
		},
		{
			name:        "invalid hostname",
			urlStr:      "http://this-should-not-resolve-ever-123456789.invalid",
			expectPriv:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.urlStr)
			require.NoError(t, err, "Failed to parse URL %s", tt.urlStr)

			isPriv, err := IsURLPrivate(ctx, u)

			if tt.expectError {
				require.Error(t, err, "IsURLPrivate(%s) expected error but got none", tt.urlStr)
			} else {
				require.NoError(t, err, "IsURLPrivate(%s) unexpected error", tt.urlStr)
				assert.Equal(t, tt.expectPriv, isPriv, "IsURLPrivate(%s)", tt.urlStr)
			}
		})
	}
}

func TestIsURLPrivate_WithDomainName(t *testing.T) {
	// Note: These tests rely on actual DNS resolution
	// They test real public domains to ensure they are not flagged as private
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	tests := []struct {
		name       string
		urlStr     string
		expectPriv bool
	}{
		{
			name:       "Google public domain",
			urlStr:     "https://www.google.com",
			expectPriv: false,
		},
		{
			name:       "GitHub public domain",
			urlStr:     "https://github.com",
			expectPriv: false,
		},
		{
			// localhost.localtest.me is a well-known domain that resolves to 127.0.0.1
			name:       "localhost.localtest.me resolves to 127.0.0.1",
			urlStr:     "http://localhost.localtest.me",
			expectPriv: true,
		},
		{
			// 10.0.0.1.nip.io resolves to 10.0.0.1 (private IP)
			name:       "nip.io domain resolving to private 10.x IP",
			urlStr:     "http://10.0.0.1.nip.io",
			expectPriv: true,
		},
		{
			// 192.168.1.1.nip.io resolves to 192.168.1.1 (private IP)
			name:       "nip.io domain resolving to private 192.168.x IP",
			urlStr:     "http://192.168.1.1.nip.io",
			expectPriv: true,
		},
		{
			// 127.0.0.1.nip.io resolves to 127.0.0.1 (localhost)
			name:       "nip.io domain resolving to localhost",
			urlStr:     "http://127.0.0.1.nip.io",
			expectPriv: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.urlStr)
			require.NoError(t, err, "Failed to parse URL %s", tt.urlStr)

			isPriv, err := IsURLPrivate(ctx, u)
			if err != nil {
				t.Skipf("DNS resolution failed for %s (network issue?): %v", tt.urlStr, err)
				return
			}

			assert.Equal(t, tt.expectPriv, isPriv, "IsURLPrivate(%s)", tt.urlStr)
		})
	}
}

func TestIsURLPrivate_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	u, err := url.Parse("http://example.com")
	require.NoError(t, err, "Failed to parse URL")

	_, err = IsURLPrivate(ctx, u)
	assert.Error(t, err, "IsURLPrivate with cancelled context expected error but got none")
}
