package iplocation

import (
	"context"
	"net"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
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
	case utils.IsLocalIPv6(ip):
		return "Internal Network", "LAN"
	case utils.IsTailscaleIP(ip):
		return "Internal Network", "Tailscale"
	case utils.IsPrivateIP(ip):
		return "Internal Network", "LAN"
	case utils.IsLocalhostIP(ip):
		return "Internal Network", "localhost"
	default:
		return "", ""
	}
}
