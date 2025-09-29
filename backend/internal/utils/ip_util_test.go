package utils

import (
	"net"
	"testing"

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
		if got := IsLocalhostIP(ip); got != tt.expected {
			t.Errorf("IsLocalhostIP(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
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
		if got := IsPrivateLanIP(ip); got != tt.expected {
			t.Errorf("IsPrivateLanIP(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
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
		if got := IsTailscaleIP(ip); got != tt.expected {
			t.Errorf("IsTailscaleIP(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
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
		if got := IsLocalIPv6(ip); got != tt.expected {
			t.Errorf("IsLocalIPv6(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestIsPrivateIP(t *testing.T) {
	// Save and restore env config
	origRanges := common.EnvConfig.LocalIPv6Ranges
	defer func() { common.EnvConfig.LocalIPv6Ranges = origRanges }()

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
		if got := IsPrivateIP(ip); got != tt.expected {
			t.Errorf("IsPrivateIP(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
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
		if got := listContainsIP(list, ip); got != tt.expected {
			t.Errorf("listContainsIP(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestInit_LocalIPv6Ranges(t *testing.T) {
	// Save and restore env config
	origRanges := common.EnvConfig.LocalIPv6Ranges
	defer func() { common.EnvConfig.LocalIPv6Ranges = origRanges }()

	common.EnvConfig.LocalIPv6Ranges = "fd00::/8, invalidCIDR ,fc00::/7"
	localIPv6Ranges = nil
	loadLocalIPv6Ranges()

	if len(localIPv6Ranges) != 2 {
		t.Errorf("expected 2 valid IPv6 ranges, got %d", len(localIPv6Ranges))
	}
}
