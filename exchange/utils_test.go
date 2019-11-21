package exchange

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/stretchr/testify/assert"
)

// permissionsMock mocks the Permissions interface for tests
//
// It only allows appnexus for GDPR consent
type permissionsMock struct{}

func (p *permissionsMock) HostCookiesAllowed(ctx context.Context, consent string) (bool, error) {
	return true, nil
}

func (p *permissionsMock) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	return true, nil
}

func (p *permissionsMock) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, consent string) (bool, error) {
	if bidder == "appnexus" {
		return true, nil
	}
	return false, nil
}

func assertReq(t *testing.T, reqByBidders map[openrtb_ext.BidderName]*openrtb.BidRequest,
	applyCOPPA bool, consentedVendors map[string]bool) {
	// assert individual bidder requests
	assert.NotEqual(t, reqByBidders, 0, "cleanOpenRTBRequest should split request into individual bidder requests")

	// assert for PI data
	// Both appnexus and brightroll should be allowed since brightroll
	// is used as an alias for appnexus in the test request
	for bidderName, bidder := range reqByBidders {
		if !applyCOPPA && consentedVendors[bidderName.String()] {
			assert.NotEqual(t, bidder.User.BuyerUID, "", "cleanOpenRTBRequest shouldn't clean PI data as per COPPA or for a consented vendor as per GDPR")
			assert.NotEqual(t, bidder.Device.DIDMD5, "", "cleanOpenRTBRequest shouldn't clean PI data as per COPPA or for a consented vendor as per GDPR")
		} else {
			assert.Equal(t, bidder.User.BuyerUID, "", "cleanOpenRTBRequest should clean PI data as per COPPA or for a non-consented vendor as per GDPR ", bidderName.String())
			assert.Equal(t, bidder.Device.DIDMD5, "", "cleanOpenRTBRequest should clean PI data as per COPPA or for a non-consented vendor as per GDPR", bidderName.String())
		}
	}
}

// Prevents #820
func TestCleanOpenRTBRequests(t *testing.T) {
	testCases := []struct {
		req              *openrtb.BidRequest
		bidReqAssertions func(t *testing.T, reqByBidders map[openrtb_ext.BidderName]*openrtb.BidRequest,
			applyCOPPA bool, consentedVendors map[string]bool)
		hasError         bool
		applyCOPPA       bool
		consentedVendors map[string]bool
	}{
		{req: newRaceCheckingRequest(t), bidReqAssertions: assertReq, hasError: false,
			applyCOPPA: true, consentedVendors: map[string]bool{"appnexus": true}},
		{req: newAdapterAliasBidRequest(t), bidReqAssertions: assertReq, hasError: false,
			applyCOPPA: false, consentedVendors: map[string]bool{"appnexus": true, "brightroll": true}},
	}

	for _, test := range testCases {
		reqByBidders, _, err := cleanOpenRTBRequests(context.Background(), test.req, &emptyUsersync{}, map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels{}, pbsmetrics.Labels{}, &permissionsMock{}, true)
		if test.hasError {
			assert.NotNil(t, err, "Error shouldn't be nil")
		} else {
			assert.Nil(t, err, "Err should be nil")
			test.bidReqAssertions(t, reqByBidders, test.applyCOPPA, test.consentedVendors)
		}
	}
}

// newAdapterAliasBidRequest builds a BidRequest with aliases
func newAdapterAliasBidRequest(t *testing.T) *openrtb.BidRequest {
	dnt := int8(1)
	return &openrtb.BidRequest{
		Site: &openrtb.Site{
			Page:   "www.some.domain.com",
			Domain: "domain.com",
			Publisher: &openrtb.Publisher{
				ID: "some-publisher-id",
			},
		},
		Device: &openrtb.Device{
			DIDMD5:   "some device ID hash",
			UA:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.87 Safari/537.36",
			IFA:      "ifa",
			IP:       "132.173.230.74",
			DNT:      &dnt,
			Language: "EN",
		},
		Source: &openrtb.Source{
			TID: "61018dc9-fa61-4c41-b7dc-f90b9ae80e87",
		},
		User: &openrtb.User{
			ID:       "our-id",
			BuyerUID: "their-id",
			Ext:      json.RawMessage(`{"consent":"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw","digitrust":{"id":"digi-id","keyv":1,"pref":1}}`),
		},
		Regs: &openrtb.Regs{
			Ext: json.RawMessage(`{"gdpr":1}`),
		},
		Imp: []openrtb.Imp{{
			ID: "some-imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: json.RawMessage(`{"appnexus": {"placementId": 10433394},"brightroll": {"placementId": 105}}`),
		}},
		Ext: json.RawMessage(`{"prebid":{"aliases":{"brightroll":"appnexus"}}}`),
	}
}

func TestRandomizeList(t *testing.T) {
	adapters := make([]openrtb_ext.BidderName, 3)
	adapters[0] = openrtb_ext.BidderName("dummy")
	adapters[1] = openrtb_ext.BidderName("dummy2")
	adapters[2] = openrtb_ext.BidderName("dummy3")

	randomizeList(adapters)

	if len(adapters) != 3 {
		t.Errorf("RandomizeList, expected a list of 3, found %d", len(adapters))
	}

	adapters = adapters[0:1]
	randomizeList(adapters)

	if len(adapters) != 1 {
		t.Errorf("RandomizeList, expected a list of 1, found %d", len(adapters))
	}

}

func TestCleanIP(t *testing.T) {
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
			description: "Should remove the lowest 8 bits for GDPR/COPPA compliance",
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
		{
			IP:          "36278042",
			cleanedIP:   "",
			description: "Should return an empty string for a bad IP",
		},
	}

	for _, test := range testCases {
		assert.Equal(t, cleanIP(test.IP), test.cleanedIP, "Should properly remove the last 8 bits of the IP")
	}
}

func TestCleanIPV6(t *testing.T) {
	testCases := []struct {
		IP          string
		cleanedIP   string
		applyGDPR   bool
		applyCOPPA  bool
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
			applyCOPPA:  true,
			description: "Should remove lowest 32 bits for COPPA compliance",
		},
		{
			IP:          "2001:0db8:0000:0000:0000:ff00:0042:8329",
			cleanedIP:   "2001:0db8:0000:0000:0000:ff00:0042:0",
			applyGDPR:   true,
			description: "Should remove lowest 16 bits for GDPR compliance",
		},
		{
			IP:          "2001:0db8:0000:0000:0000:ff00:0042:0",
			cleanedIP:   "2001:0db8:0000:0000:0000:ff00:0042:0",
			applyGDPR:   true,
			description: "Shouldn't do anything if the lowest 16 bits are already 0 for GDPR compliance",
		},
		{
			IP:          "2001:0db8:0000:0000:0000:ff00:0:0",
			cleanedIP:   "2001:0db8:0000:0000:0000:ff00:0:0",
			applyGDPR:   true,
			description: "Shouldn't do anything if the lowest 16 bits are already 0 for GDPR compliance",
		},
		{
			IP:          "2001:0db8:0000:0000:0000:ff00:0042:0",
			cleanedIP:   "2001:0db8:0000:0000:0000:ff00:0:0",
			applyCOPPA:  true,
			description: "Shouldn't do anything if the lowest 32 bits are already 0 for COPPA compliance",
		},
		{
			IP:          "2001:0db8:0000:0000:0000:ff00:0:0",
			cleanedIP:   "2001:0db8:0000:0000:0000:ff00:0:0",
			applyCOPPA:  true,
			description: "Shouldn't do anything if the lowest 32 bits are already 0 for COPPA compliance",
		},
		{
			IP:          "not an ip",
			cleanedIP:   "",
			applyCOPPA:  true,
			description: "Should return an empty string for a bad IP",
		},
		{
			IP:          "not an ip",
			cleanedIP:   "",
			applyGDPR:   true,
			description: "Should return an empty string for a bad IP",
		},
		{
			IP:          "",
			cleanedIP:   "",
			applyCOPPA:  true,
			description: "Should return an empty string for a bad IP",
		},
	}

	for _, test := range testCases {
		assert.Equal(t, cleanIPV6(test.IP, test.applyGDPR, test.applyCOPPA), test.cleanedIP, "Should properly remove the last 8 bits of the IP")
	}
}

func TestCleanGeo(t *testing.T) {
	testCases := []struct {
		geo         *openrtb.Geo
		cleanedGeo  *openrtb.Geo
		applyGDPR   bool
		applyCOPPA  bool
		description string
	}{
		{
			geo: &openrtb.Geo{
				Lat:   123.456,
				Lon:   678.89,
				Metro: "some metro",
				City:  "some city",
				ZIP:   "some zip",
			},
			cleanedGeo: &openrtb.Geo{
				Lat:   123.46,
				Lon:   678.89,
				Metro: "some metro",
				City:  "some city",
				ZIP:   "some zip",
			},
			applyGDPR:   true,
			description: "Should only round off Lat and Lon values for GDPR compliance",
		},
		{
			geo: &openrtb.Geo{
				Lat:   123.456,
				Lon:   678.89,
				Metro: "some metro",
				City:  "some city",
				ZIP:   "some zip",
			},
			cleanedGeo: &openrtb.Geo{
				Lat:   0,
				Lon:   0,
				Metro: "",
				City:  "",
				ZIP:   "",
			},
			applyCOPPA:  true,
			description: "Should suppress all Geo values for GDPR compliance",
		},
		{
			geo: &openrtb.Geo{
				Lat:   123.456,
				Lon:   678.89,
				Metro: "some metro",
				City:  "some city",
				ZIP:   "some zip",
			},
			cleanedGeo: &openrtb.Geo{
				Lat:   123.456,
				Lon:   678.89,
				Metro: "some metro",
				City:  "some city",
				ZIP:   "some zip",
			},
			applyGDPR:   false,
			applyCOPPA:  false,
			description: "Should do nothing if neither GDPR nor COPPA applies",
		},
	}

	for _, test := range testCases {
		cleanedGeo := cleanGeo(test.geo, test.applyGDPR, test.applyCOPPA)
		assert.Equal(t, cleanedGeo, test.cleanedGeo, test.description)
	}
}

func TestApplyRegs(t *testing.T) {
	bidReqOrig := openrtb.BidRequest{
		User: &openrtb.User{
			BuyerUID: "abc123",
			ID:       "123",
			Yob:      2050,
			Gender:   "Female",
		},
		Device: &openrtb.Device{
			DIDMD5:  "teapot",
			MACSHA1: "someshahash",
			IP:      "12.123.56.128",
			IPv6:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			Geo: &openrtb.Geo{
				Lat: 123.4567,
				Lon: 7.9836,
			},
		},
	}

	testCases := []struct {
		bidReq      openrtb.BidRequest
		cleanedIP   string
		cleanedIPv6 string
		applyCOPPA  bool
		applyGDPR   bool
		isAMP       bool
		description string
	}{
		{
			bidReq:      bidReqOrig,
			cleanedIP:   "12.123.56.0",
			cleanedIPv6: "2001:0db8:85a3:0000:0000:8a2e:0370:0",
			applyGDPR:   true,
			description: "Should clean recommended personal information for GDPR compliance",
		},
		{
			bidReq:      bidReqOrig,
			cleanedIP:   "12.123.56.0",
			cleanedIPv6: "2001:0db8:85a3:0000:0000:8a2e:0370:0",
			applyGDPR:   true,
			isAMP:       true,
			description: "Should clean recommended personal information for GDPR compliance",
		},
		{
			bidReq:      bidReqOrig,
			cleanedIP:   "12.123.56.0",
			cleanedIPv6: "2001:0db8:85a3:0000:0000:8a2e:0:0",
			applyGDPR:   true,
			isAMP:       true,
			applyCOPPA:  true,
			description: "Should clean recommended personal information for GDPR compliance",
		},
		{
			bidReq:      bidReqOrig,
			cleanedIP:   "12.123.56.0",
			cleanedIPv6: "2001:0db8:85a3:0000:0000:8a2e:0:0",
			applyCOPPA:  true,
			description: "Should clean recommended personal information for COPPA compliance",
		},
		{
			bidReq:      openrtb.BidRequest{},
			cleanedIP:   "",
			cleanedIPv6: "",
			description: "Shouldn't do anything for an empty bid request",
		},
	}

	for _, test := range testCases {
		// Make a shallow copy
		bidReqCopy := test.bidReq
		applyRegs(&bidReqCopy, test.isAMP, test.applyGDPR, test.applyCOPPA)

		if bidReqCopy.User != nil {
			if test.isAMP && !test.applyCOPPA {
				assert.Equal(t, "abc123", bidReqCopy.User.BuyerUID, test.description)
			} else {
				assert.Empty(t, bidReqCopy.User.BuyerUID, test.description)
			}

			if test.applyCOPPA {
				assert.Empty(t, bidReqCopy.User.ID, test.description)
				assert.Empty(t, bidReqCopy.User.Yob, test.description)
				assert.Empty(t, bidReqCopy.User.Gender, test.description)
			}
		}

		if bidReqCopy.Device != nil {
			if test.applyCOPPA {
				assert.Empty(t, bidReqCopy.Device.MACSHA1, test.description)
			}
			assert.Empty(t, bidReqCopy.Device.DIDMD5, test.description)
			assert.Equal(t, test.cleanedIP, bidReqCopy.Device.IP, test.description)
			assert.Equal(t, test.cleanedIPv6, bidReqCopy.Device.IPv6, test.description)
		}

		// verify original untouched, as we want to only modify the cleaned copy for the bidder
		assert.Equal(t, "abc123", bidReqOrig.User.BuyerUID)
		assert.Equal(t, "teapot", bidReqOrig.Device.DIDMD5)
		assert.Equal(t, "12.123.56.128", bidReqOrig.Device.IP)
		assert.Equal(t, "2001:0db8:85a3:0000:0000:8a2e:0370:7334", bidReqOrig.Device.IPv6)
		assert.Equal(t, 123.4567, bidReqOrig.Device.Geo.Lat)
		assert.Equal(t, 7.9836, bidReqOrig.Device.Geo.Lon)
	}
}
