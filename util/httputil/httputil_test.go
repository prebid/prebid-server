package httputil

import (
	"crypto/tls"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSecure(t *testing.T) {
	testCases := []struct {
		description     string
		url             string
		xForwardedProto string
		tls             bool
		expectIsSecure  bool
	}{
		{
			description:    "HTTP",
			url:            "http://host.com",
			expectIsSecure: false,
		},
		{
			description:     "HTTPS - Forwarded Protocol",
			url:             "http://host.com",
			xForwardedProto: "https",
			expectIsSecure:  true,
		},
		{
			description:     "HTTPS - Forwarded Protocol - Case Insensitive",
			url:             "http://host.com",
			xForwardedProto: "HTTPS",
			expectIsSecure:  true,
		},
		{
			description:    "HTTPS - Protocol",
			url:            "https://host.com",
			expectIsSecure: true,
		},
		{
			description:    "HTTPS - Protocol - Case Insensitive",
			url:            "HTTPS://host.com",
			expectIsSecure: true,
		},
		{
			description:    "HTTPS - TLS",
			url:            "http://host.com",
			tls:            true,
			expectIsSecure: true,
		},
	}

	for _, test := range testCases {
		// Build Request
		request, err := http.NewRequest("GET", test.url, nil)
		if err != nil {
			t.Fatalf("Unable to create test http request. Err: %v", err)
		}
		if test.xForwardedProto != "" {
			request.Header.Add("X-Forwarded-Proto", test.xForwardedProto)
		}
		if test.tls {
			request.TLS = &tls.ConnectionState{}
		}

		result := IsSecure(request)
		assert.Equal(t, test.expectIsSecure, result, test.description)
	}
}

func TestGetIP(t *testing.T) {
	testCases := []struct {
		description     string
		xForwardedFor   string
		xRealIP         string
		remoteAddr      string
		privateNetworks []string
		expectedIP      string
	}{
		{
			description: "No Address Found",
			expectedIP:  "",
		},
		{
			description:   "IPv4 - XForwardedFor - Single Address - Not Private",
			xForwardedFor: "192.168.1.2",
			expectedIP:    "192.168.1.2",
		},
		{
			description:     "IPv4 - XForwardedFor - Single Address - Private",
			xForwardedFor:   "192.168.1.2",
			privateNetworks: []string{"192.168.1.2/16"},
			expectedIP:      "",
		},
		{
			description:   "IPv4 - XForwardedFor - First Address - Not Private",
			xForwardedFor: "192.168.1.2, 192.168.1.3",
			expectedIP:    "192.168.1.2",
		},
		{
			description:     "IPv4 - XForwardedFor - First Address- Private",
			xForwardedFor:   "192.168.1.2, 193.1.2.3",
			privateNetworks: []string{"192.168.1.2/16"},
			expectedIP:      "193.1.2.3",
		},
		{
			description: "IPv4 - XRealIP - Not Private",
			xRealIP:     "192.168.1.2",
			expectedIP:  "192.168.1.2",
		},
		{
			description:     "IPv4 - XRealIP - Private",
			xRealIP:         "192.168.1.2",
			privateNetworks: []string{"192.168.1.2/16"},
			expectedIP:      "",
		},
		{
			description: "IPv4 - RemoteAddress - Not Private",
			remoteAddr:  "192.168.1.2:8080",
			expectedIP:  "192.168.1.2",
		},
		{
			description:     "IPv4 - RemoteAddress - Private",
			remoteAddr:      "192.168.1.2:8080",
			privateNetworks: []string{"192.168.1.2/16"},
			expectedIP:      "",
		},
		// ipv6
		{
			description:   "Malformed - XForwardedFor",
			xForwardedFor: "malformed",
			expectedIP:    "",
		},
		{
			description: "Malformed - XRealIP",
			xRealIP:     "malformed",
			expectedIP:  "",
		},
		{
			description: "Malformed - RemoteAddress",
			remoteAddr:  "malformed",
			expectedIP:  "",
		},
	}

	for _, test := range testCases {
		// Build Request
		request, err := http.NewRequest("GET", "http://anyurl.com", nil)
		if err != nil {
			t.Fatalf("Unable to create test http request. Err: %v", err)
		}
		if test.xForwardedFor != "" {
			request.Header.Add("X-Forwarded-For", test.xForwardedFor)
		}
		if test.xRealIP != "" {
			request.Header.Add("X-Real-IP", test.xRealIP)
		}
		request.RemoteAddr = test.remoteAddr

		// Parse Private Networks
		privateNetworks := make([]*net.IPNet, 0, len(test.privateNetworks))
		for _, n := range test.privateNetworks {
			_, ipNet, err := net.ParseCIDR(n)
			if err != nil {
				t.Fatalf("%s: %v", test.description, err)
			}
			privateNetworks = append(privateNetworks, ipNet)
		}

		// Run Test
		result := GetIP(request, privateNetworks)

		// Assertions
		if test.expectedIP == "" {
			assert.Empty(t, result, test.description+":result")
		} else {
			assert.Equal(t, net.ParseIP(test.expectedIP), result, test.description+":result")
		}
	}
}
