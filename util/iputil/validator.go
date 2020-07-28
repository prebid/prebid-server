package iputil

import (
	"net"
)

// IPValidator is the interface for validating an ip address and version.
type IPValidator interface {
	// IsValid returns true when an IP address is determined to be valid.
	IsValid(net.IP, IPVersion) bool
}

// PublicNetworkIPValidator validates an ip address which is not contained in the list of known private networks.
type PublicNetworkIPValidator struct {
	IPv4PrivateNetworks []net.IPNet
	IPv6PrivateNetworks []net.IPNet
}

// IsValid implements the IPValidator interface.
func (v PublicNetworkIPValidator) IsValid(ip net.IP, ver IPVersion) bool {
	var privateNetworks []net.IPNet
	switch ver {
	case IPv4:
		privateNetworks = v.IPv4PrivateNetworks
	case IPv6:
		privateNetworks = v.IPv6PrivateNetworks
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

// VersionIPValidator validates an ip address based on the desired ip version.
type VersionIPValidator struct {
	Version IPVersion
}

// IsValid implements the IPValidator interface.
func (v VersionIPValidator) IsValid(ip net.IP, ver IPVersion) bool {
	return ver == v.Version
}
