package service

import (
	"net"
	"net/http"
	"testing"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

func TestGeoLiteService_IPv6LocalRanges(t *testing.T) {
	tests := []struct {
		name            string
		localRanges     string
		testIP          string
		expectedCountry string
		expectedCity    string
		expectError     bool
	}{
		{
			name:            "IPv6 in local range",
			localRanges:     "2001:0db8:abcd:000::/56,2001:0db8:abcd:001::/56",
			testIP:          "2001:0db8:abcd:000::1",
			expectedCountry: "Internal Network",
			expectedCity:    "LAN",
			expectError:     false,
		},
		{
			name:        "IPv6 not in local range",
			localRanges: "2001:0db8:abcd:000::/56",
			testIP:      "2001:0db8:ffff:000::1",
			expectError: true,
		},
		{
			name:            "Multiple ranges - second range match",
			localRanges:     "2001:0db8:abcd:000::/56,2001:0db8:abcd:001::/56",
			testIP:          "2001:0db8:abcd:001::1",
			expectedCountry: "Internal Network",
			expectedCity:    "LAN",
			expectError:     false,
		},
		{
			name:        "Empty local ranges",
			localRanges: "",
			testIP:      "2001:0db8:abcd:000::1",
			expectError: true,
		},
		{
			name:            "IPv4 private address still works",
			localRanges:     "2001:0db8:abcd:000::/56",
			testIP:          "192.168.1.1",
			expectedCountry: "Internal Network",
			expectedCity:    "LAN",
			expectError:     false,
		},
		{
			name:            "IPv6 loopback",
			localRanges:     "2001:0db8:abcd:000::/56",
			testIP:          "::1",
			expectedCountry: "Internal Network",
			expectedCity:    "localhost",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalConfig := common.EnvConfig.LocalIPv6Ranges
			common.EnvConfig.LocalIPv6Ranges = tt.localRanges
			defer func() {
				common.EnvConfig.LocalIPv6Ranges = originalConfig
			}()

			service := NewGeoLiteService(&http.Client{})

			country, city, err := service.GetLocationByIP(tt.testIP)

			if tt.expectError {
				if err == nil && country != "Internal Network" {
					t.Errorf("Expected error or internal network classification for external IP")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for local IP, got: %v", err)
				}
				if country != tt.expectedCountry {
					t.Errorf("Expected country %s, got %s", tt.expectedCountry, country)
				}
				if city != tt.expectedCity {
					t.Errorf("Expected city %s, got %s", tt.expectedCity, city)
				}
			}
		})
	}
}

func TestGeoLiteService_isLocalIPv6(t *testing.T) {
	tests := []struct {
		name        string
		localRanges string
		testIP      string
		expected    bool
	}{
		{
			name:        "Valid IPv6 in range",
			localRanges: "2001:0db8:abcd:000::/56",
			testIP:      "2001:0db8:abcd:000::1",
			expected:    true,
		},
		{
			name:        "Valid IPv6 not in range",
			localRanges: "2001:0db8:abcd:000::/56",
			testIP:      "2001:0db8:ffff:000::1",
			expected:    false,
		},
		{
			name:        "IPv4 address should return false",
			localRanges: "2001:0db8:abcd:000::/56",
			testIP:      "192.168.1.1",
			expected:    false,
		},
		{
			name:        "No ranges configured",
			localRanges: "",
			testIP:      "2001:0db8:abcd:000::1",
			expected:    false,
		},
		{
			name:        "Edge of range",
			localRanges: "2001:0db8:abcd:000::/56",
			testIP:      "2001:0db8:abcd:00ff:ffff:ffff:ffff:ffff",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalConfig := common.EnvConfig.LocalIPv6Ranges
			common.EnvConfig.LocalIPv6Ranges = tt.localRanges
			defer func() {
				common.EnvConfig.LocalIPv6Ranges = originalConfig
			}()

			service := NewGeoLiteService(&http.Client{})
			ip := net.ParseIP(tt.testIP)
			if ip == nil {
				t.Fatalf("Invalid test IP: %s", tt.testIP)
			}

			result := service.isLocalIPv6(ip)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for IP %s", tt.expected, result, tt.testIP)
			}
		})
	}
}

func TestGeoLiteService_initializeIPv6LocalRanges(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		expectError bool
		expectCount int
	}{
		{
			name:        "Valid IPv6 ranges",
			envValue:    "2001:0db8:abcd:000::/56,2001:0db8:abcd:001::/56",
			expectError: false,
			expectCount: 2,
		},
		{
			name:        "Empty environment variable",
			envValue:    "",
			expectError: false,
			expectCount: 0,
		},
		{
			name:        "Invalid CIDR notation",
			envValue:    "2001:0db8:abcd:000::/999",
			expectError: true,
			expectCount: 0,
		},
		{
			name:        "IPv4 range in IPv6 env var",
			envValue:    "192.168.1.0/24",
			expectError: true,
			expectCount: 0,
		},
		{
			name:        "Mixed valid and invalid ranges",
			envValue:    "2001:0db8:abcd:000::/56,invalid-range",
			expectError: true,
			expectCount: 0,
		},
		{
			name:        "Whitespace handling",
			envValue:    " 2001:0db8:abcd:000::/56 , 2001:0db8:abcd:001::/56 ",
			expectError: false,
			expectCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalConfig := common.EnvConfig.LocalIPv6Ranges
			common.EnvConfig.LocalIPv6Ranges = tt.envValue
			defer func() {
				common.EnvConfig.LocalIPv6Ranges = originalConfig
			}()

			service := &GeoLiteService{
				httpClient: &http.Client{},
			}

			err := service.initializeIPv6LocalRanges()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			rangeCount := len(service.localIPv6Ranges)

			if rangeCount != tt.expectCount {
				t.Errorf("Expected %d ranges, got %d", tt.expectCount, rangeCount)
			}
		})
	}
}
