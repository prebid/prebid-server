package iputil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIP(t *testing.T) {
	testCases := []struct {
		input       string
		expectedVer IPVersion
		expectedIP  net.IP
	}{
		{"", IPvUnknown, nil},
		{"1.1.1.1", IPv4, net.IPv4(1, 1, 1, 1)},
		{"-1.-1.-1.-1", IPvUnknown, nil},
		{"256.256.256.256", IPvUnknown, nil},
		{"::ffff:1.1.1.1", IPv6, net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 1, 1, 1, 1}},
		{"0101::", IPv6, net.IP{1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"zzzz::", IPvUnknown, nil},
	}

	for _, test := range testCases {
		ip, ver := ParseIP(test.input)
		assert.Equal(t, test.expectedVer, ver)
		assert.Equal(t, test.expectedIP, ip)
	}
}
