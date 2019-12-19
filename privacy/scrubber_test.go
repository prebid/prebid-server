package privacy

import (
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

func TestScrubDevice(t *testing.T) {
	device := &openrtb.Device{
		DIDMD5:   "anyDIDMD5",
		DIDSHA1:  "anyDIDSHA1",
		DPIDMD5:  "anyDPIDMD5",
		DPIDSHA1: "anyDPIDSHA1",
		MACSHA1:  "anyMACSHA1",
		MACMD5:   "anyMACMD5",
		IFA:      "anyIFA",
		IP:       "1.2.3.4",
		IPv6:     "2001:0db8:0000:0000:0000:ff00:0042:8329",
		Geo: &openrtb.Geo{
			Lat:   123.456,
			Lon:   678.89,
			Metro: "some metro",
			City:  "some city",
			ZIP:   "some zip",
		},
	}

	testCases := []struct {
		expected    *openrtb.Device
		isMacAndIFA bool
		ipv6        ScrubStrategyIPV6
		geo         ScrubStrategyGeo
		description string
	}{
		{
			expected: &openrtb.Device{
				DIDMD5:   "",
				DIDSHA1:  "",
				DPIDMD5:  "",
				DPIDSHA1: "",
				MACSHA1:  "",
				MACMD5:   "",
				IFA:      "",
				IP:       "1.2.3.0",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0:0",
				Geo:      &openrtb.Geo{},
			},
			isMacAndIFA: true,
			ipv6:        ScrubStrategyIPV6Lowest32,
			geo:         ScrubStrategyGeoFull,
			description: "Full Scrubbing",
		},
		{
			expected: &openrtb.Device{
				DIDMD5:   "",
				DIDSHA1:  "",
				DPIDMD5:  "",
				DPIDSHA1: "",
				MACSHA1:  "",
				MACMD5:   "",
				IFA:      "",
				IP:       "1.2.3.0",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0042:0",
				Geo:      &openrtb.Geo{},
			},
			isMacAndIFA: true,
			ipv6:        ScrubStrategyIPV6Lowest16,
			geo:         ScrubStrategyGeoFull,
			description: "IPv6 Lowest 16",
		},
		{
			expected: &openrtb.Device{
				DIDMD5:   "",
				DIDSHA1:  "",
				DPIDMD5:  "",
				DPIDSHA1: "",
				MACSHA1:  "",
				MACMD5:   "",
				IFA:      "",
				IP:       "1.2.3.0",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0042:8329",
				Geo:      &openrtb.Geo{},
			},
			isMacAndIFA: true,
			ipv6:        ScrubStrategyIPV6None,
			geo:         ScrubStrategyGeoFull,
			description: "IPv6 None",
		},
		{
			expected: &openrtb.Device{
				DIDMD5:   "",
				DIDSHA1:  "",
				DPIDMD5:  "",
				DPIDSHA1: "",
				MACSHA1:  "",
				MACMD5:   "",
				IFA:      "",
				IP:       "1.2.3.0",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0:0",
				Geo: &openrtb.Geo{
					Lat:   123.46,
					Lon:   678.89,
					Metro: "some metro",
					City:  "some city",
					ZIP:   "some zip",
				},
			},
			isMacAndIFA: true,
			ipv6:        ScrubStrategyIPV6Lowest32,
			geo:         ScrubStrategyGeoReducedPrecision,
			description: "Geo Reduced Precision",
		},
		{
			expected: &openrtb.Device{
				DIDMD5:   "",
				DIDSHA1:  "",
				DPIDMD5:  "",
				DPIDSHA1: "",
				MACSHA1:  "",
				MACMD5:   "",
				IFA:      "",
				IP:       "1.2.3.0",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0:0",
				Geo: &openrtb.Geo{
					Lat:   123.456,
					Lon:   678.89,
					Metro: "some metro",
					City:  "some city",
					ZIP:   "some zip",
				},
			},
			isMacAndIFA: true,
			ipv6:        ScrubStrategyIPV6Lowest32,
			geo:         ScrubStrategyGeoNone,
			description: "Geo None",
		},
		{
			expected: &openrtb.Device{
				DIDMD5:   "",
				DIDSHA1:  "",
				DPIDMD5:  "",
				DPIDSHA1: "",
				MACSHA1:  "anyMACSHA1",
				MACMD5:   "anyMACMD5",
				IFA:      "anyIFA",
				IP:       "1.2.3.0",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0:0",
				Geo:      &openrtb.Geo{},
			},
			isMacAndIFA: false,
			ipv6:        ScrubStrategyIPV6Lowest32,
			geo:         ScrubStrategyGeoFull,
			description: "Without MAC Address And IFA Scrubbing",
		},
	}

	for _, test := range testCases {
		result := NewScrubber().ScrubDevice(device, test.isMacAndIFA, test.ipv6, test.geo)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestScrubUser(t *testing.T) {
	user := &openrtb.User{
		BuyerUID: "anyBuyerUID",
		ID:       "anyID",
		Yob:      42,
		Gender:   "anyGender",
		Geo: &openrtb.Geo{
			Lat:   123.456,
			Lon:   678.89,
			Metro: "some metro",
			City:  "some city",
			ZIP:   "some zip",
		},
	}

	testCases := []struct {
		expected    *openrtb.User
		strategy    ScrubStrategyUser
		geo         ScrubStrategyGeo
		description string
	}{
		{
			expected: &openrtb.User{
				BuyerUID: "",
				ID:       "",
				Yob:      0,
				Gender:   "",
				Geo:      &openrtb.Geo{},
			},
			strategy:    ScrubStrategyUserFull,
			geo:         ScrubStrategyGeoFull,
			description: "Full Scrubbing",
		},
		{
			expected: &openrtb.User{
				BuyerUID: "",
				ID:       "anyID",
				Yob:      42,
				Gender:   "anyGender",
				Geo:      &openrtb.Geo{},
			},
			strategy:    ScrubStrategyUserBuyerIDOnly,
			geo:         ScrubStrategyGeoFull,
			description: "User Buyer ID Only",
		},
		{
			expected: &openrtb.User{
				BuyerUID: "anyBuyerUID",
				ID:       "anyID",
				Yob:      42,
				Gender:   "anyGender",
				Geo:      &openrtb.Geo{},
			},
			strategy:    ScrubStrategyUserNone,
			geo:         ScrubStrategyGeoFull,
			description: "User None",
		},
		{
			expected: &openrtb.User{
				BuyerUID: "",
				ID:       "",
				Yob:      0,
				Gender:   "",
				Geo: &openrtb.Geo{
					Lat:   123.46,
					Lon:   678.89,
					Metro: "some metro",
					City:  "some city",
					ZIP:   "some zip",
				},
			},
			strategy:    ScrubStrategyUserFull,
			geo:         ScrubStrategyGeoReducedPrecision,
			description: "Geo Reduced Precision",
		},
		{
			expected: &openrtb.User{
				BuyerUID: "",
				ID:       "",
				Yob:      0,
				Gender:   "",
				Geo: &openrtb.Geo{
					Lat:   123.456,
					Lon:   678.89,
					Metro: "some metro",
					City:  "some city",
					ZIP:   "some zip",
				},
			},
			strategy:    ScrubStrategyUserFull,
			geo:         ScrubStrategyGeoNone,
			description: "Geo None",
		},
	}

	for _, test := range testCases {
		result := NewScrubber().ScrubUser(user, test.strategy, test.geo)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestScrubIPV4(t *testing.T) {
	testCases := []struct {
		IP          string
		cleanedIP   string
		description string
	}{
		{
			IP:          "0.0.0.0",
			cleanedIP:   "0.0.0.0",
			description: "Shouldn't do anything for a 0.0.0.0 IP address",
		},
		{
			IP:          "192.127.111.134",
			cleanedIP:   "192.127.111.0",
			description: "Should remove the lowest 8 bits",
		},
		{
			IP:          "192.127.111.0",
			cleanedIP:   "192.127.111.0",
			description: "Shouldn't change anything if the lowest 8 bits are already 0",
		},
		{
			IP:          "not an ip",
			cleanedIP:   "",
			description: "Should return an empty string for a bad IP",
		},
		{
			IP:          "",
			cleanedIP:   "",
			description: "Should return an empty string for a bad IP",
		},
	}

	for _, test := range testCases {
		result := scrubIPV4(test.IP)
		assert.Equal(t, test.cleanedIP, result, test.description)
	}
}

func TestScrubIPV6Lowest16Bits(t *testing.T) {
	testCases := []struct {
		IP          string
		cleanedIP   string
		description string
	}{
		{
			IP:          "0:0:0:0",
			cleanedIP:   "0:0:0:0",
			description: "Shouldn't do anything for a 0:0:0:0 IP address",
		},
		{
			IP:          "2001:0db8:0000:0000:0000:ff00:0042:8329",
			cleanedIP:   "2001:0db8:0000:0000:0000:ff00:0042:0",
			description: "Should remove lowest 16 bits",
		},
		{
			IP:          "2001:0db8:0000:0000:0000:ff00:0042:0",
			cleanedIP:   "2001:0db8:0000:0000:0000:ff00:0042:0",
			description: "Shouldn't do anything if the lowest 16 bits are already 0",
		},
		{
			IP:          "not an ip",
			cleanedIP:   "",
			description: "Should return an empty string for a bad IP",
		},
		{
			IP:          "",
			cleanedIP:   "",
			description: "Should return an empty string for a bad IP",
		},
	}

	for _, test := range testCases {
		result := scrubIPV6Lowest16Bits(test.IP)
		assert.Equal(t, test.cleanedIP, result, test.description)
	}
}

func TestScrubIPV6Lowest32Bits(t *testing.T) {
	testCases := []struct {
		IP          string
		cleanedIP   string
		description string
	}{
		{
			IP:          "0:0:0:0",
			cleanedIP:   "0:0:0:0",
			description: "Shouldn't do anything for a 0:0:0:0 IP address",
		},
		{
			IP:          "2001:0db8:0000:0000:0000:ff00:0042:8329",
			cleanedIP:   "2001:0db8:0000:0000:0000:ff00:0:0",
			description: "Should remove lowest 32 bits",
		},
		{
			IP:          "2001:0db8:0000:0000:0000:ff00:0:0",
			cleanedIP:   "2001:0db8:0000:0000:0000:ff00:0:0",
			description: "Shouldn't do anything if the lowest 32 bits are already 0",
		},

		{
			IP:          "not an ip",
			cleanedIP:   "",
			description: "Should return an empty string for a bad IP",
		},
		{
			IP:          "",
			cleanedIP:   "",
			description: "Should return an empty string for a bad IP",
		},
	}

	for _, test := range testCases {
		result := scrubIPV6Lowest32Bits(test.IP)
		assert.Equal(t, test.cleanedIP, result, test.description)
	}
}

func TestScrubGeoFull(t *testing.T) {
	geo := &openrtb.Geo{
		Lat:   123.456,
		Lon:   678.89,
		Metro: "some metro",
		City:  "some city",
		ZIP:   "some zip",
	}
	geoExpected := &openrtb.Geo{
		Lat:   0,
		Lon:   0,
		Metro: "",
		City:  "",
		ZIP:   "",
	}

	result := scrubGeoFull(geo)

	assert.Equal(t, geoExpected, result)
}

func TestScrubGeoFullWhenNil(t *testing.T) {
	result := scrubGeoFull(nil)
	assert.Nil(t, result)
}

func TestScrubGeoPrecision(t *testing.T) {
	geo := &openrtb.Geo{
		Lat:   123.456,
		Lon:   678.89,
		Metro: "some metro",
		City:  "some city",
		ZIP:   "some zip",
	}
	geoExpected := &openrtb.Geo{
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
