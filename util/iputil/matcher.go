package iputil

import (
	"net"
)

// IPMatcher is the interface for matching on an ip address and version.
type IPMatcher interface {
	// Match returns true when a desired IP address is found.
	Match(net.IP, IPVersion) bool
}

// PublicNetworkIPMatcher matches an ip address which is not contained in the list of known private networks.
type PublicNetworkIPMatcher struct {
	IPv4PrivateNetworks []net.IPNet
	IPv6PrivateNetworks []net.IPNet
}

// Match implements the IPMatcher interface.
func (m PublicNetworkIPMatcher) Match(ip net.IP, ver IPVersion) bool {
	var privateNetworks []net.IPNet
	switch ver {
	case IPv4:
		privateNetworks = m.IPv4PrivateNetworks
	case IPv6:
		privateNetworks = m.IPv6PrivateNetworks
	default:
		return false
	}

	for _, ipNet := range privateNetworks {
		if ipNet.Contains(ip) {
			return false
		}
	}

	return true
}

// VersionIPMatcher matches an ip address based on the desired ip version.
type VersionIPMatcher struct {
	Version IPVersion
}

// Match implements the IPMatcher interface.
func (m VersionIPMatcher) Match(ip net.IP, ver IPVersion) bool {
	return ver == m.Version
}
