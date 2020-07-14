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
type permissionsMock struct {
	personalInfoAllowed bool
}

func (p *permissionsMock) HostCookiesAllowed(ctx context.Context, consent string) (bool, error) {
	return true, nil
}

func (p *permissionsMock) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	return true, nil
}

func (p *permissionsMock) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, consent string) (bool, bool, error) {
	return p.personalInfoAllowed, p.personalInfoAllowed, nil
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
		reqByBidders, _, _, err := cleanOpenRTBRequests(context.Background(), test.req, &emptyUsersync{}, map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels{}, pbsmetrics.Labels{}, &permissionsMock{personalInfoAllowed: true}, true, privacyConfig)
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
		description         string
		ccpaConsent         string
		enforceCCPA         bool
		expectDataScrub     bool
		expectPrivacyLabels pbsmetrics.PrivacyLabels
	}{
		{
			description:     "Feature Flag Enabled - Opt Out",
			ccpaConsent:     "1-Y-",
			enforceCCPA:     true,
			expectDataScrub: true,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				CCPAProvided: true,
				CCPAEnforced: true,
			},
		},
		{
			description:     "Feature Flag Enabled - Opt In",
			ccpaConsent:     "1-N-",
			enforceCCPA:     true,
			expectDataScrub: false,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				CCPAProvided: true,
				CCPAEnforced: false,
			},
		},
		{
			description:     "Feature Flag Disabled",
			ccpaConsent:     "1-Y-",
			enforceCCPA:     false,
			expectDataScrub: false,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				CCPAProvided: false,
				CCPAEnforced: false,
			},
		},
	}

	for _, test := range testCases {
		req := newBidRequest(t)
		req.Regs = &openrtb.Regs{
			Ext: json.RawMessage(`{"us_privacy":"` + test.ccpaConsent + `"}`),
		}

		privacyConfig := config.Privacy{
			CCPA: config.CCPA{
				Enforce: test.enforceCCPA,
			},
		}

		results, _, privacyLabels, errs := cleanOpenRTBRequests(context.Background(), req, &emptyUsersync{}, map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels{}, pbsmetrics.Labels{}, &permissionsMock{personalInfoAllowed: true}, true, privacyConfig)
		result := results["appnexus"]

		assert.Nil(t, errs)
		if test.expectDataScrub {
			assert.Equal(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		} else {
			assert.NotEqual(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		}
		assert.Equal(t, test.expectPrivacyLabels, privacyLabels, test.description+":PrivacyLabels")
	}
}

func TestCleanOpenRTBRequestsCOPPA(t *testing.T) {
	testCases := []struct {
		description         string
		coppa               int8
		expectDataScrub     bool
		expectPrivacyLabels pbsmetrics.PrivacyLabels
	}{
		{
			description:     "Enabled",
			coppa:           1,
			expectDataScrub: true,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				COPPAEnforced: true,
			},
		},
		{
			description:     "Disabled",
			coppa:           0,
			expectDataScrub: false,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				COPPAEnforced: false,
			},
		},
	}

	for _, test := range testCases {
		req := newBidRequest(t)
		req.Regs = &openrtb.Regs{COPPA: test.coppa}

		results, _, privacyLabels, errs := cleanOpenRTBRequests(context.Background(), req, &emptyUsersync{}, map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels{}, pbsmetrics.Labels{}, &permissionsMock{personalInfoAllowed: true}, true, config.Privacy{})
		result := results["appnexus"]

		assert.Nil(t, errs)
		if test.expectDataScrub {
			assert.Equal(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.User.Yob, int64(0), test.description+":User.Yob")
		} else {
			assert.NotEqual(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.User.Yob, int64(0), test.description+":User.Yob")
		}
		assert.Equal(t, test.expectPrivacyLabels, privacyLabels, test.description+":PrivacyLabels")
	}
}

func TestCleanOpenRTBRequestsLMT(t *testing.T) {
	var (
		enabled  int8 = 1
		disabled int8 = 0
	)
	testCases := []struct {
		description         string
		lmt                 *int8
		enforceLMT          bool
		expectDataScrub     bool
		expectPrivacyLabels pbsmetrics.PrivacyLabels
	}{
		{
			description:     "Feature Flag Enabled - OpenTRB Enabled",
			lmt:             &enabled,
			enforceLMT:      true,
			expectDataScrub: true,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				LMTEnforced: true,
			},
		},
		{
			description:     "Feature Flag Disabled - OpenTRB Enabled",
			lmt:             &enabled,
			enforceLMT:      false,
			expectDataScrub: false,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				LMTEnforced: false,
			},
		},
		{
			description:     "Feature Flag Enabled - OpenTRB Disabled",
			lmt:             &disabled,
			enforceLMT:      true,
			expectDataScrub: false,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				LMTEnforced: false,
			},
		},
		{
			description:     "Feature Flag Disabled - OpenTRB Disabled",
			lmt:             &disabled,
			enforceLMT:      false,
			expectDataScrub: false,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				LMTEnforced: false,
			},
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

		results, _, privacyLabels, errs := cleanOpenRTBRequests(context.Background(), req, &emptyUsersync{}, map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels{}, pbsmetrics.Labels{}, &permissionsMock{personalInfoAllowed: true}, true, privacyConfig)
		result := results["appnexus"]

		assert.Nil(t, errs)
		if test.expectDataScrub {
			assert.Equal(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		} else {
			assert.NotEqual(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		}
		assert.Equal(t, test.expectPrivacyLabels, privacyLabels, test.description+":PrivacyLabels")
	}
}

func TestCleanOpenRTBRequestsGDPR(t *testing.T) {
	testCases := []struct {
		description         string
		gdpr                string
		gdprConsent         string
		gdprScrub           bool
		enforceGDPR         bool
		expectPrivacyLabels pbsmetrics.PrivacyLabels
	}{
		{
			description: "Enforce - TCF Invalid",
			gdpr:        "1",
			gdprConsent: "malformed",
			gdprScrub:   false,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: "",
			},
		},
		{
			description: "Enforce - TCF 1",
			gdpr:        "1",
			gdprConsent: "BONV8oqONXwgmADACHENAO7pqzAAppY",
			gdprScrub:   true,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: pbsmetrics.TCFVersionV1,
			},
		},
		{
			description: "Enforce - TCF 2",
			gdpr:        "1",
			gdprConsent: "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			gdprScrub:   true,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: pbsmetrics.TCFVersionV2,
			},
		},
		{
			description: "Not Enforce - TCF 1",
			gdpr:        "0",
			gdprConsent: "BONV8oqONXwgmADACHENAO7pqzAAppY",
			gdprScrub:   false,
			expectPrivacyLabels: pbsmetrics.PrivacyLabels{
				GDPREnforced:   false,
				GDPRTCFVersion: "",
			},
		},
	}

	for _, test := range testCases {
		req := newBidRequest(t)
		req.User.Ext = json.RawMessage(`{"consent":"` + test.gdprConsent + `"}`)
		req.Regs = &openrtb.Regs{
			Ext: json.RawMessage(`{"gdpr":` + test.gdpr + `}`),
		}

		privacyConfig := config.Privacy{
			GDPR: config.GDPR{
				TCF2: config.TCF2{
					Enabled: true,
				},
			},
		}

		results, _, privacyLabels, errs := cleanOpenRTBRequests(context.Background(), req, &emptyUsersync{}, map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels{}, pbsmetrics.Labels{}, &permissionsMock{personalInfoAllowed: !test.gdprScrub}, true, privacyConfig)
		result := results["appnexus"]

		assert.Nil(t, errs)
		if test.gdprScrub {
			assert.Equal(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		} else {
			assert.NotEqual(t, result.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		}
		assert.Equal(t, test.expectPrivacyLabels, privacyLabels, test.description+":PrivacyLabels")
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
			Yob:      1982,
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
