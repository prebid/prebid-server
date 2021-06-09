package config

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// RequestValidation specifies the request validation options.
type RequestValidation struct {
	IPv4PrivateNetworks       []string `mapstructure:"ipv4_private_networks,flow"`
	IPv4PrivateNetworksParsed []net.IPNet

	IPv6PrivateNetworks       []string `mapstructure:"ipv6_private_networks,flow"`
	IPv6PrivateNetworksParsed []net.IPNet
}

// Parse converts the CIDR representation of the IPv4 and IPv6 private networks as net.IPNet structs, or returns an error if at least one is invalid.
func (r *RequestValidation) Parse() error {
	ipv4Nets, err := parseNetworks(r.IPv4PrivateNetworks, net.IPv4len)
	if err != nil {
		return errors.New("Invalid private IPv4 network: " + err.Error())
	}

	ipv6Nets, err := parseNetworks(r.IPv6PrivateNetworks, net.IPv6len)
	if err != nil {
		return errors.New("Invalid private IPv6 network: " + err.Error())
	}

	r.IPv4PrivateNetworksParsed = ipv4Nets
	r.IPv6PrivateNetworksParsed = ipv6Nets
	return nil
}

func parseNetworks(networks []string, networksLen int) ([]net.IPNet, error) {
	ipNetworks := make([]net.IPNet, 0, len(networks))
	errMsg := strings.Builder{}

	for _, v := range networks {
		v := strings.TrimSpace(v)

		if _, ipNet, err := net.ParseCIDR(v); err != nil || len(ipNet.IP) != networksLen {
			fmt.Fprintf(&errMsg, "'%s',", v)
		} else {
			ipNetworks = append(ipNetworks, *ipNet)
		}
	}

	if errMsg.Len() > 0 {
		return nil, errors.New(errMsg.String()[:errMsg.Len()-1])
	}

	return ipNetworks, nil
}
