package privacy

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestScrubDevice(t *testing.T) {
	device := getTestDevice()

	testCases := []struct {
		description string
		expected    *openrtb2.Device
		id          ScrubStrategyDeviceID
		ipv4        ScrubStrategyIPV4
		ipv6        ScrubStrategyIPV6
		geo         ScrubStrategyGeo
	}{
		{
			description: "All Strategies - None",
			expected:    device,
			id:          ScrubStrategyDeviceIDNone,
			ipv4:        ScrubStrategyIPV4None,
			ipv6:        ScrubStrategyIPV6None,
			geo:         ScrubStrategyGeoNone,
		},
		{
			description: "All Strategies - Strictest",
			expected: &openrtb2.Device{
				DIDMD5:   "",
				DIDSHA1:  "",
				DPIDMD5:  "",
				DPIDSHA1: "",
				MACSHA1:  "",
				MACMD5:   "",
				IFA:      "",
				IP:       "1.2.3.0",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0:0",
				Geo:      &openrtb2.Geo{},
			},
			id:   ScrubStrategyDeviceIDAll,
			ipv4: ScrubStrategyIPV4Lowest8,
			ipv6: ScrubStrategyIPV6Lowest32,
			geo:  ScrubStrategyGeoFull,
		},
		{
			description: "Isolated - ID - All",
			expected: &openrtb2.Device{
				DIDMD5:   "",
				DIDSHA1:  "",
				DPIDMD5:  "",
				DPIDSHA1: "",
				MACSHA1:  "",
				MACMD5:   "",
				IFA:      "",
				IP:       "1.2.3.4",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0042:8329",
				Geo:      device.Geo,
			},
			id:   ScrubStrategyDeviceIDAll,
			ipv4: ScrubStrategyIPV4None,
			ipv6: ScrubStrategyIPV6None,
			geo:  ScrubStrategyGeoNone,
		},
		{
			description: "Isolated - IPv4 - Lowest 8",
			expected: &openrtb2.Device{
				DIDMD5:   "anyDIDMD5",
				DIDSHA1:  "anyDIDSHA1",
				DPIDMD5:  "anyDPIDMD5",
				DPIDSHA1: "anyDPIDSHA1",
				MACSHA1:  "anyMACSHA1",
				MACMD5:   "anyMACMD5",
				IFA:      "anyIFA",
				IP:       "1.2.3.0",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0042:8329",
				Geo:      device.Geo,
			},
			id:   ScrubStrategyDeviceIDNone,
			ipv4: ScrubStrategyIPV4Lowest8,
			ipv6: ScrubStrategyIPV6None,
			geo:  ScrubStrategyGeoNone,
		},
		{
			description: "Isolated - IPv6 - Lowest 16",
			expected: &openrtb2.Device{
				DIDMD5:   "anyDIDMD5",
				DIDSHA1:  "anyDIDSHA1",
				DPIDMD5:  "anyDPIDMD5",
				DPIDSHA1: "anyDPIDSHA1",
				MACSHA1:  "anyMACSHA1",
				MACMD5:   "anyMACMD5",
				IFA:      "anyIFA",
				IP:       "1.2.3.4",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0042:0",
				Geo:      device.Geo,
			},
			id:   ScrubStrategyDeviceIDNone,
			ipv4: ScrubStrategyIPV4None,
			ipv6: ScrubStrategyIPV6Lowest16,
			geo:  ScrubStrategyGeoNone,
		},
		{
			description: "Isolated - IPv6 - Lowest 32",
			expected: &openrtb2.Device{
				DIDMD5:   "anyDIDMD5",
				DIDSHA1:  "anyDIDSHA1",
				DPIDMD5:  "anyDPIDMD5",
				DPIDSHA1: "anyDPIDSHA1",
				MACSHA1:  "anyMACSHA1",
				MACMD5:   "anyMACMD5",
				IFA:      "anyIFA",
				IP:       "1.2.3.4",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0:0",
				Geo:      device.Geo,
			},
			id:   ScrubStrategyDeviceIDNone,
			ipv4: ScrubStrategyIPV4None,
			ipv6: ScrubStrategyIPV6Lowest32,
			geo:  ScrubStrategyGeoNone,
		},
		{
			description: "Isolated - Geo - Reduced Precision",
			expected: &openrtb2.Device{
				DIDMD5:   "anyDIDMD5",
				DIDSHA1:  "anyDIDSHA1",
				DPIDMD5:  "anyDPIDMD5",
				DPIDSHA1: "anyDPIDSHA1",
				MACSHA1:  "anyMACSHA1",
				MACMD5:   "anyMACMD5",
				IFA:      "anyIFA",
				IP:       "1.2.3.4",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0042:8329",
				Geo: &openrtb2.Geo{
					Lat:   123.46,
					Lon:   678.89,
					Metro: "some metro",
					City:  "some city",
					ZIP:   "some zip",
				},
			},
			id:   ScrubStrategyDeviceIDNone,
			ipv4: ScrubStrategyIPV4None,
			ipv6: ScrubStrategyIPV6None,
			geo:  ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "Isolated - Geo - Full",
			expected: &openrtb2.Device{
				DIDMD5:   "anyDIDMD5",
				DIDSHA1:  "anyDIDSHA1",
				DPIDMD5:  "anyDPIDMD5",
				DPIDSHA1: "anyDPIDSHA1",
				MACSHA1:  "anyMACSHA1",
				MACMD5:   "anyMACMD5",
				IFA:      "anyIFA",
				IP:       "1.2.3.4",
				IPv6:     "2001:0db8:0000:0000:0000:ff00:0042:8329",
				Geo:      &openrtb2.Geo{},
			},
			id:   ScrubStrategyDeviceIDNone,
			ipv4: ScrubStrategyIPV4None,
			ipv6: ScrubStrategyIPV6None,
			geo:  ScrubStrategyGeoFull,
		},
	}

	for _, test := range testCases {
		result := NewScrubber().ScrubDevice(device, test.id, test.ipv4, test.ipv6, test.geo)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestScrubDeviceNil(t *testing.T) {
	result := NewScrubber().ScrubDevice(nil, ScrubStrategyDeviceIDNone, ScrubStrategyIPV4None, ScrubStrategyIPV6None, ScrubStrategyGeoNone)
	assert.Nil(t, result)
}

func TestScrubUser(t *testing.T) {
	user := getTestUser()

	testCases := []struct {
		description string
		expected    *openrtb2.User
		scrubUser   ScrubStrategyUser
		scrubGeo    ScrubStrategyGeo
	}{
		{
			description: "User ID And Demographic & Geo Full",
			expected: &openrtb2.User{
				ID:       "",
				BuyerUID: "",
				Yob:      0,
				Gender:   "",
				Ext:      json.RawMessage(`{}`),
				Geo:      &openrtb2.Geo{},
			},
			scrubUser: ScrubStrategyUserIDAndDemographic,
			scrubGeo:  ScrubStrategyGeoFull,
		},
		{
			description: "User ID And Demographic & Geo Reduced",
			expected: &openrtb2.User{
				ID:       "",
				BuyerUID: "",
				Yob:      0,
				Gender:   "",
				Ext:      json.RawMessage(`{}`),
				Geo: &openrtb2.Geo{
					Lat:   123.46,
					Lon:   678.89,
					Metro: "some metro",
					City:  "some city",
					ZIP:   "some zip",
				},
			},
			scrubUser: ScrubStrategyUserIDAndDemographic,
			scrubGeo:  ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "User ID And Demographic & Geo None",
			expected: &openrtb2.User{
				ID:       "",
				BuyerUID: "",
				Yob:      0,
				Gender:   "",
				Ext:      json.RawMessage(`{}`),
				Geo: &openrtb2.Geo{
					Lat:   123.456,
					Lon:   678.89,
					Metro: "some metro",
					City:  "some city",
					ZIP:   "some zip",
				},
			},
			scrubUser: ScrubStrategyUserIDAndDemographic,
			scrubGeo:  ScrubStrategyGeoNone,
		},
		{
			description: "User None & Geo Full",
			expected: &openrtb2.User{
				ID:       "anyID",
				BuyerUID: "anyBuyerUID",
				Yob:      42,
				Gender:   "anyGender",
				Ext:      json.RawMessage(`{}`),
				Geo:      &openrtb2.Geo{},
			},
			scrubUser: ScrubStrategyUserNone,
			scrubGeo:  ScrubStrategyGeoFull,
		},
		{
			description: "User None & Geo Reduced",
			expected: &openrtb2.User{
				ID:       "anyID",
				BuyerUID: "anyBuyerUID",
				Yob:      42,
				Gender:   "anyGender",
				Ext:      json.RawMessage(`{}`),
				Geo: &openrtb2.Geo{
					Lat:   123.46,
					Lon:   678.89,
					Metro: "some metro",
					City:  "some city",
					ZIP:   "some zip",
				},
			},
			scrubUser: ScrubStrategyUserNone,
			scrubGeo:  ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "User None & Geo None",
			expected: &openrtb2.User{
				ID:       "anyID",
				BuyerUID: "anyBuyerUID",
				Yob:      42,
				Gender:   "anyGender",
				Ext:      json.RawMessage(`{}`),
				Geo: &openrtb2.Geo{
					Lat:   123.456,
					Lon:   678.89,
					Metro: "some metro",
					City:  "some city",
					ZIP:   "some zip",
				},
			},
			scrubUser: ScrubStrategyUserNone,
			scrubGeo:  ScrubStrategyGeoNone,
		},
	}

	for _, test := range testCases {
		result := NewScrubber().ScrubUser(user, test.scrubUser, test.scrubGeo)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestScrubUserNil(t *testing.T) {
	result := NewScrubber().ScrubUser(nil, ScrubStrategyUserNone, ScrubStrategyGeoNone)
	assert.Nil(t, result)
}

func TestScrubRequest(t *testing.T) {

	imps := []openrtb2.Imp{
		{ID: "testId", Ext: json.RawMessage(`{"test": 1, "tid": 2}`)},
	}
	source := &openrtb2.Source{
		TID: "testTid",
	}
	device := getTestDevice()
	user := getTestUser()
	user.Ext = json.RawMessage(`{"data": 1, "eids": 2}`)
	user.EIDs = []openrtb2.EID{{Source: "test"}}

	testCases := []struct {
		description    string
		enforcement    Enforcement
		userExtPresent bool
		expected       *openrtb2.BidRequest
	}{
		{
			description:    "enforce transmitUFPD with user.ext",
			enforcement:    Enforcement{UFPD: true},
			userExtPresent: true,
			expected: &openrtb2.BidRequest{
				Imp:    imps,
				Source: source,
				User: &openrtb2.User{
					EIDs: []openrtb2.EID{{Source: "test"}},
					Geo:  user.Geo,
					Ext:  json.RawMessage(`{"eids":2}`),
				},
				Device: &openrtb2.Device{
					IP:   "1.2.3.4",
					IPv6: "2001:0db8:0000:0000:0000:ff00:0042:8329",
					Geo:  device.Geo,
				},
			},
		},
		{
			description:    "enforce transmitUFPD without user.ext",
			enforcement:    Enforcement{UFPD: true},
			userExtPresent: false,
			expected: &openrtb2.BidRequest{
				Imp:    imps,
				Source: source,
				User: &openrtb2.User{
					EIDs: []openrtb2.EID{{Source: "test"}},
					Geo:  user.Geo,
				},
				Device: &openrtb2.Device{
					IP:   "1.2.3.4",
					IPv6: "2001:0db8:0000:0000:0000:ff00:0042:8329",
					Geo:  device.Geo,
				},
			},
		},
		{
			description:    "enforce transmitEids",
			enforcement:    Enforcement{Eids: true},
			userExtPresent: true,
			expected: &openrtb2.BidRequest{
				Imp:    imps,
				Source: source,
				Device: device,
				User: &openrtb2.User{
					ID:       "anyID",
					BuyerUID: "anyBuyerUID",
					Yob:      42,
					Gender:   "anyGender",
					Geo:      user.Geo,
					EIDs:     nil,
					Ext:      json.RawMessage(`{"data":1}`),
				},
			},
		},
		{
			description:    "enforce transmitTid",
			enforcement:    Enforcement{TID: true},
			userExtPresent: true,
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "testId", Ext: json.RawMessage(`{"test":1}`)},
				},
				Source: &openrtb2.Source{
					TID: "",
				},
				Device: device,
				User: &openrtb2.User{
					ID:       "anyID",
					BuyerUID: "anyBuyerUID",
					Yob:      42,
					Gender:   "anyGender",
					Geo:      user.Geo,
					EIDs:     []openrtb2.EID{{Source: "test"}},
					Ext:      json.RawMessage(`{"data": 1, "eids": 2}`),
				},
			},
		},
		{
			description:    "enforce precise Geo",
			enforcement:    Enforcement{PreciseGeo: true},
			userExtPresent: true,
			expected: &openrtb2.BidRequest{
				Imp:    imps,
				Source: source,
				User: &openrtb2.User{
					ID:       "anyID",
					BuyerUID: "anyBuyerUID",
					Yob:      42,
					Gender:   "anyGender",
					Geo: &openrtb2.Geo{
						Lat: 123.46, Lon: 678.89,
						Metro: "some metro",
						City:  "some city",
						ZIP:   "some zip",
					},
					EIDs: []openrtb2.EID{{Source: "test"}},
					Ext:  json.RawMessage(`{"data": 1, "eids": 2}`),
				},
				Device: &openrtb2.Device{
					IFA:      "anyIFA",
					DIDSHA1:  "anyDIDSHA1",
					DIDMD5:   "anyDIDMD5",
					DPIDSHA1: "anyDPIDSHA1",
					DPIDMD5:  "anyDPIDMD5",
					MACSHA1:  "anyMACSHA1",
					MACMD5:   "anyMACMD5",
					IP:       "1.2.3.0",
					IPv6:     "2001:0db8:0000:0000:0000:ff00:0:0",
					Geo: &openrtb2.Geo{
						Lat: 123.46, Lon: 678.89,
						Metro: "some metro",
						City:  "some city",
						ZIP:   "some zip",
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			bidRequest := &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "testId", Ext: json.RawMessage(`{"test": 1, "tid": 2}`)},
				},
				Source: &openrtb2.Source{
					TID: "testTid",
				},
				User:   getTestUser(),
				Device: getTestDevice(),
			}
			if test.userExtPresent {
				bidRequest.User.Ext = json.RawMessage(`{"data": 1, "eids": 2}`)
			} else {
				bidRequest.User.Ext = nil
			}
			bidRequest.User.EIDs = []openrtb2.EID{{Source: "test"}}

			result := NewScrubber().ScrubRequest(bidRequest, test.enforcement)
			assert.Equal(t, test.expected, result, test.description)
		})
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
		result := scrubIPV4Lowest8(test.IP)
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
	geo := &openrtb2.Geo{
		Lat:   123.456,
		Lon:   678.89,
		Metro: "some metro",
		City:  "some city",
		ZIP:   "some zip",
	}
	geoExpected := &openrtb2.Geo{
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

func getTestUser() *openrtb2.User {
	return &openrtb2.User{
		ID:       "anyID",
		BuyerUID: "anyBuyerUID",
		Yob:      42,
		Gender:   "anyGender",
		Ext:      json.RawMessage(`{}`),
		Geo: &openrtb2.Geo{
			Lat:   123.456,
			Lon:   678.89,
			Metro: "some metro",
			City:  "some city",
			ZIP:   "some zip",
		},
	}
}

func getTestDevice() *openrtb2.Device {
	return &openrtb2.Device{
		DIDMD5:   "anyDIDMD5",
		DIDSHA1:  "anyDIDSHA1",
		DPIDMD5:  "anyDPIDMD5",
		DPIDSHA1: "anyDPIDSHA1",
		MACSHA1:  "anyMACSHA1",
		MACMD5:   "anyMACMD5",
		IFA:      "anyIFA",
		IP:       "1.2.3.4",
		IPv6:     "2001:0db8:0000:0000:0000:ff00:0042:8329",
		Geo: &openrtb2.Geo{
			Lat:   123.456,
			Lon:   678.89,
			Metro: "some metro",
			City:  "some city",
			ZIP:   "some zip",
		},
	}

}
