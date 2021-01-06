package exchange

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/ix"
	"github.com/prebid/prebid-server/adapters/lifestreet"
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/adapters/rubicon"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	metrics "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

var (
	infoActive   = adapters.BidderInfo{Status: adapters.StatusActive}
	infoDisabled = adapters.BidderInfo{Status: adapters.StatusDisabled}
	infoUnknown  = adapters.BidderInfo{Status: adapters.StatusUnknown}
)

func TestBuildAdaptersSuccess(t *testing.T) {
	client := &http.Client{}
	cfg := &config.Configuration{Adapters: map[string]config.Adapter{
		"appnexus": {},
		"ix":       {Endpoint: "anyEndpoint"},
	}}
	infos := map[string]adapters.BidderInfo{
		"appnexus": infoActive,
		"ix":       infoActive,
	}
	metricEngine := &metrics.DummyMetricsEngine{}

	bidders, errs := BuildAdapters(client, cfg, infos, metricEngine)

	appnexusBidder, _ := appnexus.Builder(openrtb_ext.BidderAppnexus, config.Adapter{})
	appnexusBidderWithInfo := adapters.EnforceBidderInfo(appnexusBidder, infoActive)
	appnexusBidderAdapted := adaptBidder(appnexusBidderWithInfo, client, &config.Configuration{}, metricEngine, openrtb_ext.BidderAppnexus)
	appnexusBidderValidated := addValidatedBidderMiddleware(appnexusBidderAdapted)

	idLegacyAdapted := &adaptedAdapter{ix.NewIxLegacyAdapter(adapters.DefaultHTTPAdapterConfig, "anyEndpoint")}
	idLegacyValidated := addValidatedBidderMiddleware(idLegacyAdapted)

	expectedBidders := map[openrtb_ext.BidderName]adaptedBidder{
		openrtb_ext.BidderAppnexus: appnexusBidderValidated,
		openrtb_ext.BidderIx:       idLegacyValidated,
	}

	assert.Equal(t, expectedBidders, bidders)
	assert.Empty(t, errs)
}

func TestBuildAdaptersErrors(t *testing.T) {
	client := &http.Client{}
	cfg := &config.Configuration{Adapters: map[string]config.Adapter{"unknown": {}}}
	infos := map[string]adapters.BidderInfo{}
	metricEngine := &metrics.DummyMetricsEngine{}

	bidders, errs := BuildAdapters(client, cfg, infos, metricEngine)

	expectedErrors := []error{
		errors.New("unknown: unknown bidder"),
	}

	assert.Empty(t, bidders)
	assert.Equal(t, expectedErrors, errs)
}

func TestBuildExchangeBidders(t *testing.T) {
	client := &http.Client{}
	metricEngine := &metrics.DummyMetricsEngine{}

	appnexusBidder, _ := appnexus.Builder(openrtb_ext.BidderAppnexus, config.Adapter{})
	appnexusBidderWithInfo := adapters.EnforceBidderInfo(appnexusBidder, infoActive)
	appnexusBidderAdapted := adaptBidder(appnexusBidderWithInfo, client, &config.Configuration{}, metricEngine, openrtb_ext.BidderAppnexus)

	rubiconBidder, _ := rubicon.Builder(openrtb_ext.BidderRubicon, config.Adapter{})
	rubiconBidderWithInfo := adapters.EnforceBidderInfo(rubiconBidder, infoActive)
	rubiconBidderAdapted := adaptBidder(rubiconBidderWithInfo, client, &config.Configuration{}, metricEngine, openrtb_ext.BidderRubicon)

	testCases := []struct {
		description     string
		adapterConfig   map[string]config.Adapter
		bidderInfos     map[string]adapters.BidderInfo
		expectedBidders map[openrtb_ext.BidderName]adaptedBidder
		expectedErrors  []error
	}{
		{
			description:   "Invalid - Builder Errors",
			adapterConfig: map[string]config.Adapter{"appnexus": {}, "unknown": {}},
			bidderInfos:   map[string]adapters.BidderInfo{},
			expectedErrors: []error{
				errors.New("appnexus: bidder info not found"),
				errors.New("unknown: unknown bidder"),
			},
		},
		{
			description:     "Success - None",
			adapterConfig:   map[string]config.Adapter{},
			bidderInfos:     map[string]adapters.BidderInfo{},
			expectedBidders: map[openrtb_ext.BidderName]adaptedBidder{},
		},
		{
			description:   "Success - One",
			adapterConfig: map[string]config.Adapter{"appnexus": {}},
			bidderInfos:   map[string]adapters.BidderInfo{"appnexus": infoActive},
			expectedBidders: map[openrtb_ext.BidderName]adaptedBidder{
				openrtb_ext.BidderAppnexus: appnexusBidderAdapted,
			},
		},
		{
			description:   "Success - Many",
			adapterConfig: map[string]config.Adapter{"appnexus": {}, "rubicon": {}},
			bidderInfos:   map[string]adapters.BidderInfo{"appnexus": infoActive, "rubicon": infoActive},
			expectedBidders: map[openrtb_ext.BidderName]adaptedBidder{
				openrtb_ext.BidderAppnexus: appnexusBidderAdapted,
				openrtb_ext.BidderRubicon:  rubiconBidderAdapted,
			},
		},
	}

	for _, test := range testCases {
		cfg := &config.Configuration{Adapters: test.adapterConfig}
		bidders, errs := buildExchangeBidders(cfg, test.bidderInfos, client, metricEngine)
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

	inconsequentialBuilder := fakeBuilder{fakeBidder{"whatevs"}, nil}.Builder

	testCases := []struct {
		description     string
		adapterConfig   map[string]config.Adapter
		bidderInfos     map[string]adapters.BidderInfo
		builders        map[openrtb_ext.BidderName]adapters.Builder
		expectedBidders map[openrtb_ext.BidderName]adapters.Bidder
		expectedErrors  []error
	}{
		{
			description:   "Invalid - Unknown Bidder",
			adapterConfig: map[string]config.Adapter{"unknown": {}},
			bidderInfos:   map[string]adapters.BidderInfo{"unknown": infoActive},
			builders:      map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder},
			expectedErrors: []error{
				errors.New("unknown: unknown bidder"),
			},
		},
		{
			description:   "Invalid - No Bidder Info",
			adapterConfig: map[string]config.Adapter{"appnexus": {}},
			bidderInfos:   map[string]adapters.BidderInfo{},
			builders:      map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder},
			expectedErrors: []error{
				errors.New("appnexus: bidder info not found"),
			},
		},
		{
			description:   "Invalid - No Builder",
			adapterConfig: map[string]config.Adapter{"appnexus": {}},
			bidderInfos:   map[string]adapters.BidderInfo{"appnexus": infoActive},
			builders:      map[openrtb_ext.BidderName]adapters.Builder{},
			expectedErrors: []error{
				errors.New("appnexus: builder not registered"),
			},
		},
		{
			description:   "Success - Builder Error",
			adapterConfig: map[string]config.Adapter{"appnexus": {}},
			bidderInfos:   map[string]adapters.BidderInfo{"appnexus": infoActive},
			builders:      map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilderWithError},
			expectedErrors: []error{
				errors.New("appnexus: anyError"),
			},
		},
		{
			description:   "Success - None",
			adapterConfig: map[string]config.Adapter{},
			bidderInfos:   map[string]adapters.BidderInfo{},
			builders:      map[openrtb_ext.BidderName]adapters.Builder{},
		},
		{
			description:   "Success - One",
			adapterConfig: map[string]config.Adapter{"appnexus": {}},
			bidderInfos:   map[string]adapters.BidderInfo{"appnexus": infoActive},
			builders:      map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder},
			expectedBidders: map[openrtb_ext.BidderName]adapters.Bidder{
				openrtb_ext.BidderAppnexus: adapters.EnforceBidderInfo(appnexusBidder, infoActive),
			},
		},
		{
			description:   "Success - Many",
			adapterConfig: map[string]config.Adapter{"appnexus": {}, "rubicon": {}},
			bidderInfos:   map[string]adapters.BidderInfo{"appnexus": infoActive, "rubicon": infoActive},
			builders:      map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder, openrtb_ext.BidderRubicon: rubiconBuilder},
			expectedBidders: map[openrtb_ext.BidderName]adapters.Bidder{
				openrtb_ext.BidderAppnexus: adapters.EnforceBidderInfo(appnexusBidder, infoActive),
				openrtb_ext.BidderRubicon:  adapters.EnforceBidderInfo(rubiconBidder, infoActive),
			},
		},
		{
			description:   "Success - Ignores Legacy",
			adapterConfig: map[string]config.Adapter{"appnexus": {}, "ix": {}, "lifestreet": {}, "pulsepoint": {}},
			bidderInfos:   map[string]adapters.BidderInfo{"appnexus": infoActive, "ix": infoActive, "lifestreet": infoActive, "pulsepoint": infoActive},
			builders:      map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder, openrtb_ext.BidderIx: inconsequentialBuilder, openrtb_ext.BidderLifestreet: inconsequentialBuilder, openrtb_ext.BidderPulsepoint: inconsequentialBuilder},
			expectedBidders: map[openrtb_ext.BidderName]adapters.Bidder{
				openrtb_ext.BidderAppnexus: adapters.EnforceBidderInfo(appnexusBidder, infoActive),
			},
		},
		{
			description:   "Success - Ignores Disabled",
			adapterConfig: map[string]config.Adapter{"appnexus": {}, "rubicon": {}},
			bidderInfos:   map[string]adapters.BidderInfo{"appnexus": infoDisabled, "rubicon": infoActive},
			builders:      map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder, openrtb_ext.BidderRubicon: rubiconBuilder},
			expectedBidders: map[openrtb_ext.BidderName]adapters.Bidder{
				openrtb_ext.BidderRubicon: adapters.EnforceBidderInfo(rubiconBidder, infoActive),
			},
		},
		{
			description:   "Success - Ignores Unknown State",
			adapterConfig: map[string]config.Adapter{"appnexus": {}, "rubicon": {}},
			bidderInfos:   map[string]adapters.BidderInfo{"appnexus": infoUnknown, "rubicon": infoActive},
			builders:      map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder, openrtb_ext.BidderRubicon: rubiconBuilder},
			expectedBidders: map[openrtb_ext.BidderName]adapters.Bidder{
				openrtb_ext.BidderRubicon: adapters.EnforceBidderInfo(rubiconBidder, infoActive),
			},
		},
		{
			description:   "Success - Ignores Adapter Config Case",
			adapterConfig: map[string]config.Adapter{"AppNexus": {}},
			bidderInfos:   map[string]adapters.BidderInfo{"appnexus": infoActive},
			builders:      map[openrtb_ext.BidderName]adapters.Builder{openrtb_ext.BidderAppnexus: appnexusBuilder},
			expectedBidders: map[openrtb_ext.BidderName]adapters.Bidder{
				openrtb_ext.BidderAppnexus: adapters.EnforceBidderInfo(appnexusBidder, infoActive),
			},
		},
	}

	for _, test := range testCases {
		bidders, errs := buildBidders(test.adapterConfig, test.bidderInfos, test.builders)

		// For Test Setup Convenience
		if test.expectedBidders == nil {
			test.expectedBidders = make(map[openrtb_ext.BidderName]adapters.Bidder)
		}

		assert.Equal(t, test.expectedBidders, bidders, test.description+":bidders")
		assert.ElementsMatch(t, test.expectedErrors, errs, test.description+":errors")
	}
}

func TestBuildExchangeBiddersLegacy(t *testing.T) {
	cfg := config.Adapter{Endpoint: "anyEndpoint"}

	expectedIx := &adaptedAdapter{ix.NewIxLegacyAdapter(adapters.DefaultHTTPAdapterConfig, "anyEndpoint")}
	expectedLifestreet := &adaptedAdapter{lifestreet.NewLifestreetLegacyAdapter(adapters.DefaultHTTPAdapterConfig, "anyEndpoint")}
	expectedPulsepoint := &adaptedAdapter{pulsepoint.NewPulsePointLegacyAdapter(adapters.DefaultHTTPAdapterConfig, "anyEndpoint")}

	testCases := []struct {
		description   string
		adapterConfig map[string]config.Adapter
		bidderInfos   map[string]adapters.BidderInfo
		expected      map[openrtb_ext.BidderName]adaptedBidder
	}{
		{
			description:   "All Active",
			adapterConfig: map[string]config.Adapter{"ix": cfg, "lifestreet": cfg, "pulsepoint": cfg},
			bidderInfos:   map[string]adapters.BidderInfo{"ix": infoActive, "lifestreet": infoActive, "pulsepoint": infoActive},
			expected:      map[openrtb_ext.BidderName]adaptedBidder{"ix": expectedIx, "lifestreet": expectedLifestreet, "pulsepoint": expectedPulsepoint},
		},
		{
			description:   "All Disabled",
			adapterConfig: map[string]config.Adapter{"ix": cfg, "lifestreet": cfg, "pulsepoint": cfg},
			bidderInfos:   map[string]adapters.BidderInfo{"ix": infoDisabled, "lifestreet": infoDisabled, "pulsepoint": infoDisabled},
			expected:      map[openrtb_ext.BidderName]adaptedBidder{},
		},
		{
			description:   "All Unknown",
			adapterConfig: map[string]config.Adapter{"ix": cfg, "lifestreet": cfg, "pulsepoint": cfg},
			bidderInfos:   map[string]adapters.BidderInfo{"ix": infoUnknown, "lifestreet": infoUnknown, "pulsepoint": infoUnknown},
			expected:      map[openrtb_ext.BidderName]adaptedBidder{},
		},
		{
			description:   "Mixed",
			adapterConfig: map[string]config.Adapter{"ix": cfg, "lifestreet": cfg, "pulsepoint": cfg},
			bidderInfos:   map[string]adapters.BidderInfo{"ix": infoActive, "lifestreet": infoDisabled, "pulsepoint": infoUnknown},
			expected:      map[openrtb_ext.BidderName]adaptedBidder{"ix": expectedIx},
		},
	}

	for _, test := range testCases {
		result := buildExchangeBiddersLegacy(test.adapterConfig, test.bidderInfos)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestWrapWithMiddleware(t *testing.T) {
	appNexusBidder := fakeAdaptedBidder{}

	bidders := map[openrtb_ext.BidderName]adaptedBidder{
		openrtb_ext.BidderAppnexus: appNexusBidder,
	}

	wrapWithMiddleware(bidders)

	expected := map[openrtb_ext.BidderName]adaptedBidder{
		openrtb_ext.BidderAppnexus: &validatedBidder{appNexusBidder},
	}

	assert.Equal(t, expected, bidders)
}

func TestGetActiveBidders(t *testing.T) {
	testCases := []struct {
		description string
		bidderInfos map[string]adapters.BidderInfo
		expected    map[string]openrtb_ext.BidderName
	}{
		{
			description: "None",
			bidderInfos: map[string]adapters.BidderInfo{},
			expected:    map[string]openrtb_ext.BidderName{},
		},
		{
			description: "Active",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": infoActive},
			expected:    map[string]openrtb_ext.BidderName{"appnexus": openrtb_ext.BidderAppnexus},
		},
		{
			description: "Disabled",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": infoDisabled},
			expected:    map[string]openrtb_ext.BidderName{},
		},
		{
			description: "Unknown",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": infoUnknown},
			expected:    map[string]openrtb_ext.BidderName{"appnexus": openrtb_ext.BidderAppnexus},
		},
		{
			description: "Mixed",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": infoDisabled, "openx": infoActive, "rubicon": infoUnknown},
			expected:    map[string]openrtb_ext.BidderName{"openx": openrtb_ext.BidderOpenx, "rubicon": openrtb_ext.BidderRubicon},
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
		bidderInfos map[string]adapters.BidderInfo
		expected    map[string]string
	}{
		{
			description: "None",
			bidderInfos: map[string]adapters.BidderInfo{},
			expected:    map[string]string{},
		},
		{
			description: "Active",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": infoActive},
			expected:    map[string]string{},
		},
		{
			description: "Disabled",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": infoDisabled},
			expected: map[string]string{
				"appnexus": `Bidder "appnexus" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`,
			},
		},
		{
			description: "Unknown",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": infoUnknown},
			expected:    map[string]string{},
		},
		{
			description: "Mixed",
			bidderInfos: map[string]adapters.BidderInfo{"appnexus": infoDisabled, "openx": infoActive, "rubicon": infoUnknown},
			expected:    map[string]string{"appnexus": `Bidder "appnexus" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`},
		},
	}

	for _, test := range testCases {
		result := GetDisabledBiddersErrorMessages(test.bidderInfos)
		assert.Equal(t, test.expected, result, test.description)
	}
}

type fakeAdaptedBidder struct{}

func (fakeAdaptedBidder) requestBid(ctx context.Context, request *openrtb.BidRequest, name openrtb_ext.BidderName, bidAdjustment float64, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo) (*pbsOrtbSeatBid, []error) {
	return nil, nil
}

type fakeBidder struct {
	name string
}

func (fakeBidder) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	return nil, nil
}

func (fakeBidder) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return nil, nil
}

type fakeBuilder struct {
	bidder adapters.Bidder
	err    error
}

func (b fakeBuilder) Builder(name openrtb_ext.BidderName, cfg config.Adapter) (adapters.Bidder, error) {
	return b.bidder, b.err
}
