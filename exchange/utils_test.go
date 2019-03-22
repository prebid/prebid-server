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

func (p *permissionsMock) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	if bidder == "appnexus" {
		return true, nil
	}
	return false, nil
}

func assertReqWithoutAliases(t *testing.T, reqByBidders map[openrtb_ext.BidderName]*openrtb.BidRequest) {
	// assert individual bidder requests
	assert.NotEqual(t, len(reqByBidders), 0, "cleanOpenRTBRequest should split request into individual bidder requests")

	// assert for PI data
	// Only appnexus should be allowed
	for bidderName, bidder := range reqByBidders {
		if bidderName == "appnexus" {
			assert.NotEqual(t, bidder.User.BuyerUID, "", "cleanOpenRTBRequest shouldn't clean PI data for a consented vendor")
		} else {
			assert.Equal(t, bidder.User.BuyerUID, "", "cleanOpenRTBRequest should clean PI data for a non-consented vendor")
		}
	}
}

func assertReqWithAliases(t *testing.T, reqByBidders map[openrtb_ext.BidderName]*openrtb.BidRequest) {
	// assert individual bidder requests
	assert.NotEqual(t, reqByBidders, 0, "cleanOpenRTBRequest should split request into individual bidder requests")

	// assert for PI data
	// Both appnexus and brightroll should be allowed since brightroll
	// is used as an alias for appnexus in the test request
	for bidderName, bidder := range reqByBidders {
		if bidderName == "appnexus" || bidderName == "brightroll" {
			assert.NotEqual(t, bidder.User.BuyerUID, "", "cleanOpenRTBRequest shouldn't clean PI data for a consented vendor")
			assert.NotEqual(t, bidder.Device.DIDMD5, "", "cleanOpenRTBRequest shouldn't clean PI data for a consented vendor")
		} else {
			assert.Equal(t, bidder.User.BuyerUID, "", "cleanOpenRTBRequest should clean PI data for a non-consented vendor")
			assert.Equal(t, bidder.Device.DIDMD5, "", "cleanOpenRTBRequest shouldn't clean PI data for a consented vendor")
		}
	}
}

// Prevents #820
func TestCleanOpenRTBRequests(t *testing.T) {
	testCases := []struct {
		req              *openrtb.BidRequest
		bidReqAssertions func(t *testing.T, reqByBidders map[openrtb_ext.BidderName]*openrtb.BidRequest)
		hasError         bool
	}{
		{req: newRaceCheckingRequest(t), bidReqAssertions: assertReqWithoutAliases, hasError: false},
		{req: newAdapterAliasBidRequest(t), bidReqAssertions: assertReqWithAliases, hasError: false},
	}

	for _, test := range testCases {
		reqByBidders, _, err := cleanOpenRTBRequests(context.Background(), test.req, &emptyUsersync{}, map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels{}, pbsmetrics.Labels{}, &permissionsMock{}, true)
		if test.hasError {
			assert.NotNil(t, err, "Error shouldn't be nil")
		} else {
			assert.Nil(t, err, "Err should be nil")
			test.bidReqAssertions(t, reqByBidders)
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
