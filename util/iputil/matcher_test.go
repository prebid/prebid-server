package iputil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicNetworkIPMatcher(t *testing.T) {
	ipv4Network1 := net.IPNet{IP: net.ParseIP("1.0.0.0"), Mask: net.IPMask{255, 0, 0, 0}}
	ipv4Network2 := net.IPNet{IP: net.ParseIP("2.0.0.0"), Mask: net.IPMask{255, 0, 0, 0}}

	ipv6Network1 := net.IPNet{IP: net.ParseIP("0300::"), Mask: net.IPMask{255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	ipv6Network2 := net.IPNet{IP: net.ParseIP("0400::"), Mask: net.IPMask{255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}

	testCases := []struct {
		description         string
		ip                  net.IP
		ver                 IPVersion
		ipv4PrivateNetworks []net.IPNet
		ipv6PrivateNetworks []net.IPNet
		expected            bool
	}{
		{
			description:         "IPv4 - Public - None",
			ip:                  net.ParseIP("1.1.1.1"),
			ver:                 IPv4,
			ipv4PrivateNetworks: []net.IPNet{},
			ipv6PrivateNetworks: []net.IPNet{},
			expected:            true,
		},
		{
			description:         "IPv4 - Public - One",
			ip:                  net.ParseIP("2.2.2.2"),
			ver:                 IPv4,
			ipv4PrivateNetworks: []net.IPNet{ipv4Network1},
			ipv6PrivateNetworks: []net.IPNet{},
			expected:            true,
		},
		{
			description:         "IPv4 - Public - Many",
			ip:                  net.ParseIP("3.3.3.3"),
			ver:                 IPv4,
			ipv4PrivateNetworks: []net.IPNet{ipv4Network1, ipv4Network2},
			ipv6PrivateNetworks: []net.IPNet{},
			expected:            true,
		},
		{
			description:         "IPv4 - Private - One",
			ip:                  net.ParseIP("1.1.1.1"),
			ver:                 IPv4,
			ipv4PrivateNetworks: []net.IPNet{ipv4Network1},
			ipv6PrivateNetworks: []net.IPNet{},
			expected:            false,
		},
		{
			description:         "IPv4 - Private - Many",
			ip:                  net.ParseIP("2.2.2.2"),
			ver:                 IPv4,
			ipv4PrivateNetworks: []net.IPNet{ipv4Network1, ipv4Network2},
			ipv6PrivateNetworks: []net.IPNet{},
			expected:            false,
		},
		{
			description:         "IPv6 - Public - None",
			ip:                  net.ParseIP("0303::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{},
			ipv6PrivateNetworks: []net.IPNet{},
			expected:            true,
		},
		{
			description:         "IPv6 - Public - One",
			ip:                  net.ParseIP("0404::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1},
			expected:            true,
		},
		{
			description:         "IPv6 - Public - Many",
			ip:                  net.ParseIP("0505::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1, ipv6Network2},
			expected:            true,
		},
		{
			description:         "IPv6 - Private - One",
			ip:                  net.ParseIP("0303::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1},
			expected:            false,
		},
		{
			description:         "IPv6 - Private - Many",
			ip:                  net.ParseIP("0404::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1, ipv6Network2},
			expected:            false,
		},
		{
			description:         "Mixed - Unknown",
			ip:                  nil,
			ver:                 IPvUnknown,
			ipv4PrivateNetworks: []net.IPNet{ipv4Network1, ipv4Network1},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1, ipv6Network2},
			expected:            false,
		},
		{
			description:         "Mixed - Public - IPv4",
			ip:                  net.ParseIP("3.3.3.3"),
			ver:                 IPv4,
			ipv4PrivateNetworks: []net.IPNet{ipv4Network1, ipv4Network1},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1, ipv6Network2},
			expected:            true,
		},
		{
			description:         "Mixed - Public - IPv6",
			ip:                  net.ParseIP("0505::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{ipv4Network1, ipv4Network1},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1, ipv6Network2},
			expected:            true,
		},
		{
			description:         "Mixed - Private - IPv4",
			ip:                  net.ParseIP("1.1.1.1"),
			ver:                 IPv4,
			ipv4PrivateNetworks: []net.IPNet{ipv4Network1, ipv4Network1},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1, ipv6Network2},
			expected:            false,
		},
		{
			description:         "Mixed - Private - IPv6",
			ip:                  net.ParseIP("0303::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{ipv4Network1, ipv4Network1},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1, ipv6Network2},
			expected:            false,
		},
		{
			description:         "Mixed - Public - IPv6 Encoded IPv4",
			ip:                  net.ParseIP("::FFFF:1.1.1.1"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{{IP: net.ParseIP("1.0.0.0"), Mask: net.IPMask{255, 0, 0, 0}}},
			ipv6PrivateNetworks: []net.IPNet{{IP: net.ParseIP("::FFFF:2.0.0.0"), Mask: net.IPMask{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0}}},
			expected:            true,
		},
		{
			description:         "Mixed - Private - IPv6 Encoded IPv4",
			ip:                  net.ParseIP("::FFFF:2.2.2.2"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{{IP: net.ParseIP("1.0.0.0"), Mask: net.IPMask{255, 0, 0, 0}}},
			ipv6PrivateNetworks: []net.IPNet{{IP: net.ParseIP("::FFFF:2.0.0.0"), Mask: net.IPMask{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0}}},
			expected:            false,
		},
	}

	for _, test := range testCases {
		requestValidation := PublicNetworkIPMatcher{
			IPv4PrivateNetworks: test.ipv4PrivateNetworks,
			IPv6PrivateNetworks: test.ipv6PrivateNetworks,
		}

		result := requestValidation.Match(test.ip, test.ver)

		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestVersionIPMatcher(t *testing.T) {
	testCases := []struct {
		description    string
		matcherVersion IPVersion
		ip             net.IP
		ipVer          IPVersion
		expectedMatch  bool
	}{
		{
			description:    "IPv4",
			matcherVersion: IPv4,
			ip:             net.ParseIP("1.1.1.1"),
			ipVer:          IPv4,
			expectedMatch:  true,
		},
		{
			description:    "IPv4 - Given Unknown",
			matcherVersion: IPv4,
			ip:             nil,
			ipVer:          IPvUnknown,
			expectedMatch:  false,
		},
		{
			description:    "IPv6",
			matcherVersion: IPv6,
			ip:             net.ParseIP("0101::"),
			ipVer:          IPv6,
			expectedMatch:  true,
		},
		{
			description:    "IPv6 - Given Unknown",
			matcherVersion: IPv6,
			ip:             nil,
			ipVer:          IPvUnknown,
			expectedMatch:  false,
		},
	}

	for _, test := range testCases {
		m := VersionIPMatcher{
			Version: test.matcherVersion,
		}

		result := m.Match(test.ip, test.ipVer)

		assert.Equal(t, test.expectedMatch, result)
	}
}
