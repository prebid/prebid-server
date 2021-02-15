package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

// permissionsMock mocks the Permissions interface for tests
//
// It only allows appnexus for GDPR consent
type permissionsMock struct {
	personalInfoAllowed      bool
	personalInfoAllowedError error
}

func (p *permissionsMock) HostCookiesAllowed(ctx context.Context, gdpr gdpr.Signal, consent string) (bool, error) {
	return true, nil
}

func (p *permissionsMock) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, gdpr gdpr.Signal, consent string) (bool, error) {
	return true, nil
}

func (p *permissionsMock) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, gdpr gdpr.Signal, consent string) (bool, bool, bool, error) {
	return p.personalInfoAllowed, p.personalInfoAllowed, p.personalInfoAllowed, p.personalInfoAllowedError
}

func assertReq(t *testing.T, bidderRequests []BidderRequest,
	applyCOPPA bool, consentedVendors map[string]bool) {
	// assert individual bidder requests
	assert.NotEqual(t, bidderRequests, 0, "cleanOpenRTBRequest should split request into individual bidder requests")

	// assert for PI data
	// Both appnexus and brightroll should be allowed since brightroll
	// is used as an alias for appnexus in the test request
	for _, req := range bidderRequests {
		if !applyCOPPA && consentedVendors[req.BidderName.String()] {
			assert.NotEqual(t, req.BidRequest.User.BuyerUID, "", "cleanOpenRTBRequest shouldn't clean PI data as per COPPA or for a consented vendor as per GDPR or per CCPA")
			assert.NotEqual(t, req.BidRequest.Device.DIDMD5, "", "cleanOpenRTBRequest shouldn't clean PI data as per COPPA or for a consented vendor as per GDPR or per CCPA")
		} else {
			assert.Equal(t, req.BidRequest.User.BuyerUID, "", "cleanOpenRTBRequest should clean PI data as per COPPA or for a non-consented vendor as per GDPR or per CCPA", req.BidderName.String())
			assert.Equal(t, req.BidRequest.Device.DIDMD5, "", "cleanOpenRTBRequest should clean PI data as per COPPA or for a non-consented vendor as per GDPR or per CCPA", req.BidderName.String())
		}
	}
}

func TestCleanOpenRTBRequests(t *testing.T) {
	testCases := []struct {
		req              AuctionRequest
		bidReqAssertions func(t *testing.T, bidderRequests []BidderRequest,
			applyCOPPA bool, consentedVendors map[string]bool)
		hasError         bool
		applyCOPPA       bool
		consentedVendors map[string]bool
	}{
		{
			req:              AuctionRequest{BidRequest: newRaceCheckingRequest(t), UserSyncs: &emptyUsersync{}},
			bidReqAssertions: assertReq,
			hasError:         false,
			applyCOPPA:       true,
			consentedVendors: map[string]bool{"appnexus": true},
		},
		{
			req:              AuctionRequest{BidRequest: newAdapterAliasBidRequest(t), UserSyncs: &emptyUsersync{}},
			bidReqAssertions: assertReq,
			hasError:         false,
			applyCOPPA:       false,
			consentedVendors: map[string]bool{"appnexus": true, "brightroll": true},
		},
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
		bidderRequests, _, err := cleanOpenRTBRequests(context.Background(), test.req, nil, &permissionsMock{personalInfoAllowed: true}, true, privacyConfig)
		if test.hasError {
			assert.NotNil(t, err, "Error shouldn't be nil")
		} else {
			assert.Nil(t, err, "Err should be nil")
			test.bidReqAssertions(t, bidderRequests, test.applyCOPPA, test.consentedVendors)
		}
	}
}

func TestCleanOpenRTBRequestsCCPA(t *testing.T) {
	trueValue, falseValue := true, false

	testCases := []struct {
		description         string
		reqExt              json.RawMessage
		ccpaConsent         string
		ccpaHostEnabled     bool
		ccpaAccountEnabled  *bool
		expectDataScrub     bool
		expectPrivacyLabels metrics.PrivacyLabels
	}{
		{
			description:        "Feature Flags Enabled - Opt Out",
			ccpaConsent:        "1-Y-",
			ccpaHostEnabled:    true,
			ccpaAccountEnabled: &trueValue,
			expectDataScrub:    true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				CCPAProvided: true,
				CCPAEnforced: true,
			},
		},
		{
			description:        "Feature Flags Enabled - Opt In",
			ccpaConsent:        "1-N-",
			ccpaHostEnabled:    true,
			ccpaAccountEnabled: &trueValue,
			expectDataScrub:    false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				CCPAProvided: true,
				CCPAEnforced: false,
			},
		},
		{
			description:        "Feature Flags Enabled - No Sale Star - Doesn't Scrub",
			reqExt:             json.RawMessage(`{"prebid":{"nosale":["*"]}}`),
			ccpaConsent:        "1-Y-",
			ccpaHostEnabled:    true,
			ccpaAccountEnabled: &trueValue,
			expectDataScrub:    false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				CCPAProvided: true,
				CCPAEnforced: false,
			},
		},
		{
			description:        "Feature Flags Enabled - No Sale Specific Bidder - Doesn't Scrub",
			reqExt:             json.RawMessage(`{"prebid":{"nosale":["appnexus"]}}`),
			ccpaConsent:        "1-Y-",
			ccpaHostEnabled:    true,
			ccpaAccountEnabled: &trueValue,
			expectDataScrub:    false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				CCPAProvided: true,
				CCPAEnforced: true,
			},
		},
		{
			description:        "Feature Flags Enabled - No Sale Different Bidder - Scrubs",
			reqExt:             json.RawMessage(`{"prebid":{"nosale":["rubicon"]}}`),
			ccpaConsent:        "1-Y-",
			ccpaHostEnabled:    true,
			ccpaAccountEnabled: &trueValue,
			expectDataScrub:    true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				CCPAProvided: true,
				CCPAEnforced: true,
			},
		},
		{
			description:        "Feature flags Account CCPA enabled, host CCPA disregarded - Opt Out",
			ccpaConsent:        "1-Y-",
			ccpaHostEnabled:    false,
			ccpaAccountEnabled: &trueValue,
			expectDataScrub:    true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				CCPAProvided: true,
				CCPAEnforced: true,
			},
		},
		{
			description:        "Feature flags Account CCPA disabled, host CCPA disregarded",
			ccpaConsent:        "1-Y-",
			ccpaHostEnabled:    true,
			ccpaAccountEnabled: &falseValue,
			expectDataScrub:    false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				CCPAProvided: true,
				CCPAEnforced: false,
			},
		},
		{
			description:        "Feature flags Account CCPA not specified, host CCPA enabled - Opt Out",
			ccpaConsent:        "1-Y-",
			ccpaHostEnabled:    true,
			ccpaAccountEnabled: nil,
			expectDataScrub:    true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				CCPAProvided: true,
				CCPAEnforced: true,
			},
		},
		{
			description:        "Feature flags Account CCPA not specified, host CCPA disabled",
			ccpaConsent:        "1-Y-",
			ccpaHostEnabled:    false,
			ccpaAccountEnabled: nil,
			expectDataScrub:    false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				CCPAProvided: true,
				CCPAEnforced: false,
			},
		},
	}

	for _, test := range testCases {
		req := newBidRequest(t)
		req.Ext = test.reqExt
		req.Regs = &openrtb.Regs{
			Ext: json.RawMessage(`{"us_privacy":"` + test.ccpaConsent + `"}`),
		}

		privacyConfig := config.Privacy{
			CCPA: config.CCPA{
				Enforce: test.ccpaHostEnabled,
			},
		}

		accountConfig := config.Account{
			CCPA: config.AccountCCPA{
				Enabled: test.ccpaAccountEnabled,
			},
		}

		auctionReq := AuctionRequest{
			BidRequest: req,
			UserSyncs:  &emptyUsersync{},
			Account:    accountConfig,
		}

		bidderRequests, privacyLabels, errs := cleanOpenRTBRequests(
			context.Background(),
			auctionReq,
			nil,
			&permissionsMock{personalInfoAllowed: true},
			true,
			privacyConfig)
		result := bidderRequests[0]

		assert.Nil(t, errs)
		if test.expectDataScrub {
			assert.Equal(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.BidRequest.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		} else {
			assert.NotEqual(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.BidRequest.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		}
		assert.Equal(t, test.expectPrivacyLabels, privacyLabels, test.description+":PrivacyLabels")
	}
}

func TestCleanOpenRTBRequestsCCPAErrors(t *testing.T) {
	testCases := []struct {
		description string
		reqExt      json.RawMessage
		reqRegsExt  json.RawMessage
		expectError error
	}{
		{
			description: "Invalid Consent",
			reqExt:      json.RawMessage(`{"prebid":{"nosale":["*"]}}`),
			reqRegsExt:  json.RawMessage(`{"us_privacy":"malformed"}`),
			expectError: &errortypes.InvalidPrivacyConsent{"request.regs.ext.us_privacy must contain 4 characters"},
		},
		{
			description: "Invalid No Sale Bidders",
			reqExt:      json.RawMessage(`{"prebid":{"nosale":["*", "another"]}}`),
			reqRegsExt:  json.RawMessage(`{"us_privacy":"1NYN"}`),
			expectError: errors.New("request.ext.prebid.nosale is invalid: can only specify all bidders if no other bidders are provided"),
		},
	}

	for _, test := range testCases {
		req := newBidRequest(t)
		req.Ext = test.reqExt
		req.Regs = &openrtb.Regs{Ext: test.reqRegsExt}

		var reqExtStruct openrtb_ext.ExtRequest
		err := json.Unmarshal(req.Ext, &reqExtStruct)
		assert.NoError(t, err, test.description+":marshal_ext")

		auctionReq := AuctionRequest{
			BidRequest: req,
			UserSyncs:  &emptyUsersync{},
		}

		privacyConfig := config.Privacy{
			CCPA: config.CCPA{
				Enforce: true,
			},
		}
		_, _, errs := cleanOpenRTBRequests(context.Background(), auctionReq, &reqExtStruct, &permissionsMock{personalInfoAllowed: true}, true, privacyConfig)

		assert.ElementsMatch(t, []error{test.expectError}, errs, test.description)
	}
}

func TestCleanOpenRTBRequestsCOPPA(t *testing.T) {
	testCases := []struct {
		description         string
		coppa               int8
		expectDataScrub     bool
		expectPrivacyLabels metrics.PrivacyLabels
	}{
		{
			description:     "Enabled",
			coppa:           1,
			expectDataScrub: true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				COPPAEnforced: true,
			},
		},
		{
			description:     "Disabled",
			coppa:           0,
			expectDataScrub: false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				COPPAEnforced: false,
			},
		},
	}

	for _, test := range testCases {
		req := newBidRequest(t)
		req.Regs = &openrtb.Regs{COPPA: test.coppa}

		auctionReq := AuctionRequest{
			BidRequest: req,
			UserSyncs:  &emptyUsersync{},
		}

		bidderRequests, privacyLabels, errs := cleanOpenRTBRequests(context.Background(), auctionReq, nil, &permissionsMock{personalInfoAllowed: true}, true, config.Privacy{})
		result := bidderRequests[0]

		assert.Nil(t, errs)
		if test.expectDataScrub {
			assert.Equal(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.BidRequest.User.Yob, int64(0), test.description+":User.Yob")
		} else {
			assert.NotEqual(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.BidRequest.User.Yob, int64(0), test.description+":User.Yob")
		}
		assert.Equal(t, test.expectPrivacyLabels, privacyLabels, test.description+":PrivacyLabels")
	}
}

func TestCleanOpenRTBRequestsSChain(t *testing.T) {
	testCases := []struct {
		description   string
		inExt         json.RawMessage
		inSourceExt   json.RawMessage
		outSourceExt  json.RawMessage
		outRequestExt json.RawMessage
		hasError      bool
	}{
		{
			description:   "Empty root ext and source ext, nil unmarshaled ext",
			inExt:         nil,
			inSourceExt:   json.RawMessage(``),
			outSourceExt:  json.RawMessage(``),
			outRequestExt: json.RawMessage(``),
			hasError:      false,
		},
		{
			description:   "Empty root ext, source ext, and unmarshaled ext",
			inExt:         json.RawMessage(``),
			inSourceExt:   json.RawMessage(``),
			outSourceExt:  json.RawMessage(``),
			outRequestExt: json.RawMessage(``),
			hasError:      false,
		},
		{
			description:   "No schains in root ext and empty source ext. Unmarshaled ext is equivalent to root ext",
			inSourceExt:   json.RawMessage(``),
			inExt:         json.RawMessage(`{"prebid":{"schains":[]}}`),
			outSourceExt:  json.RawMessage(``),
			outRequestExt: json.RawMessage(`{"prebid":{}}`),
			hasError:      false,
		},
		{
			description:   "Use source schain -- no bidder schain or wildcard schain in ext.prebid.schains. Unmarshaled ext is equivalent to root ext",
			inSourceExt:   json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"example.com","sid":"example1","rid":"ExampleReq1","hp":1}],"ver":"1.0"}}`),
			inExt:         json.RawMessage(`{"prebid":{"schains":[{"bidders":["bidder1"],"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}]}}`),
			outSourceExt:  json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"example.com","sid":"example1","rid":"ExampleReq1","hp":1}],"ver":"1.0"}}`),
			outRequestExt: json.RawMessage(`{"prebid":{}}`),
			hasError:      false,
		},
		{
			description:   "Use schain for bidder in ext.prebid.schains. Unmarshaled ext is equivalent to root ext",
			inSourceExt:   json.RawMessage(``),
			inExt:         json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}]}}`),
			outSourceExt:  json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}`),
			outRequestExt: json.RawMessage(`{"prebid":{}}`),
			hasError:      false,
		},
		{
			description:   "Use wildcard schain in ext.prebid.schains. Unmarshaled ext is equivalent to root ext",
			inSourceExt:   json.RawMessage(``),
			inExt:         json.RawMessage(`{"prebid":{"schains":[{"bidders":["*"],"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}]}}`),
			outSourceExt:  json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}`),
			outRequestExt: json.RawMessage(`{"prebid":{}}`),
			hasError:      false,
		},
		{
			description:   "Use schain for bidder in ext.prebid.schains instead of wildcard. Unmarshaled ext is equivalent to root ext",
			inSourceExt:   json.RawMessage(``),
			inExt:         json.RawMessage(`{"prebid":{"aliases":{"appnexus":"alias1"},"schains":[{"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}, {"bidders":["*"],"schain":{"complete":1,"nodes":[{"asi":"wildcard.com","sid":"wildcard1","rid":"WildcardReq1","hp":1}],"ver":"1.0"}} ]}}`),
			outSourceExt:  json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"directseller.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}`),
			outRequestExt: json.RawMessage(`{"prebid":{"aliases":{"appnexus":"alias1"}}}`),
			hasError:      false,
		},
		{
			description:   "Use source schain -- multiple (two) bidder schains in ext.prebid.schains. Unmarshaled ext is equivalent to root ext",
			inSourceExt:   json.RawMessage(`{"schain":{"complete":1,"nodes":[{"asi":"example.com","sid":"example1","rid":"ExampleReq1","hp":1}],"ver":"1.0"}}`),
			inExt:         json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller1.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}, {"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller2.com","sid":"00002","rid":"BidRequest2","hp":1}],"ver":"1.0"}}]}}`),
			outSourceExt:  nil,
			outRequestExt: nil,
			hasError:      true,
		},
	}

	for _, test := range testCases {
		req := newBidRequest(t)
		req.Source.Ext = test.inSourceExt

		var extRequest *openrtb_ext.ExtRequest
		if test.inExt != nil {
			req.Ext = test.inExt
			unmarshaledExt, err := extractBidRequestExt(req)
			assert.NoErrorf(t, err, test.description+":Error unmarshaling inExt")
			extRequest = unmarshaledExt
		}

		auctionReq := AuctionRequest{
			BidRequest: req,
			UserSyncs:  &emptyUsersync{},
		}

		bidderRequests, _, errs := cleanOpenRTBRequests(context.Background(), auctionReq, extRequest, &permissionsMock{}, true, config.Privacy{})
		if test.hasError == true {
			assert.NotNil(t, errs)
			assert.Len(t, bidderRequests, 0)
		} else {
			result := bidderRequests[0]
			assert.Nil(t, errs)
			assert.Equal(t, test.outSourceExt, result.BidRequest.Source.Ext, test.description+":Source.Ext")
			assert.Equal(t, test.outRequestExt, result.BidRequest.Ext, test.description+":Ext")
		}
	}
}

func TestExtractBidRequestExt(t *testing.T) {
	var boolFalse, boolTrue *bool = new(bool), new(bool)
	*boolFalse = false
	*boolTrue = true

	testCases := []struct {
		desc          string
		inBidRequest  *openrtb.BidRequest
		outRequestExt *openrtb_ext.ExtRequest
		outError      error
	}{
		{
			desc:         "Valid vastxml.returnCreative set to false",
			inBidRequest: &openrtb.BidRequest{Ext: json.RawMessage(`{"prebid":{"debug":true,"cache":{"vastxml":{"returnCreative":false}}}}`)},
			outRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Debug: true,
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{
							ReturnCreative: boolFalse,
						},
					},
				},
			},
			outError: nil,
		},
		{
			desc:         "Valid vastxml.returnCreative set to true",
			inBidRequest: &openrtb.BidRequest{Ext: json.RawMessage(`{"prebid":{"debug":true,"cache":{"vastxml":{"returnCreative":true}}}}`)},
			outRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Debug: true,
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{
							ReturnCreative: boolTrue,
						},
					},
				},
			},
			outError: nil,
		},
		{
			desc:          "bidRequest nil, we expect an error",
			inBidRequest:  nil,
			outRequestExt: &openrtb_ext.ExtRequest{},
			outError:      fmt.Errorf("Error bidRequest should not be nil"),
		},
		{
			desc:          "Non-nil bidRequest with empty Ext, we expect a blank requestExt",
			inBidRequest:  &openrtb.BidRequest{},
			outRequestExt: &openrtb_ext.ExtRequest{},
			outError:      nil,
		},
		{
			desc:          "Non-nil bidRequest with non-empty, invalid Ext, we expect unmarshaling error",
			inBidRequest:  &openrtb.BidRequest{Ext: json.RawMessage(`invalid`)},
			outRequestExt: &openrtb_ext.ExtRequest{},
			outError:      fmt.Errorf("Error decoding Request.ext : invalid character 'i' looking for beginning of value"),
		},
	}
	for _, test := range testCases {
		actualRequestExt, actualErr := extractBidRequestExt(test.inBidRequest)

		// Given that assert.Equal asserts pointer variable equality based on the equality of the referenced values (as opposed to
		// the memory addresses) the call below asserts whether or not *test.outRequestExt.Prebid.Cache.VastXML.ReturnCreative boolean
		// value is equal to that of *actualRequestExt.Prebid.Cache.VastXML.ReturnCreative
		assert.Equal(t, test.outRequestExt, actualRequestExt, "%s. Unexpected RequestExt value. \n", test.desc)
		assert.Equal(t, test.outError, actualErr, "%s. Unexpected error value. \n", test.desc)
	}
}

func TestGetExtCacheInstructions(t *testing.T) {
	var boolFalse, boolTrue *bool = new(bool), new(bool)
	*boolFalse = false
	*boolTrue = true

	testCases := []struct {
		desc                 string
		inRequestExt         *openrtb_ext.ExtRequest
		outCacheInstructions extCacheInstructions
	}{
		{
			desc:         "Nil inRequestExt, all cache flags false except for returnCreative that defaults to true",
			inRequestExt: nil,
			outCacheInstructions: extCacheInstructions{
				cacheBids:      false,
				cacheVAST:      false,
				returnCreative: true,
			},
		},
		{
			desc:         "Non-nil inRequestExt, nil Cache field, all cache flags false except for returnCreative that defaults to true",
			inRequestExt: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Cache: nil}},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      false,
				cacheVAST:      false,
				returnCreative: true,
			},
		},
		{
			desc: "Non-nil Cache field, both ExtRequestPrebidCacheBids and ExtRequestPrebidCacheVAST nil returnCreative that defaults to true",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    nil,
						VastXML: nil,
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      false,
				cacheVAST:      false,
				returnCreative: true,
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheVAST with unspecified ReturnCreative field, cacheVAST = true and returnCreative defaults to true",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    nil,
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{},
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      false,
				cacheVAST:      true,
				returnCreative: true, // default value
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheVAST where ReturnCreative is set to false, cacheVAST = true and returnCreative = false",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    nil,
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolFalse},
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      false,
				cacheVAST:      true,
				returnCreative: false,
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheVAST where ReturnCreative is set to true, cacheVAST = true and returnCreative = true",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    nil,
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolTrue},
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      false,
				cacheVAST:      true,
				returnCreative: true,
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheBids with unspecified ReturnCreative field, cacheBids = true and returnCreative defaults to true",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{},
						VastXML: nil,
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      true,
				cacheVAST:      false,
				returnCreative: true, // default value
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheBids where ReturnCreative is set to false, cacheBids = true and returnCreative  = false",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolFalse},
						VastXML: nil,
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      true,
				cacheVAST:      false,
				returnCreative: false,
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheBids where ReturnCreative is set to true, cacheBids = true and returnCreative  = true",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolTrue},
						VastXML: nil,
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      true,
				cacheVAST:      false,
				returnCreative: true,
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheBids and ExtRequest.Cache.ExtRequestPrebidCacheVAST, neither specify a ReturnCreative field value, all extCacheInstructions fields set to true",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{},
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{},
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      true,
				cacheVAST:      true,
				returnCreative: true,
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheBids and ExtRequest.Cache.ExtRequestPrebidCacheVAST sets ReturnCreative to true, all extCacheInstructions fields set to true",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{},
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolTrue},
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      true,
				cacheVAST:      true,
				returnCreative: true,
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheBids and ExtRequest.Cache.ExtRequestPrebidCacheVAST sets ReturnCreative to false, returnCreative = false",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{},
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolFalse},
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      true,
				cacheVAST:      true,
				returnCreative: false,
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheVAST and ExtRequest.Cache.ExtRequestPrebidCacheBids sets ReturnCreative to true, all extCacheInstructions fields set to true",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolTrue},
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{},
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      true,
				cacheVAST:      true,
				returnCreative: true,
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheVAST and ExtRequest.Cache.ExtRequestPrebidCacheBids sets ReturnCreative to false, returnCreative = false",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolFalse},
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{},
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      true,
				cacheVAST:      true,
				returnCreative: false,
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheVAST and ExtRequest.Cache.ExtRequestPrebidCacheBids set different ReturnCreative values, returnCreative = true because one of them is true",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolFalse},
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolTrue},
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      true,
				cacheVAST:      true,
				returnCreative: true,
			},
		},
		{
			desc: "Non-nil ExtRequest.Cache.ExtRequestPrebidCacheVAST and ExtRequest.Cache.ExtRequestPrebidCacheBids set different ReturnCreative values, returnCreative = true because one of them is true",
			inRequestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Cache: &openrtb_ext.ExtRequestPrebidCache{
						Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolTrue},
						VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolFalse},
					},
				},
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      true,
				cacheVAST:      true,
				returnCreative: true,
			},
		},
	}

	for _, test := range testCases {
		cacheInstructions := getExtCacheInstructions(test.inRequestExt)

		assert.Equal(t, test.outCacheInstructions.cacheBids, cacheInstructions.cacheBids, "%s. Unexpected shouldCacheBids value. \n", test.desc)
		assert.Equal(t, test.outCacheInstructions.cacheVAST, cacheInstructions.cacheVAST, "%s. Unexpected shouldCacheVAST value. \n", test.desc)
		assert.Equal(t, test.outCacheInstructions.returnCreative, cacheInstructions.returnCreative, "%s. Unexpected returnCreative value. \n", test.desc)
	}
}

func TestGetExtTargetData(t *testing.T) {
	type inTest struct {
		requestExt        *openrtb_ext.ExtRequest
		cacheInstructions *extCacheInstructions
	}
	type outTest struct {
		targetData    *targetData
		nilTargetData bool
	}
	testCases := []struct {
		desc string
		in   inTest
		out  outTest
	}{
		{
			"nil requestExt, nil outTargetData",
			inTest{
				requestExt: nil,
				cacheInstructions: &extCacheInstructions{
					cacheBids: true,
					cacheVAST: true,
				},
			},
			outTest{targetData: nil, nilTargetData: true},
		},
		{
			"Valid requestExt, nil Targeting field, nil outTargetData",
			inTest{
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Targeting: nil,
					},
				},
				cacheInstructions: &extCacheInstructions{
					cacheBids: true,
					cacheVAST: true,
				},
			},
			outTest{targetData: nil, nilTargetData: true},
		},
		{
			"Valid targeting data in requestExt, valid outTargetData",
			inTest{
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Targeting: &openrtb_ext.ExtRequestTargeting{
							PriceGranularity: openrtb_ext.PriceGranularity{
								Precision: 2,
								Ranges:    []openrtb_ext.GranularityRange{{Min: 0.00, Max: 5.00, Increment: 1.00}},
							},
							IncludeWinners:    true,
							IncludeBidderKeys: true,
						},
					},
				},
				cacheInstructions: &extCacheInstructions{
					cacheBids: true,
					cacheVAST: true,
				},
			},
			outTest{
				targetData: &targetData{
					priceGranularity: openrtb_ext.PriceGranularity{
						Precision: 2,
						Ranges:    []openrtb_ext.GranularityRange{{Min: 0.00, Max: 5.00, Increment: 1.00}},
					},
					includeWinners:    true,
					includeBidderKeys: true,
					includeCacheBids:  true,
					includeCacheVast:  true,
				},
				nilTargetData: false,
			},
		},
	}
	for _, test := range testCases {
		actualTargetData := getExtTargetData(test.in.requestExt, test.in.cacheInstructions)

		if test.out.nilTargetData {
			assert.Nil(t, actualTargetData, "%s. Targeting data should be nil. \n", test.desc)
		} else {
			assert.NotNil(t, actualTargetData, "%s. Targeting data should NOT be nil. \n", test.desc)
			assert.Equal(t, *test.out.targetData, *actualTargetData, "%s. Unexpected targeting data value. \n", test.desc)
		}
	}
}

func TestGetDebugInfo(t *testing.T) {
	type inTest struct {
		bidRequest *openrtb.BidRequest
		requestExt *openrtb_ext.ExtRequest
	}
	testCases := []struct {
		desc string
		in   inTest
		out  bool
	}{
		{
			desc: "Nil bid request, nil requestExt",
			in:   inTest{nil, nil},
			out:  false,
		},
		{
			desc: "bid request test == 0, nil requestExt",
			in:   inTest{&openrtb.BidRequest{Test: 0}, nil},
			out:  false,
		},
		{
			desc: "Nil bid request, requestExt debug flag false",
			in:   inTest{nil, &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Debug: false}}},
			out:  false,
		},
		{
			desc: "bid request test == 0, requestExt debug flag false",
			in:   inTest{&openrtb.BidRequest{Test: 0}, &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Debug: false}}},
			out:  false,
		},
		{
			desc: "bid request test == 1, requestExt debug flag false",
			in:   inTest{&openrtb.BidRequest{Test: 1}, &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Debug: false}}},
			out:  true,
		},
		{
			desc: "bid request test == 0, requestExt debug flag true",
			in:   inTest{&openrtb.BidRequest{Test: 0}, &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Debug: true}}},
			out:  true,
		},
		{
			desc: "bid request test == 1, requestExt debug flag true",
			in:   inTest{&openrtb.BidRequest{Test: 1}, &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Debug: true}}},
			out:  true,
		},
	}
	for _, test := range testCases {
		actualDebugInfo := getDebugInfo(test.in.bidRequest, test.in.requestExt)

		assert.Equal(t, test.out, actualDebugInfo, "%s. Unexpected debug value. \n", test.desc)
	}
}

func TestGetExtBidAdjustmentFactors(t *testing.T) {
	testCases := []struct {
		desc                    string
		inRequestExt            *openrtb_ext.ExtRequest
		outBidAdjustmentFactors map[string]float64
	}{
		{
			desc:                    "Nil request ext",
			inRequestExt:            nil,
			outBidAdjustmentFactors: nil,
		},
		{
			desc:                    "Non-nil request ext, nil BidAdjustmentFactors field",
			inRequestExt:            &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{BidAdjustmentFactors: nil}},
			outBidAdjustmentFactors: nil,
		},
		{
			desc:                    "Non-nil request ext, valid BidAdjustmentFactors field",
			inRequestExt:            &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{BidAdjustmentFactors: map[string]float64{"bid-factor": 1.0}}},
			outBidAdjustmentFactors: map[string]float64{"bid-factor": 1.0},
		},
	}
	for _, test := range testCases {
		actualBidAdjustmentFactors := getExtBidAdjustmentFactors(test.inRequestExt)

		assert.Equal(t, test.outBidAdjustmentFactors, actualBidAdjustmentFactors, "%s. Unexpected BidAdjustmentFactors value. \n", test.desc)
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
		expectPrivacyLabels metrics.PrivacyLabels
	}{
		{
			description:     "Feature Flag Enabled - OpenTRB Enabled",
			lmt:             &enabled,
			enforceLMT:      true,
			expectDataScrub: true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				LMTEnforced: true,
			},
		},
		{
			description:     "Feature Flag Disabled - OpenTRB Enabled",
			lmt:             &enabled,
			enforceLMT:      false,
			expectDataScrub: false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				LMTEnforced: false,
			},
		},
		{
			description:     "Feature Flag Enabled - OpenTRB Disabled",
			lmt:             &disabled,
			enforceLMT:      true,
			expectDataScrub: false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				LMTEnforced: false,
			},
		},
		{
			description:     "Feature Flag Disabled - OpenTRB Disabled",
			lmt:             &disabled,
			enforceLMT:      false,
			expectDataScrub: false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				LMTEnforced: false,
			},
		},
	}

	for _, test := range testCases {
		req := newBidRequest(t)
		req.Device.Lmt = test.lmt

		auctionReq := AuctionRequest{
			BidRequest: req,
			UserSyncs:  &emptyUsersync{},
		}

		privacyConfig := config.Privacy{
			LMT: config.LMT{
				Enforce: test.enforceLMT,
			},
		}

		results, privacyLabels, errs := cleanOpenRTBRequests(context.Background(), auctionReq, nil, &permissionsMock{personalInfoAllowed: true}, true, privacyConfig)
		result := results[0]

		assert.Nil(t, errs)
		if test.expectDataScrub {
			assert.Equal(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.BidRequest.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		} else {
			assert.NotEqual(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.BidRequest.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		}
		assert.Equal(t, test.expectPrivacyLabels, privacyLabels, test.description+":PrivacyLabels")
	}
}

func TestCleanOpenRTBRequestsGDPR(t *testing.T) {
	tcf1Consent := "BONV8oqONXwgmADACHENAO7pqzAAppY"
	tcf2Consent := "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA"
	trueValue, falseValue := true, false

	testCases := []struct {
		description         string
		gdprAccountEnabled  *bool
		gdprHostEnabled     bool
		gdpr                string
		gdprConsent         string
		gdprScrub           bool
		permissionsError    error
		userSyncIfAmbiguous bool
		expectPrivacyLabels metrics.PrivacyLabels
		expectError         bool
	}{
		{
			description:        "Enforce - TCF Invalid",
			gdprAccountEnabled: &trueValue,
			gdprHostEnabled:    true,
			gdpr:               "1",
			gdprConsent:        "malformed",
			gdprScrub:          false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: "",
			},
		},
		{
			description:        "Enforce - TCF 1",
			gdprAccountEnabled: &trueValue,
			gdprHostEnabled:    true,
			gdpr:               "1",
			gdprConsent:        tcf1Consent,
			gdprScrub:          true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: metrics.TCFVersionV1,
			},
		},
		{
			description:        "Enforce - TCF 2",
			gdprAccountEnabled: &trueValue,
			gdprHostEnabled:    true,
			gdpr:               "1",
			gdprConsent:        tcf2Consent,
			gdprScrub:          true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: metrics.TCFVersionV2,
			},
		},
		{
			description:        "Not Enforce - TCF 1",
			gdprAccountEnabled: &trueValue,
			gdprHostEnabled:    true,
			gdpr:               "0",
			gdprConsent:        tcf1Consent,
			gdprScrub:          false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   false,
				GDPRTCFVersion: "",
			},
		},
		{
			description:        "Enforce - TCF 1; GDPR signal extraction error",
			gdprAccountEnabled: &trueValue,
			gdprHostEnabled:    true,
			gdpr:               "0{",
			gdprConsent:        "BONV8oqONXwgmADACHENAO7pqzAAppY",
			gdprScrub:          true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: metrics.TCFVersionV1,
			},
			expectError: true,
		},
		{
			description:        "Enforce - TCF 1; account GDPR enabled, host GDPR setting disregarded",
			gdprAccountEnabled: &trueValue,
			gdprHostEnabled:    false,
			gdpr:               "1",
			gdprConsent:        tcf1Consent,
			gdprScrub:          true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: metrics.TCFVersionV1,
			},
		},
		{
			description:        "Not Enforce - TCF 1; account GDPR disabled, host GDPR setting disregarded",
			gdprAccountEnabled: &falseValue,
			gdprHostEnabled:    true,
			gdpr:               "1",
			gdprConsent:        tcf1Consent,
			gdprScrub:          false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   false,
				GDPRTCFVersion: "",
			},
		},
		{
			description:        "Enforce - TCF 1; account GDPR not specified, host GDPR enabled",
			gdprAccountEnabled: nil,
			gdprHostEnabled:    true,
			gdpr:               "1",
			gdprConsent:        tcf1Consent,
			gdprScrub:          true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: metrics.TCFVersionV1,
			},
		},
		{
			description:        "Not Enforce - TCF 1; account GDPR not specified, host GDPR disabled",
			gdprAccountEnabled: nil,
			gdprHostEnabled:    false,
			gdpr:               "1",
			gdprConsent:        tcf1Consent,
			gdprScrub:          false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   false,
				GDPRTCFVersion: "",
			},
		},
		{
			description:         "Enforce - Ambiguous signal, don't sync user if ambiguous",
			gdprAccountEnabled:  nil,
			gdprHostEnabled:     true,
			gdpr:                "null",
			gdprConsent:         tcf1Consent,
			gdprScrub:           true,
			userSyncIfAmbiguous: false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: metrics.TCFVersionV1,
			},
		},
		{
			description:         "Not Enforce - Ambiguous signal, sync user if ambiguous",
			gdprAccountEnabled:  nil,
			gdprHostEnabled:     true,
			gdpr:                "null",
			gdprConsent:         tcf1Consent,
			gdprScrub:           false,
			userSyncIfAmbiguous: true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   false,
				GDPRTCFVersion: "",
			},
		},
		{
			description:        "Enforce - error while checking if personal info is allowed",
			gdprAccountEnabled: nil,
			gdprHostEnabled:    true,
			gdpr:               "1",
			gdprConsent:        tcf1Consent,
			gdprScrub:          true,
			permissionsError:   errors.New("Some error"),
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: metrics.TCFVersionV1,
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
				Enabled:             test.gdprHostEnabled,
				UsersyncIfAmbiguous: test.userSyncIfAmbiguous,
				TCF2: config.TCF2{
					Enabled: true,
				},
			},
		}

		accountConfig := config.Account{
			GDPR: config.AccountGDPR{
				Enabled: test.gdprAccountEnabled,
			},
		}

		auctionReq := AuctionRequest{
			BidRequest: req,
			UserSyncs:  &emptyUsersync{},
			Account:    accountConfig,
		}

		results, privacyLabels, errs := cleanOpenRTBRequests(
			context.Background(),
			auctionReq,
			nil,
			&permissionsMock{personalInfoAllowed: !test.gdprScrub, personalInfoAllowedError: test.permissionsError},
			test.userSyncIfAmbiguous,
			privacyConfig)
		result := results[0]

		if test.expectError {
			assert.NotNil(t, errs)
		} else {
			assert.Nil(t, errs)
		}

		if test.gdprScrub {
			assert.Equal(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.BidRequest.Device.DIDMD5, "", test.description+":Device.DIDMD5")
		} else {
			assert.NotEqual(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.BidRequest.Device.DIDMD5, "", test.description+":Device.DIDMD5")
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

func TestRemoveUnpermissionedEids(t *testing.T) {
	bidder := "bidderA"

	testCases := []struct {
		description     string
		userExt         json.RawMessage
		eidPermissions  []openrtb_ext.ExtRequestPrebidDataEidPermission
		expectedUserExt json.RawMessage
	}{
		{
			description: "Extension Nil",
			userExt:     nil,
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"bidderA"}},
			},
			expectedUserExt: nil,
		},
		{
			description: "Extension Empty",
			userExt:     json.RawMessage(`{}`),
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"bidderA"}},
			},
			expectedUserExt: json.RawMessage(`{}`),
		},
		{
			description: "Extension Empty - Keep Other Data",
			userExt:     json.RawMessage(`{"other":42}`),
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"bidderA"}},
			},
			expectedUserExt: json.RawMessage(`{"other":42}`),
		},
		{
			description: "Eids Empty",
			userExt:     json.RawMessage(`{"eids":[]}`),
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"bidderA"}},
			},
			expectedUserExt: json.RawMessage(`{"eids":[]}`),
		},
		{
			description: "Eids Empty - Keep Other Data",
			userExt:     json.RawMessage(`{"eids":[],"other":42}`),
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"bidderA"}},
			},
			expectedUserExt: json.RawMessage(`{"eids":[],"other":42}`),
		},
		{
			description:     "Allowed By Nil Permissions",
			userExt:         json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`),
			eidPermissions:  nil,
			expectedUserExt: json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`),
		},
		{
			description:     "Allowed By Empty Permissions",
			userExt:         json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`),
			eidPermissions:  []openrtb_ext.ExtRequestPrebidDataEidPermission{},
			expectedUserExt: json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`),
		},
		{
			description: "Allowed By Specific Bidder",
			userExt:     json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`),
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"bidderA"}},
			},
			expectedUserExt: json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`),
		},
		{
			description: "Allowed By All Bidders",
			userExt:     json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`),
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"*"}},
			},
			expectedUserExt: json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`),
		},
		{
			description: "Allowed By Lack Of Matching Source",
			userExt:     json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`),
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source2", Bidders: []string{"otherBidder"}},
			},
			expectedUserExt: json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`),
		},
		{
			description: "Allowed - Keep Other Data",
			userExt:     json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}],"other":42}`),
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"bidderA"}},
			},
			expectedUserExt: json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}],"other":42}`),
		},
		{
			description: "Denied",
			userExt:     json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`),
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"otherBidder"}},
			},
			expectedUserExt: nil,
		},
		{
			description: "Denied - Keep Other Data",
			userExt:     json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}],"otherdata":42}`),
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"otherBidder"}},
			},
			expectedUserExt: json.RawMessage(`{"otherdata":42}`),
		},
		{
			description: "Mix Of Allowed By Specific Bidder, Allowed By Lack Of Matching Source, Denied, Keep Other Data",
			userExt:     json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"},{"source":"source2","id":"anyID"},{"source":"source3","id":"anyID"}],"other":42}`),
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"bidderA"}},
				{Source: "source3", Bidders: []string{"otherBidder"}},
			},
			expectedUserExt: json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"},{"source":"source2","id":"anyID"}],"other":42}`),
		},
	}

	for _, test := range testCases {
		request := &openrtb.BidRequest{
			User: &openrtb.User{Ext: test.userExt},
		}

		requestExt := &openrtb_ext.ExtRequest{
			Prebid: openrtb_ext.ExtRequestPrebid{
				Data: &openrtb_ext.ExtRequestPrebidData{
					EidPermissions: test.eidPermissions,
				},
			},
		}

		expectedRequest := &openrtb.BidRequest{
			User: &openrtb.User{Ext: test.expectedUserExt},
		}

		resultErr := removeUnpermissionedEids(request, bidder, requestExt)
		assert.NoError(t, resultErr, test.description)
		assert.Equal(t, expectedRequest, request, test.description)
	}
}

func TestRemoveUnpermissionedEidsUnmarshalErrors(t *testing.T) {
	testCases := []struct {
		description string
		userExt     json.RawMessage
		expectedErr string
	}{
		{
			description: "Malformed Ext",
			userExt:     json.RawMessage(`malformed`),
			expectedErr: "invalid character 'm' looking for beginning of value",
		},
		{
			description: "Malformed Eid Array Type",
			userExt:     json.RawMessage(`{"eids":[42]}`),
			expectedErr: "json: cannot unmarshal number into Go value of type openrtb_ext.ExtUserEid",
		},
		{
			description: "Malformed Eid Item Type",
			userExt:     json.RawMessage(`{"eids":[{"source":42,"id":"anyID"}]}`),
			expectedErr: "json: cannot unmarshal number into Go struct field ExtUserEid.source of type string",
		},
	}

	for _, test := range testCases {
		request := &openrtb.BidRequest{
			User: &openrtb.User{Ext: test.userExt},
		}

		requestExt := &openrtb_ext.ExtRequest{
			Prebid: openrtb_ext.ExtRequestPrebid{
				Data: &openrtb_ext.ExtRequestPrebidData{
					EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
						{Source: "source1", Bidders: []string{"*"}},
					},
				},
			},
		}

		resultErr := removeUnpermissionedEids(request, "bidderA", requestExt)
		assert.EqualError(t, resultErr, test.expectedErr, test.description)
	}
}

func TestRemoveUnpermissionedEidsEmptyValidations(t *testing.T) {
	testCases := []struct {
		description string
		request     *openrtb.BidRequest
		requestExt  *openrtb_ext.ExtRequest
	}{
		{
			description: "Nil User",
			request: &openrtb.BidRequest{
				User: nil,
			},
			requestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Data: &openrtb_ext.ExtRequestPrebidData{
						EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
							{Source: "source1", Bidders: []string{"*"}},
						},
					},
				},
			},
		},
		{
			description: "Empty User",
			request: &openrtb.BidRequest{
				User: &openrtb.User{},
			},
			requestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Data: &openrtb_ext.ExtRequestPrebidData{
						EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
							{Source: "source1", Bidders: []string{"*"}},
						},
					},
				},
			},
		},
		{
			description: "Nil Ext",
			request: &openrtb.BidRequest{
				User: &openrtb.User{Ext: json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`)},
			},
			requestExt: nil,
		},
		{
			description: "Nil Prebid Data",
			request: &openrtb.BidRequest{
				User: &openrtb.User{Ext: json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`)},
			},
			requestExt: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Data: nil,
				},
			},
		},
	}

	for _, test := range testCases {
		requestExpected := *test.request

		resultErr := removeUnpermissionedEids(test.request, "bidderA", test.requestExt)
		assert.NoError(t, resultErr, test.description+":err")
		assert.Equal(t, &requestExpected, test.request, test.description+":request")
	}
}
