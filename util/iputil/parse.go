package iputil

import (
	"net"
	"strings"
)

// IPVersion is the numerical version of the IP address spec (4 or 6).
type IPVersion int

// IP address versions.
const (
	IPvUnknown IPVersion = 0
	IPv4       IPVersion = 4
	IPv6       IPVersion = 6
)

const (
	IPv4BitSize = 32
	IPv6BitSize = 128

	IPv4DefaultMaskingBitSize = 24
	IPv6DefaultMaskingBitSize = 56
)

// ParseIP parses v as an ip address returning the result and version, or nil and unknown if invalid.
func ParseIP(v string) (net.IP, IPVersion) {
	if ip := net.ParseIP(v); ip != nil {
		if strings.ContainsRune(v, ':') {
			return ip, IPv6
		}
		return ip, IPv4
	}
	return nil, IPvUnknown
}
