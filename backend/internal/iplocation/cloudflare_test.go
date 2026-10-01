package iplocation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloudflareLocation(t *testing.T) {
	resolver := NewCloudflareResolver()

	tests := []struct {
		name        string
		countryCode string
		city        string
		country     string
		wantCity    string
	}{
		{name: "country and city", countryCode: "CH", city: "Zürich", country: "Switzerland", wantCity: "Zürich"},
		{name: "country only", countryCode: "US", country: "United States"},
		{name: "city only", city: "London", wantCity: "London"},
		{name: "normalized headers", countryCode: " gb ", city: " London ", country: "United Kingdom", wantCity: "London"},
		{name: "missing headers"},
		{name: "unknown country", countryCode: "XX"},
		{name: "Tor country", countryCode: "T1"},
		{name: "unspecified country", countryCode: "ZZ"},
		{name: "invalid country", countryCode: "invalid"},
		{name: "continent is not a country", countryCode: "EU"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithCloudflareLocation(t.Context(), "81.2.69.142", tt.countryCode, tt.city)
			country, city, err := resolver.GetLocationByIP(ctx, "81.2.69.142")
			require.NoError(t, err)
			require.Equal(t, tt.country, country)
			require.Equal(t, tt.wantCity, city)
		})
	}
}

func TestCloudflareLocationOnlyDescribesRequestClient(t *testing.T) {
	resolver := NewCloudflareResolver()
	ctx := WithCloudflareLocation(t.Context(), "81.2.69.142", "GB", "London")

	// Looking up another device must never reuse the approving device's headers
	country, city, err := resolver.GetLocationByIP(ctx, "216.160.83.56")
	require.NoError(t, err)
	require.Empty(t, country)
	require.Empty(t, city)

	// Non-request contexts have no header location to resolve
	country, city, err = resolver.GetLocationByIP(t.Context(), "81.2.69.142")
	require.NoError(t, err)
	require.Empty(t, country)
	require.Empty(t, city)
}

func TestCloudflareLocationPreservesIPHandling(t *testing.T) {
	resolver := NewCloudflareResolver()

	for _, ip := range []string{"192.168.1.20", "100.101.102.103", "fd00::1", "127.0.0.1"} {
		t.Run(ip, func(t *testing.T) {
			ctx := WithCloudflareLocation(t.Context(), ip, "CH", "Zürich")
			country, city, err := resolver.GetLocationByIP(ctx, ip)
			require.NoError(t, err)
			require.Equal(t, "Internal Network", country)
			if ip == "100.101.102.103" {
				require.Equal(t, "Tailscale", city)
			} else {
				require.Equal(t, "LAN", city)
			}
		})
	}

	ctx := WithCloudflareLocation(t.Context(), "2001:218::1", "JP", "Tokyo")
	country, city, err := resolver.GetLocationByIP(ctx, "2001:218::1")
	require.NoError(t, err)
	require.Equal(t, "Japan", country)
	require.Equal(t, "Tokyo", city)

	_, _, err = resolver.GetLocationByIP(ctx, "not-an-ip")
	require.ErrorContains(t, err, "failed to parse IP address")

	country, city, err = resolver.GetLocationByIP(ctx, "")
	require.NoError(t, err)
	require.Empty(t, country)
	require.Empty(t, city)
}
