package ortb

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestValidateImpExt(t *testing.T) {
	type testCase struct {
		description         string
		impExt              json.RawMessage
		cfg                 ValidationConfig
		paramValidatorError error
		expectedImpExt      string
		expectedErrs        []error
	}
	testGroups := []struct {
		description string
		testCases   []testCase
	}{
		{
			"Empty",
			[]testCase{
				{
					description:    "Empty",
					impExt:         nil,
					expectedImpExt: "",
					expectedErrs:   []error{errors.New("request.imp[0].ext is required")},
				},
			},
		},
		{
			"Unknown bidder tests",
			[]testCase{
				{
					description:    "Unknown Bidder + Empty Prebid Bidder",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{}}, "unknownbidder":{"placement_id":555}}`),
					expectedImpExt: `{"prebid":{"bidder":{}}, "unknownbidder":{"placement_id":555}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder")},
				},
				{
					description:    "Unknown Bidder only",
					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555}}`),
					expectedImpExt: `{"unknownbidder":{"placement_id":555}}`,
					expectedErrs: []error{&errortypes.Warning{Message: ("request.imp[0].ext contains unknown bidder: 'unknownbidder', ignoring")},
						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder")},
				},
				{
					description:    "Unknown Prebid Ext Bidder only",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
				{
					description:    "Unknown Prebid Ext Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
				{
					description:    "Unknown Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555} ,"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"unknownbidder":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs: []error{&errortypes.Warning{Message: ("request.imp[0].ext contains unknown bidder: 'unknownbidder', ignoring")},
						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
					},
				},
				{
					description:    "Unknown Bidder + Disabled Bidder",
					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555},"disabledbidder":{"foo":"bar"}}`),
					expectedImpExt: `{"unknownbidder":{"placement_id":555},"disabledbidder":{"foo":"bar"}}`,
					expectedErrs: []error{&errortypes.Warning{Message: ("request.imp[0].ext contains unknown bidder: 'unknownbidder', ignoring")},
						&errortypes.BidderTemporarilyDisabled{Message: ("The bidder 'disabledbidder' has been disabled.")},
						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
					},
				},
				{
					description:    "Unknown Bidder + Disabled Prebid Ext Bidder",
					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555},"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`),
					expectedImpExt: `{"unknownbidder":{"placement_id":555},"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`,
					expectedErrs: []error{&errortypes.BidderTemporarilyDisabled{Message: ("The bidder 'disabledbidder' has been disabled.")},
						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
					},
				},
			},
		},
		{
			"Disabled bidder tests",
			[]testCase{
				{
					description:    "Disabled Bidder",
					impExt:         json.RawMessage(`{"disabledbidder":{"foo":"bar"}}`),
					expectedImpExt: `{"disabledbidder":{"foo":"bar"}}`,
					expectedErrs: []error{
						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
					},
					// if only bidder(s) found in request.imp[x].ext.{biddername} or request.imp[x].ext.prebid.bidder.{biddername} are disabled, return error
				},
				{
					description:    "Disabled Prebid Ext Bidder",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`,
					expectedErrs: []error{
						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
					},
				},
				{
					description:    "Disabled Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"disabledbidder":{"foo":"bar"},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"disabledbidder":{"foo":"bar"},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs: []error{
						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
					},
				},
				{
					description:    "Disabled Prebid Ext Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs: []error{
						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
					},
				},
			},
		},
		{
			"First Party only",
			[]testCase{
				{
					description:    "First Party Data Context",
					impExt:         json.RawMessage(`{"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs: []error{
						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
					},
				},
			},
		},
		{
			"Valid bidder tests",
			[]testCase{
				{
					description:    "Valid bidder root ext",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}}`,
					expectedErrs:   []error{},
				},
				{
					// Since prebid.bidder object is present - bidders expected to be within it
					// So even though appnexus is a valid bidder - it is ignored and considered to be an arbitrary field
					// If there was no prebid.bidder then appnexus would have been considered a bidder.
					description:    "Valid bidder root ext + Empty Prebid Bidder",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{}}, "appnexus":{"placement_id":555}}`),
					expectedImpExt: `{"prebid":{"bidder":{}}, "appnexus":{"placement_id":555}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder")},
				},
				{
					description:    "Valid bidder in prebid field",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}}`,
					expectedErrs:   []error{},
				},
				{
					description:    "Valid Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{},
				},
				{
					description:    "Valid Prebid Ext Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555}}} ,"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{},
				},
				{
					description:    "Valid Bidder + Unknown Bidder",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"unknownbidder":{"placement_id":555}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}, "unknownbidder":{"placement_id":555}}`,
					expectedErrs:   []error{&errortypes.Warning{Message: ("request.imp[0].ext contains unknown bidder: 'unknownbidder', ignoring")}},
				},
				{
					description:    "Valid Bidder + Disabled Bidder",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}}`,
					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
				},
				{
					description:    "Valid Bidder + Disabled Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
				},
				{
					description:    "Valid Bidder + Disabled Bidder + Unknown Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"unknownbidder":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}, "unknownbidder":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs: []error{&errortypes.Warning{Message: ("request.imp[0].ext contains unknown bidder: 'unknownbidder', ignoring")},
						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
					},
				},
				{
					description:    "Valid Prebid Ext Bidder + Disabled Prebid Bidder Ext",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"}}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id": 555}}}}`,
					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
				},
				{
					description:    "Valid Prebid Ext Bidder + Arbitrary Key Ext",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id": 555}}},"arbitraryKey":{"placement_id":555}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id": 555}}},"arbitraryKey":{"placement_id":555}}`,
					expectedErrs:   []error{},
				},
				{
					description:    "Valid Prebid Ext Bidder + Disabled Ext Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"}}},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id": 555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
				},
				{
					description:    "Valid Prebid Ext Bidder + Disabled Prebid Ext Bidder + Unknown Prebid Ext + First Party Data Context",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"unknownbidder":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"unknownbidder":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
			},
		},
		{
			"Config tests",
			[]testCase{
				{
					description: "Invalid Params",
					impExt:      json.RawMessage(`{"appnexus":{"placement_id_wrong_format":[]}}`),
					cfg: ValidationConfig{
						SkipBidderParams: false,
					},
					paramValidatorError: errors.New("params error"),
					expectedImpExt:      `{"appnexus":{"placement_id_wrong_format":[]}}`,
					expectedErrs:        []error{errors.New("request.imp[0].ext.prebid.bidder.appnexus failed validation.\nparams error")},
				},
				{
					description: "Invalid Params - Skip Params Validation",
					impExt:      json.RawMessage(`{"appnexus":{"placement_id_wrong_format":[]}}`),
					cfg: ValidationConfig{
						SkipBidderParams: true,
					},
					paramValidatorError: errors.New("params error"),
					expectedImpExt:      `{"prebid":{"bidder":{"appnexus":{"placement_id_wrong_format":[]}}}}`,
					expectedErrs:        []error{},
				},
			},
		},
	}

	for _, group := range testGroups {
		for _, test := range group.testCases {
			t.Run(test.description, func(t *testing.T) {
				imp := &openrtb2.Imp{Ext: test.impExt}
				impWrapper := &openrtb_ext.ImpWrapper{Imp: imp}

				disabledBidders := map[string]string{"disabledbidder": "The bidder 'disabledbidder' has been disabled."}
				rv := standardRequestValidator{
					bidderMap:       openrtb_ext.BuildBidderMap(),
					disabledBidders: disabledBidders,
					paramsValidator: mockBidderParamValidator{
						Error: test.paramValidatorError,
					},
				}
				errs := rv.validateImpExt(impWrapper, test.cfg, nil, 0, false, nil)

				assert.NoError(t, impWrapper.RebuildImp(), test.description+":rebuild_imp")

				if len(test.expectedImpExt) > 0 {
					assert.JSONEq(t, test.expectedImpExt, string(imp.Ext), "imp.ext JSON does not match expected. Test: %s. %s\n", group.description, test.description)
				} else {
					assert.Empty(t, imp.Ext, "imp.ext expected to be empty but was: %s. Test: %s. %s\n", string(imp.Ext), group.description, test.description)
				}
				assert.ElementsMatch(t, test.expectedErrs, errs, "errs slice does not match expected. Test: %s. %s\n", group.description, test.description)
			})
		}
	}
}

type mockBidderParamValidator struct {
	Error error
}

func (v mockBidderParamValidator) Validate(name openrtb_ext.BidderName, ext json.RawMessage) error {
	return v.Error
}
func (v mockBidderParamValidator) Schema(name openrtb_ext.BidderName) string { return "" }
