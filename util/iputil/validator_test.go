package iputil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicNetworkIPValidator(t *testing.T) {
	ipv4Network1 := net.IPNet{IP: net.ParseIP("1.0.0.0"), Mask: net.CIDRMask(8, 32)}
	ipv4Network2 := net.IPNet{IP: net.ParseIP("2.0.0.0"), Mask: net.CIDRMask(8, 32)}

	ipv6Network1 := net.IPNet{IP: net.ParseIP("3300::"), Mask: net.CIDRMask(8, 128)}
	ipv6Network2 := net.IPNet{IP: net.ParseIP("4400::"), Mask: net.CIDRMask(8, 128)}

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
			ip:                  net.ParseIP("3333::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{},
			ipv6PrivateNetworks: []net.IPNet{},
			expected:            true,
		},
		{
			description:         "IPv6 - Public - One",
			ip:                  net.ParseIP("4444::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1},
			expected:            true,
		},
		{
			description:         "IPv6 - Public - Many",
			ip:                  net.ParseIP("5555::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1, ipv6Network2},
			expected:            true,
		},
		{
			description:         "IPv6 - Private - One",
			ip:                  net.ParseIP("3333::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1},
			expected:            false,
		},
		{
			description:         "IPv6 - Private - Many",
			ip:                  net.ParseIP("4444::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1, ipv6Network2},
			expected:            false,
		},
		{
			description:         "Mixed - Unknown",
			ip:                  net.ParseIP("3.3.3.3"),
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
			ip:                  net.ParseIP("5555::"),
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
			ip:                  net.ParseIP("3333::"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{ipv4Network1, ipv4Network1},
			ipv6PrivateNetworks: []net.IPNet{ipv6Network1, ipv6Network2},
			expected:            false,
		},
		{
			description:         "Mixed - Public - IPv6 Encoded IPv4",
			ip:                  net.ParseIP("::FFFF:1.1.1.1"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{{IP: net.ParseIP("1.0.0.0"), Mask: net.CIDRMask(8, 32)}},
			ipv6PrivateNetworks: []net.IPNet{{IP: net.ParseIP("::FFFF:2.0.0.0"), Mask: net.CIDRMask(108, 128)}},
			expected:            true,
		},
		{
			description:         "Mixed - Private - IPv6 Encoded IPv4",
			ip:                  net.ParseIP("::FFFF:2.2.2.2"),
			ver:                 IPv6,
			ipv4PrivateNetworks: []net.IPNet{{IP: net.ParseIP("1.0.0.0"), Mask: net.CIDRMask(8, 32)}},
			ipv6PrivateNetworks: []net.IPNet{{IP: net.ParseIP("::FFFF:2.0.0.0"), Mask: net.CIDRMask(108, 128)}},
			expected:            false,
		},
	}

	for _, test := range testCases {
		requestValidation := PublicNetworkIPValidator{
			IPv4PrivateNetworks: test.ipv4PrivateNetworks,
			IPv6PrivateNetworks: test.ipv6PrivateNetworks,
		}

		result := requestValidation.IsValid(test.ip, test.ver)

		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestVersionIPValidator(t *testing.T) {
	testCases := []struct {
		description      string
		validatorVersion IPVersion
		ip               net.IP
		ipVer            IPVersion
		expected         bool
	}{
		{
			description:      "IPv4",
			validatorVersion: IPv4,
			ip:               net.ParseIP("1.1.1.1"),
			ipVer:            IPv4,
			expected:         true,
		},
		{
			description:      "IPv4 - Given Unknown",
			validatorVersion: IPv4,
			ip:               nil,
			ipVer:            IPvUnknown,
			expected:         false,
		},
		{
			description:      "IPv6",
			validatorVersion: IPv6,
			ip:               net.ParseIP("1111::"),
			ipVer:            IPv6,
			expected:         true,
		},
		{
			description:      "IPv6 - Given Unknown",
			validatorVersion: IPv6,
			ip:               nil,
			ipVer:            IPvUnknown,
			expected:         false,
		},
	}

	for _, test := range testCases {
		m := VersionIPValidator{
			Version: test.validatorVersion,
		}

		result := m.IsValid(test.ip, test.ipVer)

		assert.Equal(t, test.expected, result)
	}
}
