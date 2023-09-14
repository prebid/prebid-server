package privacy

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestScrubIP(t *testing.T) {
	testCases := []struct {
		IP        string
		cleanedIP string
		bits      int
		maskBits  int
	}{
		{
			IP:        "0:0:0:0:0:0:0:0",
			cleanedIP: "::",
			bits:      128,
			maskBits:  56,
		},
		{
			IP:        "",
			cleanedIP: "",
			bits:      128,
			maskBits:  56,
		},
		{
			IP:        "1111:2222:3333:4444:5555:6666:7777:8888",
			cleanedIP: "1111:2222:3333:4400::",
			bits:      128,
			maskBits:  56,
		},
		{
			IP:        "1111:2222:3333:4444:5555:6666:7777:8888",
			cleanedIP: "1111:2222::",
			bits:      128,
			maskBits:  34,
		},
		{
			IP:        "1111:0:3333:4444:5555:6666:7777:8888",
			cleanedIP: "1111:0:3333:4400::",
			bits:      128,
			maskBits:  56,
		},
		{
			IP:        "1111::6666:7777:8888",
			cleanedIP: "1111::",
			bits:      128,
			maskBits:  56,
		},
		{
			IP:        "2001:1db8:0000:0000:0000:ff00:0042:8329",
			cleanedIP: "2001:1db8::ff00:0:0",
			bits:      128,
			maskBits:  96,
		},
		{
			IP:        "2001:1db8:0000:0000:0000:ff00:0:0",
			cleanedIP: "2001:1db8::ff00:0:0",
			bits:      128,
			maskBits:  96,
		},
		{
			IP:        "2001:1db8:0000:0000:0000:ff00:0042:8329",
			cleanedIP: "2001:1db8::ff00:42:0",
			bits:      128,
			maskBits:  112,
		},
		{
			IP:        "2001:1db8:0000:0000:0000:ff00:0042:0",
			cleanedIP: "2001:1db8::ff00:42:0",
			bits:      128,
			maskBits:  112,
		},
		{
			IP:        "127.0.0.1",
			cleanedIP: "127.0.0.0",
			bits:      32,
			maskBits:  24,
		},
		{
			IP:        "0.0.0.0",
			cleanedIP: "0.0.0.0",
			bits:      32,
			maskBits:  24,
		},
		{
			IP:        "192.127.111.134",
			cleanedIP: "192.127.111.0",
			bits:      32,
			maskBits:  24,
		},
		{
			IP:        "192.127.111.0",
			cleanedIP: "192.127.111.0",
			bits:      32,
			maskBits:  24,
		},
	}
	for _, test := range testCases {
		t.Run(test.IP, func(t *testing.T) {
			// bits: ipv6 - 128, ipv4 - 32
			result := scrubIP(test.IP, test.maskBits, test.bits)
			assert.Equal(t, test.cleanedIP, result)
		})
	}
}

func TestScrubGeoPrecision(t *testing.T) {
	geo := &openrtb2.Geo{
		Lat:   123.456,
		Lon:   678.89,
		Metro: "some metro",
		City:  "some city",
		ZIP:   "some zip",
	}
	geoExpected := &openrtb2.Geo{
		Lat:   123.46,
		Lon:   678.89,
		Metro: "some metro",
		City:  "some city",
		ZIP:   "some zip",
	}

	result := scrubGeoPrecision(geo)

	assert.Equal(t, geoExpected, result)
}

func TestScrubGeoPrecisionWhenNil(t *testing.T) {
	result := scrubGeoPrecision(nil)
	assert.Nil(t, result)
}

func TestScrubUserExtIDs(t *testing.T) {
	testCases := []struct {
		description string
		userExt     json.RawMessage
		expected    json.RawMessage
	}{
		{
			description: "Nil",
			userExt:     nil,
			expected:    nil,
		},
		{
			description: "Empty String",
			userExt:     json.RawMessage(``),
			expected:    json.RawMessage(``),
		},
		{
			description: "Empty Object",
			userExt:     json.RawMessage(`{}`),
			expected:    json.RawMessage(`{}`),
		},
		{
			description: "Do Nothing When Malformed",
			userExt:     json.RawMessage(`malformed`),
			expected:    json.RawMessage(`malformed`),
		},
		{
			description: "Do Nothing When No IDs Present",
			userExt:     json.RawMessage(`{"anyExisting":42}}`),
			expected:    json.RawMessage(`{"anyExisting":42}}`),
		},
		{
			description: "Remove eids",
			userExt:     json.RawMessage(`{"eids":[{"source":"anySource","id":"anyId","uids":[{"id":"anyId","ext":{"id":42}}],"ext":{"id":42}}]}`),
			expected:    json.RawMessage(`{}`),
		},
		{
			description: "Remove eids - With Other Data",
			userExt:     json.RawMessage(`{"anyExisting":42,"eids":[{"source":"anySource","id":"anyId","uids":[{"id":"anyId","ext":{"id":42}}],"ext":{"id":42}}]}`),
			expected:    json.RawMessage(`{"anyExisting":42}`),
		},
		{
			description: "Remove eids - With Other Nested Data",
			userExt:     json.RawMessage(`{"anyExisting":{"existing":42},"eids":[{"source":"anySource","id":"anyId","uids":[{"id":"anyId","ext":{"id":42}}],"ext":{"id":42}}]}`),
			expected:    json.RawMessage(`{"anyExisting":{"existing":42}}`),
		},
		{
			description: "Remove eids Only",
			userExt:     json.RawMessage(`{"eids":[{"source":"anySource","id":"anyId","uids":[{"id":"anyId","ext":{"id":42}}],"ext":{"id":42}}]}`),
			expected:    json.RawMessage(`{}`),
		},
		{
			description: "Remove eids Only - Empty Array",
			userExt:     json.RawMessage(`{"eids":[]}`),
			expected:    json.RawMessage(`{}`),
		},
		{
			description: "Remove eids Only - With Other Data",
			userExt:     json.RawMessage(`{"anyExisting":42,"eids":[{"source":"anySource","id":"anyId","uids":[{"id":"anyId","ext":{"id":42}}],"ext":{"id":42}}]}`),
			expected:    json.RawMessage(`{"anyExisting":42}`),
		},
		{
			description: "Remove eids Only - With Other Nested Data",
			userExt:     json.RawMessage(`{"anyExisting":{"existing":42},"eids":[{"source":"anySource","id":"anyId","uids":[{"id":"anyId","ext":{"id":42}}],"ext":{"id":42}}]}`),
			expected:    json.RawMessage(`{"anyExisting":{"existing":42}}`),
		},
	}

	for _, test := range testCases {
		result := scrubExtIDs(test.userExt, "eids")
		assert.Equal(t, test.expected, result, test.description)
	}
}
