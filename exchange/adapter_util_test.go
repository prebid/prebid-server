package exchange

import (
	"errors"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/appnexus"
	"github.com/prebid/prebid-server/v3/adapters/rubicon"
	"github.com/prebid/prebid-server/v3/config"
	metrics "github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	falseValue          = false
	infoEnabled         = config.BidderInfo{Disabled: false}
	infoDisabled        = config.BidderInfo{Disabled: true}
	multiformatDisabled = config.BidderInfo{
		Disabled: false,
		OpenRTB: &config.OpenRTBInfo{
			MultiformatSupported: &falseValue,
		},
	}
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
		description                 string
		bidderInfos                 map[string]config.BidderInfo
		expectedBidders             map[openrtb_ext.BidderName]AdaptedBidder
		expectedSingleFormatBidders map[openrtb_ext.BidderName]struct{}
		expectedErrors              []error
	}{
		{
			description:                 "No Bidders",
			bidderInfos:                 map[string]config.BidderInfo{},
			expectedBidders:             map[openrtb_ext.BidderName]AdaptedBidder{},
			expectedSingleFormatBidders: map[openrtb_ext.BidderName]struct{}{},
		},
		{
			description: "One Bidder",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoEnabled},
			expectedBidders: map[openrtb_ext.BidderName]AdaptedBidder{
				openrtb_ext.BidderAppnexus: appnexusValidated,
			},
			expectedSingleFormatBidders: map[openrtb_ext.BidderName]struct{}{},
		},
		{
			description: "Many Bidders",
			bidderInfos: map[string]config.BidderInfo{"appnexus": infoEnabled, "rubicon": infoEnabled},
			expectedBidders: map[openrtb_ext.BidderName]AdaptedBidder{
				openrtb_ext.BidderAppnexus: appnexusValidated,
				openrtb_ext.BidderRubicon:  rubiconBidderValidated,
			},
			expectedSingleFormatBidders: map[openrtb_ext.BidderName]struct{}{},
		},
		{
			description: "Invalid - Builder Errors",
			bidderInfos: map[string]config.BidderInfo{"unknown": {}, "appNexus": {}},
			expectedErrors: []error{
				errors.New("unknown: unknown bidder"),
			},
			expectedSingleFormatBidders: nil,
		},
		{
			description: "Bidders with multiformat Support Disabled",
			bidderInfos: map[string]config.BidderInfo{"appnexus": multiformatDisabled, "rubicon": multiformatDisabled},
			expectedBidders: map[openrtb_ext.BidderName]AdaptedBidder{
				openrtb_ext.BidderAppnexus: appnexusValidated,
				openrtb_ext.BidderRubicon:  rubiconBidderValidated,
			},
			expectedSingleFormatBidders: map[openrtb_ext.BidderName]struct{}{
				openrtb_ext.BidderAppnexus: {},
				openrtb_ext.BidderRubicon:  {},
			},
		},
	}

	cfg := &config.Configuration{}
	for _, test := range testCases {
		bidders, singleFormatBidders, errs := BuildAdapters(client, cfg, test.bidderInfos, metricEngine)
		assert.Equal(t, test.expectedBidders, bidders, test.description+":bidders")

		assert.Equal(t, test.expectedSingleFormatBidders, singleFormatBidders, test.description+":singleFormatBidders")

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
		bidders, _, errs := buildBidders(test.bidderInfos, test.builders, server)

		// For Test Setup Convenience
		if test.expectedBidders == nil {
			test.expectedBidders = make(map[openrtb_ext.BidderName]adapters.Bidder)
		}

		assert.Equal(t, test.expectedBidders, bidders, test.description+":bidders")
		assert.ElementsMatch(t, test.expectedErrors, errs, test.description+":errors")
	}
}

func TestSetAliasBuilder(t *testing.T) {
	rubiconBidder := fakeBidder{"b"}
	ixBidder := fakeBidder{"ix"}
	rubiconBuilder := fakeBuilder{rubiconBidder, nil}.Builder
	ixBuilder := fakeBuilder{ixBidder, nil}.Builder

	testCases := []struct {
		description      string
		bidderInfo       config.BidderInfo
		builders         map[openrtb_ext.BidderName]adapters.Builder
		bidderName       openrtb_ext.BidderName
		expectedBuilders map[openrtb_ext.BidderName]adapters.Builder
		expectedError    error
	}{
		{
			description:      "Success - Alias builder",
			bidderInfo:       config.BidderInfo{Disabled: false, AliasOf: "rubicon"},
			bidderName:       openrtb_ext.BidderName("appnexus"),
			builders:         map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderRubicon: rubiconBuilder},
			expectedBuilders: map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderRubicon: rubiconBuilder, openrtb_ext.BidderAppnexus: rubiconBuilder},
		},
		{
			description:   "Failure - Invalid parent bidder builder",
			bidderInfo:    config.BidderInfo{Disabled: false, AliasOf: "rubicon"},
			bidderName:    openrtb_ext.BidderName("appnexus"),
			builders:      map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderIx: ixBuilder},
			expectedError: errors.New("rubicon: parent builder not registered"),
		},
		{
			description:   "Failure - Invalid parent for alias",
			bidderInfo:    config.BidderInfo{Disabled: false, AliasOf: "unknown"},
			bidderName:    openrtb_ext.BidderName("appnexus"),
			builders:      map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderIx: ixBuilder},
			expectedError: errors.New("unknown parent bidder: unknown for alias: appnexus"),
		},
	}

	for _, test := range testCases {
		err := setAliasBuilder(test.bidderInfo, test.builders, test.bidderName)

		if test.expectedBuilders != nil {
			assert.ObjectsAreEqual(test.builders, test.expectedBuilders)
		}
		if test.expectedError != nil {
			assert.EqualError(t, test.expectedError, err.Error(), test.description+":errors")
		}
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

func TestGetDisabledBidderWarningMessages(t *testing.T) {
	t.Run("removed", func(t *testing.T) {
		result := GetDisabledBidderWarningMessages(nil)

		// test proper construction by verifying one expected bidder is in the list
		require.Contains(t, result, "groupm")
		assert.Equal(t, result["groupm"], `Bidder "groupm" is no longer available in Prebid Server. Please update your configuration.`)
	})

	t.Run("removed-and-disabled", func(t *testing.T) {
		result := GetDisabledBidderWarningMessages(map[string]config.BidderInfo{"bidderA": infoDisabled})

		// test proper construction by verifying one expected bidder is in the list with the disabled bidder
		require.Contains(t, result, "groupm")
		assert.Equal(t, result["groupm"], `Bidder "groupm" is no longer available in Prebid Server. Please update your configuration.`)

		require.Contains(t, result, "bidderA")
		assert.Equal(t, result["bidderA"], `Bidder "bidderA" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`)
	})
}

func TestMergeRemovedAndDisabledBidderWarningMessages(t *testing.T) {
	testCases := []struct {
		name             string
		givenRemoved     map[string]string
		givenBidderInfos map[string]config.BidderInfo
		expected         map[string]string
	}{
		{
			name:             "none",
			givenRemoved:     map[string]string{},
			givenBidderInfos: map[string]config.BidderInfo{},
			expected:         map[string]string{},
		},
		{
			name:             "removed",
			givenRemoved:     map[string]string{"bidderA": `Bidder A Message`},
			givenBidderInfos: map[string]config.BidderInfo{},
			expected:         map[string]string{"bidderA": `Bidder A Message`},
		},
		{
			name:             "enabled",
			givenRemoved:     map[string]string{},
			givenBidderInfos: map[string]config.BidderInfo{"bidderA": infoEnabled},
			expected:         map[string]string{},
		},
		{
			name:             "disabled",
			givenRemoved:     map[string]string{},
			givenBidderInfos: map[string]config.BidderInfo{"bidderA": infoDisabled},
			expected:         map[string]string{"bidderA": `Bidder "bidderA" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`},
		},
		{
			name:             "mixed",
			givenRemoved:     map[string]string{"bidderA": `Bidder A Message`},
			givenBidderInfos: map[string]config.BidderInfo{"bidderB": infoEnabled, "bidderC": infoDisabled},
			expected:         map[string]string{"bidderA": `Bidder A Message`, "bidderC": `Bidder "bidderC" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := mergeRemovedAndDisabledBidderWarningMessages(test.givenRemoved, test.givenBidderInfos)
			assert.Equal(t, test.expected, result, test.name)
		})
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
