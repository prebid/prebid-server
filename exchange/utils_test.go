package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"

	gpplib "github.com/prebid/go-gpp"
	"github.com/prebid/go-gpp/constants"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/firstpartydata"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const deviceUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.87 Safari/537.36"

// permissionsMock mocks the Permissions interface for tests
type permissionsMock struct {
	allowAllBidders bool
	allowedBidders  []openrtb_ext.BidderName
	passGeo         bool
	passID          bool
	activitiesError error
}

func (p *permissionsMock) HostCookiesAllowed(ctx context.Context) (bool, error) {
	return true, nil
}

func (p *permissionsMock) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName) (bool, error) {
	return true, nil
}

func (p *permissionsMock) AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) gdpr.AuctionPermissions {
	permissions := gdpr.AuctionPermissions{
		PassGeo: p.passGeo,
		PassID:  p.passID,
	}

	if p.allowAllBidders {
		permissions.AllowBidRequest = true
		return permissions
	}

	for _, allowedBidder := range p.allowedBidders {
		if bidder == allowedBidder {
			permissions.AllowBidRequest = true
		}
	}

	return permissions
}

type fakePermissionsBuilder struct {
	permissions gdpr.Permissions
}

func (fpb fakePermissionsBuilder) Builder(gdpr.TCF2ConfigReader, gdpr.RequestInfo) gdpr.Permissions {
	return fpb.permissions
}

func assertReq(t *testing.T, bidderRequests []BidderRequest,
	applyCOPPA bool, consentedVendors map[string]bool,
) {
	// assert individual bidder requests
	assert.NotEqual(t, bidderRequests, 0, "cleanOpenRTBRequest should split request into individual bidder requests")

	// assert for PI data
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

func TestSplitImps(t *testing.T) {
	testCases := []struct {
		description     string
		givenImps       []openrtb2.Imp
		validatorErrors []error
		expectedImps    map[string][]openrtb2.Imp
		expectedError   string
	}{
		{
			description:   "Nil",
			givenImps:     nil,
			expectedImps:  map[string][]openrtb2.Imp{},
			expectedError: "",
		},
		{
			description:   "Empty",
			givenImps:     []openrtb2.Imp{},
			expectedImps:  map[string][]openrtb2.Imp{},
			expectedError: "",
		},
		{
			description: "1 Imp, 1 Bidder",
			givenImps: []openrtb2.Imp{
				{ID: "imp1", Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"imp1ParamA":"imp1ValueA"}}}}`)},
			},
			expectedImps: map[string][]openrtb2.Imp{
				"bidderA": {
					{ID: "imp1", Ext: json.RawMessage(`{"bidder":{"imp1ParamA":"imp1ValueA"}}`)},
				},
			},
			expectedError: "",
		},
		{
			description: "1 Imp, 2 Bidders",
			givenImps: []openrtb2.Imp{
				{ID: "imp1", Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"imp1ParamA":"imp1ValueA"},"bidderB":{"imp1ParamB":"imp1ValueB"}}}}`)},
			},
			expectedImps: map[string][]openrtb2.Imp{
				"bidderA": {
					{ID: "imp1", Ext: json.RawMessage(`{"bidder":{"imp1ParamA":"imp1ValueA"}}`)},
				},
				"bidderB": {
					{ID: "imp1", Ext: json.RawMessage(`{"bidder":{"imp1ParamB":"imp1ValueB"}}`)},
				},
			},
			expectedError: "",
		},
		{
			description: "2 Imps, 1 Bidders Each",
			givenImps: []openrtb2.Imp{
				{ID: "imp1", Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"imp1ParamA":"imp1ValueA"}}}}`)},
				{ID: "imp2", Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"imp2ParamA":"imp2ValueA"}}}}`)},
			},
			expectedImps: map[string][]openrtb2.Imp{
				"bidderA": {
					{ID: "imp1", Ext: json.RawMessage(`{"bidder":{"imp1ParamA":"imp1ValueA"}}`)},
					{ID: "imp2", Ext: json.RawMessage(`{"bidder":{"imp2ParamA":"imp2ValueA"}}`)},
				},
			},
			expectedError: "",
		},
		{
			description: "2 Imps, 2 Bidders Each",
			givenImps: []openrtb2.Imp{
				{ID: "imp1", Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"imp1paramA":"imp1valueA"},"bidderB":{"imp1paramB":"imp1valueB"}}}}`)},
				{ID: "imp2", Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"imp2paramA":"imp2valueA"},"bidderB":{"imp2paramB":"imp2valueB"}}}}`)},
			},
			expectedImps: map[string][]openrtb2.Imp{
				"bidderA": {
					{ID: "imp1", Ext: json.RawMessage(`{"bidder":{"imp1paramA":"imp1valueA"}}`)},
					{ID: "imp2", Ext: json.RawMessage(`{"bidder":{"imp2paramA":"imp2valueA"}}`)},
				},
				"bidderB": {
					{ID: "imp1", Ext: json.RawMessage(`{"bidder":{"imp1paramB":"imp1valueB"}}`)},
					{ID: "imp2", Ext: json.RawMessage(`{"bidder":{"imp2paramB":"imp2valueB"}}`)},
				},
			},
			expectedError: "",
		},
		{
			// This is a "happy path" integration test. Functionality is covered in detail by TestCreateSanitizedImpExt.
			description: "Other Fields - 2 Imps, 2 Bidders Each",
			givenImps: []openrtb2.Imp{
				{ID: "imp1", Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"imp1paramA":"imp1valueA"},"bidderB":{"imp1paramB":"imp1valueB"}}},"skadn":"imp1SkAdN"}`)},
				{ID: "imp2", Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"imp2paramA":"imp2valueA"},"bidderB":{"imp2paramB":"imp2valueB"}}},"skadn":"imp2SkAdN"}`)},
			},
			expectedImps: map[string][]openrtb2.Imp{
				"bidderA": {
					{ID: "imp1", Ext: json.RawMessage(`{"bidder":{"imp1paramA":"imp1valueA"},"skadn":"imp1SkAdN"}`)},
					{ID: "imp2", Ext: json.RawMessage(`{"bidder":{"imp2paramA":"imp2valueA"},"skadn":"imp2SkAdN"}`)},
				},
				"bidderB": {
					{ID: "imp1", Ext: json.RawMessage(`{"bidder":{"imp1paramB":"imp1valueB"},"skadn":"imp1SkAdN"}`)},
					{ID: "imp2", Ext: json.RawMessage(`{"bidder":{"imp2paramB":"imp2valueB"},"skadn":"imp2SkAdN"}`)},
				},
			},
			expectedError: "",
		},
		{
			description: "Malformed imp.ext",
			givenImps: []openrtb2.Imp{
				{ID: "imp1", Ext: json.RawMessage(`malformed`)},
			},
			expectedError: "invalid json for imp[0]: expect { or n, but found m",
		},
		{
			description: "Malformed imp.ext.prebid",
			givenImps: []openrtb2.Imp{
				{ID: "imp1", Ext: json.RawMessage(`{"prebid": malformed}`)},
			},
			expectedError: "invalid json for imp[0]: do not know how to skip: 109",
		},
		{
			description: "Malformed imp.ext.prebid.bidder",
			givenImps: []openrtb2.Imp{
				{ID: "imp1", Ext: json.RawMessage(`{"prebid": {"bidder": malformed}}`)},
			},
			expectedError: "invalid json for imp[0]: do not know how to skip: 109",
		},
		{
			description: "Malformed imp.ext.prebid.imp",
			givenImps: []openrtb2.Imp{
				{ID: "imp1", Ext: json.RawMessage(`{"prebid": {"imp": malformed}}`)},
			},
			expectedError: "invalid json for imp[0]: do not know how to skip: 109",
		},
		{
			description: "valid FPD at imp.ext.prebid.imp for valid bidder",
			givenImps: []openrtb2.Imp{
				{
					ID: "imp1",
					Banner: &openrtb2.Banner{
						Format: []openrtb2.Format{
							{
								W: 10,
								H: 20,
							},
						},
					},
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"imp1paramA":"imp1valueA"}},"imp":{"bidderA":{"id":"impFPD", "banner":{"format":[{"w":30,"h":40}]}}}}}`),
				},
			},
			expectedImps: map[string][]openrtb2.Imp{
				"bidderA": {
					{
						ID: "impFPD",
						Banner: &openrtb2.Banner{
							Format: []openrtb2.Format{
								{
									W: 30,
									H: 40,
								},
							},
						},
						Ext: json.RawMessage(`{"bidder":{"imp1paramA":"imp1valueA"}}`),
					},
				},
			},
			expectedError: "",
		},
		{
			description: "valid FPD at imp.ext.prebid.imp for unknown bidder",
			givenImps: []openrtb2.Imp{
				{
					ID: "imp1",
					Banner: &openrtb2.Banner{
						Format: []openrtb2.Format{
							{
								W: 10,
								H: 20,
							},
						},
					},
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderB":{"imp1paramB":"imp1valueB"}},"imp":{"bidderA":{"id":"impFPD", "banner":{"format":[{"w":30,"h":40}]}}}}}`),
				},
			},
			expectedImps: map[string][]openrtb2.Imp{
				"bidderB": {
					{
						ID: "imp1",
						Banner: &openrtb2.Banner{
							Format: []openrtb2.Format{
								{
									W: 10,
									H: 20,
								},
							},
						},
						Ext: json.RawMessage(`{"bidder":{"imp1paramB":"imp1valueB"}}`),
					},
				},
			},
			expectedError: "",
		},
		{
			description: "invalid FPD at imp.ext.prebid.imp for valid bidder",
			givenImps: []openrtb2.Imp{
				{
					ID: "imp1",
					Banner: &openrtb2.Banner{
						Format: []openrtb2.Format{
							{
								W: 10,
								H: 20,
							},
						},
					},
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"imp1paramA":"imp1valueA"}},"imp":{"bidderA":{"id":"impFPD", "banner":{"format":[{"w":0,"h":0}]}}}}}`),
				},
			},
			validatorErrors: []error{errors.New("some error")},
			expectedImps:    nil,
			expectedError:   "merging bidder imp first party data for imp imp1 results in an invalid imp: [some error]",
		},
	}

	for _, test := range testCases {
		imps, err := splitImps(test.givenImps, &mockRequestValidator{errors: test.validatorErrors}, nil, false, nil)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}

		assert.Equal(t, test.expectedImps, imps, test.description+":imps")
	}
}

func TestMergeImpFPD(t *testing.T) {
	imp1 := &openrtb2.Imp{
		ID: "imp1",
		Banner: &openrtb2.Banner{
			W: ptrutil.ToPtr[int64](200),
			H: ptrutil.ToPtr[int64](400),
		},
	}

	tests := []struct {
		description string
		imp         *openrtb2.Imp
		fpd         json.RawMessage
		wantImp     *openrtb2.Imp
		wantError   bool
	}{
		{
			description: "nil",
			imp:         nil,
			fpd:         nil,
			wantImp:     nil,
			wantError:   true,
		},
		{
			description: "nil_fpd",
			imp:         imp1,
			fpd:         nil,
			wantImp:     imp1,
			wantError:   true,
		},
		{
			description: "empty_fpd",
			imp:         imp1,
			fpd:         json.RawMessage(`{}`),
			wantImp:     imp1,
			wantError:   false,
		},
		{
			description: "nil_imp",
			imp:         nil,
			fpd:         json.RawMessage(`{}`),
			wantImp:     nil,
			wantError:   true,
		},
		{
			description: "zero_value_imp",
			imp:         &openrtb2.Imp{},
			fpd:         json.RawMessage(`{}`),
			wantImp:     &openrtb2.Imp{},
			wantError:   false,
		},
		{
			description: "invalid_json_on_existing_imp",
			imp: &openrtb2.Imp{
				Ext: json.RawMessage(`malformed`),
			},
			fpd: json.RawMessage(`{"ext": {"a":1}}`),
			wantImp: &openrtb2.Imp{
				Ext: json.RawMessage(`malformed`),
			},
			wantError: true,
		},
		{
			description: "invalid_json_in_fpd",
			imp: &openrtb2.Imp{
				Ext: json.RawMessage(`{"ext": {"a":1}}`),
			},
			fpd: json.RawMessage(`malformed`),
			wantImp: &openrtb2.Imp{
				Ext: json.RawMessage(`{"ext": {"a":1}}`),
			},
			wantError: true,
		},
		{
			description: "override_everything",
			imp: &openrtb2.Imp{
				ID:     "id1",
				Metric: []openrtb2.Metric{{Type: "type1", Value: 1, Vendor: "vendor1"}},
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](1),
					H: ptrutil.ToPtr[int64](2),
					Format: []openrtb2.Format{
						{
							W:   10,
							H:   20,
							Ext: json.RawMessage(`{"formatkey1":"formatval1"}`),
						},
					},
				},
				Instl:    1,
				BidFloor: 1,
				Ext:      json.RawMessage(`{"cool":"test"}`),
			},
			fpd: json.RawMessage(`{"id": "id2", "metric": [{"type":"type2", "value":2, "vendor":"vendor2"}], "banner": {"w":100, "h": 200, "format": [{"w":1000, "h":2000, "ext":{"formatkey1":"formatval2"}}]}, "instl":2, "bidfloor":2, "ext":{"cool":"test2"} }`),
			wantImp: &openrtb2.Imp{
				ID:     "id2",
				Metric: []openrtb2.Metric{{Type: "type2", Value: 2, Vendor: "vendor2"}},
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](100),
					H: ptrutil.ToPtr[int64](200),
					Format: []openrtb2.Format{
						{
							W:   1000,
							H:   2000,
							Ext: json.RawMessage(`{"formatkey1":"formatval2"}`),
						},
					},
				},
				Instl:    2,
				BidFloor: 2,
				Ext:      json.RawMessage(`{"cool":"test2"}`),
			},
		},
		{
			description: "override_partial_simple",
			imp:         imp1,
			fpd:         json.RawMessage(`{"id": "456", "banner": {"format": [{"w":1, "h":2}]} }`),
			wantImp: &openrtb2.Imp{
				ID: "456",
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](200),
					H: ptrutil.ToPtr[int64](400),
					Format: []openrtb2.Format{
						{
							W: 1,
							H: 2,
						},
					},
				},
			},
		},
		{
			description: "override_partial_complex",
			imp: &openrtb2.Imp{
				ID:     "id1",
				Metric: []openrtb2.Metric{{Type: "type1", Value: 1, Vendor: "vendor1"}},
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](1),
					H: ptrutil.ToPtr[int64](2),
					Format: []openrtb2.Format{
						{
							W:   10,
							H:   20,
							Ext: json.RawMessage(`{"formatkey1":"formatval1"}`),
						},
					},
				},
				Instl:        1,
				TagID:        "tag1",
				BidFloor:     1,
				Rwdd:         1,
				DT:           1,
				IframeBuster: []string{"buster1", "buster2"},
				Ext:          json.RawMessage(`{"cool1":"test1", "cool2":"test2"}`),
			},
			fpd: json.RawMessage(`{"id": "id2", "metric": [{"type":"type2", "value":2, "vendor":"vendor2"}], "banner": {"w":100, "format": [{"w":1000, "h":2000, "ext":{"formatkey1":"formatval11"}}]}, "instl":2, "bidfloor":2, "ext":{"cool1":"test11"} }`),
			wantImp: &openrtb2.Imp{
				ID:     "id2",
				Metric: []openrtb2.Metric{{Type: "type2", Value: 2, Vendor: "vendor2"}},
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](100),
					H: ptrutil.ToPtr[int64](2),
					Format: []openrtb2.Format{
						{
							W:   1000,
							H:   2000,
							Ext: json.RawMessage(`{"formatkey1":"formatval11"}`),
						},
					},
				},
				Instl:        2,
				TagID:        "tag1",
				BidFloor:     2,
				Rwdd:         1,
				DT:           1,
				IframeBuster: []string{"buster1", "buster2"},
				Ext:          json.RawMessage(`{"cool1":"test11","cool2":"test2"}`),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := mergeImpFPD(test.imp, test.fpd, 1)
			assert.Equal(t, test.wantImp, test.imp)

			if test.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateSanitizedImpExt(t *testing.T) {
	testCases := []struct {
		description       string
		givenImpExt       map[string]json.RawMessage
		givenImpExtPrebid map[string]json.RawMessage
		expected          map[string]json.RawMessage
		expectedError     string
	}{
		{
			description:       "Nil",
			givenImpExt:       nil,
			givenImpExtPrebid: nil,
			expected:          map[string]json.RawMessage{},
			expectedError:     "",
		},
		{
			description:       "Empty",
			givenImpExt:       map[string]json.RawMessage{},
			givenImpExtPrebid: map[string]json.RawMessage{},
			expected:          map[string]json.RawMessage{},
			expectedError:     "",
		},
		{
			description: "imp.ext.prebid - Bidder Only",
			givenImpExt: map[string]json.RawMessage{
				"prebid":  json.RawMessage(`"ignoredInFavorOfSeparatelyUnmarshalledImpExtPrebid"`),
				"data":    json.RawMessage(`"anyData"`),
				"context": json.RawMessage(`"anyContext"`),
				"skadn":   json.RawMessage(`"anySKAdNetwork"`),
				"gpid":    json.RawMessage(`"anyGPID"`),
				"tid":     json.RawMessage(`"anyTID"`),
			},
			givenImpExtPrebid: map[string]json.RawMessage{
				"bidder": json.RawMessage(`"anyBidder"`),
			},
			expected: map[string]json.RawMessage{
				"data":    json.RawMessage(`"anyData"`),
				"context": json.RawMessage(`"anyContext"`),
				"skadn":   json.RawMessage(`"anySKAdNetwork"`),
				"gpid":    json.RawMessage(`"anyGPID"`),
				"tid":     json.RawMessage(`"anyTID"`),
			},
			expectedError: "",
		},
		{
			description: "imp.ext.prebid - Bidder + Other Forbidden Value",
			givenImpExt: map[string]json.RawMessage{
				"prebid":  json.RawMessage(`"ignoredInFavorOfSeparatelyUnmarshalledImpExtPrebid"`),
				"data":    json.RawMessage(`"anyData"`),
				"context": json.RawMessage(`"anyContext"`),
				"skadn":   json.RawMessage(`"anySKAdNetwork"`),
				"gpid":    json.RawMessage(`"anyGPID"`),
				"tid":     json.RawMessage(`"anyTID"`),
			},
			givenImpExtPrebid: map[string]json.RawMessage{
				"bidder":    json.RawMessage(`"anyBidder"`),
				"forbidden": json.RawMessage(`"anyValue"`),
			},
			expected: map[string]json.RawMessage{
				"data":    json.RawMessage(`"anyData"`),
				"context": json.RawMessage(`"anyContext"`),
				"skadn":   json.RawMessage(`"anySKAdNetwork"`),
				"gpid":    json.RawMessage(`"anyGPID"`),
				"tid":     json.RawMessage(`"anyTID"`),
			},
			expectedError: "",
		},
		{
			description: "imp.ext.prebid - Bidder + Other Allowed Values",
			givenImpExt: map[string]json.RawMessage{
				"prebid":  json.RawMessage(`"ignoredInFavorOfSeparatelyUnmarshalledImpExtPrebid"`),
				"data":    json.RawMessage(`"anyData"`),
				"context": json.RawMessage(`"anyContext"`),
				"skadn":   json.RawMessage(`"anySKAdNetwork"`),
				"gpid":    json.RawMessage(`"anyGPID"`),
				"tid":     json.RawMessage(`"anyTID"`),
			},
			givenImpExtPrebid: map[string]json.RawMessage{
				"bidder":                json.RawMessage(`"anyBidder"`),
				"is_rewarded_inventory": json.RawMessage(`"anyIsRewardedInventory"`),
				"options":               json.RawMessage(`"anyOptions"`),
			},
			expected: map[string]json.RawMessage{
				"prebid":  json.RawMessage(`{"is_rewarded_inventory":"anyIsRewardedInventory","options":"anyOptions"}`),
				"data":    json.RawMessage(`"anyData"`),
				"context": json.RawMessage(`"anyContext"`),
				"skadn":   json.RawMessage(`"anySKAdNetwork"`),
				"gpid":    json.RawMessage(`"anyGPID"`),
				"tid":     json.RawMessage(`"anyTID"`),
			},
			expectedError: "",
		},
		{
			description: "imp.ext",
			givenImpExt: map[string]json.RawMessage{
				"arbitraryField": json.RawMessage(`"arbitraryValue"`),
				"data":           json.RawMessage(`"anyData"`),
				"context":        json.RawMessage(`"anyContext"`),
				"skadn":          json.RawMessage(`"anySKAdNetwork"`),
				"gpid":           json.RawMessage(`"anyGPID"`),
				"tid":            json.RawMessage(`"anyTID"`),
			},
			givenImpExtPrebid: map[string]json.RawMessage{},
			expected: map[string]json.RawMessage{
				"arbitraryField": json.RawMessage(`"arbitraryValue"`),
				"data":           json.RawMessage(`"anyData"`),
				"context":        json.RawMessage(`"anyContext"`),
				"skadn":          json.RawMessage(`"anySKAdNetwork"`),
				"gpid":           json.RawMessage(`"anyGPID"`),
				"tid":            json.RawMessage(`"anyTID"`),
			},
			expectedError: "",
		},
		{
			description: "imp.ext + imp.ext.prebid - Prebid Bidder Only",
			givenImpExt: map[string]json.RawMessage{
				"arbitraryField": json.RawMessage(`"arbitraryValue"`),
				"prebid":         json.RawMessage(`"ignoredInFavorOfSeparatelyUnmarshalledImpExtPrebid"`),
				"data":           json.RawMessage(`"anyData"`),
				"context":        json.RawMessage(`"anyContext"`),
				"skadn":          json.RawMessage(`"anySKAdNetwork"`),
				"gpid":           json.RawMessage(`"anyGPID"`),
				"tid":            json.RawMessage(`"anyTID"`),
			},
			givenImpExtPrebid: map[string]json.RawMessage{
				"bidder": json.RawMessage(`"anyBidder"`),
			},
			expected: map[string]json.RawMessage{
				"arbitraryField": json.RawMessage(`"arbitraryValue"`),
				"data":           json.RawMessage(`"anyData"`),
				"context":        json.RawMessage(`"anyContext"`),
				"skadn":          json.RawMessage(`"anySKAdNetwork"`),
				"gpid":           json.RawMessage(`"anyGPID"`),
				"tid":            json.RawMessage(`"anyTID"`),
			},
			expectedError: "",
		},
		{
			description: "imp.ext + imp.ext.prebid - Prebid Bidder + Other Forbidden Value",
			givenImpExt: map[string]json.RawMessage{
				"arbitraryField": json.RawMessage(`"arbitraryValue"`),
				"prebid":         json.RawMessage(`"ignoredInFavorOfSeparatelyUnmarshalledImpExtPrebid"`),
				"data":           json.RawMessage(`"anyData"`),
				"context":        json.RawMessage(`"anyContext"`),
				"skadn":          json.RawMessage(`"anySKAdNetwork"`),
				"gpid":           json.RawMessage(`"anyGPID"`),
				"tid":            json.RawMessage(`"anyTID"`),
			},
			givenImpExtPrebid: map[string]json.RawMessage{
				"bidder":    json.RawMessage(`"anyBidder"`),
				"forbidden": json.RawMessage(`"anyValue"`),
			},
			expected: map[string]json.RawMessage{
				"arbitraryField": json.RawMessage(`"arbitraryValue"`),
				"data":           json.RawMessage(`"anyData"`),
				"context":        json.RawMessage(`"anyContext"`),
				"skadn":          json.RawMessage(`"anySKAdNetwork"`),
				"gpid":           json.RawMessage(`"anyGPID"`),
				"tid":            json.RawMessage(`"anyTID"`),
			},
			expectedError: "",
		},
		{
			description: "imp.ext + imp.ext.prebid - Prebid Bidder + Other Allowed Values",
			givenImpExt: map[string]json.RawMessage{
				"arbitraryField": json.RawMessage(`"arbitraryValue"`),
				"prebid":         json.RawMessage(`"ignoredInFavorOfSeparatelyUnmarshalledImpExtPrebid"`),
				"data":           json.RawMessage(`"anyData"`),
				"context":        json.RawMessage(`"anyContext"`),
				"skadn":          json.RawMessage(`"anySKAdNetwork"`),
				"gpid":           json.RawMessage(`"anyGPID"`),
				"tid":            json.RawMessage(`"anyTID"`),
			},
			givenImpExtPrebid: map[string]json.RawMessage{
				"bidder":                json.RawMessage(`"anyBidder"`),
				"is_rewarded_inventory": json.RawMessage(`"anyIsRewardedInventory"`),
				"options":               json.RawMessage(`"anyOptions"`),
			},
			expected: map[string]json.RawMessage{
				"arbitraryField": json.RawMessage(`"arbitraryValue"`),
				"prebid":         json.RawMessage(`{"is_rewarded_inventory":"anyIsRewardedInventory","options":"anyOptions"}`),
				"data":           json.RawMessage(`"anyData"`),
				"context":        json.RawMessage(`"anyContext"`),
				"skadn":          json.RawMessage(`"anySKAdNetwork"`),
				"gpid":           json.RawMessage(`"anyGPID"`),
				"tid":            json.RawMessage(`"anyTID"`),
			},
			expectedError: "",
		},
	}

	for _, test := range testCases {
		result, err := createSanitizedImpExt(test.givenImpExt, test.givenImpExtPrebid)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}

		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestCleanOpenRTBRequests(t *testing.T) {
	emptyTCF2Config := gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{})

	testCases := []struct {
		req              AuctionRequest
		bidReqAssertions func(t *testing.T, bidderRequests []BidderRequest,
			applyCOPPA bool, consentedVendors map[string]bool)
		hasError         bool
		applyCOPPA       bool
		consentedVendors map[string]bool
	}{
		{
			req:              AuctionRequest{BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: getTestBuildRequest(t)}, UserSyncs: &emptyUsersync{}, TCF2Config: emptyTCF2Config},
			bidReqAssertions: assertReq,
			hasError:         false,
			applyCOPPA:       true,
			consentedVendors: map[string]bool{"appnexus": true},
		},
		{
			req:              AuctionRequest{BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: newAdapterAliasBidRequest()}, UserSyncs: &emptyUsersync{}, TCF2Config: emptyTCF2Config},
			bidReqAssertions: assertReq,
			hasError:         false,
			applyCOPPA:       false,
			consentedVendors: map[string]bool{"appnexus": true},
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

		gdprPermsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder

		reqSplitter := &requestSplitter{
			bidderToSyncerKey: map[string]string{},
			me:                &metrics.MetricsEngineMock{},
			privacyConfig:     privacyConfig,
			gdprPermsBuilder:  gdprPermsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}
		bidderRequests, _, err := reqSplitter.cleanOpenRTBRequests(context.Background(), test.req, nil, gdpr.SignalNo, false, map[string]float64{})
		if test.hasError {
			assert.NotNil(t, err, "Error shouldn't be nil")
		} else {
			assert.Nil(t, err, "Err should be nil")
			test.bidReqAssertions(t, bidderRequests, test.applyCOPPA, test.consentedVendors)
		}
	}
}

func TestCleanOpenRTBRequestsWithFPD(t *testing.T) {
	fpd := make(map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData)

	apnFpd := firstpartydata.ResolvedFirstPartyData{
		Site: &openrtb2.Site{Name: "fpdApnSite"},
		App:  &openrtb2.App{Name: "fpdApnApp"},
		User: &openrtb2.User{Keywords: "fpdApnUser"},
	}
	fpd[openrtb_ext.BidderName("rubicon")] = &apnFpd

	emptyTCF2Config := gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{})

	testCases := []struct {
		description string
		req         AuctionRequest
		fpdExpected bool
	}{
		{
			description: "Pass valid FPD data for bidder not found in the request",
			req:         AuctionRequest{BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: getTestBuildRequest(t)}, UserSyncs: &emptyUsersync{}, FirstPartyData: fpd, TCF2Config: emptyTCF2Config},
			fpdExpected: false,
		},
		{
			description: "Pass valid FPD data for bidders specified in request",
			req:         AuctionRequest{BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: newAdapterAliasBidRequest()}, UserSyncs: &emptyUsersync{}, FirstPartyData: fpd, TCF2Config: emptyTCF2Config},
			fpdExpected: true,
		},
		{
			description: "Bidders specified in request but there is no fpd data for this bidder",
			req:         AuctionRequest{BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: newAdapterAliasBidRequest()}, UserSyncs: &emptyUsersync{}, FirstPartyData: make(map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData), TCF2Config: emptyTCF2Config},
			fpdExpected: false,
		},
		{
			description: "No FPD data passed",
			req:         AuctionRequest{BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: newAdapterAliasBidRequest()}, UserSyncs: &emptyUsersync{}, FirstPartyData: nil, TCF2Config: emptyTCF2Config},
			fpdExpected: false,
		},
	}

	for _, test := range testCases {

		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder

		reqSplitter := &requestSplitter{
			bidderToSyncerKey: map[string]string{},
			me:                &metrics.MetricsEngineMock{},
			privacyConfig:     config.Privacy{},
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}

		bidderRequests, _, err := reqSplitter.cleanOpenRTBRequests(context.Background(), test.req, nil, gdpr.SignalNo, false, map[string]float64{})
		assert.Empty(t, err, "No errors should be returned")
		for _, bidderRequest := range bidderRequests {
			bidderName := bidderRequest.BidderName
			if test.fpdExpected {
				assert.Equal(t, fpd[bidderName].Site.Name, bidderRequest.BidRequest.Site.Name, "Incorrect FPD site name")
				assert.Equal(t, fpd[bidderName].App.Name, bidderRequest.BidRequest.App.Name, "Incorrect FPD app name")
				assert.Equal(t, fpd[bidderName].User.Keywords, bidderRequest.BidRequest.User.Keywords, "Incorrect FPD user keywords")
				assert.Equal(t, test.req.BidRequestWrapper.User.BuyerUID, bidderRequest.BidRequest.User.BuyerUID, "Incorrect FPD user buyerUID")
			} else {
				assert.Equal(t, "", bidderRequest.BidRequest.Site.Name, "Incorrect FPD site name")
				assert.Equal(t, "", bidderRequest.BidRequest.User.Keywords, "Incorrect FPD user keywords")
			}
		}
	}
}

func TestExtractAdapterReqBidderParamsMap(t *testing.T) {
	tests := []struct {
		name            string
		givenBidRequest *openrtb2.BidRequest
		want            map[string]json.RawMessage
		wantErr         error
	}{
		{
			name:            "nil req",
			givenBidRequest: nil,
			want:            nil,
			wantErr:         errors.New("error bidRequest should not be nil"),
		},
		{
			name:            "nil req.ext",
			givenBidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{}}`)},
			want:            nil,
			wantErr:         nil,
		},
		{
			name:            "malformed req.ext",
			givenBidRequest: &openrtb2.BidRequest{Ext: json.RawMessage("malformed")},
			want:            nil,
			wantErr:         errors.New("error decoding Request.ext : expect { or n, but found m"),
		},
		{
			name:            "extract bidder params from req.Ext for input request in adapter code",
			givenBidRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"bidderparams": {"profile": 1234, "version": 1}}}`)},
			want:            map[string]json.RawMessage{"profile": json.RawMessage(`1234`), "version": json.RawMessage(`1`)},
			wantErr:         nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractReqExtBidderParamsMap(tt.givenBidRequest)
			assert.Equal(t, tt.wantErr, err, "err")
			assert.Equal(t, tt.want, got, "result")
		})
	}
}

func TestCleanOpenRTBRequestsWithBidResponses(t *testing.T) {
	bidRespId1 := json.RawMessage(`{"id": "resp_id1"}`)
	bidRespId2 := json.RawMessage(`{"id": "resp_id2"}`)

	testCases := []struct {
		description            string
		storedBidResponses     map[string]map[string]json.RawMessage
		imps                   []openrtb2.Imp
		expectedBidderRequests map[string]BidderRequest
	}{
		{
			description: "Request with imp with one bidder stored bid response",
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"bidderA": bidRespId1},
			},
			imps: []openrtb2.Imp{
				{
					ID: "imp-id1",
					Video: &openrtb2.Video{
						W: ptrutil.ToPtr[int64](300),
						H: ptrutil.ToPtr[int64](250),
					},
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"}}}}`),
				},
			},
			expectedBidderRequests: map[string]BidderRequest{
				"bidderA": {
					BidRequest: &openrtb2.BidRequest{Imp: nil},
					BidderName: "bidderA",
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId1,
					},
				},
			},
		},
		{
			description: "Request with imps with and without stored bid response for one bidder",
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"bidderA": bidRespId1},
			},
			imps: []openrtb2.Imp{
				{
					ID: "imp-id1",
					Video: &openrtb2.Video{
						W: ptrutil.ToPtr[int64](300),
						H: ptrutil.ToPtr[int64](250),
					},
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"}}}}`),
				},
				{
					ID:  "imp-id2",
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"}}}}`),
				},
			},
			expectedBidderRequests: map[string]BidderRequest{
				"bidderA": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id2", Ext: json.RawMessage(`{"bidder":{"placementId":"123"}}`)},
					}},
					BidderName: "bidderA",
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId1,
					},
				},
			},
		},
		{
			description: "Request with imp with 2 bidders stored bid response",
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"bidderA": bidRespId1, "bidderB": bidRespId2},
			},
			imps: []openrtb2.Imp{
				{
					ID: "imp-id1",
					Video: &openrtb2.Video{
						W: ptrutil.ToPtr[int64](300),
						H: ptrutil.ToPtr[int64](250),
					},
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"},"bidderB":{"placementId":"456"}}}}`),
				},
			},
			expectedBidderRequests: map[string]BidderRequest{
				"bidderA": {
					BidRequest: &openrtb2.BidRequest{Imp: nil},
					BidderName: "bidderA",
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId1,
					},
				},
				"bidderB": {
					BidRequest: &openrtb2.BidRequest{Imp: nil},
					BidderName: "bidderB",
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId2,
					},
				},
			},
		},
		{
			description: "Request with 2 imps: with 2 bidders stored bid response and imp without stored responses",
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"bidderA": bidRespId1, "bidderB": bidRespId2},
			},
			imps: []openrtb2.Imp{
				{
					ID: "imp-id1",
					Video: &openrtb2.Video{
						W: ptrutil.ToPtr[int64](300),
						H: ptrutil.ToPtr[int64](250),
					},
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"},"bidderB":{"placementId":"456"}}}}`),
				},
				{
					ID:  "imp-id2",
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"}}}}`),
				},
			},
			expectedBidderRequests: map[string]BidderRequest{
				"bidderA": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id2", Ext: json.RawMessage(`{"bidder":{"placementId":"123"}}`)},
					}},
					BidderName: "bidderA",
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId1,
					},
				},
				"bidderB": {
					BidRequest: &openrtb2.BidRequest{Imp: nil},
					BidderName: "bidderB",
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId2,
					},
				},
			},
		},
		{
			description: "Request with 3 imps: with 2 bidders stored bid response and 2 imps without stored responses",
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"bidderA": bidRespId1, "bidderB": bidRespId2},
			},
			imps: []openrtb2.Imp{
				{
					ID:  "imp-id3",
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderC":{"placementId":"1234"}}}}`),
				},
				{
					ID: "imp-id1",
					Video: &openrtb2.Video{
						W: ptrutil.ToPtr[int64](300),
						H: ptrutil.ToPtr[int64](250),
					},
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"},"bidderB":{"placementId":"456"}}}}`),
				},
				{
					ID:  "imp-id2",
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"}}}}`),
				},
			},
			expectedBidderRequests: map[string]BidderRequest{
				"bidderA": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id2", Ext: json.RawMessage(`{"bidder":{"placementId":"123"}}`)},
					}},
					BidderName: "bidderA",
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId1,
					},
				},
				"bidderB": {
					BidRequest: &openrtb2.BidRequest{Imp: nil},
					BidderName: "bidderB",
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId2,
					},
				},
				"bidderC": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id3", Ext: json.RawMessage(`{"bidder":{"placementId":"1234"}}`)},
					}},
					BidderName:            "bidderC",
					BidderStoredResponses: nil,
				},
			},
		},
		{
			description: "Request with 2 imps: with 1 bidders stored bid response and imp without stored responses and with the same bidder",
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id2": {"bidderA": bidRespId2},
			},
			imps: []openrtb2.Imp{
				{
					ID:  "imp-id1",
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"}}}}`),
				},
				{
					ID:  "imp-id2",
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"}}}}`),
				},
			},
			expectedBidderRequests: map[string]BidderRequest{
				"bidderA": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id1", Ext: json.RawMessage(`{"bidder":{"placementId":"123"}}`)},
					}},
					BidderName: "bidderA",
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id2": bidRespId2,
					},
				},
			},
		},
		{
			description: "Request with 2 imps with stored responses and with the same bidder",
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"bidderA": bidRespId1},
				"imp-id2": {"bidderA": bidRespId2},
			},
			imps: []openrtb2.Imp{
				{
					ID:  "imp-id1",
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"}}}}`),
				},
				{
					ID:  "imp-id2",
					Ext: json.RawMessage(`{"prebid":{"bidder":{"bidderA":{"placementId":"123"}}}}`),
				},
			},
			expectedBidderRequests: map[string]BidderRequest{
				"bidderA": {
					BidRequest: &openrtb2.BidRequest{Imp: nil},
					BidderName: "bidderA",
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId1,
						"imp-id2": bidRespId2,
					},
				},
			},
		},
	}

	for _, test := range testCases {

		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder

		auctionReq := AuctionRequest{
			BidRequestWrapper:  &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Imp: test.imps}},
			UserSyncs:          &emptyUsersync{},
			StoredBidResponses: test.storedBidResponses,
			TCF2Config:         gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}

		reqSplitter := &requestSplitter{
			bidderToSyncerKey: map[string]string{},
			me:                &metrics.MetricsEngineMock{},
			privacyConfig:     config.Privacy{},
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}

		actualBidderRequests, _, err := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, nil, gdpr.SignalNo, false, map[string]float64{})
		assert.Empty(t, err, "No errors should be returned")
		assert.Len(t, actualBidderRequests, len(test.expectedBidderRequests), "result len doesn't match for testCase %s", test.description)
		for _, actualBidderRequest := range actualBidderRequests {
			bidderName := string(actualBidderRequest.BidderName)
			assert.Equal(t, test.expectedBidderRequests[bidderName].BidRequest.Imp, actualBidderRequest.BidRequest.Imp, "incorrect Impressions for testCase %s", test.description)
			assert.Equal(t, test.expectedBidderRequests[bidderName].BidderStoredResponses, actualBidderRequest.BidderStoredResponses, "incorrect Bidder Stored Responses for testCase %s", test.description)
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
		req := newBidRequest()
		req.Ext = test.reqExt
		req.Regs = &openrtb2.Regs{
			USPrivacy: test.ccpaConsent,
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
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
			UserSyncs:         &emptyUsersync{},
			Account:           accountConfig,
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, accountConfig.GDPR),
		}

		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder

		metricsMock := metrics.MetricsEngineMock{}
		metricsMock.Mock.On("RecordAdapterBuyerUIDScrubbed", mock.Anything).Return()

		bidderToSyncerKey := map[string]string{}
		reqSplitter := &requestSplitter{
			bidderToSyncerKey: bidderToSyncerKey,
			me:                &metricsMock,
			privacyConfig:     privacyConfig,
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}

		bidderRequests, privacyLabels, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, nil, gdpr.SignalNo, false, map[string]float64{})
		result := bidderRequests[0]

		assert.Nil(t, errs)
		if test.expectDataScrub {
			assert.Equal(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.BidRequest.Device.DIDMD5, "", test.description+":Device.DIDMD5")
			metricsMock.AssertCalled(t, "RecordAdapterBuyerUIDScrubbed", openrtb_ext.BidderAppnexus)
		} else {
			assert.NotEqual(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.BidRequest.Device.DIDMD5, "", test.description+":Device.DIDMD5")
			metricsMock.AssertNotCalled(t, "RecordAdapterBuyerUIDScrubbed", openrtb_ext.BidderAppnexus)
		}
		assert.Equal(t, test.expectPrivacyLabels, privacyLabels, test.description+":PrivacyLabels")
	}
}

func TestCleanOpenRTBRequestsCCPAErrors(t *testing.T) {
	testCases := []struct {
		description    string
		reqExt         json.RawMessage
		reqRegsPrivacy string
		expectError    error
	}{
		{
			description:    "Invalid Consent",
			reqExt:         json.RawMessage(`{"prebid":{"nosale":["*"]}}`),
			reqRegsPrivacy: "malformed",
			expectError: &errortypes.Warning{
				Message:     "request.regs.ext.us_privacy must contain 4 characters",
				WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
			},
		},
		{
			description:    "Invalid No Sale Bidders",
			reqExt:         json.RawMessage(`{"prebid":{"nosale":["*", "another"]}}`),
			reqRegsPrivacy: "1NYN",
			expectError:    errors.New("request.ext.prebid.nosale is invalid: can only specify all bidders if no other bidders are provided"),
		},
	}

	for _, test := range testCases {
		req := newBidRequest()
		req.Ext = test.reqExt
		req.Regs = &openrtb2.Regs{USPrivacy: test.reqRegsPrivacy}

		var reqExtStruct openrtb_ext.ExtRequest
		err := jsonutil.UnmarshalValid(req.Ext, &reqExtStruct)
		assert.NoError(t, err, test.description+":marshal_ext")

		auctionReq := AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
			UserSyncs:         &emptyUsersync{},
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}

		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder

		privacyConfig := config.Privacy{
			CCPA: config.CCPA{
				Enforce: true,
			},
		}
		bidderToSyncerKey := map[string]string{}
		metrics := metrics.MetricsEngineMock{}

		reqSplitter := &requestSplitter{
			bidderToSyncerKey: bidderToSyncerKey,
			me:                &metrics,
			privacyConfig:     privacyConfig,
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}

		_, _, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, &reqExtStruct, gdpr.SignalNo, false, map[string]float64{})

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
		req := newBidRequest()
		req.Regs = &openrtb2.Regs{COPPA: test.coppa}

		auctionReq := AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
			UserSyncs:         &emptyUsersync{},
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}

		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder

		bidderToSyncerKey := map[string]string{}
		metrics := metrics.MetricsEngineMock{}

		reqSplitter := &requestSplitter{
			bidderToSyncerKey: bidderToSyncerKey,
			me:                &metrics,
			privacyConfig:     config.Privacy{},
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}

		bidderRequests, privacyLabels, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, nil, gdpr.SignalNo, false, map[string]float64{})
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
	const seller1SChain string = `"schain":{"complete":1,"nodes":[{"asi":"directseller1.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}`
	const seller2SChain string = `"schain":{"complete":2,"nodes":[{"asi":"directseller2.com","sid":"00002","rid":"BidRequest2","hp":2}],"ver":"2.0"}`

	testCases := []struct {
		description   string
		inExt         json.RawMessage
		inSChain      *openrtb2.SupplyChain
		outRequestExt json.RawMessage
		outSource     *openrtb2.Source
		hasError      bool
		ortbVersion   string
	}{
		{
			description:   "nil",
			inExt:         nil,
			inSChain:      nil,
			outRequestExt: nil,
			outSource: &openrtb2.Source{
				TID:    "testTID",
				SChain: nil,
				Ext:    nil,
			},
		},
		{
			description: "Supply Chain defined in request.Source.supplyChain",
			inExt:       nil,
			inSChain: &openrtb2.SupplyChain{
				Complete: 1,
				Ver:      "1.0",
				Ext:      nil,
				Nodes: []openrtb2.SupplyChainNode{
					{
						ASI: "directseller1.com",
						SID: "00001",
						RID: "BidRequest1",
						HP:  openrtb2.Int8Ptr(1),
						Ext: nil,
					},
				},
			},
			outRequestExt: nil,
			outSource: &openrtb2.Source{
				TID: "testTID",
				SChain: &openrtb2.SupplyChain{
					Complete: 1,
					Ver:      "1.0",
					Ext:      nil,
					Nodes: []openrtb2.SupplyChainNode{
						{
							ASI: "directseller1.com",
							SID: "00001",
							RID: "BidRequest1",
							HP:  openrtb2.Int8Ptr(1),
							Ext: nil,
						},
					},
				},
				Ext: nil,
			},
			ortbVersion: "2.6",
		},
		{
			description:   "Supply Chain defined in request.ext.prebid.schains",
			inExt:         json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `}]}}`),
			inSChain:      nil,
			outRequestExt: nil,
			outSource: &openrtb2.Source{
				TID: "testTID",
				SChain: &openrtb2.SupplyChain{
					Complete: 1,
					Ver:      "1.0",
					Ext:      nil,
					Nodes: []openrtb2.SupplyChainNode{
						{
							ASI: "directseller1.com",
							SID: "00001",
							RID: "BidRequest1",
							HP:  openrtb2.Int8Ptr(1),
							Ext: nil,
						},
					},
				},
				Ext: nil,
			},
			ortbVersion: "2.6",
		},
		{
			description: "schainwriter instantation error -- multiple bidder schains in ext.prebid.schains.",
			inExt:       json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `},{"bidders":["appnexus"],` + seller2SChain + `}]}}`),
			inSChain: &openrtb2.SupplyChain{
				Complete: 1,
				Ver:      "1.0",
				Ext:      nil,
				Nodes: []openrtb2.SupplyChainNode{
					{
						ASI: "directseller1.com",
						SID: "00001",
						RID: "BidRequest1",
						HP:  openrtb2.Int8Ptr(1),
						Ext: nil,
					},
				},
			},

			outRequestExt: nil,
			outSource:     nil,
			hasError:      true,
		},
	}

	for _, test := range testCases {
		req := newBidRequest()
		if test.inSChain != nil {
			req.Source.SChain = test.inSChain
		}

		var extRequest *openrtb_ext.ExtRequest
		if test.inExt != nil {
			req.Ext = test.inExt
			extRequest = &openrtb_ext.ExtRequest{}
			err := jsonutil.UnmarshalValid(req.Ext, extRequest)
			assert.NoErrorf(t, err, test.description+":Error unmarshaling inExt")
		}

		auctionReq := AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
			UserSyncs:         &emptyUsersync{},
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}

		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder

		reqSplitter := &requestSplitter{
			bidderToSyncerKey: map[string]string{},
			me:                &metrics.MetricsEngineMock{},
			privacyConfig:     config.Privacy{},
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{"appnexus": config.BidderInfo{OpenRTB: &config.OpenRTBInfo{Version: test.ortbVersion}}},
		}

		bidderRequests, _, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, extRequest, gdpr.SignalNo, false, map[string]float64{})
		if test.hasError == true {
			assert.NotNil(t, errs)
			assert.Len(t, bidderRequests, 0)
		} else {
			result := bidderRequests[0]
			assert.Nil(t, errs)
			assert.Equal(t, test.outSource, result.BidRequest.Source, test.description+":Source")
			assert.Equal(t, test.outRequestExt, result.BidRequest.Ext, test.description+":Ext")
		}
	}
}

func TestCleanOpenRTBRequestsBidderParams(t *testing.T) {
	testCases := []struct {
		description string
		inExt       json.RawMessage
		expectedExt map[string]json.RawMessage
		hasError    bool
	}{
		{
			description: "Nil Bidder params",
			inExt:       nil,
			expectedExt: getExpectedReqExt(true, false, false),
			hasError:    false,
		},
		{
			description: "Bidder params for single partner",
			inExt:       json.RawMessage(`{"prebid":{"bidderparams":{"pubmatic":{"profile":1234,"version":2}}}}`),
			expectedExt: getExpectedReqExt(false, true, false),
			hasError:    false,
		},
		{
			description: "Bidder params for two partners",
			inExt:       json.RawMessage(`{"prebid":{"bidderparams":{"pubmatic":{"profile":1234,"version":2},"appnexus":{"key1":123,"key2":{"innerKey1":"innerValue1"}}}}}`),
			expectedExt: getExpectedReqExt(false, true, true),
			hasError:    false,
		},
	}

	for _, test := range testCases {
		req := newBidRequestWithBidderParams()
		var extRequest *openrtb_ext.ExtRequest
		if test.inExt != nil {
			req.Ext = test.inExt
			extRequest = &openrtb_ext.ExtRequest{}
			err := jsonutil.UnmarshalValid(req.Ext, extRequest)
			assert.NoErrorf(t, err, test.description+":Error unmarshaling inExt")
		}

		auctionReq := AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
			UserSyncs:         &emptyUsersync{},
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}

		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder

		reqSplitter := &requestSplitter{
			bidderToSyncerKey: map[string]string{},
			me:                &metrics.MetricsEngineMock{},
			privacyConfig:     config.Privacy{},
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}

		bidderRequests, _, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, extRequest, gdpr.SignalNo, false, map[string]float64{})
		if test.hasError == true {
			assert.NotNil(t, errs)
			assert.Len(t, bidderRequests, 0)
		} else {
			assert.Nil(t, errs)
			for _, r := range bidderRequests {
				expected := test.expectedExt[r.BidderName.String()]
				actual := r.BidRequest.Ext
				assert.Equal(t, expected, actual, test.description+" Req:Ext.Prebid.BidderParams")
			}
		}
	}
}

func getExpectedReqExt(nilExt, includePubmaticParams, includeAppnexusParams bool) map[string]json.RawMessage {
	bidderParamsMap := make(map[string]json.RawMessage)

	if nilExt {
		bidderParamsMap["pubmatic"] = nil
		bidderParamsMap["appnexus"] = nil
		return bidderParamsMap
	}

	if includePubmaticParams {
		bidderParamsMap["pubmatic"] = json.RawMessage(`{"prebid":{"bidderparams":{"profile":1234,"version":2}}}`)
	} else {
		bidderParamsMap["pubmatic"] = nil
	}

	if includeAppnexusParams {
		bidderParamsMap["appnexus"] = json.RawMessage(`{"prebid":{"bidderparams":{"key1":123,"key2":{"innerKey1":"innerValue1"}}}}`)
	} else {
		bidderParamsMap["appnexus"] = nil
	}

	return bidderParamsMap
}

func TestGetExtCacheInstructions(t *testing.T) {
	var boolFalse, boolTrue *bool = new(bool), new(bool)
	*boolFalse = false
	*boolTrue = true

	testCases := []struct {
		desc                 string
		requestExtPrebid     *openrtb_ext.ExtRequestPrebid
		outCacheInstructions extCacheInstructions
	}{
		{
			desc:             "Nil request ext, all cache flags false except for returnCreative that defaults to true",
			requestExtPrebid: nil,
			outCacheInstructions: extCacheInstructions{
				cacheBids:      false,
				cacheVAST:      false,
				returnCreative: true,
			},
		},
		{
			desc: "Non-nil request ext, nil Cache field, all cache flags false except for returnCreative that defaults to true",
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: nil,
			},
			outCacheInstructions: extCacheInstructions{
				cacheBids:      false,
				cacheVAST:      false,
				returnCreative: true,
			},
		},
		{
			desc: "Non-nil Cache field, both ExtRequestPrebidCacheBids and ExtRequestPrebidCacheVAST nil returnCreative that defaults to true",
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    nil,
					VastXML: nil,
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    nil,
					VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{},
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    nil,
					VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolFalse},
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    nil,
					VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolTrue},
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{},
					VastXML: nil,
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolFalse},
					VastXML: nil,
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolTrue},
					VastXML: nil,
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{},
					VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{},
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{},
					VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolTrue},
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{},
					VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolFalse},
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolTrue},
					VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{},
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolFalse},
					VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{},
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolFalse},
					VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolTrue},
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
			requestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids:    &openrtb_ext.ExtRequestPrebidCacheBids{ReturnCreative: boolTrue},
					VastXML: &openrtb_ext.ExtRequestPrebidCacheVAST{ReturnCreative: boolFalse},
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
		cacheInstructions := getExtCacheInstructions(test.requestExtPrebid)

		assert.Equal(t, test.outCacheInstructions.cacheBids, cacheInstructions.cacheBids, "%s. Unexpected shouldCacheBids value. \n", test.desc)
		assert.Equal(t, test.outCacheInstructions.cacheVAST, cacheInstructions.cacheVAST, "%s. Unexpected shouldCacheVAST value. \n", test.desc)
		assert.Equal(t, test.outCacheInstructions.returnCreative, cacheInstructions.returnCreative, "%s. Unexpected returnCreative value. \n", test.desc)
	}
}

func TestGetExtTargetData(t *testing.T) {
	testCases := []struct {
		name                   string
		givenRequestExtPrebid  *openrtb_ext.ExtRequestPrebid
		givenCacheInstructions extCacheInstructions
		givenAccount           config.Account
		givenWarning           []*errortypes.Warning
		expectTargetData       *targetData
	}{
		{
			name:                   "nil",
			givenRequestExtPrebid:  nil,
			givenCacheInstructions: extCacheInstructions{cacheBids: true, cacheVAST: true},
			givenAccount:           config.Account{},
			givenWarning:           nil,
			expectTargetData:       nil,
		},
		{
			name:                   "nil-targeting",
			givenRequestExtPrebid:  &openrtb_ext.ExtRequestPrebid{Targeting: nil},
			givenCacheInstructions: extCacheInstructions{cacheBids: true, cacheVAST: true},
			givenAccount:           config.Account{},
			givenWarning:           nil,
			expectTargetData:       nil,
		},
		{
			name: "populated-full",
			givenRequestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Targeting: &openrtb_ext.ExtRequestTargeting{
					AlwaysIncludeDeals:        true,
					IncludeBidderKeys:         ptrutil.ToPtr(true),
					IncludeFormat:             true,
					IncludeWinners:            ptrutil.ToPtr(true),
					MediaTypePriceGranularity: &openrtb_ext.MediaTypePriceGranularity{},
					PreferDeals:               true,
					PriceGranularity: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges:    []openrtb_ext.GranularityRange{{Min: 0.00, Max: 5.00, Increment: 1.00}},
					},
				},
			},
			givenCacheInstructions: extCacheInstructions{
				cacheBids: true,
				cacheVAST: true,
			},
			givenAccount: config.Account{},
			givenWarning: nil,
			expectTargetData: &targetData{
				alwaysIncludeDeals:        true,
				includeBidderKeys:         true,
				includeCacheBids:          true,
				includeCacheVast:          true,
				includeFormat:             true,
				includeWinners:            true,
				mediaTypePriceGranularity: openrtb_ext.MediaTypePriceGranularity{},
				preferDeals:               true,
				priceGranularity: openrtb_ext.PriceGranularity{
					Precision: ptrutil.ToPtr(2),
					Ranges:    []openrtb_ext.GranularityRange{{Min: 0.00, Max: 5.00, Increment: 1.00}},
				},
				prefix: DefaultKeyPrefix,
			},
		},
		{
			name: "populated-pointers-nil",
			givenRequestExtPrebid: &openrtb_ext.ExtRequestPrebid{
				Targeting: &openrtb_ext.ExtRequestTargeting{
					AlwaysIncludeDeals:        true,
					IncludeBidderKeys:         nil,
					IncludeFormat:             true,
					IncludeWinners:            nil,
					MediaTypePriceGranularity: nil,
					PreferDeals:               true,
					PriceGranularity:          nil,
				},
			},
			givenCacheInstructions: extCacheInstructions{
				cacheBids: true,
				cacheVAST: true,
			},
			givenAccount: config.Account{},
			givenWarning: nil,
			expectTargetData: &targetData{
				alwaysIncludeDeals:        true,
				includeBidderKeys:         false,
				includeCacheBids:          true,
				includeCacheVast:          true,
				includeFormat:             true,
				includeWinners:            false,
				mediaTypePriceGranularity: openrtb_ext.MediaTypePriceGranularity{},
				preferDeals:               true,
				priceGranularity:          openrtb_ext.PriceGranularity{},
				prefix:                    DefaultKeyPrefix,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, warnings := getExtTargetData(test.givenRequestExtPrebid, test.givenCacheInstructions, test.givenAccount)
			assert.Equal(t, test.givenWarning, warnings)
			assert.Equal(t, test.expectTargetData, result)
		})
	}
}

func TestParseRequestDebugValues(t *testing.T) {
	testCases := []struct {
		desc           string
		givenTest      int8
		givenExtPrebid *openrtb_ext.ExtRequestPrebid
		expected       bool
	}{
		{
			desc:           "bid request test == 0, nil requestExt",
			givenTest:      0,
			givenExtPrebid: nil,
			expected:       false,
		},
		{
			desc:           "bid request test == 0, requestExt debug flag false",
			givenTest:      0,
			givenExtPrebid: &openrtb_ext.ExtRequestPrebid{Debug: false},
			expected:       false,
		},
		{
			desc:           "bid request test == 1, requestExt debug flag false",
			givenTest:      1,
			givenExtPrebid: &openrtb_ext.ExtRequestPrebid{Debug: false},
			expected:       true,
		},
		{
			desc:           "bid request test == 0, requestExt debug flag true",
			givenTest:      0,
			givenExtPrebid: &openrtb_ext.ExtRequestPrebid{Debug: true},
			expected:       true,
		},
		{
			desc:           "bid request test == 1, requestExt debug flag true",
			givenTest:      1,
			givenExtPrebid: &openrtb_ext.ExtRequestPrebid{Debug: true},
			expected:       true,
		},
	}
	for _, test := range testCases {
		actualDebugInfo := parseRequestDebugValues(test.givenTest, test.givenExtPrebid)

		assert.Equal(t, test.expected, actualDebugInfo, "%s. Unexpected debug value. \n", test.desc)
	}
}

func TestSetDebugLogValues(t *testing.T) {
	type aTest struct {
		desc               string
		inAccountDebugFlag bool
		inDebugLog         *DebugLog
		expectedDebugLog   *DebugLog
	}

	testGroups := []struct {
		desc      string
		testCases []aTest
	}{
		{
			"nil debug log",
			[]aTest{
				{
					desc:               "accountDebugFlag false, expect all false flags in resulting debugLog",
					inAccountDebugFlag: false,
					inDebugLog:         nil,
					expectedDebugLog:   &DebugLog{},
				},
				{
					desc:               "accountDebugFlag true, expect debugLog.Enabled to be true",
					inAccountDebugFlag: true,
					inDebugLog:         nil,
					expectedDebugLog:   &DebugLog{Enabled: true},
				},
			},
		},
		{
			"non-nil debug log",
			[]aTest{
				{
					desc:               "both accountDebugFlag and DebugEnabledOrOverridden are false, expect debugLog.Enabled to be false",
					inAccountDebugFlag: false,
					inDebugLog:         &DebugLog{},
					expectedDebugLog:   &DebugLog{},
				},
				{
					desc:               "accountDebugFlag false but DebugEnabledOrOverridden is true, expect debugLog.Enabled to be true",
					inAccountDebugFlag: false,
					inDebugLog:         &DebugLog{DebugEnabledOrOverridden: true},
					expectedDebugLog:   &DebugLog{DebugEnabledOrOverridden: true, Enabled: true},
				},
				{
					desc:               "accountDebugFlag true but DebugEnabledOrOverridden is false, expect debugLog.Enabled to be true",
					inAccountDebugFlag: true,
					inDebugLog:         &DebugLog{},
					expectedDebugLog:   &DebugLog{Enabled: true},
				},
				{
					desc:               "Both accountDebugFlag and DebugEnabledOrOverridden are true, expect debugLog.Enabled to be true",
					inAccountDebugFlag: true,
					inDebugLog:         &DebugLog{DebugEnabledOrOverridden: true},
					expectedDebugLog:   &DebugLog{DebugEnabledOrOverridden: true, Enabled: true},
				},
			},
		},
	}

	for _, group := range testGroups {
		for _, tc := range group.testCases {
			// run
			actualDebugLog := setDebugLogValues(tc.inAccountDebugFlag, tc.inDebugLog)
			// assertions
			assert.Equal(t, tc.expectedDebugLog, actualDebugLog, "%s. %s", group.desc, tc.desc)
		}
	}
}

func TestGetExtBidAdjustmentFactors(t *testing.T) {
	testCases := []struct {
		desc                    string
		requestExtPrebid        *openrtb_ext.ExtRequestPrebid
		outBidAdjustmentFactors map[string]float64
	}{
		{
			desc:                    "Nil request ext",
			requestExtPrebid:        nil,
			outBidAdjustmentFactors: nil,
		},
		{
			desc:                    "Non-nil request ext, nil BidAdjustmentFactors field",
			requestExtPrebid:        &openrtb_ext.ExtRequestPrebid{BidAdjustmentFactors: nil},
			outBidAdjustmentFactors: nil,
		},
		{
			desc:                    "Non-nil request ext, valid BidAdjustmentFactors field",
			requestExtPrebid:        &openrtb_ext.ExtRequestPrebid{BidAdjustmentFactors: map[string]float64{"bid-factor": 1.0}},
			outBidAdjustmentFactors: map[string]float64{"bid-factor": 1.0},
		},
		{
			desc:                    "BidAdjustmentFactors contains uppercase bidders, expect case insensitve map returned",
			requestExtPrebid:        &openrtb_ext.ExtRequestPrebid{BidAdjustmentFactors: map[string]float64{"Bidder": 1.0, "APPNEXUS": 2.0}},
			outBidAdjustmentFactors: map[string]float64{"bidder": 1.0, "appnexus": 2.0},
		},
	}
	for _, test := range testCases {
		actualBidAdjustmentFactors := getExtBidAdjustmentFactors(test.requestExtPrebid)

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
		req := newBidRequest()
		req.Device.Lmt = test.lmt

		auctionReq := AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
			UserSyncs:         &emptyUsersync{},
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}

		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder

		privacyConfig := config.Privacy{
			LMT: config.LMT{
				Enforce: test.enforceLMT,
			},
		}

		reqSplitter := &requestSplitter{
			bidderToSyncerKey: map[string]string{},
			me:                &metrics.MetricsEngineMock{},
			privacyConfig:     privacyConfig,
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}

		results, privacyLabels, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, nil, gdpr.SignalNo, false, map[string]float64{})
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
	tcf2Consent := "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA"

	testCases := []struct {
		description         string
		gdprConsent         string
		gdprScrub           bool
		gdprSignal          gdpr.Signal
		gdprEnforced        bool
		permissionsError    error
		expectPrivacyLabels metrics.PrivacyLabels
		expectError         bool
	}{
		{
			description:  "enforce no scrub - TCF invalid",
			gdprConsent:  "malformed",
			gdprScrub:    false,
			gdprSignal:   gdpr.SignalYes,
			gdprEnforced: true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: "",
			},
		},
		{
			description:  "enforce and scrub",
			gdprConsent:  tcf2Consent,
			gdprScrub:    true,
			gdprSignal:   gdpr.SignalYes,
			gdprEnforced: true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: metrics.TCFVersionV2,
			},
		},
		{
			description:  "not enforce",
			gdprConsent:  tcf2Consent,
			gdprScrub:    false,
			gdprSignal:   gdpr.SignalYes,
			gdprEnforced: false,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   false,
				GDPRTCFVersion: "",
			},
		},
		{
			description:      "enforce - error while checking if personal info is allowed",
			gdprConsent:      tcf2Consent,
			gdprScrub:        true,
			permissionsError: errors.New("Some error"),
			gdprSignal:       gdpr.SignalYes,
			gdprEnforced:     true,
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   true,
				GDPRTCFVersion: metrics.TCFVersionV2,
			},
		},
	}

	for _, test := range testCases {
		req := newBidRequest()
		req.User.Consent = test.gdprConsent

		privacyConfig := config.Privacy{}
		accountConfig := config.Account{}

		auctionReq := AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
			UserSyncs:         &emptyUsersync{},
			Account:           accountConfig,
			TCF2Config: gdpr.NewTCF2Config(
				privacyConfig.GDPR.TCF2,
				accountConfig.GDPR,
			),
		}

		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
				passGeo:         !test.gdprScrub,
				passID:          !test.gdprScrub,
				activitiesError: test.permissionsError,
			},
		}.Builder

		metricsMock := metrics.MetricsEngineMock{}
		metricsMock.Mock.On("RecordAdapterBuyerUIDScrubbed", mock.Anything).Return()

		reqSplitter := &requestSplitter{
			bidderToSyncerKey: map[string]string{},
			me:                &metricsMock,
			privacyConfig:     privacyConfig,
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}

		results, privacyLabels, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, nil, test.gdprSignal, test.gdprEnforced, map[string]float64{})
		result := results[0]

		if test.expectError {
			assert.NotNil(t, errs)
		} else {
			assert.Nil(t, errs)
		}

		if test.gdprScrub {
			assert.Equal(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.Equal(t, result.BidRequest.Device.DIDMD5, "", test.description+":Device.DIDMD5")
			metricsMock.AssertCalled(t, "RecordAdapterBuyerUIDScrubbed", openrtb_ext.BidderAppnexus)
		} else {
			assert.NotEqual(t, result.BidRequest.User.BuyerUID, "", test.description+":User.BuyerUID")
			assert.NotEqual(t, result.BidRequest.Device.DIDMD5, "", test.description+":Device.DIDMD5")
			metricsMock.AssertNotCalled(t, "RecordAdapterBuyerUIDScrubbed", openrtb_ext.BidderAppnexus)
		}
		assert.Equal(t, test.expectPrivacyLabels, privacyLabels, test.description+":PrivacyLabels")
	}
}

func TestCleanOpenRTBRequestsGDPRBlockBidRequest(t *testing.T) {
	testCases := []struct {
		description            string
		gdprEnforced           bool
		gdprAllowedBidders     []openrtb_ext.BidderName
		expectedBidders        []openrtb_ext.BidderName
		expectedBlockedBidders []openrtb_ext.BidderName
	}{
		{
			description:            "gdpr enforced, one request allowed and one request blocked",
			gdprEnforced:           true,
			gdprAllowedBidders:     []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus},
			expectedBidders:        []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus},
			expectedBlockedBidders: []openrtb_ext.BidderName{openrtb_ext.BidderRubicon},
		},
		{
			description:            "gdpr enforced, two requests allowed and no requests blocked",
			gdprEnforced:           true,
			gdprAllowedBidders:     []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon},
			expectedBidders:        []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon},
			expectedBlockedBidders: []openrtb_ext.BidderName{},
		},
		{
			description:            "gdpr not enforced, two requests allowed and no requests blocked",
			gdprEnforced:           false,
			gdprAllowedBidders:     []openrtb_ext.BidderName{},
			expectedBidders:        []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon},
			expectedBlockedBidders: []openrtb_ext.BidderName{},
		},
	}

	for _, test := range testCases {
		req := newBidRequest()
		req.Regs = &openrtb2.Regs{
			Ext: json.RawMessage(`{"gdpr":1}`),
		}
		req.Imp[0].Ext = json.RawMessage(`{"prebid":{"bidder":{"appnexus": {"placementId": 1}, "rubicon": {}}}}`)

		privacyConfig := config.Privacy{}
		accountConfig := config.Account{
			GDPR: config.AccountGDPR{
				Enabled: nil,
			},
		}

		auctionReq := AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
			UserSyncs:         &emptyUsersync{},
			Account:           accountConfig,
			TCF2Config:        gdpr.NewTCF2Config(privacyConfig.GDPR.TCF2, accountConfig.GDPR),
		}

		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowedBidders:  test.gdprAllowedBidders,
				passGeo:         true,
				passID:          true,
				activitiesError: nil,
			},
		}.Builder

		metricsMock := metrics.MetricsEngineMock{}
		metricsMock.Mock.On("RecordAdapterGDPRRequestBlocked", mock.Anything).Return()

		reqSplitter := &requestSplitter{
			bidderToSyncerKey: map[string]string{},
			me:                &metricsMock,
			privacyConfig:     privacyConfig,
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}

		results, _, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, nil, gdpr.SignalYes, test.gdprEnforced, map[string]float64{})

		// extract bidder name from each request in the results
		bidders := []openrtb_ext.BidderName{}
		for _, req := range results {
			bidders = append(bidders, req.BidderName)
		}

		assert.Empty(t, errs, test.description)
		assert.ElementsMatch(t, bidders, test.expectedBidders, test.description)

		for _, blockedBidder := range test.expectedBlockedBidders {
			metricsMock.AssertCalled(t, "RecordAdapterGDPRRequestBlocked", blockedBidder)
		}
		for _, allowedBidder := range test.expectedBidders {
			metricsMock.AssertNotCalled(t, "RecordAdapterGDPRRequestBlocked", allowedBidder)
		}
	}
}

func TestCleanOpenRTBRequestsWithOpenRTBDowngrade(t *testing.T) {
	emptyTCF2Config := gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{})

	bidReq := newBidRequest()
	bidReq.Regs = &openrtb2.Regs{}
	bidReq.Regs.GPP = "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1NYN"
	bidReq.Regs.GPPSID = []int8{6}
	bidReq.User.ID = ""
	bidReq.User.BuyerUID = ""
	bidReq.User.Yob = 0
	bidReq.User.Gender = ""
	bidReq.User.Geo = &openrtb2.Geo{Lat: ptrutil.ToPtr(123.46)}

	downgradedRegs := *bidReq.Regs
	downgradedUser := *bidReq.User
	downgradedRegs.GDPR = ptrutil.ToPtr[int8](0)
	downgradedRegs.USPrivacy = "1NYN"
	downgradedUser.Consent = "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA"

	testCases := []struct {
		name        string
		req         AuctionRequest
		expectRegs  *openrtb2.Regs
		expectUser  *openrtb2.User
		bidderInfos config.BidderInfos
	}{
		{
			name:        "NotSupported",
			req:         AuctionRequest{BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: bidReq}, UserSyncs: &emptyUsersync{}, TCF2Config: emptyTCF2Config},
			expectRegs:  &downgradedRegs,
			expectUser:  &downgradedUser,
			bidderInfos: config.BidderInfos{"appnexus": config.BidderInfo{OpenRTB: &config.OpenRTBInfo{GPPSupported: false, Version: "2.6"}}},
		},
		{
			name:        "Supported",
			req:         AuctionRequest{BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: bidReq}, UserSyncs: &emptyUsersync{}, TCF2Config: emptyTCF2Config},
			expectRegs:  bidReq.Regs,
			expectUser:  bidReq.User,
			bidderInfos: config.BidderInfos{"appnexus": config.BidderInfo{OpenRTB: &config.OpenRTBInfo{GPPSupported: true, Version: "2.6"}}},
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
		t.Run(test.name, func(t *testing.T) {
			gdprPermsBuilder := fakePermissionsBuilder{
				permissions: &permissionsMock{
					allowAllBidders: true,
				},
			}.Builder

			reqSplitter := &requestSplitter{
				bidderToSyncerKey: map[string]string{},
				me:                &metrics.MetricsEngineMock{},
				privacyConfig:     privacyConfig,
				gdprPermsBuilder:  gdprPermsBuilder,
				hostSChainNode:    nil,
				bidderInfo:        test.bidderInfos,
			}
			bidderRequests, _, err := reqSplitter.cleanOpenRTBRequests(context.Background(), test.req, nil, gdpr.SignalNo, false, map[string]float64{})
			assert.Nil(t, err, "Err should be nil")
			bidRequest := bidderRequests[0]
			assert.Equal(t, test.expectRegs, bidRequest.BidRequest.Regs)
			assert.Equal(t, test.expectUser, bidRequest.BidRequest.User)
		})
	}
}

func TestBuildRequestExtForBidder(t *testing.T) {
	var (
		bidder       = "foo"
		bidderParams = json.RawMessage(`"bar"`)
	)

	testCases := []struct {
		name                 string
		requestExt           json.RawMessage
		bidderParams         map[string]json.RawMessage
		alternateBidderCodes *openrtb_ext.ExtAlternateBidderCodes
		expectedJson         json.RawMessage
	}{
		{
			name:                 "Nil",
			bidderParams:         nil,
			requestExt:           nil,
			alternateBidderCodes: nil,
			expectedJson:         nil,
		},
		{
			name:                 "Empty",
			bidderParams:         nil,
			alternateBidderCodes: nil,
			requestExt:           json.RawMessage(`{}`),
			expectedJson:         nil,
		},
		{
			name:         "Prebid - Allowed Fields Only",
			bidderParams: nil,
			requestExt:   json.RawMessage(`{"prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true}, "server": {"externalurl": "url", "gvlid": 1, "datacenter": "2"}, "sdk": {"renderers": [{"name": "r1"}]}}}`),
			expectedJson: json.RawMessage(`{"prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true}, "server": {"externalurl": "url", "gvlid": 1, "datacenter": "2"}, "sdk": {"renderers": [{"name": "r1"}]}}}`),
		},
		{
			name:         "Prebid - Allowed Fields + Bidder Params",
			bidderParams: map[string]json.RawMessage{bidder: bidderParams},
			requestExt:   json.RawMessage(`{"prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true}, "server": {"externalurl": "url", "gvlid": 1, "datacenter": "2"}, "sdk": {"renderers": [{"name": "r1"}]}}}`),
			expectedJson: json.RawMessage(`{"prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true}, "server": {"externalurl": "url", "gvlid": 1, "datacenter": "2"}, "sdk": {"renderers": [{"name": "r1"}]}, "bidderparams":"bar"}}`),
		},
		{
			name:         "Other",
			bidderParams: nil,
			requestExt:   json.RawMessage(`{"other":"foo"}`),
			expectedJson: json.RawMessage(`{"other":"foo"}`),
		},
		{
			name:         "Prebid + Other + Bider Params",
			bidderParams: map[string]json.RawMessage{bidder: bidderParams},
			requestExt:   json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true}, "server": {"externalurl": "url", "gvlid": 1, "datacenter": "2"}, "sdk": {"renderers": [{"name": "r1"}]}}}`),
			expectedJson: json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true}, "server": {"externalurl": "url", "gvlid": 1, "datacenter": "2"}, "sdk": {"renderers": [{"name": "r1"}]}, "bidderparams":"bar"}}`),
		},
		{
			name:                 "Prebid + AlternateBidderCodes in pbs config but current bidder not in AlternateBidderCodes config",
			bidderParams:         map[string]json.RawMessage{bidder: bidderParams},
			alternateBidderCodes: &openrtb_ext.ExtAlternateBidderCodes{Enabled: true, Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{"bar": {Enabled: true, AllowedBidderCodes: []string{"*"}}}},
			requestExt:           json.RawMessage(`{"other":"foo"}`),
			expectedJson:         json.RawMessage(`{"other":"foo","prebid":{"alternatebiddercodes":{"enabled":true,"bidders":null},"bidderparams":"bar"}}`),
		},
		{
			name:                 "Prebid + AlternateBidderCodes in request",
			bidderParams:         map[string]json.RawMessage{bidder: bidderParams},
			alternateBidderCodes: &openrtb_ext.ExtAlternateBidderCodes{},
			requestExt:           json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["foo2"]},"bar":{"enabled":true,"allowedbiddercodes":["ix"]}}}}}`),
			expectedJson:         json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["foo2"]}}},"bidderparams":"bar"}}`),
		},
		{
			name:                 "Prebid + AlternateBidderCodes in request but current bidder not in AlternateBidderCodes config",
			bidderParams:         map[string]json.RawMessage{bidder: bidderParams},
			alternateBidderCodes: &openrtb_ext.ExtAlternateBidderCodes{},
			requestExt:           json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"bar":{"enabled":true,"allowedbiddercodes":["ix"]}}}}}`),
			expectedJson:         json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":null},"bidderparams":"bar"}}`),
		},
		{
			name:                 "Prebid + AlternateBidderCodes in both pbs config and in the request",
			bidderParams:         map[string]json.RawMessage{bidder: bidderParams},
			alternateBidderCodes: &openrtb_ext.ExtAlternateBidderCodes{Enabled: true, Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{"foo": {Enabled: true, AllowedBidderCodes: []string{"*"}}}},
			requestExt:           json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["foo2"]},"bar":{"enabled":true,"allowedbiddercodes":["ix"]}}}}}`),
			expectedJson:         json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["foo2"]}}},"bidderparams":"bar"}}`),
		},
		{
			name:         "Prebid + Other + Bider Params + MultiBid.Bidder",
			bidderParams: map[string]json.RawMessage{bidder: bidderParams},
			requestExt:   json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"multibid":[{"bidder":"foo","maxbids":2,"targetbiddercodeprefix":"fmb"},{"bidders":["appnexus","groupm"],"maxbids":2}]}}`),
			expectedJson: json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"multibid":[{"bidder":"foo","maxbids":2,"targetbiddercodeprefix":"fmb"}],"bidderparams":"bar"}}`),
		},
		{
			name:         "Prebid + Other + Bider Params + MultiBid.Bidders",
			bidderParams: map[string]json.RawMessage{bidder: bidderParams},
			requestExt:   json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"multibid":[{"bidder":"pubmatic","maxbids":3,"targetbiddercodeprefix":"pubM"},{"bidders":["foo","groupm"],"maxbids":4}]}}`),
			expectedJson: json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"multibid":[{"bidders":["foo"],"maxbids":4}],"bidderparams":"bar"}}`),
		},
		{
			name:         "Prebid + Other + Bider Params + MultiBid (foo not in MultiBid)",
			bidderParams: map[string]json.RawMessage{bidder: bidderParams},
			requestExt:   json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"multibid":[{"bidder":"foo2","maxbids":2,"targetbiddercodeprefix":"fmb"},{"bidders":["appnexus","groupm"],"maxbids":2}]}}`),
			expectedJson: json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"bidderparams":"bar"}}`),
		},
		{
			name:         "Prebid + Other + Bider Params + MultiBid (foo not in MultiBid)",
			bidderParams: map[string]json.RawMessage{bidder: bidderParams},
			requestExt:   json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"multibid":[{"bidder":"foo2","maxbids":2,"targetbiddercodeprefix":"fmb"},{"bidders":["appnexus","groupm"],"maxbids":2}]}}`),
			expectedJson: json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"bidderparams":"bar"}}`),
		},
		{
			name:         "Prebid + AlternateBidderCodes.MultiBid.Bidder",
			requestExt:   json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["pubmatic"]}}},"multibid":[{"bidder":"foo","maxbids":3,"targetbiddercodeprefix":"fmb"},{"bidder":"foo2","maxbids":4,"targetbiddercodeprefix":"fmb2"},{"bidder":"pubmatic","maxbids":5,"targetbiddercodeprefix":"pm"}]}}`),
			expectedJson: json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["pubmatic"]}}},"multibid":[{"bidder":"foo","maxbids":3,"targetbiddercodeprefix":"fmb"},{"bidder":"pubmatic","maxbids":5,"targetbiddercodeprefix":"pm"}]}}`),
		},
		{
			name:         "Prebid + AlternateBidderCodes.MultiBid.Bidders",
			requestExt:   json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["pubmatic"]}}},"multibid":[{"bidder":"foo","maxbids":3,"targetbiddercodeprefix":"fmb"},{"bidders":["pubmatic","groupm"],"maxbids":4}]}}`),
			expectedJson: json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["pubmatic"]}}},"multibid":[{"bidder":"foo","maxbids":3,"targetbiddercodeprefix":"fmb"},{"bidders":["pubmatic"],"maxbids":4}]}}`),
		},
		{
			name:         "Prebid + AlternateBidderCodes.MultiBid.Bidder with *",
			requestExt:   json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["*"]}}},"multibid":[{"bidder":"foo","maxbids":3,"targetbiddercodeprefix":"fmb"},{"bidder":"foo2","maxbids":4,"targetbiddercodeprefix":"fmb2"},{"bidder":"pubmatic","maxbids":5,"targetbiddercodeprefix":"pm"}]}}`),
			expectedJson: json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["*"]}}},"multibid":[{"bidder":"foo","maxbids":3,"targetbiddercodeprefix":"fmb"},{"bidder":"foo2","maxbids":4,"targetbiddercodeprefix":"fmb2"},{"bidder":"pubmatic","maxbids":5,"targetbiddercodeprefix":"pm"}]}}`),
		},
		{
			name:         "Prebid + AlternateBidderCodes.MultiBid.Bidders with *",
			requestExt:   json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["*"]}}},"multibid":[{"bidder":"foo","maxbids":3,"targetbiddercodeprefix":"fmb"},{"bidders":["pubmatic","groupm"],"maxbids":4}]}}`),
			expectedJson: json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["*"]}}},"multibid":[{"bidder":"foo","maxbids":3,"targetbiddercodeprefix":"fmb"},{"bidders":["pubmatic"],"maxbids":4},{"bidders":["groupm"],"maxbids":4}]}}`),
		},
		{
			name:         "Prebid + AlternateBidderCodes + MultiBid",
			requestExt:   json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["foo2"]}}},"multibid":[{"bidder":"foo3","maxbids":3,"targetbiddercodeprefix":"fmb"},{"bidders":["pubmatic","groupm"],"maxbids":4}]}}`),
			expectedJson: json.RawMessage(`{"other":"foo","prebid":{"integration":"a","channel":{"name":"b","version":"c"},"debug":true,"currency":{"rates":{"FOO":{"BAR":42}},"usepbsrates":true},"alternatebiddercodes":{"enabled":true,"bidders":{"foo":{"enabled":true,"allowedbiddercodes":["foo2"]}}}}}`),
		},
		{
			name:         "targeting",
			requestExt:   json.RawMessage(`{"prebid":{"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":20,"increment":0.1}]},"mediatypepricegranularity":{},"includebidderkeys":true,"includewinners":true,"includebrandcategory":{"primaryadserver":1,"publisher":"anyPublisher","withcategory":true}}}}`),
			expectedJson: json.RawMessage(`{"prebid":{"targeting":{"includebrandcategory":{"primaryadserver":1,"publisher":"anyPublisher","withcategory":true}}}}`),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			req := openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: test.requestExt,
				},
			}
			err := buildRequestExtForBidder(bidder, &req, test.bidderParams, test.alternateBidderCodes)
			assert.NoError(t, req.RebuildRequest())
			assert.NoError(t, err)

			if len(test.expectedJson) > 0 {
				assert.JSONEq(t, string(test.expectedJson), string(req.Ext))
			} else {
				assert.Equal(t, test.expectedJson, req.Ext)
			}
		})
	}
}

func TestBuildRequestExtForBidder_RequestExtParsedNil(t *testing.T) {
	var (
		bidder               = "foo"
		requestExt           = json.RawMessage(`{}`)
		bidderParams         map[string]json.RawMessage
		alternateBidderCodes *openrtb_ext.ExtAlternateBidderCodes
	)

	req := openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Ext: requestExt,
		},
	}
	err := buildRequestExtForBidder(bidder, &req, bidderParams, alternateBidderCodes)
	assert.NoError(t, req.RebuildRequest())
	assert.Nil(t, req.Ext)
	assert.NoError(t, err)
}

func TestBuildRequestExtForBidder_RequestExtMalformed(t *testing.T) {
	var (
		bidder               = "foo"
		requestExt           = json.RawMessage(`malformed`)
		bidderParams         map[string]json.RawMessage
		alternateBidderCodes *openrtb_ext.ExtAlternateBidderCodes
	)

	req := openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Ext: requestExt,
		},
	}
	err := buildRequestExtForBidder(bidder, &req, bidderParams, alternateBidderCodes)
	assert.NoError(t, req.RebuildRequest())
	assert.EqualError(t, err, "expect { or n, but found m")
}

func TestBuildRequestExtTargeting(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := buildRequestExtTargeting(nil)
		assert.Nil(t, result)
	})

	t.Run("brandcategory-nil", func(t *testing.T) {
		given := &openrtb_ext.ExtRequestTargeting{}

		result := buildRequestExtTargeting(given)
		assert.Nil(t, result)
	})

	t.Run("brandcategory-populated", func(t *testing.T) {
		brandCatgory := &openrtb_ext.ExtIncludeBrandCategory{
			PrimaryAdServer:     1,
			Publisher:           "anyPublisher",
			WithCategory:        true,
			TranslateCategories: ptrutil.ToPtr(true),
		}

		given := &openrtb_ext.ExtRequestTargeting{
			PriceGranularity:     &openrtb_ext.PriceGranularity{},
			IncludeBrandCategory: brandCatgory,
			IncludeWinners:       ptrutil.ToPtr(true),
		}

		expected := &openrtb_ext.ExtRequestTargeting{
			PriceGranularity:     nil,
			IncludeBrandCategory: brandCatgory,
			IncludeWinners:       nil,
		}

		result := buildRequestExtTargeting(given)
		assert.Equal(t, expected, result)
	})
}

// newAdapterAliasBidRequest builds a BidRequest with aliases
func newAdapterAliasBidRequest() *openrtb2.BidRequest {
	dnt := int8(1)
	return &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Page:   "www.some.domain.com",
			Domain: "domain.com",
			Publisher: &openrtb2.Publisher{
				ID: "some-publisher-id",
			},
		},
		Device: &openrtb2.Device{
			DIDMD5:   "some device ID hash",
			UA:       deviceUA,
			IFA:      "ifa",
			IP:       "132.173.230.74",
			DNT:      &dnt,
			Language: "EN",
		},
		Source: &openrtb2.Source{
			TID: "testTID",
		},
		User: &openrtb2.User{
			ID:       "our-id",
			BuyerUID: "their-id",
			Ext:      json.RawMessage(`{"consent":"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw"}`),
		},
		Regs: &openrtb2.Regs{
			Ext: json.RawMessage(`{"gdpr":1}`),
		},
		Imp: []openrtb2.Imp{{
			ID: "some-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: json.RawMessage(`{"appnexus": {"placementId": 1},"somealias": {"placementId": 105}}`),
		}},
		Ext: json.RawMessage(`{"prebid":{"aliases":{"somealias":"appnexus"}}}`),
	}
}

func newBidRequest() *openrtb2.BidRequest {
	return &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Page:   "www.some.domain.com",
			Domain: "domain.com",
			Publisher: &openrtb2.Publisher{
				ID: "some-publisher-id",
			},
		},
		Device: &openrtb2.Device{
			UA:       deviceUA,
			IP:       "132.173.230.74",
			Language: "EN",
			DIDMD5:   "DIDMD5",
			IFA:      "IFA",
			DIDSHA1:  "DIDSHA1",
			DPIDMD5:  "DPIDMD5",
			DPIDSHA1: "DPIDSHA1",
			MACMD5:   "MACMD5",
			MACSHA1:  "MACSHA1",
			Geo:      &openrtb2.Geo{Lat: ptrutil.ToPtr(123.456), Lon: ptrutil.ToPtr(11.278)},
		},
		Source: &openrtb2.Source{
			TID: "testTID",
		},
		User: &openrtb2.User{
			ID:       "our-id",
			BuyerUID: "their-id",
			Yob:      1982,
			Gender:   "test",
			Ext:      json.RawMessage(`{"data": 1, "test": 2}`),
			Geo:      &openrtb2.Geo{Lat: ptrutil.ToPtr(123.456), Lon: ptrutil.ToPtr(11.278)},
			EIDs: []openrtb2.EID{
				{Source: "eids-source"},
			},
			Data: []openrtb2.Data{{ID: "data-id"}},
		},
		Imp: []openrtb2.Imp{{
			BidFloor: 100,
			ID:       "some-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: json.RawMessage(`{"prebid":{"tid":"1234567", "bidder":{"appnexus": {"placementId": 1}}}}`),
		}},
	}
}

func newBidRequestWithBidderParams() *openrtb2.BidRequest {
	return &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Page:   "www.some.domain.com",
			Domain: "domain.com",
			Publisher: &openrtb2.Publisher{
				ID: "some-publisher-id",
			},
		},
		Device: &openrtb2.Device{
			DIDMD5:   "some device ID hash",
			UA:       deviceUA,
			IFA:      "ifa",
			IP:       "132.173.230.74",
			Language: "EN",
		},
		Source: &openrtb2.Source{
			TID: "testTID",
		},
		User: &openrtb2.User{
			ID:       "our-id",
			BuyerUID: "their-id",
			Yob:      1982,
			Ext:      json.RawMessage(`{}`),
		},
		Imp: []openrtb2.Imp{{
			ID: "some-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: json.RawMessage(`{"prebid":{"bidder":{"appnexus": {"placementId": 1}, "pubmatic":{"publisherId": "1234"}}}}`),
		}},
	}
}

func TestRandomizeList(t *testing.T) {
	var (
		bidder1 = openrtb_ext.BidderName("bidder1")
		bidder2 = openrtb_ext.BidderName("bidder2")
		bidder3 = openrtb_ext.BidderName("bidder3")
	)

	testCases := []struct {
		description string
		bidders     []openrtb_ext.BidderName
	}{
		{
			description: "None",
			bidders:     []openrtb_ext.BidderName{},
		},
		{
			description: "One",
			bidders:     []openrtb_ext.BidderName{bidder1},
		},
		{
			description: "Many",
			bidders:     []openrtb_ext.BidderName{bidder1, bidder2, bidder3},
		},
	}

	for _, test := range testCases {
		biddersWorkingCopy := make([]openrtb_ext.BidderName, len(test.bidders))
		copy(biddersWorkingCopy, test.bidders)

		randomizeList(biddersWorkingCopy)

		// test all bidders are still present, ignoring order. we are testing the algorithm doesn't loose
		// elements. we are not testing the random number generator itself.
		assert.ElementsMatch(t, test.bidders, biddersWorkingCopy)
	}
}

func TestRemoveUnpermissionedEids(t *testing.T) {
	bidder := "bidderA"

	testCases := []struct {
		description      string
		userEids         []openrtb2.EID
		eidPermissions   []openrtb_ext.ExtRequestPrebidDataEidPermission
		expectedUserEids []openrtb2.EID
	}{
		{
			description: "Eids Empty",
			userEids:    []openrtb2.EID{},
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"bidderA"}},
			},
			expectedUserEids: []openrtb2.EID{},
		},
		{
			description:      "Allowed By Nil Permissions",
			userEids:         []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
			eidPermissions:   nil,
			expectedUserEids: []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
		},
		{
			description:      "Allowed By Empty Permissions",
			userEids:         []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
			eidPermissions:   []openrtb_ext.ExtRequestPrebidDataEidPermission{},
			expectedUserEids: []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
		},
		{
			description: "Allowed By Specific Bidder",
			userEids:    []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"bidderA"}},
			},
			expectedUserEids: []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
		},
		{
			description: "Allowed By Specific Bidder - Case Insensitive",
			userEids:    []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"BIDDERA"}},
			},
			expectedUserEids: []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
		},
		{
			description: "Allowed By All Bidders",
			userEids:    []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"*"}},
			},
			expectedUserEids: []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
		},
		{
			description: "Allowed By Lack Of Matching Source",
			userEids:    []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source2", Bidders: []string{"otherBidder"}},
			},
			expectedUserEids: []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
		},
		{
			description: "Denied",
			userEids:    []openrtb2.EID{{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID"}}}},
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"otherBidder"}},
			},
			expectedUserEids: nil,
		},
		{
			description: "Mix Of Allowed By Specific Bidder, Allowed By Lack Of Matching Source, Denied",
			userEids: []openrtb2.EID{
				{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID1"}}},
				{Source: "source2", UIDs: []openrtb2.UID{{ID: "anyID2"}}},
				{Source: "source3", UIDs: []openrtb2.UID{{ID: "anyID3"}}},
			},
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"bidderA"}},
				{Source: "source3", Bidders: []string{"otherBidder"}},
			},
			expectedUserEids: []openrtb2.EID{
				{Source: "source1", UIDs: []openrtb2.UID{{ID: "anyID1"}}},
				{Source: "source2", UIDs: []openrtb2.UID{{ID: "anyID2"}}},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			request := &openrtb2.BidRequest{
				User: &openrtb2.User{EIDs: test.userEids},
			}

			reqWrapper := openrtb_ext.RequestWrapper{BidRequest: request}
			re, _ := reqWrapper.GetRequestExt()
			re.SetPrebid(&openrtb_ext.ExtRequestPrebid{
				Data: &openrtb_ext.ExtRequestPrebidData{
					EidPermissions: test.eidPermissions,
				},
			})

			expectedRequest := &openrtb2.BidRequest{
				User: &openrtb2.User{EIDs: test.expectedUserEids},
			}

			resultErr := removeUnpermissionedEids(&reqWrapper, bidder)
			assert.NoError(t, resultErr, test.description)
			assert.Equal(t, expectedRequest, reqWrapper.BidRequest)
		})
	}
}

func TestGetDebugInfo(t *testing.T) {
	type testInput struct {
		debugEnabledOrOverridden bool
		accountDebugFlag         bool
	}
	type testOut struct {
		responseDebugAllow bool
		accountDebugAllow  bool
		debugLog           *DebugLog
	}
	type testCase struct {
		in       testInput
		expected testOut
	}

	testGroups := []struct {
		description   string
		isTestRequest int8
		testCases     []testCase
	}{
		{
			description:   "Bid request doesn't call for debug info",
			isTestRequest: 0,
			testCases: []testCase{
				{
					testInput{debugEnabledOrOverridden: false, accountDebugFlag: false},
					testOut{
						responseDebugAllow: false,
						accountDebugAllow:  false,
						debugLog:           &DebugLog{Enabled: false},
					},
				},
				{
					testInput{debugEnabledOrOverridden: false, accountDebugFlag: true},
					testOut{
						responseDebugAllow: false,
						accountDebugAllow:  false,
						debugLog:           &DebugLog{Enabled: true},
					},
				},
				{
					testInput{debugEnabledOrOverridden: true, accountDebugFlag: false},
					testOut{
						responseDebugAllow: true,
						accountDebugAllow:  false,
						debugLog:           &DebugLog{DebugEnabledOrOverridden: true, Enabled: true},
					},
				},
				{
					testInput{debugEnabledOrOverridden: true, accountDebugFlag: true},
					testOut{
						responseDebugAllow: true,
						accountDebugAllow:  true,
						debugLog:           &DebugLog{DebugEnabledOrOverridden: true, Enabled: true},
					},
				},
			},
		},
		{
			description:   "Bid request requires debug info",
			isTestRequest: 1,
			testCases: []testCase{
				{
					testInput{debugEnabledOrOverridden: false, accountDebugFlag: false},
					testOut{
						responseDebugAllow: false,
						accountDebugAllow:  false,
						debugLog:           &DebugLog{Enabled: false},
					},
				},
				{
					testInput{debugEnabledOrOverridden: false, accountDebugFlag: true},
					testOut{
						responseDebugAllow: true,
						accountDebugAllow:  true,
						debugLog:           &DebugLog{Enabled: true},
					},
				},
				{
					testInput{debugEnabledOrOverridden: true, accountDebugFlag: false},
					testOut{
						responseDebugAllow: true,
						accountDebugAllow:  false,
						debugLog:           &DebugLog{DebugEnabledOrOverridden: true, Enabled: true},
					},
				},
				{
					testInput{debugEnabledOrOverridden: true, accountDebugFlag: true},
					testOut{
						responseDebugAllow: true,
						accountDebugAllow:  true,
						debugLog:           &DebugLog{DebugEnabledOrOverridden: true, Enabled: true},
					},
				},
			},
		},
	}
	for _, group := range testGroups {
		for i, tc := range group.testCases {
			inDebugLog := &DebugLog{DebugEnabledOrOverridden: tc.in.debugEnabledOrOverridden}

			// run
			responseDebugAllow, accountDebugAllow, debugLog := getDebugInfo(group.isTestRequest, nil, tc.in.accountDebugFlag, inDebugLog)

			// assertions
			assert.Equal(t, tc.expected.responseDebugAllow, responseDebugAllow, "%s - %d", group.description, i)
			assert.Equal(t, tc.expected.accountDebugAllow, accountDebugAllow, "%s - %d", group.description, i)
			assert.Equal(t, tc.expected.debugLog, debugLog, "%s - %d", group.description, i)
		}
	}
}

func TestRemoveUnpermissionedEidsEmptyValidations(t *testing.T) {
	testCases := []struct {
		description    string
		request        *openrtb2.BidRequest
		eidPermissions []openrtb_ext.ExtRequestPrebidDataEidPermission
	}{
		{
			description: "Nil User",
			request: &openrtb2.BidRequest{
				User: nil,
			},
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"*"}},
			},
		},
		{
			description: "Empty User",
			request: &openrtb2.BidRequest{
				User: &openrtb2.User{},
			},
			eidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "source1", Bidders: []string{"*"}},
			},
		},
		{
			description: "Nil Ext",
			request: &openrtb2.BidRequest{
				User: &openrtb2.User{Ext: json.RawMessage(`{"eids":[{"source":"source1","id":"anyID"}]}`)},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			requestExpected := *test.request
			reqWrapper := openrtb_ext.RequestWrapper{BidRequest: test.request}

			re, _ := reqWrapper.GetRequestExt()
			re.SetPrebid(&openrtb_ext.ExtRequestPrebid{
				Data: &openrtb_ext.ExtRequestPrebidData{
					EidPermissions: test.eidPermissions,
				},
			})

			resultErr := removeUnpermissionedEids(&reqWrapper, "bidderA")
			assert.NoError(t, resultErr, test.description+":err")
			assert.Equal(t, &requestExpected, reqWrapper.BidRequest, test.description+":request")
		})
	}
}

func TestCleanOpenRTBRequestsSChainMultipleBidders(t *testing.T) {
	req := &openrtb2.BidRequest{
		Site: &openrtb2.Site{},
		Source: &openrtb2.Source{
			TID: "testTID",
		},
		Imp: []openrtb2.Imp{{
			Ext: json.RawMessage(`{"prebid":{"bidder":{"appnexus": {"placementId": 1}, "axonix": { "supplyId": "123"}}}}`),
		}},
		Ext: json.RawMessage(`{"prebid":{"schains":[{ "bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller1.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}, {"bidders":["axonix"],"schain":{"complete":1,"nodes":[{"asi":"directseller2.com","sid":"00002","rid":"BidRequest2","hp":1}],"ver":"1.0"}}]}}`),
	}

	extRequest := &openrtb_ext.ExtRequest{}
	err := jsonutil.UnmarshalValid(req.Ext, extRequest)
	assert.NoErrorf(t, err, "Error unmarshaling inExt")

	auctionReq := AuctionRequest{
		BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
		UserSyncs:         &emptyUsersync{},
		TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
	}

	gdprPermissionsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
			passGeo:         true,
			passID:          true,
			activitiesError: nil,
		},
	}.Builder

	ortb26enabled := config.BidderInfo{OpenRTB: &config.OpenRTBInfo{Version: "2.6"}}
	reqSplitter := &requestSplitter{
		bidderToSyncerKey: map[string]string{},
		me:                &metrics.MetricsEngineMock{},
		privacyConfig:     config.Privacy{},
		gdprPermsBuilder:  gdprPermissionsBuilder,
		hostSChainNode:    nil,
		bidderInfo:        config.BidderInfos{"appnexus": ortb26enabled, "axonix": ortb26enabled},
	}
	bidderRequests, _, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, extRequest, gdpr.SignalNo, false, map[string]float64{})

	assert.Nil(t, errs)
	assert.Len(t, bidderRequests, 2, "Bid request count is not 2")

	bidRequestSourceSupplyChain := map[openrtb_ext.BidderName]*openrtb2.SupplyChain{}
	for _, bidderRequest := range bidderRequests {
		bidRequestSourceSupplyChain[bidderRequest.BidderName] = bidderRequest.BidRequest.Source.SChain
	}

	appnexusSchainsSchainExpected := &openrtb2.SupplyChain{
		Complete: 1,
		Ver:      "1.0",
		Ext:      nil,
		Nodes: []openrtb2.SupplyChainNode{
			{
				ASI: "directseller1.com",
				SID: "00001",
				RID: "BidRequest1",
				HP:  openrtb2.Int8Ptr(1),
				Ext: nil,
			},
		},
	}

	axonixSchainsSchainExpected := &openrtb2.SupplyChain{
		Complete: 1,
		Ver:      "1.0",
		Ext:      nil,
		Nodes: []openrtb2.SupplyChainNode{
			{
				ASI: "directseller2.com",
				SID: "00002",
				RID: "BidRequest2",
				HP:  openrtb2.Int8Ptr(1),
				Ext: nil,
			},
		},
	}

	assert.Equal(t, appnexusSchainsSchainExpected, bidRequestSourceSupplyChain["appnexus"], "Incorrect appnexus bid request schain ")
	assert.Equal(t, axonixSchainsSchainExpected, bidRequestSourceSupplyChain["axonix"], "Incorrect axonix bid request schain")
}

func TestCleanOpenRTBRequestsBidAdjustment(t *testing.T) {
	tcf2Consent := "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA"
	falseValue := false
	testCases := []struct {
		description         string
		gdprAccountEnabled  *bool
		gdprHostEnabled     bool
		gdpr                string
		gdprConsent         string
		gdprScrub           bool
		permissionsError    error
		gdprDefaultValue    string
		expectPrivacyLabels metrics.PrivacyLabels
		expectError         bool
		bidAdjustmentFactor map[string]float64
		expectedImp         []openrtb2.Imp
	}{
		{
			description:        "BidFloor Adjustment Done for Appnexus",
			gdprAccountEnabled: &falseValue,
			gdprHostEnabled:    true,
			gdpr:               "1",
			gdprConsent:        tcf2Consent,
			gdprScrub:          false,
			gdprDefaultValue:   "1",
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   false,
				GDPRTCFVersion: "",
			},
			bidAdjustmentFactor: map[string]float64{"appnexus": 0.50},
			expectedImp: []openrtb2.Imp{{
				BidFloor: 200,
				ID:       "some-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{{
						W: 300,
						H: 250,
					}, {
						W: 300,
						H: 600,
					}},
				},
				Ext: json.RawMessage(`{"bidder":{"placementId": 1}}`),
			}},
		},
		{
			description:        "bidAdjustment Not provided",
			gdprAccountEnabled: &falseValue,
			gdprHostEnabled:    true,
			gdpr:               "1",
			gdprConsent:        tcf2Consent,
			gdprScrub:          false,
			gdprDefaultValue:   "1",
			expectPrivacyLabels: metrics.PrivacyLabels{
				GDPREnforced:   false,
				GDPRTCFVersion: "",
			},
			bidAdjustmentFactor: map[string]float64{},
			expectedImp: []openrtb2.Imp{{
				BidFloor: 100,
				ID:       "some-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{{
						W: 300,
						H: 250,
					}, {
						W: 300,
						H: 600,
					}},
				},
				Ext: json.RawMessage(`{"bidder":{"placementId": 1}}`),
			}},
		},
	}
	for _, test := range testCases {
		req := newBidRequest()
		accountConfig := config.Account{
			GDPR: config.AccountGDPR{
				Enabled: &falseValue,
			},
			PriceFloors: config.AccountPriceFloors{
				AdjustForBidAdjustment: true,
			},
		}
		auctionReq := AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
			UserSyncs:         &emptyUsersync{},
			Account:           accountConfig,
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}
		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
				passGeo:         !test.gdprScrub,
				passID:          !test.gdprScrub,
				activitiesError: test.permissionsError,
			},
		}.Builder
		reqSplitter := &requestSplitter{
			bidderToSyncerKey: map[string]string{},
			me:                &metrics.MetricsEngineMock{},
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}
		results, _, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, nil, gdpr.SignalNo, false, test.bidAdjustmentFactor)
		result := results[0]
		assert.Nil(t, errs)
		assert.Equal(t, test.expectedImp, result.BidRequest.Imp, test.description)
	}
}

func TestCleanOpenRTBRequestsBuyerUID(t *testing.T) {
	tcf2Consent := "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA"

	buyerUIDAppnexus := `{"appnexus": "a"}`
	buyerUIDAppnexusMixedCase := `{"aPpNeXuS": "a"}`
	buyerUIDBoth := `{"appnexus": "a", "pubmatic": "b"}`

	bidderParamsAppnexus := `{"appnexus": {"placementId": 1}}`
	bidderParamsBoth := `{"appnexus": {"placementId": 1}, "pubmatic": {"publisherId": "abc"}}`

	tests := []struct {
		name          string
		bidderParams  string
		user          openrtb2.User
		expectedUsers map[string]openrtb2.User
	}{
		{
			name:         "one-bidder-with-prebid-buyeruid",
			bidderParams: bidderParamsAppnexus,
			user: openrtb2.User{
				ID:      "some-id",
				Ext:     json.RawMessage(`{"data": 1, "test": 2, "prebid": {"buyeruids": ` + buyerUIDAppnexus + `}}`),
				Consent: tcf2Consent,
			},
			expectedUsers: map[string]openrtb2.User{
				"appnexus": {
					ID:       "some-id",
					BuyerUID: "a",
					Ext:      json.RawMessage(`{"consent":"` + tcf2Consent + `","data":1,"test":2}`),
				},
			},
		},
		{
			name:         "one-bidder-with-prebid-buyeruid-mixed-case",
			bidderParams: bidderParamsAppnexus,
			user: openrtb2.User{
				ID:      "some-id",
				Ext:     json.RawMessage(`{"data": 1, "test": 2, "prebid": {"buyeruids": ` + buyerUIDAppnexusMixedCase + `}}`),
				Consent: tcf2Consent,
			},
			expectedUsers: map[string]openrtb2.User{
				"appnexus": {
					ID:       "some-id",
					BuyerUID: "a",
					Ext:      json.RawMessage(`{"consent":"` + tcf2Consent + `","data":1,"test":2}`),
				},
			},
		},
		{
			name:         "one-bidder-with-buyeruid-already-set",
			bidderParams: bidderParamsAppnexus,
			user: openrtb2.User{
				ID:       "some-id",
				BuyerUID: "already-set-buyeruid",
				Ext:      json.RawMessage(`{"data": 1, "test": 2, "prebid": {"buyeruids": ` + buyerUIDAppnexus + `}}`),
				Consent:  tcf2Consent,
			},
			expectedUsers: map[string]openrtb2.User{
				"appnexus": {
					ID:       "some-id",
					BuyerUID: "already-set-buyeruid",
					Ext:      json.RawMessage(`{"consent":"` + tcf2Consent + `","data":1,"test":2}`),
				},
			},
		},
		{
			name:         "two-bidder-with-prebid-buyeruids",
			bidderParams: bidderParamsBoth,
			user: openrtb2.User{
				ID:      "some-id",
				Ext:     json.RawMessage(`{"data": 1, "test": 2, "prebid": {"buyeruids": ` + buyerUIDBoth + `}}`),
				Consent: tcf2Consent,
			},
			expectedUsers: map[string]openrtb2.User{
				"appnexus": {
					ID:       "some-id",
					BuyerUID: "a",
					Ext:      json.RawMessage(`{"consent":"` + tcf2Consent + `","data":1,"test":2}`),
				},
				"pubmatic": {
					ID:       "some-id",
					BuyerUID: "b",
					Ext:      json.RawMessage(`{"consent":"` + tcf2Consent + `","data":1,"test":2}`),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "some-publisher-id",
					},
				},
				Imp: []openrtb2.Imp{{
					ID: "some-imp-id",
					Banner: &openrtb2.Banner{
						Format: []openrtb2.Format{{
							W: 300,
							H: 250,
						}},
					},
					Ext: json.RawMessage(`{"prebid":{"tid":"123", "bidder":` + string(test.bidderParams) + `}}`),
				}},
				User: &test.user,
			}

			auctionReq := AuctionRequest{
				BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
				UserSyncs:         &emptyUsersync{},
				Account: config.Account{
					GDPR: config.AccountGDPR{
						Enabled: ptrutil.ToPtr(false),
					},
				},
				TCF2Config: gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			}
			gdprPermissionsBuilder := fakePermissionsBuilder{
				permissions: &permissionsMock{
					allowAllBidders: true,
					passGeo:         false,
					passID:          false,
					activitiesError: nil,
				},
			}.Builder

			reqSplitter := &requestSplitter{
				bidderToSyncerKey: map[string]string{},
				me:                &metrics.MetricsEngineMock{},
				gdprPermsBuilder:  gdprPermissionsBuilder,
				hostSChainNode:    nil,
				bidderInfo: config.BidderInfos{
					"appnexus": config.BidderInfo{
						OpenRTB: &config.OpenRTBInfo{
							Version: "2.5",
						},
					},
					"pubmatic": config.BidderInfo{
						OpenRTB: &config.OpenRTBInfo{
							Version: "2.5",
						},
					},
				},
			}

			results, _, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, nil, gdpr.SignalNo, false, nil)

			assert.Empty(t, errs)
			for _, v := range results {
				require.NotNil(t, v.BidRequest, "bidrequest")
				require.NotNil(t, v.BidRequest.User, "bidrequest.user")
				assert.Equal(t, test.expectedUsers[string(v.BidderName)], *v.BidRequest.User)
			}
		})
	}
}

func TestApplyFPD(t *testing.T) {
	testCases := []struct {
		description               string
		inputFpd                  map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData
		inputBidderName           string
		inputBidderCoreName       string
		inputBidderIsRequestAlias bool
		inputRequest              openrtb2.BidRequest
		expectedRequest           openrtb2.BidRequest
		fpdUserEIDsExisted        bool
	}{
		{
			description:               "fpd-nil",
			inputFpd:                  nil,
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}},
			expectedRequest:           openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}},
		},
		{
			description: "fpd-bidderdata-nil",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": nil,
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}},
			expectedRequest:           openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}},
		},
		{
			description: "fpd-bidderdata-notdefined",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"differentBidder": {App: &openrtb2.App{ID: "AppId"}},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}},
			expectedRequest:           openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}},
		},
		{
			description: "fpd-bidderdata-alias",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"alias": {App: &openrtb2.App{ID: "AppId"}},
			},
			inputBidderName:           "alias",
			inputBidderCoreName:       "bidder",
			inputBidderIsRequestAlias: true,
			inputRequest:              openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}},
			expectedRequest:           openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}},
		},
		{
			description: "req.Site defined; bidderFPD.Site not defined; expect request.Site remains the same",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": {Site: nil, App: nil, User: nil},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}},
			expectedRequest:           openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}},
		},
		{
			description: "req.Site, req.App, req.User are not defined; bidderFPD.App, bidderFPD.Site and bidderFPD.User defined; " +
				"expect req.Site, req.App, req.User to be overriden by bidderFPD.App, bidderFPD.Site and bidderFPD.User",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": {Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}, User: &openrtb2.User{ID: "UserId"}},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{},
			expectedRequest:           openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}, User: &openrtb2.User{ID: "UserId"}},
		},
		{
			description: "req.Site, defined; bidderFPD.App defined; expect request.App to be overriden by bidderFPD.App; expect req.Site remains the same",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": {App: &openrtb2.App{ID: "AppId"}},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}},
			expectedRequest:           openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}},
		},
		{
			description: "req.Site, req.App defined; bidderFPD.App defined; expect request.App to be overriden by bidderFPD.App",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": {App: &openrtb2.App{ID: "AppId"}},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "TestAppId"}},
			expectedRequest:           openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}},
		},
		{
			description: "req.User is defined; bidderFPD.User defined; req.User has BuyerUID. Expect to see user.BuyerUID in result request",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": {Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}, User: &openrtb2.User{ID: "UserId"}},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{User: &openrtb2.User{ID: "UserIdIn", BuyerUID: "12345"}},
			expectedRequest:           openrtb2.BidRequest{User: &openrtb2.User{ID: "UserId", BuyerUID: "12345"}, Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}},
		},
		{
			description: "req.User is defined; bidderFPD.User defined; req.User has BuyerUID with zero length. Expect to see empty user.BuyerUID in result request",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": {Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}, User: &openrtb2.User{ID: "UserId"}},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{User: &openrtb2.User{ID: "UserIdIn", BuyerUID: ""}},
			expectedRequest:           openrtb2.BidRequest{User: &openrtb2.User{ID: "UserId"}, Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}},
		},
		{
			description: "req.User is not defined; bidderFPD.User defined and has BuyerUID. Expect to see user.BuyerUID in result request",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": {Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}, User: &openrtb2.User{ID: "UserId", BuyerUID: "FPDBuyerUID"}},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{},
			expectedRequest:           openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}, User: &openrtb2.User{ID: "UserId", BuyerUID: "FPDBuyerUID"}},
		},
		{
			description: "req.User is defined and had bidder fpd user eids (fpdUserEIDsExisted); bidderFPD.User defined and has EIDs. Expect to see user.EIDs in result request taken from fpd",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": {Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}, User: &openrtb2.User{ID: "UserId", EIDs: []openrtb2.EID{{Source: "source1"}, {Source: "source2"}}}},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{User: &openrtb2.User{ID: "UserId", EIDs: []openrtb2.EID{{Source: "source3"}, {Source: "source4"}}}},
			expectedRequest:           openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}, User: &openrtb2.User{ID: "UserId", EIDs: []openrtb2.EID{{Source: "source1"}, {Source: "source2"}}}},
			fpdUserEIDsExisted:        true,
		},
		{
			description: "req.User is defined and doesn't have fpr user eids (fpdUserEIDsExisted); bidderFPD.User defined and has EIDs. Expect to see user.EIDs in result request taken from original req",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": {Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}, User: &openrtb2.User{ID: "UserId", EIDs: []openrtb2.EID{{Source: "source1"}, {Source: "source2"}}}},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{User: &openrtb2.User{ID: "UserId", EIDs: []openrtb2.EID{{Source: "source3"}, {Source: "source4"}}}},
			expectedRequest:           openrtb2.BidRequest{Site: &openrtb2.Site{ID: "SiteId"}, App: &openrtb2.App{ID: "AppId"}, User: &openrtb2.User{ID: "UserId", EIDs: []openrtb2.EID{{Source: "source3"}, {Source: "source4"}}}},
			fpdUserEIDsExisted:        false,
		},
		{
			description: "req.Device defined; bidderFPD.Device defined; expect request.Device to be overriden by bidderFPD.Device",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": {Device: &openrtb2.Device{Make: "DeviceMake"}},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{Device: &openrtb2.Device{Make: "TestDeviceMake"}},
			expectedRequest:           openrtb2.BidRequest{Device: &openrtb2.Device{Make: "DeviceMake"}},
		},
		{
			description: "req.Device defined; bidderFPD.Device not defined; expect request.Device remains the same",
			inputFpd: map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData{
				"bidderNormalized": {Device: nil},
			},
			inputBidderName:           "bidderFromRequest",
			inputBidderCoreName:       "bidderNormalized",
			inputBidderIsRequestAlias: false,
			inputRequest:              openrtb2.BidRequest{Device: &openrtb2.Device{Make: "TestDeviceMake"}},
			expectedRequest:           openrtb2.BidRequest{Device: &openrtb2.Device{Make: "TestDeviceMake"}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			reqWrapper := &openrtb_ext.RequestWrapper{BidRequest: &testCase.inputRequest}
			applyFPD(
				testCase.inputFpd,
				openrtb_ext.BidderName(testCase.inputBidderCoreName),
				openrtb_ext.BidderName(testCase.inputBidderName),
				testCase.inputBidderIsRequestAlias,
				reqWrapper,
				testCase.fpdUserEIDsExisted,
			)
			assert.Equal(t, &testCase.expectedRequest, reqWrapper.BidRequest)
		})
	}
}

func TestGetRequestAliases(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest openrtb_ext.RequestWrapper
		wantAliases  map[string]string
		wantGVLIDs   map[string]uint16
		wantError    string
	}{
		{
			name: "nil",
			givenRequest: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			wantAliases: nil,
			wantGVLIDs:  nil,
			wantError:   "",
		},
		{
			name: "empty",
			givenRequest: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{}`),
				},
			},
			wantAliases: nil,
			wantGVLIDs:  nil,
			wantError:   "",
		},
		{
			name: "empty-prebid",
			givenRequest: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{}}`),
				},
			},
			wantAliases: nil,
			wantGVLIDs:  nil,
			wantError:   "",
		},
		{
			name: "aliases-and-gvlids",
			givenRequest: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"aliases":{"alias1":"bidder1"}, "aliasgvlids":{"alias1":1}}}`),
				},
			},
			wantAliases: map[string]string{"alias1": "bidder1"},
			wantGVLIDs:  map[string]uint16{"alias1": 1},
			wantError:   "",
		},
		{
			name: "malformed",
			givenRequest: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`malformed`),
				},
			},
			wantAliases: nil,
			wantGVLIDs:  nil,
			wantError:   "request.ext is invalid",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotAliases, gotGVLIDs, err := getRequestAliases(&test.givenRequest)

			assert.Equal(t, test.wantAliases, gotAliases, "aliases")
			assert.Equal(t, test.wantGVLIDs, gotGVLIDs, "gvlids")

			if len(test.wantError) > 0 {
				require.Len(t, err, 1, "error-len")
				assert.EqualError(t, err[0], test.wantError, "error")
			} else {
				assert.Empty(t, err, "error")
			}
		})
	}
}

func TestBuildExtData(t *testing.T) {
	testCases := []struct {
		description string
		input       []byte
		expectedRes string
	}{
		{
			description: "Input object with int value",
			input:       []byte(`{"someData": 123}`),
			expectedRes: `{"data": {"someData": 123}}`,
		},
		{
			description: "Input object with bool value",
			input:       []byte(`{"someData": true}`),
			expectedRes: `{"data": {"someData": true}}`,
		},
		{
			description: "Input object with string value",
			input:       []byte(`{"someData": "true"}`),
			expectedRes: `{"data": {"someData": "true"}}`,
		},
		{
			description: "No input object",
			input:       []byte(`{}`),
			expectedRes: `{"data": {}}`,
		},
		{
			description: "Input object with object value",
			input:       []byte(`{"someData": {"moreFpdData": "fpddata"}}`),
			expectedRes: `{"data": {"someData": {"moreFpdData": "fpddata"}}}`,
		},
	}

	for _, test := range testCases {
		actualRes := WrapJSONInData(test.input)
		assert.JSONEq(t, test.expectedRes, string(actualRes), "Incorrect result data")
	}
}

func TestCleanOpenRTBRequestsFilterBidderRequestExt(t *testing.T) {
	testCases := []struct {
		desc      string
		inExt     json.RawMessage
		inCfgABC  *openrtb_ext.ExtAlternateBidderCodes
		wantExt   []json.RawMessage
		wantError bool
	}{
		{
			desc:      "Nil request ext, default account alternatebiddercodes config (nil)",
			inExt:     nil,
			inCfgABC:  nil,
			wantExt:   nil,
			wantError: false,
		},
		{
			desc:     "Nil request ext, default account alternatebiddercodes config (explicity defined)",
			inExt:    nil,
			inCfgABC: &openrtb_ext.ExtAlternateBidderCodes{Enabled: false},
			wantExt: []json.RawMessage{
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":null}}}`),
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":null}}}`),
			},
			wantError: false,
		},
		{
			desc:     "request ext, default account alternatebiddercodes config (explicity defined)",
			inExt:    json.RawMessage(`{"prebid":{}}`),
			inCfgABC: &openrtb_ext.ExtAlternateBidderCodes{Enabled: false},
			wantExt: []json.RawMessage{
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":null}}}`),
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":null}}}`),
			},
			wantError: false,
		},
		{
			desc:  "Nil request ext, account alternatebiddercodes config disabled with biddercodes defined",
			inExt: nil,
			inCfgABC: &openrtb_ext.ExtAlternateBidderCodes{
				Enabled: false,
				Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
					"pubmatic": {Enabled: true},
				},
			},
			wantExt: []json.RawMessage{
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":null}}}`),
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":null}}}}}`),
			},
			wantError: false,
		},
		{
			desc:  "Nil request ext, account alternatebiddercodes config disabled with biddercodes defined (not participant bidder)",
			inExt: nil,
			inCfgABC: &openrtb_ext.ExtAlternateBidderCodes{
				Enabled: false,
				Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
					"ix": {Enabled: true},
				},
			},
			wantExt: []json.RawMessage{
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":null}}}`),
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":null}}}`),
			},
			wantError: false,
		},
		{
			desc:     "Nil request ext, alternatebiddercodes config enabled but bidder not present",
			inExt:    nil,
			inCfgABC: &openrtb_ext.ExtAlternateBidderCodes{Enabled: true},
			wantExt: []json.RawMessage{
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":null}}}`),
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":null}}}`),
			},
			wantError: false,
		},
		{
			desc:      "request ext with default alternatebiddercodes values (nil)",
			inExt:     json.RawMessage(`{"prebid":{}}`),
			inCfgABC:  nil,
			wantExt:   nil,
			wantError: false,
		},
		{
			desc:     "request ext w/o alternatebiddercodes",
			inExt:    json.RawMessage(`{"prebid":{}}`),
			inCfgABC: &openrtb_ext.ExtAlternateBidderCodes{},
			wantExt: []json.RawMessage{
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":null}}}`),
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":false,"bidders":null}}}`),
			},
			wantError: false,
		},
		{
			desc:     "request ext having alternatebiddercodes for only one bidder",
			inExt:    json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]}}}}}`),
			inCfgABC: &openrtb_ext.ExtAlternateBidderCodes{Enabled: false},
			wantExt: []json.RawMessage{
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":null}}}`),
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]}}}}}`),
			},
			wantError: false,
		},
		{
			desc:     "request ext having alternatebiddercodes for multiple bidder",
			inExt:    json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]},"appnexus":{"enabled":true,"allowedbiddercodes":["ix"]}}}}}`),
			inCfgABC: &openrtb_ext.ExtAlternateBidderCodes{Enabled: false},
			wantExt: []json.RawMessage{
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"appnexus":{"enabled":true,"allowedbiddercodes":["ix"]}}}}}`),
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]}}}}}`),
			},
			wantError: false,
		},
		{
			desc:     "request ext having alternatebiddercodes for multiple bidder (config alternatebiddercodes not defined)",
			inExt:    json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]},"appnexus":{"enabled":true,"allowedbiddercodes":["ix"]}}}}}`),
			inCfgABC: &openrtb_ext.ExtAlternateBidderCodes{Enabled: false},
			wantExt: []json.RawMessage{
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"appnexus":{"enabled":true,"allowedbiddercodes":["ix"]}}}}}`),
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]}}}}}`),
			},
			wantError: false,
		},
		{
			desc:  "Nil request ext, alternatebiddercodes config enabled with bidder code for only one bidder",
			inExt: nil,
			inCfgABC: &openrtb_ext.ExtAlternateBidderCodes{
				Enabled: true,
				Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
					"pubmatic": {
						Enabled:            true,
						AllowedBidderCodes: []string{"groupm"},
					},
				},
			},
			wantExt: []json.RawMessage{
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":null}}}`),
				json.RawMessage(`{"prebid":{"alternatebiddercodes":{"enabled":true,"bidders":{"pubmatic":{"enabled":true,"allowedbiddercodes":["groupm"]}}}}}`),
			},
			wantError: false,
		},
	}

	for _, test := range testCases {
		req := newBidRequestWithBidderParams()
		req.Ext = nil
		var extRequest *openrtb_ext.ExtRequest
		if test.inExt != nil {
			req.Ext = test.inExt
			extRequest = &openrtb_ext.ExtRequest{}
			err := jsonutil.UnmarshalValid(req.Ext, extRequest)
			assert.NoErrorf(t, err, test.desc+":Error unmarshaling inExt")
		}

		auctionReq := AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
			UserSyncs:         &emptyUsersync{},
			Account:           config.Account{AlternateBidderCodes: test.inCfgABC},
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}
		gdprPermissionsBuilder := fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder

		reqSplitter := &requestSplitter{
			bidderToSyncerKey: map[string]string{},
			me:                &metrics.MetricsEngineMock{},
			privacyConfig:     config.Privacy{},
			gdprPermsBuilder:  gdprPermissionsBuilder,
			hostSChainNode:    nil,
			bidderInfo:        config.BidderInfos{},
		}

		bidderRequests, _, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, extRequest, gdpr.SignalNo, false, map[string]float64{})
		assert.Equal(t, test.wantError, len(errs) != 0, test.desc)
		sort.Slice(bidderRequests, func(i, j int) bool {
			return bidderRequests[i].BidderCoreName < bidderRequests[j].BidderCoreName
		})
		for i, wantBidderRequest := range test.wantExt {
			assert.Equal(t, wantBidderRequest, bidderRequests[i].BidRequest.Ext, test.desc+" : "+string(bidderRequests[i].BidderCoreName)+"\n\t\tGotRequestExt : "+string(bidderRequests[i].BidRequest.Ext))
		}
	}
}

func TestGetTargetDataPrefix(t *testing.T) {
	testCases := []struct {
		description      string
		requestPrefix    string
		account          config.Account
		expectedResult   string
		expectedWarnings int
	}{
		{
			description:   "TruncateTargetAttribute set is nil",
			requestPrefix: "",
			account: config.Account{
				TargetingPrefix:         "hb",
				TruncateTargetAttribute: nil,
			},
			expectedResult:   "hb",
			expectedWarnings: 0,
		},
		{
			description:   "TargetingPrefix set in Account",
			requestPrefix: "tst",
			account: config.Account{
				TargetingPrefix:         "tst",
				TruncateTargetAttribute: intPtr(15),
			},
			expectedResult:   "tst",
			expectedWarnings: 0,
		},
		{
			description:   "TargetingPrefix is longer than expected",
			requestPrefix: "test",
			account: config.Account{
				TargetingPrefix:         "test",
				TruncateTargetAttribute: intPtr(15),
			},
			expectedResult:   "hb",
			expectedWarnings: 1,
		},
		{
			description:   "TruncateTargetAttribute is smaller than expected",
			requestPrefix: "test",
			account: config.Account{
				TargetingPrefix:         "test",
				TruncateTargetAttribute: intPtr(1),
			},
			expectedResult:   "hb",
			expectedWarnings: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result, warnings := getTargetDataPrefix(tc.requestPrefix, tc.account)
			assert.Equal(t, tc.expectedResult, result)
			assert.Len(t, warnings, tc.expectedWarnings)
		})
	}
}

type GPPMockSection struct {
	sectionID constants.SectionID
	value     string
}

func (gs GPPMockSection) GetID() constants.SectionID {
	return gs.sectionID
}

func (gs GPPMockSection) GetValue() string {
	return gs.value
}

func (gs GPPMockSection) Encode(bool) []byte {
	return nil
}

func TestGdprFromGPP(t *testing.T) {
	testCases := []struct {
		name            string
		initialRequest  *openrtb_ext.RequestWrapper
		gpp             gpplib.GppContainer
		expectedRequest *openrtb_ext.RequestWrapper
	}{
		{
			name: "Empty", // Empty Request
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			gpp: gpplib.GppContainer{},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
		},
		{
			name: "GDPR_Downgrade", // GDPR from GPP, into empty
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{2},
					},
				},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{2},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 2,
						value:     "GDPRConsent",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{2},
						GDPR:   ptrutil.ToPtr[int8](1),
					},
					User: &openrtb2.User{
						Consent: "GDPRConsent",
					},
				},
			},
		},
		{
			name: "GDPR_Downgrade", // GDPR from GPP, into empty legacy, existing objects
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID:    []int8{2},
						USPrivacy: "LegacyUSP",
					},
					User: &openrtb2.User{
						ID: "1234",
					},
				},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{2},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 2,
						value:     "GDPRConsent",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID:    []int8{2},
						GDPR:      ptrutil.ToPtr[int8](1),
						USPrivacy: "LegacyUSP",
					},
					User: &openrtb2.User{
						ID:      "1234",
						Consent: "GDPRConsent",
					},
				},
			},
		},
		{
			name: "Downgrade_Blocked_By_Existing", // GDPR from GPP blocked by existing GDPR",
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{2},
						GDPR:   ptrutil.ToPtr[int8](1),
					},
					User: &openrtb2.User{
						Consent: "LegacyConsent",
					},
				},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{2},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 2,
						value:     "GDPRConsent",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{2},
						GDPR:   ptrutil.ToPtr[int8](1),
					},
					User: &openrtb2.User{
						Consent: "LegacyConsent",
					},
				},
			},
		},
		{
			name: "Downgrade_Partial", // GDPR from GPP partially blocked by existing GDPR
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{2},
						GDPR:   ptrutil.ToPtr[int8](0),
					},
				},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{2},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 2,
						value:     "GDPRConsent",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{2},
						GDPR:   ptrutil.ToPtr[int8](0),
					},
					User: &openrtb2.User{
						Consent: "GDPRConsent",
					},
				},
			},
		},
		{
			name: "No_GDPR", // Downgrade not possible due to missing GDPR
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{6},
					},
				},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{6},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 6,
						value:     "USPrivacy",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{6},
						GDPR:   ptrutil.ToPtr[int8](0),
					},
				},
			},
		},
		{
			name: "No_SID", // GDPR from GPP partially blocked by no SID
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{6},
					},
				},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{2, 6},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 2,
						value:     "GDPRConsent",
					},
					GPPMockSection{
						sectionID: 6,
						value:     "USPrivacy",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{6},
						GDPR:   ptrutil.ToPtr[int8](0),
					},
					User: &openrtb2.User{
						Consent: "GDPRConsent",
					},
				},
			},
		},
		{
			name: "GDPR_Nil_SID", // GDPR from GPP, into empty, but with nil SID
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{2},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 2,
						value:     "GDPRConsent",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Consent: "GDPRConsent",
					},
				},
			},
		},
		{
			name: "Downgrade_Nil_SID_Blocked_By_Existing", // GDPR from GPP blocked by existing GDPR, with nil SID",
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GDPR: ptrutil.ToPtr[int8](1),
					},
					User: &openrtb2.User{
						Consent: "LegacyConsent",
					},
				},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{2},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 2,
						value:     "GDPRConsent",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GDPR: ptrutil.ToPtr[int8](1),
					},
					User: &openrtb2.User{
						Consent: "LegacyConsent",
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			setLegacyGDPRFromGPP(test.initialRequest, test.gpp)
			assert.Equal(t, test.expectedRequest, test.initialRequest)
		})
	}
}

func TestPrivacyFromGPP(t *testing.T) {
	testCases := []struct {
		name            string
		initialRequest  *openrtb_ext.RequestWrapper
		gpp             gpplib.GppContainer
		expectedRequest *openrtb_ext.RequestWrapper
	}{
		{
			name: "Empty", // Empty Request
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			gpp: gpplib.GppContainer{},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
		},
		{
			name: "Privacy_Downgrade", // US Privacy from GPP, into empty
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{6},
					},
				},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{6},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 6,
						value:     "USPrivacy",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID:    []int8{6},
						USPrivacy: "USPrivacy",
					},
				},
			},
		},
		{
			name: "Downgrade_Blocked_By_Existing", // US Privacy from GPP blocked by existing US Privacy
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID:    []int8{6},
						USPrivacy: "LegacyPrivacy",
					},
				},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{6},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 6,
						value:     "USPrivacy",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID:    []int8{6},
						USPrivacy: "LegacyPrivacy",
					},
				},
			},
		},
		{
			name: "No_USPrivacy", // Downgrade not possible due to missing USPrivacy
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{2},
					},
				},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{2},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 2,
						value:     "GDPRConsent",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{2},
					},
				},
			},
		},
		{
			name: "No_SID", // US Privacy from GPP partially blocked by no SID
			initialRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{2},
					},
				},
			},
			gpp: gpplib.GppContainer{
				SectionTypes: []constants.SectionID{2, 6},
				Sections: []gpplib.Section{
					GPPMockSection{
						sectionID: 2,
						value:     "GDPRConsent",
					},
					GPPMockSection{
						sectionID: 6,
						value:     "USPrivacy",
					},
				},
			},
			expectedRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{2},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			setLegacyUSPFromGPP(test.initialRequest, test.gpp)
			assert.Equal(t, test.expectedRequest, test.initialRequest)
		})
	}
}

func Test_isBidderInExtAlternateBidderCodes(t *testing.T) {
	type args struct {
		adapter               string
		currentMultiBidBidder string
		adapterABC            *openrtb_ext.ExtAlternateBidderCodes
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "alternatebiddercodes not defined",
			want: false,
		},
		{
			name: "adapter not defined in alternatebiddercodes",
			args: args{
				adapter: string(openrtb_ext.BidderPubmatic),
				adapterABC: &openrtb_ext.ExtAlternateBidderCodes{
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{string(openrtb_ext.BidderAppnexus): {}},
				},
			},
			want: false,
		},
		{
			name: "adapter defined in alternatebiddercodes but currentMultiBidBidder not in AllowedBidders list",
			args: args{
				adapter:               string(openrtb_ext.BidderPubmatic),
				currentMultiBidBidder: "groupm",
				adapterABC: &openrtb_ext.ExtAlternateBidderCodes{
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						string(openrtb_ext.BidderPubmatic): {
							AllowedBidderCodes: []string{string(openrtb_ext.BidderAppnexus)},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "adapter defined in alternatebiddercodes with currentMultiBidBidder mentioned in AllowedBidders list",
			args: args{
				adapter:               string(openrtb_ext.BidderPubmatic),
				currentMultiBidBidder: "groupm",
				adapterABC: &openrtb_ext.ExtAlternateBidderCodes{
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						string(openrtb_ext.BidderPubmatic): {
							AllowedBidderCodes: []string{"groupm"},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "adapter defined in alternatebiddercodes with AllowedBidders list as *",
			args: args{
				adapter:               string(openrtb_ext.BidderPubmatic),
				currentMultiBidBidder: "groupm",
				adapterABC: &openrtb_ext.ExtAlternateBidderCodes{
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						string(openrtb_ext.BidderPubmatic): {
							AllowedBidderCodes: []string{"*"},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBidderInExtAlternateBidderCodes(tt.args.adapter, tt.args.currentMultiBidBidder, tt.args.adapterABC); got != tt.want {
				t.Errorf("isBidderInExtAlternateBidderCodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildRequestExtMultiBid(t *testing.T) {
	type args struct {
		adapter     string
		reqMultiBid []*openrtb_ext.ExtMultiBid
		adapterABC  *openrtb_ext.ExtAlternateBidderCodes
	}
	tests := []struct {
		name string
		args args
		want []*openrtb_ext.ExtMultiBid
	}{
		{
			name: "multi-bid config not defined",
			args: args{
				adapter:     string(openrtb_ext.BidderPubmatic),
				reqMultiBid: nil,
			},
			want: nil,
		},
		{
			name: "adapter not defined in multi-bid config",
			args: args{
				adapter: string(openrtb_ext.BidderPubmatic),
				reqMultiBid: []*openrtb_ext.ExtMultiBid{
					{
						Bidder:  string(openrtb_ext.BidderAppnexus),
						MaxBids: ptrutil.ToPtr(2),
					},
				},
			},
			want: nil,
		},
		{
			name: "adapter defined in multi-bid config as Bidder object along with other bidders",
			args: args{
				adapter: string(openrtb_ext.BidderPubmatic),
				reqMultiBid: []*openrtb_ext.ExtMultiBid{
					{
						Bidder:  string(openrtb_ext.BidderAppnexus),
						MaxBids: ptrutil.ToPtr(3),
					},
					{
						Bidder:  string(openrtb_ext.BidderPubmatic),
						MaxBids: ptrutil.ToPtr(2),
					},
					{
						Bidders: []string{string(openrtb_ext.Bidder33Across), string(openrtb_ext.BidderRubicon)},
						MaxBids: ptrutil.ToPtr(2),
					},
				},
			},
			want: []*openrtb_ext.ExtMultiBid{
				{
					Bidder:  string(openrtb_ext.BidderPubmatic),
					MaxBids: ptrutil.ToPtr(2),
				},
			},
		},
		{
			name: "adapter defined in multi-bid config as a entry of Bidders list along with other bidders",
			args: args{
				adapter: string(openrtb_ext.BidderRubicon),
				reqMultiBid: []*openrtb_ext.ExtMultiBid{
					{
						Bidder:  string(openrtb_ext.BidderAppnexus),
						MaxBids: ptrutil.ToPtr(3),
					},
					{
						Bidder:  string(openrtb_ext.BidderPubmatic),
						MaxBids: ptrutil.ToPtr(2),
					},
					{
						Bidders: []string{string(openrtb_ext.Bidder33Across), string(openrtb_ext.BidderRubicon)},
						MaxBids: ptrutil.ToPtr(4),
					},
				},
			},
			want: []*openrtb_ext.ExtMultiBid{
				{
					Bidders: []string{string(openrtb_ext.BidderRubicon)},
					MaxBids: ptrutil.ToPtr(4),
				},
			},
		},
		{
			name: "adapter defined in multi-bid config as Bidder object along with other bidders with alternateBiddercode",
			args: args{
				adapter: string(openrtb_ext.BidderPubmatic),
				reqMultiBid: []*openrtb_ext.ExtMultiBid{
					{
						Bidder:  "groupm",
						MaxBids: ptrutil.ToPtr(3),
					},
					{
						Bidder:  string(openrtb_ext.BidderPubmatic),
						MaxBids: ptrutil.ToPtr(2),
					},
					{
						Bidders: []string{string(openrtb_ext.Bidder33Across), string(openrtb_ext.BidderRubicon)},
						MaxBids: ptrutil.ToPtr(2),
					},
				},
				adapterABC: &openrtb_ext.ExtAlternateBidderCodes{
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						string(openrtb_ext.BidderPubmatic): {
							AllowedBidderCodes: []string{"groupm"},
						},
					},
				},
			},
			want: []*openrtb_ext.ExtMultiBid{
				{
					Bidder:  "groupm",
					MaxBids: ptrutil.ToPtr(3),
				},
				{
					Bidder:  string(openrtb_ext.BidderPubmatic),
					MaxBids: ptrutil.ToPtr(2),
				},
			},
		},
		{
			name: "adapter defined in multi-bid config as a entry of Bidders list along with other bidders with alternateBiddercode",
			args: args{
				adapter: string(openrtb_ext.BidderAppnexus),
				reqMultiBid: []*openrtb_ext.ExtMultiBid{
					{
						Bidder:  "groupm",
						MaxBids: ptrutil.ToPtr(3),
					},
					{
						Bidder:  string(openrtb_ext.BidderPubmatic),
						MaxBids: ptrutil.ToPtr(2),
					},
					{
						Bidders: []string{string(openrtb_ext.Bidder33Across), string(openrtb_ext.BidderAppnexus)},
						MaxBids: ptrutil.ToPtr(4),
					},
				},
				adapterABC: &openrtb_ext.ExtAlternateBidderCodes{
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						string(openrtb_ext.BidderAppnexus): {
							AllowedBidderCodes: []string{"groupm"},
						},
					},
				},
			},
			want: []*openrtb_ext.ExtMultiBid{
				{
					Bidder:  "groupm",
					MaxBids: ptrutil.ToPtr(3),
				},
				{
					Bidders: []string{string(openrtb_ext.BidderAppnexus)},
					MaxBids: ptrutil.ToPtr(4),
				},
			},
		},
		{
			name: "adapter defined in multi-bid config as Bidder object along with other bidders with alternateBiddercode.AllowedBidders as *",
			args: args{
				adapter: string(openrtb_ext.BidderPubmatic),
				reqMultiBid: []*openrtb_ext.ExtMultiBid{
					{
						Bidder:  "groupm",
						MaxBids: ptrutil.ToPtr(3),
					},
					{
						Bidder:  string(openrtb_ext.BidderPubmatic),
						MaxBids: ptrutil.ToPtr(2),
					},
					{
						Bidders: []string{string(openrtb_ext.Bidder33Across), string(openrtb_ext.BidderRubicon)},
						MaxBids: ptrutil.ToPtr(2),
					},
				},
				adapterABC: &openrtb_ext.ExtAlternateBidderCodes{
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						string(openrtb_ext.BidderPubmatic): {
							AllowedBidderCodes: []string{"*"},
						},
					},
				},
			},
			want: []*openrtb_ext.ExtMultiBid{
				{
					Bidder:  "groupm",
					MaxBids: ptrutil.ToPtr(3),
				},
				{
					Bidder:  string(openrtb_ext.BidderPubmatic),
					MaxBids: ptrutil.ToPtr(2),
				},
				{
					Bidders: []string{string(openrtb_ext.Bidder33Across)},
					MaxBids: ptrutil.ToPtr(2),
				},
				{
					Bidders: []string{string(openrtb_ext.BidderRubicon)},
					MaxBids: ptrutil.ToPtr(2),
				},
			},
		},
		{
			name: "adapter defined in multi-bid config as a entry of Bidders list along with other bidders with alternateBiddercode.AllowedBidders as *",
			args: args{
				adapter: string(openrtb_ext.BidderAppnexus),
				reqMultiBid: []*openrtb_ext.ExtMultiBid{
					{
						Bidder:  "groupm",
						MaxBids: ptrutil.ToPtr(3),
					},
					{
						Bidder:  string(openrtb_ext.BidderPubmatic),
						MaxBids: ptrutil.ToPtr(2),
					},
					{
						Bidders: []string{string(openrtb_ext.Bidder33Across), string(openrtb_ext.BidderAppnexus)},
						MaxBids: ptrutil.ToPtr(4),
					},
				},
				adapterABC: &openrtb_ext.ExtAlternateBidderCodes{
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						string(openrtb_ext.BidderAppnexus): {
							AllowedBidderCodes: []string{"*"},
						},
					},
				},
			},
			want: []*openrtb_ext.ExtMultiBid{
				{
					Bidder:  "groupm",
					MaxBids: ptrutil.ToPtr(3),
				},
				{
					Bidder:  string(openrtb_ext.BidderPubmatic),
					MaxBids: ptrutil.ToPtr(2),
				},
				{
					Bidders: []string{string(openrtb_ext.Bidder33Across)},
					MaxBids: ptrutil.ToPtr(4),
				},
				{
					Bidders: []string{string(openrtb_ext.BidderAppnexus)},
					MaxBids: ptrutil.ToPtr(4),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRequestExtMultiBid(tt.args.adapter, tt.args.reqMultiBid, tt.args.adapterABC)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPrebidMediaTypeForBid(t *testing.T) {
	tests := []struct {
		description     string
		inputBid        openrtb2.Bid
		expectedBidType openrtb_ext.BidType
		expectedError   string
	}{
		{
			description:     "Valid bid ext with bid type native",
			inputBid:        openrtb2.Bid{ID: "bidId", ImpID: "impId", Ext: json.RawMessage(`{"prebid": {"type": "native"}}`)},
			expectedBidType: openrtb_ext.BidTypeNative,
		},
		{
			description:   "Valid bid ext with non-existing bid type",
			inputBid:      openrtb2.Bid{ID: "bidId", ImpID: "impId", Ext: json.RawMessage(`{"prebid": {"type": "unknown"}}`)},
			expectedError: "Failed to parse bid mediatype for impression \"impId\", invalid BidType: unknown",
		},
		{
			description:   "Invalid bid ext",
			inputBid:      openrtb2.Bid{ID: "bidId", ImpID: "impId", Ext: json.RawMessage(`[true`)},
			expectedError: "Failed to parse bid mediatype for impression \"impId\", expect { or n, but found [",
		},
		{
			description:   "Bid ext is nil",
			inputBid:      openrtb2.Bid{ID: "bidId", ImpID: "impId", Ext: nil},
			expectedError: "Failed to parse bid mediatype for impression \"impId\"",
		},
		{
			description:   "Empty bid ext",
			inputBid:      openrtb2.Bid{ID: "bidId", ImpID: "impId", Ext: json.RawMessage(`{}`)},
			expectedError: "Failed to parse bid mediatype for impression \"impId\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			bidType, err := getPrebidMediaTypeForBid(tt.inputBid)
			if len(tt.expectedError) == 0 {
				assert.Equal(t, tt.expectedBidType, bidType)
			} else {
				assert.Equal(t, tt.expectedError, err.Error())
			}
		})
	}
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		description     string
		inputBid        openrtb2.Bid
		expectedBidType openrtb_ext.BidType
		expectedError   string
	}{
		{
			description:     "Valid bid ext with bid type native",
			inputBid:        openrtb2.Bid{ID: "bidId", ImpID: "impId", Ext: json.RawMessage(`{"prebid": {"type": "native"}}`)},
			expectedBidType: openrtb_ext.BidTypeNative,
		},
		{
			description:   "invalid bid ext",
			inputBid:      openrtb2.Bid{ID: "bidId", ImpID: "impId", Ext: json.RawMessage(`{"prebid"`)},
			expectedError: "Failed to parse bid mediatype for impression \"impId\", expect :, but found \x00",
		},
		{
			description:     "Valid bid ext with mtype native",
			inputBid:        openrtb2.Bid{ID: "bidId", ImpID: "impId", MType: openrtb2.MarkupNative},
			expectedBidType: openrtb_ext.BidTypeNative,
		},
		{
			description:     "Valid bid ext with mtype banner",
			inputBid:        openrtb2.Bid{ID: "bidId", ImpID: "impId", MType: openrtb2.MarkupBanner},
			expectedBidType: openrtb_ext.BidTypeBanner,
		},
		{
			description:     "Valid bid ext with mtype video",
			inputBid:        openrtb2.Bid{ID: "bidId", ImpID: "impId", MType: openrtb2.MarkupVideo},
			expectedBidType: openrtb_ext.BidTypeVideo,
		},
		{
			description:     "Valid bid ext with mtype audio",
			inputBid:        openrtb2.Bid{ID: "bidId", ImpID: "impId", MType: openrtb2.MarkupAudio},
			expectedBidType: openrtb_ext.BidTypeAudio,
		},
		{
			description:   "Valid bid ext with mtype unknown",
			inputBid:      openrtb2.Bid{ID: "bidId", ImpID: "impId", MType: 8},
			expectedError: "Failed to parse bid mType for impression \"impId\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			bidType, err := getMediaTypeForBid(tt.inputBid)
			if len(tt.expectedError) == 0 {
				assert.Equal(t, tt.expectedBidType, bidType)
			} else {
				assert.Equal(t, tt.expectedError, err.Error())
			}
		})
	}
}

func TestCleanOpenRTBRequestsActivities(t *testing.T) {
	expectedUserDefault := openrtb2.User{
		ID:       "our-id",
		BuyerUID: "their-id",
		Yob:      1982,
		Gender:   "test",
		Ext:      json.RawMessage(`{"data": 1, "test": 2}`),
		Geo:      &openrtb2.Geo{Lat: ptrutil.ToPtr(123.456), Lon: ptrutil.ToPtr(11.278)},
		EIDs: []openrtb2.EID{
			{Source: "eids-source"},
		},
		Data: []openrtb2.Data{{ID: "data-id"}},
	}
	expectedDeviceDefault := openrtb2.Device{
		UA:       deviceUA,
		IP:       "132.173.230.74",
		Language: "EN",
		DIDMD5:   "DIDMD5",
		IFA:      "IFA",
		DIDSHA1:  "DIDSHA1",
		DPIDMD5:  "DPIDMD5",
		DPIDSHA1: "DPIDSHA1",
		MACMD5:   "MACMD5",
		MACSHA1:  "MACSHA1",
		Geo:      &openrtb2.Geo{Lat: ptrutil.ToPtr(123.456), Lon: ptrutil.ToPtr(11.278)},
	}

	expectedSourceDefault := openrtb2.Source{
		TID: "testTID",
	}

	testCases := []struct {
		name              string
		req               *openrtb2.BidRequest
		privacyConfig     config.AccountPrivacy
		componentName     string
		allow             bool
		ortbVersion       string
		expectedReqNumber int
		expectedUser      openrtb2.User
		expectUserScrub   bool
		expectedDevice    openrtb2.Device
		expectedSource    openrtb2.Source
		expectedImpExt    json.RawMessage
	}{
		{
			name:              "fetch_bids_request_with_one_bidder_allowed",
			req:               newBidRequest(),
			privacyConfig:     getFetchBidsActivityConfig("appnexus", true),
			ortbVersion:       "2.6",
			expectedReqNumber: 1,
			expectedUser:      expectedUserDefault,
			expectedDevice:    expectedDeviceDefault,
			expectedSource:    expectedSourceDefault,
		},
		{
			name:              "fetch_bids_request_with_one_bidder_not_allowed",
			req:               newBidRequest(),
			privacyConfig:     getFetchBidsActivityConfig("appnexus", false),
			expectedReqNumber: 0,
			expectedUser:      expectedUserDefault,
			expectedDevice:    expectedDeviceDefault,
			expectedSource:    expectedSourceDefault,
		},
		{
			name:              "transmit_ufpd_allowed",
			req:               newBidRequest(),
			privacyConfig:     getTransmitUFPDActivityConfig("appnexus", true),
			ortbVersion:       "2.6",
			expectedReqNumber: 1,
			expectedUser:      expectedUserDefault,
			expectedDevice:    expectedDeviceDefault,
			expectedSource:    expectedSourceDefault,
		},
		{
			// remove user.eids, user.ext.data.*, user.data.*, user.{id, buyeruid, yob, gender}
			// and device-specific IDs
			name:              "transmit_ufpd_deny",
			req:               newBidRequest(),
			privacyConfig:     getTransmitUFPDActivityConfig("appnexus", false),
			expectedReqNumber: 1,
			expectedUser: openrtb2.User{
				ID:       "",
				BuyerUID: "",
				Yob:      0,
				Geo:      &openrtb2.Geo{Lat: ptrutil.ToPtr(123.456), Lon: ptrutil.ToPtr(11.278)},
				EIDs:     nil,
				Ext:      json.RawMessage(`{"test":2}`),
				Data:     nil,
			},
			expectUserScrub: true,
			expectedDevice: openrtb2.Device{
				UA:       deviceUA,
				Language: "EN",
				IP:       "132.173.230.74",
				DIDMD5:   "",
				IFA:      "",
				DIDSHA1:  "",
				DPIDMD5:  "",
				DPIDSHA1: "",
				MACMD5:   "",
				MACSHA1:  "",
				Geo:      &openrtb2.Geo{Lat: ptrutil.ToPtr(123.456), Lon: ptrutil.ToPtr(11.278)},
			},
			expectedSource: expectedSourceDefault,
		},
		{
			name:              "transmit_precise_geo_allowed",
			req:               newBidRequest(),
			privacyConfig:     getTransmitPreciseGeoActivityConfig("appnexus", true),
			ortbVersion:       "2.6",
			expectedReqNumber: 1,
			expectedUser:      expectedUserDefault,
			expectedDevice:    expectedDeviceDefault,
			expectedSource:    expectedSourceDefault,
		},
		{
			// round user's geographic location by rounding off IP address and lat/lng data.
			// this applies to both device.geo and user.geo
			name:              "transmit_precise_geo_deny",
			req:               newBidRequest(),
			privacyConfig:     getTransmitPreciseGeoActivityConfig("appnexus", false),
			ortbVersion:       "2.6",
			expectedReqNumber: 1,
			expectedUser: openrtb2.User{
				ID:       "our-id",
				BuyerUID: "their-id",
				Yob:      1982,
				Geo:      &openrtb2.Geo{Lat: ptrutil.ToPtr(123.46), Lon: ptrutil.ToPtr(11.28)},
				Gender:   "test",
				Ext:      json.RawMessage(`{"data": 1, "test": 2}`),
				EIDs: []openrtb2.EID{
					{Source: "eids-source"},
				},
				Data: []openrtb2.Data{{ID: "data-id"}},
			},
			expectedDevice: openrtb2.Device{
				UA:       deviceUA,
				IP:       "132.173.0.0",
				Language: "EN",
				DIDMD5:   "DIDMD5",
				IFA:      "IFA",
				DIDSHA1:  "DIDSHA1",
				DPIDMD5:  "DPIDMD5",
				DPIDSHA1: "DPIDSHA1",
				MACMD5:   "MACMD5",
				MACSHA1:  "MACSHA1",
				Geo:      &openrtb2.Geo{Lat: ptrutil.ToPtr(123.46), Lon: ptrutil.ToPtr(11.28)},
			},
			expectedSource: expectedSourceDefault,
		},
		{
			name:              "transmit_tid_allowed",
			req:               newBidRequest(),
			privacyConfig:     getTransmitTIDActivityConfig("appnexus", true),
			ortbVersion:       "2.6",
			expectedReqNumber: 1,
			expectedUser:      expectedUserDefault,
			expectedDevice:    expectedDeviceDefault,
			expectedSource:    expectedSourceDefault,
		},
		{
			// remove source.tid and imp.ext.tid
			name:              "transmit_tid_deny",
			req:               newBidRequest(),
			privacyConfig:     getTransmitTIDActivityConfig("appnexus", false),
			ortbVersion:       "2.6",
			expectedReqNumber: 1,
			expectedUser:      expectedUserDefault,
			expectedDevice:    expectedDeviceDefault,
			expectedSource: openrtb2.Source{
				TID: "",
			},
			expectedImpExt: json.RawMessage(`{"bidder": {"placementId": 1}}`),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			activities := privacy.NewActivityControl(&test.privacyConfig)
			auctionReq := AuctionRequest{
				BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: test.req},
				UserSyncs:         &emptyUsersync{},
				Activities:        activities,
				Account: config.Account{Privacy: config.AccountPrivacy{
					IPv6Config: config.IPv6{
						AnonKeepBits: 32,
					},
					IPv4Config: config.IPv4{
						AnonKeepBits: 16,
					},
				}},
				TCF2Config: gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			}

			metricsMock := metrics.MetricsEngineMock{}
			metricsMock.Mock.On("RecordAdapterBuyerUIDScrubbed", mock.Anything).Return()

			bidderToSyncerKey := map[string]string{}
			reqSplitter := &requestSplitter{
				bidderToSyncerKey: bidderToSyncerKey,
				me:                &metricsMock,
				hostSChainNode:    nil,
				bidderInfo:        config.BidderInfos{"appnexus": config.BidderInfo{OpenRTB: &config.OpenRTBInfo{Version: test.ortbVersion}}},
			}

			bidderRequests, _, errs := reqSplitter.cleanOpenRTBRequests(context.Background(), auctionReq, nil, gdpr.SignalNo, false, map[string]float64{})
			assert.Empty(t, errs)
			assert.Len(t, bidderRequests, test.expectedReqNumber)

			if test.expectedReqNumber == 1 {
				assert.Equal(t, &test.expectedUser, bidderRequests[0].BidRequest.User)
				assert.Equal(t, &test.expectedDevice, bidderRequests[0].BidRequest.Device)
				assert.Equal(t, &test.expectedSource, bidderRequests[0].BidRequest.Source)

				if len(test.expectedImpExt) > 0 {
					assert.JSONEq(t, string(test.expectedImpExt), string(bidderRequests[0].BidRequest.Imp[0].Ext))
				}
				if test.expectUserScrub {
					metricsMock.AssertCalled(t, "RecordAdapterBuyerUIDScrubbed", openrtb_ext.BidderAppnexus)
				} else {
					metricsMock.AssertNotCalled(t, "RecordAdapterBuyerUIDScrubbed", openrtb_ext.BidderAppnexus)
				}
			}
		})
	}
}

func buildDefaultActivityConfig(componentName string, allow bool) config.Activity {
	return config.Activity{
		Default: ptrutil.ToPtr(true),
		Rules: []config.ActivityRule{
			{
				Allow: allow,
				Condition: config.ActivityCondition{
					ComponentName: []string{componentName},
					ComponentType: []string{"bidder"},
				},
			},
		},
	}
}

func getFetchBidsActivityConfig(componentName string, allow bool) config.AccountPrivacy {
	return config.AccountPrivacy{
		AllowActivities: &config.AllowActivities{
			FetchBids: buildDefaultActivityConfig(componentName, allow),
		},
	}
}

func getTransmitUFPDActivityConfig(componentName string, allow bool) config.AccountPrivacy {
	return config.AccountPrivacy{
		AllowActivities: &config.AllowActivities{
			TransmitUserFPD: buildDefaultActivityConfig(componentName, allow),
		},
	}
}

func getTransmitPreciseGeoActivityConfig(componentName string, allow bool) config.AccountPrivacy {
	return config.AccountPrivacy{
		AllowActivities: &config.AllowActivities{
			TransmitPreciseGeo: buildDefaultActivityConfig(componentName, allow),
		},
	}
}

func getTransmitTIDActivityConfig(componentName string, allow bool) config.AccountPrivacy {
	return config.AccountPrivacy{
		AllowActivities: &config.AllowActivities{
			TransmitTids: buildDefaultActivityConfig(componentName, allow),
		},
	}
}

func TestApplyBidAdjustmentToFloor(t *testing.T) {
	type args struct {
		bidRequestWrapper    *openrtb_ext.RequestWrapper
		bidderName           string
		bidAdjustmentFactors map[string]float64
	}
	tests := []struct {
		name               string
		args               args
		expectedBidRequest *openrtb2.BidRequest
	}{
		{
			name: "bid_adjustment_factor_is_nil",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Imp: []openrtb2.Imp{{BidFloor: 100}, {BidFloor: 150}},
					},
				},
				bidderName:           "appnexus",
				bidAdjustmentFactors: nil,
			},
			expectedBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{BidFloor: 100}, {BidFloor: 150}},
			},
		},
		{
			name: "bid_adjustment_factor_is_empty",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Imp: []openrtb2.Imp{{BidFloor: 100}, {BidFloor: 150}},
					},
				},
				bidderName:           "appnexus",
				bidAdjustmentFactors: map[string]float64{},
			},
			expectedBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{BidFloor: 100}, {BidFloor: 150}},
			},
		},
		{
			name: "bid_adjustment_factor_not_present",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Imp: []openrtb2.Imp{{BidFloor: 100}, {BidFloor: 150}},
					},
				},
				bidderName:           "appnexus",
				bidAdjustmentFactors: map[string]float64{"pubmatic": 1.0},
			},
			expectedBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{BidFloor: 100}, {BidFloor: 150}},
			},
		},
		{
			name: "bid_adjustment_factor_present",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Imp: []openrtb2.Imp{{BidFloor: 100}, {BidFloor: 150}},
					},
				},
				bidderName:           "appnexus",
				bidAdjustmentFactors: map[string]float64{"pubmatic": 1.0, "appnexus": 0.75},
			},
			expectedBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{BidFloor: 133.33333333333334}, {BidFloor: 200}},
			},
		},
		{
			name: "bid_adjustment_factor_present_and_zero",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Imp: []openrtb2.Imp{{BidFloor: 100}, {BidFloor: 150}},
					},
				},
				bidderName:           "appnexus",
				bidAdjustmentFactors: map[string]float64{"pubmatic": 1.0, "appnexus": 0.0},
			},
			expectedBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{BidFloor: 100}, {BidFloor: 150}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyBidAdjustmentToFloor(tt.args.bidRequestWrapper, tt.args.bidderName, tt.args.bidAdjustmentFactors)
			assert.NoError(t, tt.args.bidRequestWrapper.RebuildRequest())
			assert.Equal(t, tt.expectedBidRequest, tt.args.bidRequestWrapper.BidRequest, tt.name)
		})
	}
}

func TestBuildRequestExtAlternateBidderCodes(t *testing.T) {
	type testInput struct {
		bidderNameRaw string
		accABC        *openrtb_ext.ExtAlternateBidderCodes
		reqABC        *openrtb_ext.ExtAlternateBidderCodes
	}
	testCases := []struct {
		desc     string
		in       testInput
		expected *openrtb_ext.ExtAlternateBidderCodes
	}{
		{
			desc:     "No biddername, nil reqABC and accABC",
			in:       testInput{},
			expected: nil,
		},
		{
			desc: "No biddername, non-nil reqABC",
			in: testInput{
				reqABC: &openrtb_ext.ExtAlternateBidderCodes{},
			},
			expected: &openrtb_ext.ExtAlternateBidderCodes{},
		},
		{
			desc: "No biddername, non-nil accABC",
			in: testInput{
				accABC: &openrtb_ext.ExtAlternateBidderCodes{},
			},
			expected: &openrtb_ext.ExtAlternateBidderCodes{},
		},
		{
			desc: "No biddername, non-nil reqABC nor accABC",
			in: testInput{
				reqABC: &openrtb_ext.ExtAlternateBidderCodes{},
				accABC: &openrtb_ext.ExtAlternateBidderCodes{},
			},
			expected: &openrtb_ext.ExtAlternateBidderCodes{},
		},
		{
			desc: "non-nil reqABC",
			in: testInput{
				bidderNameRaw: "pubmatic",
				reqABC:        &openrtb_ext.ExtAlternateBidderCodes{},
			},
			expected: &openrtb_ext.ExtAlternateBidderCodes{},
		},
		{
			desc: "non-nil accABC",
			in: testInput{
				bidderNameRaw: "pubmatic",
				accABC:        &openrtb_ext.ExtAlternateBidderCodes{},
			},
			expected: &openrtb_ext.ExtAlternateBidderCodes{},
		},
		{
			desc: "both reqABC and accABC enabled and bidder matches elements in accABC but reqABC comes first",
			in: testInput{
				bidderNameRaw: "PUBmatic",
				reqABC: &openrtb_ext.ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						"appnexus": {
							AllowedBidderCodes: []string{"pubCode1"},
						},
					},
				},
				accABC: &openrtb_ext.ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						"PubMatic": {
							AllowedBidderCodes: []string{"pubCode2"},
						},
					},
				},
			},
			expected: &openrtb_ext.ExtAlternateBidderCodes{Enabled: true},
		},
		{
			desc: "both reqABC and accABC enabled and bidder matches elements in both but we prioritize reqABC",
			in: testInput{
				bidderNameRaw: "pubmatic",
				reqABC: &openrtb_ext.ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						"PubMatic": {
							AllowedBidderCodes: []string{"pubCode"},
						},
					},
				},
				accABC: &openrtb_ext.ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						"appnexus": {
							AllowedBidderCodes: []string{"anxsCode"},
						},
					},
				},
			},
			expected: &openrtb_ext.ExtAlternateBidderCodes{
				Enabled: true,
				Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
					"pubmatic": {
						AllowedBidderCodes: []string{"pubCode"},
					},
				},
			},
		},
		{
			desc: "nil reqABC non-nil accABC enabled and bidder matches elements in accABC",
			in: testInput{
				bidderNameRaw: "APPnexus",
				accABC: &openrtb_ext.ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						"appnexus": {
							AllowedBidderCodes: []string{"anxsCode"},
						},
					},
				},
			},
			expected: &openrtb_ext.ExtAlternateBidderCodes{
				Enabled: true,
				Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
					"APPnexus": {
						AllowedBidderCodes: []string{"anxsCode"},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			alternateBidderCodes := buildRequestExtAlternateBidderCodes(tc.in.bidderNameRaw, tc.in.accABC, tc.in.reqABC)
			assert.Equal(t, tc.expected, alternateBidderCodes)
		})
	}
}

func TestCopyExtAlternateBidderCodes(t *testing.T) {
	type testInput struct {
		bidder               string
		alternateBidderCodes *openrtb_ext.ExtAlternateBidderCodes
	}
	testCases := []struct {
		desc     string
		in       testInput
		expected *openrtb_ext.ExtAlternateBidderCodes
	}{
		{
			desc:     "pass a nil alternateBidderCodes argument, expect nil output",
			in:       testInput{},
			expected: nil,
		},
		{
			desc: "non-nil alternateBidderCodes argument but bidder doesn't match",
			in: testInput{
				alternateBidderCodes: &openrtb_ext.ExtAlternateBidderCodes{
					Enabled: true,
				},
			},
			expected: &openrtb_ext.ExtAlternateBidderCodes{
				Enabled: true,
			},
		},
		{
			desc: "non-nil alternateBidderCodes argument bidder is identical to one element in map",
			in: testInput{
				bidder: "appnexus",
				alternateBidderCodes: &openrtb_ext.ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						"appnexus": {
							AllowedBidderCodes: []string{"adnxs"},
						},
					},
				},
			},
			expected: &openrtb_ext.ExtAlternateBidderCodes{
				Enabled: true,
				Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
					"appnexus": {
						AllowedBidderCodes: []string{"adnxs"},
					},
				},
			},
		},
		{
			desc: "case insensitive match, keep bidder casing in output",
			in: testInput{
				bidder: "AppNexus",
				alternateBidderCodes: &openrtb_ext.ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						"appnexus": {
							AllowedBidderCodes: []string{"adnxs"},
						},
					},
				},
			},
			expected: &openrtb_ext.ExtAlternateBidderCodes{
				Enabled: true,
				Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
					"AppNexus": {
						AllowedBidderCodes: []string{"adnxs"},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			alternateBidderCodes := copyExtAlternateBidderCodes(tc.in.bidder, tc.in.alternateBidderCodes)
			assert.Equal(t, tc.expected, alternateBidderCodes)
		})
	}
}

func TestRemoveImpsWithStoredResponses(t *testing.T) {
	bidRespId1 := json.RawMessage(`{"id": "resp_id1"}`)
	testCases := []struct {
		description        string
		req                *openrtb_ext.RequestWrapper
		storedBidResponses map[string]json.RawMessage
		expectedImps       []openrtb2.Imp
	}{
		{
			description: "request with imps and stored bid response for this imp",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp-id1"},
					},
				},
			},
			storedBidResponses: map[string]json.RawMessage{
				"imp-id1": bidRespId1,
			},
			expectedImps: nil,
		},
		{
			description: "request with imps and stored bid response for one of these imp",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp-id1"},
						{ID: "imp-id2"},
					},
				},
			},
			storedBidResponses: map[string]json.RawMessage{
				"imp-id1": bidRespId1,
			},
			expectedImps: []openrtb2.Imp{
				{
					ID: "imp-id2",
				},
			},
		},
		{
			description: "request with imps and stored bid response for both of these imp",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp-id1"},
						{ID: "imp-id2"},
					},
				},
			},
			storedBidResponses: map[string]json.RawMessage{
				"imp-id1": bidRespId1,
				"imp-id2": bidRespId1,
			},
			expectedImps: nil,
		},
		{
			description: "request with imps and no stored bid responses",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp-id1"},
						{ID: "imp-id2"},
					},
				},
			},
			storedBidResponses: nil,

			expectedImps: []openrtb2.Imp{
				{ID: "imp-id1"},
				{ID: "imp-id2"},
			},
		},
	}
	for _, testCase := range testCases {
		request := testCase.req
		removeImpsWithStoredResponses(request, testCase.storedBidResponses)
		assert.NoError(t, request.RebuildRequest())
		assert.Equal(t, testCase.expectedImps, request.Imp, "incorrect Impressions for testCase %s", testCase.description)
	}
}

func TestExtractAndCleanBuyerUIDs(t *testing.T) {
	tests := []struct {
		name              string
		user              *openrtb2.User
		expectedBuyerUIDs map[string]string
		expectedUser      *openrtb2.User
		expectError       bool
	}{
		{
			name:              "user_is_nil",
			user:              nil,
			expectedBuyerUIDs: nil,
			expectedUser:      nil,
			expectError:       false,
		},
		{
			name: "user.ext_is_nil",
			user: &openrtb2.User{
				Ext: nil,
			},
			expectedBuyerUIDs: nil,
			expectedUser: &openrtb2.User{
				Ext: nil,
			},
			expectError: false,
		},
		{
			name: "user.ext_malformed",
			user: &openrtb2.User{
				Ext: json.RawMessage(`{"prebid":}`),
			},
			expectedBuyerUIDs: nil,
			expectedUser: &openrtb2.User{
				Ext: json.RawMessage(`{"prebid":}`),
			},
			expectError: true,
		},
		{
			name: "user.ext.prebid_is_nil",
			user: &openrtb2.User{
				Ext: json.RawMessage(`{"prebid":null}`),
			},
			expectedBuyerUIDs: nil,
			expectedUser: &openrtb2.User{
				Ext: nil,
			},
			expectError: false,
		},
		{
			name: "user.ext.prebid.buyeruids_is_nil",
			user: &openrtb2.User{
				Ext: json.RawMessage(`{"prebid":{"buyeruids": null}}`),
			},
			expectedBuyerUIDs: nil,
			expectedUser: &openrtb2.User{
				Ext: nil,
			},
			expectError: false,
		},
		{
			name: "user.ext.prebid.buyeruids_has_one",
			user: &openrtb2.User{
				Ext: json.RawMessage(`{"prebid":{"buyeruids": {"appnexus":"a"}}}`),
			},
			expectedBuyerUIDs: map[string]string{"appnexus": "a"},
			expectedUser: &openrtb2.User{
				Ext: nil,
			},
			expectError: false,
		},
		{
			name: "user.ext.prebid.buyeruids_has_many",
			user: &openrtb2.User{
				Ext: json.RawMessage(`{"prebid":{"buyeruids": {"appnexus":"a", "pubmatic":"b"}}}`),
			},
			expectedBuyerUIDs: map[string]string{"appnexus": "a", "pubmatic": "b"},
			expectedUser: &openrtb2.User{
				Ext: nil,
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: test.user,
				},
			}

			result, err := extractAndCleanBuyerUIDs(&req)
			if test.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.NoError(t, req.RebuildRequest())

			assert.Equal(t, req.User, test.expectedUser)
			assert.Equal(t, test.expectedBuyerUIDs, result)
		})
	}
}

func intPtr(i int) *int {
	return &i
}
