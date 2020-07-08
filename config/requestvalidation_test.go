package config

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	ipv4Mask16 := net.CIDRMask(16, 32)
	ipv4Mask24 := net.CIDRMask(24, 32)

	ipv6Mask16 := net.CIDRMask(16, 128)
	ipv6Mask32 := net.CIDRMask(32, 128)

	testCases := []struct {
		description  string
		ipv4         []string
		ipv4Expected []net.IPNet
		ipv6         []string
		ipv6Expected []net.IPNet
		expectedErr  string
	}{
		{
			description:  "Empty",
			ipv4:         []string{},
			ipv4Expected: []net.IPNet{},
			ipv6:         []string{},
			ipv6Expected: []net.IPNet{},
		},
		{
			description:  "One",
			ipv4:         []string{"1.1.1.1/24"},
			ipv4Expected: []net.IPNet{{IP: net.IP{1, 1, 1, 0}, Mask: ipv4Mask24}},
			ipv6:         []string{"1111:2222::/16"},
			ipv6Expected: []net.IPNet{{IP: net.ParseIP("1111::"), Mask: ipv6Mask16}},
		},
		{
			description:  "One - Ignore Whitespace",
			ipv4:         []string{"   1.1.1.1/24 "},
			ipv4Expected: []net.IPNet{{IP: net.IP{1, 1, 1, 0}, Mask: ipv4Mask24}},
			ipv6:         []string{"   1111:2222::/16 "},
			ipv6Expected: []net.IPNet{{IP: net.ParseIP("1111::"), Mask: ipv6Mask16}},
		},
		{
			description:  "Many",
			ipv4:         []string{"1.1.1.1/24", "2.2.2.2/16"},
			ipv4Expected: []net.IPNet{{IP: net.IP{1, 1, 1, 0}, Mask: ipv4Mask24}, {IP: net.IP{2, 2, 0, 0}, Mask: ipv4Mask16}},
			ipv6:         []string{"1111:2222::/16", "1111:2222:3333::/32"},
			ipv6Expected: []net.IPNet{{IP: net.ParseIP("1111::"), Mask: ipv6Mask16}, {IP: net.ParseIP("1111:2222::"), Mask: ipv6Mask32}},
		},
		{
			description: "Malformed - IPv4 - One",
			ipv4:        []string{"malformed1"},
			ipv6:        []string{},
			expectedErr: "Invalid private IPv4 network: 'malformed1'",
		},
		{
			description: "Malformed - IPv4 - Many",
			ipv4:        []string{"malformed1", "malformed2"},
			ipv6:        []string{},
			expectedErr: "Invalid private IPv4 network: 'malformed1','malformed2'",
		},
		{
			description: "Malformed - IPv6 - One",
			ipv4:        []string{},
			ipv6:        []string{"malformed2"},
			expectedErr: "Invalid private IPv6 network: 'malformed2'",
		},
		{
			description: "Malformed - IPv6 - Many",
			ipv4:        []string{},
			ipv6:        []string{"malformed1", "malformed2"},
			expectedErr: "Invalid private IPv6 network: 'malformed1','malformed2'",
		},
		{
			description: "Malformed - Mixed",
			ipv4:        []string{"malformed1"},
			ipv6:        []string{"malformed2"},
			expectedErr: "Invalid private IPv4 network: 'malformed1'",
		},
		{
			description: "Malformed - IPv4 - Ignore Whitespace",
			ipv4:        []string{"   malformed1 "},
			ipv6:        []string{},
			expectedErr: "Invalid private IPv4 network: 'malformed1'",
		},
		{
			description: "Malformed - IPv6 - Ignore Whitespace",
			ipv4:        []string{},
			ipv6:        []string{"   malformed2 "},
			expectedErr: "Invalid private IPv6 network: 'malformed2'",
		},
		{
			description: "Malformed - IPv4 - Missing Network Mask",
			ipv4:        []string{"1.1.1.1"},
			ipv6:        []string{},
			expectedErr: "Invalid private IPv4 network: '1.1.1.1'",
		},
		{
			description: "Malformed - IPv6 - Missing Network Mask",
			ipv4:        []string{},
			ipv6:        []string{"1111::"},
			expectedErr: "Invalid private IPv6 network: '1111::'",
		},
		{
			description: "Malformed - IPv4 - Wrong IP Version",
			ipv4:        []string{"1111::/16"},
			ipv6:        []string{},
			expectedErr: "Invalid private IPv4 network: '1111::/16'",
		},
		{
			description: "Malformed - IPv6 - Wrong IP Version",
			ipv4:        []string{},
			ipv6:        []string{"1.1.1.1/16"},
			expectedErr: "Invalid private IPv6 network: '1.1.1.1/16'",
		},
		{
			description: "Malformed - IPv6 Mapped IPv4",
			ipv4:        []string{"::FFFF:1.1.1.1"},
			ipv6:        []string{},
			expectedErr: "Invalid private IPv4 network: '::FFFF:1.1.1.1'",
		},
	}

	for _, test := range testCases {
		requestValidation := &RequestValidation{
			IPv4PrivateNetworks: test.ipv4,
			IPv6PrivateNetworks: test.ipv6,
		}

		err := requestValidation.Parse()

		if test.expectedErr == "" {
			assert.NoError(t, err, test.description+":err")
		} else {
			assert.Error(t, err, test.description+":err")
			assert.Equal(t, test.expectedErr, err.Error(), test.description+":err_msg")
		}

		assert.ElementsMatch(t, requestValidation.IPv4PrivateNetworksParsed, test.ipv4Expected, test.description+":ipv4")
		assert.ElementsMatch(t, requestValidation.IPv6PrivateNetworksParsed, test.ipv6Expected, test.description+":ipv6")
	}
}
