package exchange

import (
	"errors"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/rubicon"
	"github.com/prebid/prebid-server/config"
	metrics "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

var (
	infoEnabled  = config.BidderInfo{Disabled: false}
	infoDisabled = config.BidderInfo{Disabled: true}
)

func TestBuildAdapters(t *testing.T) {
	client := &http.Client{}
	metricEngine := &metrics.NilMetricsEngine{}

	appnexusBidder, _ := appnexus.Builder(openrtb_ext.BidderAppnexus, config.Adapter{}, config.Server{})
	appnexusBidderWithInfo := adapters.BuildInfoAwareBidder(appnexusBidder, infoEnabled)
	appnexusBidderAdapted := AdaptBidder(appnexusBidderWithInfo, client, &config.Configuration{}, metricEngine, openrtb_ext.BidderAppnexus, nil, "")
	appnexusValidated := addValidatedBidderMiddleware(appnexusBidderAdapted)

	rubiconBidder, _ := rubicon.Builder(openrtb_ext.BidderRubicon, config.Adapter{}, config.Server{})
	rubiconBidderWithInfo := adapters.BuildInfoAwareBidder(rubiconBidder, infoEnabled)
	rubiconBidderAdapted := AdaptBidder(rubiconBidderWithInfo, client, &config.Configuration{}, metricEngine, openrtb_ext.BidderRubicon, nil, "")
	rubiconBidderValidated := addValidatedBidderMiddleware(rubiconBidderAdapted)

	testCases := []struct {
		description     string
		bidderInfos     map[string]config.BidderInfo
		expectedBidders map[openrtb_ext.BidderName]AdaptedBidder
		expectedErrors  []error
	}{
		{
			description:     "No Bidders",
			bidderInfos:     map[string]config.BidderInfo{},
			expectedBidders: map[openrtb_ext.BidderName]AdaptedBidder{},
		},
		{
			description: "One Bidder",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoEnabled},
			expectedBidders: map[openrtb_ext.BidderName]AdaptedBidder{
				openrtb_ext.BidderAppnexus: appnexusValidated,
			},
		},
		{
			description: "Many Bidders",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoEnabled, "rubicon": infoEnabled},
			expectedBidders: map[openrtb_ext.BidderName]AdaptedBidder{
				openrtb_ext.BidderAppnexus: appnexusValidated,
				openrtb_ext.BidderRubicon:  rubiconBidderValidated,
			},
		},
		{
			description: "Invalid - Builder Errors",
			bidderInfos: map[string]config.BidderInfo{"unknown": {}, "appNexus": {}},
			expectedErrors: []error{
				errors.New("unknown: unknown bidder"),
			},
		},
	}

	cfg := &config.Configuration{}
	for _, test := range testCases {
		bidders, errs := BuildAdapters(client, cfg, test.bidderInfos, metricEngine)
		assert.Equal(t, test.expectedBidders, bidders, test.description+":bidders")
		assert.ElementsMatch(t, test.expectedErrors, errs, test.description+":errors")
	}
}

func TestBuildBidders(t *testing.T) {
	appnexusBidder := fakeBidder{"a"}
	appnexusBuilder := fakeBuilder{appnexusBidder, nil}.Builder
	appnexusBuilderWithError := fakeBuilder{appnexusBidder, errors.New("anyError")}.Builder

	rubiconBidder := fakeBidder{"b"}
	rubiconBuilder := fakeBuilder{rubiconBidder, nil}.Builder

	server := config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"}

	testCases := []struct {
		description     string
		bidderInfos     map[string]config.BidderInfo
		builders        map[openrtb_ext.BidderName]adapters.Builder
		expectedBidders map[openrtb_ext.BidderName]adapters.Bidder
		expectedErrors  []error
	}{
		{
			description: "Invalid - Unknown Bidder",
			bidderInfos: map[string]config.BidderInfo{"unknown": infoEnabled},
			builders:    map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder},
			expectedErrors: []error{
				errors.New("unknown: unknown bidder"),
			},
		},
		{
			description: "Invalid - No Builder",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoEnabled},
			builders:    map[openrtb_ext.BidderName]adapters.Builder{},
			expectedErrors: []error{
				errors.New("appnexus: builder not registered"),
			},
		},
		{
			description: "Success - Builder Error",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoEnabled},
			builders:    map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilderWithError},
			expectedErrors: []error{
				errors.New("appnexus: anyError"),
			},
		},
		{
			description: "Success - None",
			bidderInfos: map[string]config.BidderInfo{},
			builders:    map[openrtb_ext.BidderName]adapters.Builder{},
		},
		{
			description: "Success - One",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoEnabled},
			builders:    map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder},
			expectedBidders: map[openrtb_ext.BidderName]adapters.Bidder{
				openrtb_ext.BidderAppnexus: adapters.BuildInfoAwareBidder(appnexusBidder, infoEnabled),
			},
		},
		{
			description: "Success - Many",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoEnabled, "rubicon": infoEnabled},
			builders:    map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder, openrtb_ext.BidderRubicon: rubiconBuilder},
			expectedBidders: map[openrtb_ext.BidderName]adapters.Bidder{
				openrtb_ext.BidderAppnexus: adapters.BuildInfoAwareBidder(appnexusBidder, infoEnabled),
				openrtb_ext.BidderRubicon:  adapters.BuildInfoAwareBidder(rubiconBidder, infoEnabled),
			},
		},
		{
			description: "Success - Ignores Disabled",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoDisabled, "rubicon": infoEnabled},
			builders:    map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder, openrtb_ext.BidderRubicon: rubiconBuilder},
			expectedBidders: map[openrtb_ext.BidderName]adapters.Bidder{
				openrtb_ext.BidderRubicon: adapters.BuildInfoAwareBidder(rubiconBidder, infoEnabled),
			},
		},
	}

	for _, test := range testCases {
		bidders, errs := buildBidders(test.bidderInfos, test.builders, server)

		// For Test Setup Convenience
		if test.expectedBidders == nil {
			test.expectedBidders = make(map[openrtb_ext.BidderName]adapters.Bidder)
		}

		assert.Equal(t, test.expectedBidders, bidders, test.description+":bidders")
		assert.ElementsMatch(t, test.expectedErrors, errs, test.description+":errors")
	}
}

func TestGetActiveBidders(t *testing.T) {
	testCases := []struct {
		description string
		bidderInfos map[string]config.BidderInfo
		expected    map[string]openrtb_ext.BidderName
	}{
		{
			description: "None",
			bidderInfos: map[string]config.BidderInfo{},
			expected:    map[string]openrtb_ext.BidderName{},
		},
		{
			description: "Enabled",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoEnabled},
			expected:    map[string]openrtb_ext.BidderName{"appnexus": openrtb_ext.BidderAppnexus},
		},
		{
			description: "Disabled",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoDisabled},
			expected:    map[string]openrtb_ext.BidderName{},
		},
		{
			description: "Mixed",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoDisabled, "openx": infoEnabled},
			expected:    map[string]openrtb_ext.BidderName{"openx": openrtb_ext.BidderOpenx},
		},
	}

	for _, test := range testCases {
		result := GetActiveBidders(test.bidderInfos)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestGetDisabledBiddersErrorMessages(t *testing.T) {
	testCases := []struct {
		description string
		bidderInfos map[string]config.BidderInfo
		expected    map[string]string
	}{
		{
			description: "None",
			bidderInfos: map[string]config.BidderInfo{},
			expected: map[string]string{
				"lifestreet":     `Bidder "lifestreet" is no longer available in Prebid Server. Please update your configuration.`,
				"adagio":         `Bidder "adagio" is no longer available in Prebid Server. Please update your configuration.`,
				"somoaudience":   `Bidder "somoaudience" is no longer available in Prebid Server. Please update your configuration.`,
				"yssp":           `Bidder "yssp" is no longer available in Prebid Server. If you're looking to use the Yahoo SSP adapter, please rename it to "yahooAds" in your configuration.`,
				"andbeyondmedia": `Bidder "andbeyondmedia" is no longer available in Prebid Server. If you're looking to use the AndBeyond.Media SSP adapter, please rename it to "beyondmedia" in your configuration.`,
				"oftmedia":       `Bidder "oftmedia" is no longer available in Prebid Server. Please update your configuration.`,
				"groupm":         `Bidder "groupm" is no longer available in Prebid Server. Please update your configuration.`,
				"verizonmedia":   `Bidder "verizonmedia" is no longer available in Prebid Server. Please update your configuration.`,
			},
		},
		{
			description: "Enabled",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoEnabled},
			expected: map[string]string{
				"lifestreet":     `Bidder "lifestreet" is no longer available in Prebid Server. Please update your configuration.`,
				"adagio":         `Bidder "adagio" is no longer available in Prebid Server. Please update your configuration.`,
				"somoaudience":   `Bidder "somoaudience" is no longer available in Prebid Server. Please update your configuration.`,
				"yssp":           `Bidder "yssp" is no longer available in Prebid Server. If you're looking to use the Yahoo SSP adapter, please rename it to "yahooAds" in your configuration.`,
				"andbeyondmedia": `Bidder "andbeyondmedia" is no longer available in Prebid Server. If you're looking to use the AndBeyond.Media SSP adapter, please rename it to "beyondmedia" in your configuration.`,
				"oftmedia":       `Bidder "oftmedia" is no longer available in Prebid Server. Please update your configuration.`,
				"groupm":         `Bidder "groupm" is no longer available in Prebid Server. Please update your configuration.`,
				"verizonmedia":   `Bidder "verizonmedia" is no longer available in Prebid Server. Please update your configuration.`,
			},
		},
		{
			description: "Disabled",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoDisabled},
			expected: map[string]string{
				"lifestreet":     `Bidder "lifestreet" is no longer available in Prebid Server. Please update your configuration.`,
				"adagio":         `Bidder "adagio" is no longer available in Prebid Server. Please update your configuration.`,
				"somoaudience":   `Bidder "somoaudience" is no longer available in Prebid Server. Please update your configuration.`,
				"yssp":           `Bidder "yssp" is no longer available in Prebid Server. If you're looking to use the Yahoo SSP adapter, please rename it to "yahooAds" in your configuration.`,
				"appnexus":       `Bidder "appnexus" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`,
				"andbeyondmedia": `Bidder "andbeyondmedia" is no longer available in Prebid Server. If you're looking to use the AndBeyond.Media SSP adapter, please rename it to "beyondmedia" in your configuration.`,
				"oftmedia":       `Bidder "oftmedia" is no longer available in Prebid Server. Please update your configuration.`,
				"groupm":         `Bidder "groupm" is no longer available in Prebid Server. Please update your configuration.`,
				"verizonmedia":   `Bidder "verizonmedia" is no longer available in Prebid Server. Please update your configuration.`,
			},
		},
		{
			description: "Mixed",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoDisabled, "openx": infoEnabled},
			expected: map[string]string{
				"lifestreet":     `Bidder "lifestreet" is no longer available in Prebid Server. Please update your configuration.`,
				"adagio":         `Bidder "adagio" is no longer available in Prebid Server. Please update your configuration.`,
				"somoaudience":   `Bidder "somoaudience" is no longer available in Prebid Server. Please update your configuration.`,
				"yssp":           `Bidder "yssp" is no longer available in Prebid Server. If you're looking to use the Yahoo SSP adapter, please rename it to "yahooAds" in your configuration.`,
				"appnexus":       `Bidder "appnexus" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`,
				"andbeyondmedia": `Bidder "andbeyondmedia" is no longer available in Prebid Server. If you're looking to use the AndBeyond.Media SSP adapter, please rename it to "beyondmedia" in your configuration.`,
				"oftmedia":       `Bidder "oftmedia" is no longer available in Prebid Server. Please update your configuration.`,
				"groupm":         `Bidder "groupm" is no longer available in Prebid Server. Please update your configuration.`,
				"verizonmedia":   `Bidder "verizonmedia" is no longer available in Prebid Server. Please update your configuration.`,
			},
		},
	}

	for _, test := range testCases {
		result := GetDisabledBiddersErrorMessages(test.bidderInfos)
		assert.Equal(t, test.expected, result, test.description)
	}
}

type fakeBidder struct {
	name string
}

func (fakeBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	return nil, nil
}

func (fakeBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return nil, nil
}

type fakeBuilder struct {
	bidder adapters.Bidder
	err    error
}

func (b fakeBuilder) Builder(name openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	return b.bidder, b.err
}
