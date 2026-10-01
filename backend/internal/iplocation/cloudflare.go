package iplocation

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

type cloudflareLocationKey struct{}

type cloudflareLocation struct {
	ipAddress string
	country   string
	city      string
}

// WithCloudflareLocation captures the client's location while the originating HTTP request is available
// Callers must only use this when the deployment explicitly trusts Cloudflare's location headers
func WithCloudflareLocation(ctx context.Context, ipAddress, countryCode, city string) context.Context {
	location := cloudflareLocation{ipAddress: ipAddress, city: strings.TrimSpace(city)}

	// Cloudflare supplies an ISO country code, while audit logs and notification emails display English country names
	code := strings.ToUpper(strings.TrimSpace(countryCode))
	if len(code) == 2 {
		region, err := language.ParseRegion(code)
		if err == nil && region.IsCountry() {
			location.country = display.English.Regions().Name(region)
		}
	}

	return context.WithValue(ctx, cloudflareLocationKey{}, location)
}

// CloudflareResolver resolves locations using trusted Cloudflare request headers
type CloudflareResolver struct{}

func NewCloudflareResolver() *CloudflareResolver {
	return &CloudflareResolver{}
}

// GetLocationByIP returns the country and city for the client making the current request
func (r *CloudflareResolver) GetLocationByIP(ctx context.Context, ipAddress string) (country string, city string, err error) {
	if ipAddress == "" {
		return "", "", nil
	}

	// Keep local network labels consistent across location providers
	if country, city := LocalLocation(net.ParseIP(ipAddress)); country != "" {
		return country, city, nil
	}
	if _, err := netip.ParseAddr(ipAddress); err != nil {
		return "", "", fmt.Errorf("failed to parse IP address: %w", err)
	}

	// Headers describe only the request's client, never another IP being inspected
	location, ok := ctx.Value(cloudflareLocationKey{}).(cloudflareLocation)
	if !ok || location.ipAddress != ipAddress {
		return "", "", nil
	}
	return location.country, location.city, nil
}
