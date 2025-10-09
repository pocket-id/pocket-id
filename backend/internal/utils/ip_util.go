package utils

import (
	"net"
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

var localIPv6Ranges []*net.IPNet

var localhostIPNets = []*net.IPNet{
	{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)}, // 127.0.0.0/8
	{IP: net.IPv6loopback, Mask: net.CIDRMask(128, 128)},    // ::1/128
}

var privateLanIPNets = []*net.IPNet{
	{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},     // 10.0.0.0/8
	{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},  // 172.16.0.0/12
	{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)}, // 192.168.0.0/16
}

var tailscaleIPNets = []*net.IPNet{
	{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}, // 100.64.0.0/10
}

func IsLocalIPv6(ip net.IP) bool {
	if ip.To4() != nil {
		return false
	}

	return listContainsIP(localIPv6Ranges, ip)
}

func IsLocalhostIP(ip net.IP) bool {
	return listContainsIP(localhostIPNets, ip)
}

func IsPrivateLanIP(ip net.IP) bool {
	if ip.To4() == nil {
		return false
	}

	return listContainsIP(privateLanIPNets, ip)
}

func IsTailscaleIP(ip net.IP) bool {
	if ip.To4() == nil {
		return false
	}

	return listContainsIP(tailscaleIPNets, ip)
}

func IsPrivateIP(ip net.IP) bool {
	return IsLocalhostIP(ip) || IsPrivateLanIP(ip) || IsTailscaleIP(ip) || IsLocalIPv6(ip)
}

func listContainsIP(ipNets []*net.IPNet, ip net.IP) bool {
	for _, ipNet := range ipNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

func loadLocalIPv6Ranges() {
	localIPv6Ranges = nil
	ranges := strings.Split(common.EnvConfig.LocalIPv6Ranges, ",")

	for _, rangeStr := range ranges {
		rangeStr = strings.TrimSpace(rangeStr)
		if rangeStr == "" {
			continue
		}

		_, ipNet, err := net.ParseCIDR(rangeStr)
		if err == nil {
			localIPv6Ranges = append(localIPv6Ranges, ipNet)
		}
	}
}

func init() {
	loadLocalIPv6Ranges()
}
