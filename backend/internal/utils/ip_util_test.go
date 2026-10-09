package utils

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

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
		{"127.0.0.1", true},              // localhost
		{"192.168.1.1", true},            // private LAN
		{"100.64.0.1", true},             // Tailscale
		{"169.254.169.254", true},        // IPv4 link-local
		{"169.254.170.2", true},          // IPv4 link-local
		{"::ffff:169.254.169.254", true}, // IPv4-mapped link-local
		{"fe80::1", true},                // IPv6 link-local
		{"ff02::1", true},                // IPv6 link-local multicast
		{"0.0.0.0", true},                // IPv4 unspecified
		{"::", true},                     // IPv6 unspecified
		{"fd00::1", true},                // private IPv6
		{"8.8.8.8", false},               // public IPv4
		{"2001:4860:4860::8888", false},  // public IPv6
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
