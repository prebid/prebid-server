package exchange

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
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

func (p *permissionsMock) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, consent string) (bool, bool, error) {
	if bidder == "appnexus" {
		return true, true, nil
	}
	return false, false, nil
}

func (p *permissionsMock) AMPException() bool {
	return false
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
			assert.NotEqual(t, bidder.User.BuyerUID, "", "cleanOpenRTBRequest shouldn't clean PI data as per COPPA or for a consented vendor as per GDPR or per CCPA")
			assert.NotEqual(t, bidder.Device.DIDMD5, "", "cleanOpenRTBRequest shouldn't clean PI data as per COPPA or for a consented vendor as per GDPR or per CCPA")
		} else {
			assert.Equal(t, bidder.User.BuyerUID, "", "cleanOpenRTBRequest should clean PI data as per COPPA or for a non-consented vendor as per GDPR or per CCPA", bidderName.String())
			assert.Equal(t, bidder.Device.DIDMD5, "", "cleanOpenRTBRequest should clean PI data as per COPPA or for a non-consented vendor as per GDPR or per CCPA", bidderName.String())
		}
	}
}

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

	privacyConfig := config.Privacy{
		CCPA: config.CCPA{
			Enforce: true,
		},
		LMT: config.LMT{
			Enforce: true,
		},
	}

	for _, test := range testCases {
		reqByBidders, _, _, err := cleanOpenRTBRequests(context.Background(), test.req, &emptyUsersync{}, map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels{}, pbsmetrics.Labels{}, &permissionsMock{}, true, privacyConfig)
		if test.hasError {
			assert.NotNil(t, err, "Error shouldn't be nil")
		} else {
			assert.Nil(t, err, "Err should be nil")
			test.bidReqAssertions(t, reqByBidders, test.applyCOPPA, test.consentedVendors)
		}
	}
}

func TestCleanOpenRTBRequestsCCPA(t *testing.T) {
	testCases := []struct {
		description     string
		enforceCCPA     bool
		expectDataScrub bool
	}{
		{
			description:     "Feature Flag Enabled",
			enforceCCPA:     true,
			expectDataScrub: true,
		},
		{
			description:     "Feature Flag Disabled",
			enforceCCPA:     false,
			expectDataScrub: false,
		},
	}

	for _, test := range testCases {
		req := newBidRequest(t)
		req.Regs = &openrtb.Regs{
			Ext: json.RawMessage(`{"us_privacy":"1-Y-"}`),
		}

		privacyConfig := config.Privacy{
			CCPA: config.CCPA{
				Enforce: test.enforceCCPA,
			},
		}

		results, _, _, errs := cleanOpenRTBRequests(context.Background(), req, &emptyUsersync{}, map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels{}, pbsmetrics.Labels{}, &permissionsMock{}, true, privacyConfig)
		result := results["appnexus"]

		assert.Nil(t, errs)

		if test.expectDataScrub {
			assert.Equal(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		} else {
			assert.NotEqual(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		}
	}
}

func TestCleanOpenRTBRequestsSChain(t *testing.T) {
	testCases := []struct {
		description  string
		inSourceExt  json.RawMessage
		inExt        json.RawMessage
		outSourceExt json.RawMessage
		outExt       json.RawMessage
		hasError     bool
	}{
		{
			description:  "Empty root ext and source ext",
			inSourceExt:  json.RawMessage(``),
			inExt:        json.RawMessage(``),
			outSourceExt: json.RawMessage(``),
			outExt:       json.RawMessage(``),
			hasError:     false,
		},
		{
			description:  "No schains in root ext and empty source ext",
			inSourceExt:  json.RawMessage(``),
			inExt:        json.RawMessage(`{"prebid":{"schains":[]}}`),
			outSourceExt: json.RawMessage(``),
			outExt:       json.RawMessage(`{"prebid":{}}`),
			hasError:     false,
		},
		{
			description:  "Use source schain -- no bidder schain or wildcard schain in ext.prebid.schains",
			inSourceExt:  json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"example.com","sid":"example1","rid":"ExampleReq1","hp":1}],"ver":"1.0"}}`),
			inExt:        json.RawMessage(`{"prebid":{"schains":[{"bidders":["bidder1"],"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}]}}`),
			outSourceExt: json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"example.com","sid":"example1","rid":"ExampleReq1","hp":1}],"ver":"1.0"}}`),
			outExt:       json.RawMessage(`{"prebid":{}}`),
			hasError:     false,
		},
		{
			description:  "Use schain for bidder in ext.prebid.schains",
			inSourceExt:  json.RawMessage(``),
			inExt:        json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}]}}`),
			outSourceExt: json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}`),
			outExt:       json.RawMessage(`{"prebid":{}}`),
			hasError:     false,
		},
		{
			description:  "Use wildcard schain in ext.prebid.schains",
			inSourceExt:  json.RawMessage(``),
			inExt:        json.RawMessage(`{"prebid":{"schains":[{"bidders":["*"],"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}]}}`),
			outSourceExt: json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}`),
			outExt:       json.RawMessage(`{"prebid":{}}`),
			hasError:     false,
		},
		{
			description:  "Use schain for bidder in ext.prebid.schains instead of wildcard",
			inSourceExt:  json.RawMessage(``),
			inExt:        json.RawMessage(`{"prebid":{"aliases":{"appnexus":"alias1"},"schains":[{"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}, {"bidders":["*"],"schain":{"complete":1,"nodes":[{"asi":"wildcard.com","sid":"wildcard1","rid":"WildcardReq1","hp":1}],"ver":"1.0"}} ]}}`),
			outSourceExt: json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}`),
			outExt:       json.RawMessage(`{"prebid":{"aliases":{"appnexus":"alias1"}}}`),
			hasError:     false,
		},
		{
			description:  "Use source schain -- multiple (two) bidder schains in ext.prebid.schains",
			inSourceExt:  json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"example.com","sid":"example1","rid":"ExampleReq1","hp":1}],"ver":"1.0"}}`),
			inExt:        json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller1.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}, {"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller2.com","sid":"00002","rid":"BidRequest2","hp":1}],"ver":"1.0"}}]}}`),
			outSourceExt: nil,
			outExt:       nil,
			hasError:     true,
		},
	}

	for _, test := range testCases {
		req := newBidRequest(t)
		req.Source.Ext = test.inSourceExt
		req.Ext = test.inExt

		results, _, _, errs := cleanOpenRTBRequests(context.Background(), req, &emptyUsersync{}, map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels{}, pbsmetrics.Labels{}, &permissionsMock{}, true, config.Privacy{})
		result := results["appnexus"]

		if test.hasError == true {
			assert.NotNil(t, errs)
			assert.Nil(t, result)
		} else {
			assert.Nil(t, errs)
			assert.Equal(t, test.outSourceExt, result.Source.Ext, test.description+":Source.Ext")
			assert.Equal(t, test.outExt, result.Ext, test.description+":Ext")
		}
	}
}

func TestCleanOpenRTBRequestsLMT(t *testing.T) {
	var (
		enabled  int8 = 1
		disabled int8 = 0
	)
	testCases := []struct {
		description     string
		lmt             *int8
		enforceLMT      bool
		expectDataScrub bool
	}{
		{
			description:     "Feature Flag Enabled - OpenTRB Enabled",
			lmt:             &enabled,
			enforceLMT:      true,
			expectDataScrub: true,
		},
		{
			description:     "Feature Flag Disabled - OpenTRB Enabled",
			lmt:             &enabled,
			enforceLMT:      false,
			expectDataScrub: false,
		},
		{
			description:     "Feature Flag Enabled - OpenTRB Disabled",
			lmt:             &disabled,
			enforceLMT:      true,
			expectDataScrub: false,
		},
		{
			description:     "Feature Flag Disabled - OpenTRB Disabled",
			lmt:             &disabled,
			enforceLMT:      false,
			expectDataScrub: false,
		},
	}

	for _, test := range testCases {
		req := newBidRequest(t)
		req.Device.Lmt = test.lmt

		privacyConfig := config.Privacy{
			LMT: config.LMT{
				Enforce: test.enforceLMT,
			},
		}

		results, _, _, errs := cleanOpenRTBRequests(context.Background(), req, &emptyUsersync{}, map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels{}, pbsmetrics.Labels{}, &permissionsMock{}, true, privacyConfig)
		result := results["appnexus"]

		assert.Nil(t, errs)

		if test.expectDataScrub {
			assert.Equal(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		} else {
			assert.NotEqual(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.Device.DIDMD5, "", test.description+":Device.DIDMD5")
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
			Ext: json.RawMessage(`{"appnexus": {"placementId": 1},"brightroll": {"placementId": 105}}`),
		}},
		Ext: json.RawMessage(`{"prebid":{"aliases":{"brightroll":"appnexus"}}}`),
	}
}

func newBidRequest(t *testing.T) *openrtb.BidRequest {
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
			Language: "EN",
		},
		Source: &openrtb.Source{
			TID: "61018dc9-fa61-4c41-b7dc-f90b9ae80e87",
		},
		User: &openrtb.User{
			ID:       "our-id",
			BuyerUID: "their-id",
			Ext:      json.RawMessage(`{"digitrust":{"id":"digi-id","keyv":1,"pref":1}}`),
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
			Ext: json.RawMessage(`{"appnexus": {"placementId": 1}}`),
		}},
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

func TestBidderToPrebidChains(t *testing.T) {
	input := openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			SChains: []*openrtb_ext.ExtRequestPrebidSChain{
				{
					Bidders: []string{"Bidder1", "Bidder2"},
					SChain: openrtb_ext.ExtRequestPrebidSChainSChain{
						Complete: 1,
						Nodes: []*openrtb_ext.ExtRequestPrebidSChainSChainNode{
							{
								ASI:    "asi1",
								SID:    "sid1",
								Name:   "name1",
								RID:    "rid1",
								Domain: "domain1",
								HP:     1,
							},
							{
								ASI:    "asi2",
								SID:    "sid2",
								Name:   "name2",
								RID:    "rid2",
								Domain: "domain2",
								HP:     2,
							},
						},
						Ver: "version1",
					},
				},
				{
					Bidders: []string{"Bidder3", "Bidder4"},
					SChain:  openrtb_ext.ExtRequestPrebidSChainSChain{},
				},
			},
		},
	}

	output, err := BidderToPrebidSChains(&input)

	assert.Nil(t, err)
	assert.Equal(t, len(output), 4)
	assert.Same(t, output["Bidder1"], &input.Prebid.SChains[0].SChain)
	assert.Same(t, output["Bidder2"], &input.Prebid.SChains[0].SChain)
	assert.Same(t, output["Bidder3"], &input.Prebid.SChains[1].SChain)
	assert.Same(t, output["Bidder4"], &input.Prebid.SChains[1].SChain)
}

func TestBidderToPrebidChainsDiscardMultipleChainsForBidder(t *testing.T) {
	input := openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			SChains: []*openrtb_ext.ExtRequestPrebidSChain{
				{
					Bidders: []string{"Bidder1"},
					SChain:  openrtb_ext.ExtRequestPrebidSChainSChain{},
				},
				{
					Bidders: []string{"Bidder1", "Bidder2"},
					SChain:  openrtb_ext.ExtRequestPrebidSChainSChain{},
				},
			},
		},
	}

	output, err := BidderToPrebidSChains(&input)

	assert.NotNil(t, err)
	assert.Nil(t, output)
}

func TestBidderToPrebidChainsNilSChains(t *testing.T) {
	input := openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			SChains: nil,
		},
	}

	output, err := BidderToPrebidSChains(&input)

	assert.Nil(t, err)
	assert.Equal(t, len(output), 0)
}

func TestBidderToPrebidChainsZeroLengthSChains(t *testing.T) {
	input := openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			SChains: []*openrtb_ext.ExtRequestPrebidSChain{},
		},
	}

	output, err := BidderToPrebidSChains(&input)

	assert.Nil(t, err)
	assert.Equal(t, len(output), 0)
}
