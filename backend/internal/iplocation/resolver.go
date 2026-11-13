package iplocation

import (
	"context"
	"net"
)

type Resolver interface {
	GetLocationByIP(ctx context.Context, ipAddress string) (country string, city string, err error)
}

// LocalLocation returns the labels shared by all providers for non-public addresses
func LocalLocation(ip net.IP) (country string, city string) {
	if ip == nil {
		return "", ""
	}

	switch {
	case isLocalIPv6(ip):
		return "Internal Network", "LAN"
	case isTailscaleIP(ip):
		return "Internal Network", "Tailscale"
	case isPrivateIP(ip):
		return "Internal Network", "LAN"
	case isLocalhostIP(ip):
		return "Internal Network", "localhost"
	default:
		return "", ""
	}
}
