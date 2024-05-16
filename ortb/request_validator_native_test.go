package ortb

import (
	"testing"

	"github.com/prebid/openrtb/v20/native1"
	nativeRequests "github.com/prebid/openrtb/v20/native1/request"
	"github.com/stretchr/testify/assert"
)

func TestValidateNativeContextTypes(t *testing.T) {
	impIndex := 4

	testCases := []struct {
		description      string
		givenContextType native1.ContextType
		givenSubType     native1.ContextSubType
		expectedError    string
	}{
		{
			description:      "No Types Specified",
			givenContextType: 0,
			givenSubType:     0,
			expectedError:    "",
		},
		{
			description:      "All Types Exchange Specific",
			givenContextType: 500,
			givenSubType:     500,
			expectedError:    "",
		},
		{
			description:      "Context Type Known Value - Sub Type Unspecified",
			givenContextType: 1,
			givenSubType:     0,
			expectedError:    "",
		},
		{
			description:      "Context Type Negative",
			givenContextType: -1,
			givenSubType:     0,
			expectedError:    "request.imp[4].native.request.context is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Context Type Just Above Range",
			givenContextType: 4, // Range is currently 1-3
			givenSubType:     0,
			expectedError:    "request.imp[4].native.request.context is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Sub Type Negative",
			givenContextType: 1,
			givenSubType:     -1,
			expectedError:    "request.imp[4].native.request.contextsubtype value can't be less than 0. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Content - Sub Type Just Below Range",
			givenContextType: 1, // Content constant
			givenSubType:     9, // Content range is currently 10-15
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Content - Sub Type In Range",
			givenContextType: 1,  // Content constant
			givenSubType:     10, // Content range is currently 10-15
			expectedError:    "",
		},
		{
			description:      "Content - Sub Type In Range - Context Type Exchange Specific Boundary",
			givenContextType: 500,
			givenSubType:     10, // Content range is currently 10-15
			expectedError:    "",
		},
		{
			description:      "Content - Sub Type In Range - Context Type Exchange Specific Boundary + 1",
			givenContextType: 501,
			givenSubType:     10, // Content range is currently 10-15
			expectedError:    "",
		},
		{
			description:      "Content - Sub Type Just Above Range",
			givenContextType: 1,  // Content constant
			givenSubType:     16, // Content range is currently 10-15
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Content - Sub Type Exchange Specific Boundary",
			givenContextType: 1, // Content constant
			givenSubType:     500,
			expectedError:    "",
		},
		{
			description:      "Content - Sub Type Exchange Specific Boundary + 1",
			givenContextType: 1, // Content constant
			givenSubType:     501,
			expectedError:    "",
		},
		{
			description:      "Content - Invalid Context Type",
			givenContextType: 2,  // Not content constant
			givenSubType:     10, // Content range is currently 10-15
			expectedError:    "request.imp[4].native.request.context is 2, but contextsubtype is 10. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Social - Sub Type Just Below Range",
			givenContextType: 2,  // Social constant
			givenSubType:     19, // Social range is currently 20-22
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Social - Sub Type In Range",
			givenContextType: 2,  // Social constant
			givenSubType:     20, // Social range is currently 20-22
			expectedError:    "",
		},
		{
			description:      "Social - Sub Type In Range - Context Type Exchange Specific Boundary",
			givenContextType: 500,
			givenSubType:     20, // Social range is currently 20-22
			expectedError:    "",
		},
		{
			description:      "Social - Sub Type In Range - Context Type Exchange Specific Boundary + 1",
			givenContextType: 501,
			givenSubType:     20, // Social range is currently 20-22
			expectedError:    "",
		},
		{
			description:      "Social - Sub Type Just Above Range",
			givenContextType: 2,  // Social constant
			givenSubType:     23, // Social range is currently 20-22
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Social - Sub Type Exchange Specific Boundary",
			givenContextType: 2, // Social constant
			givenSubType:     500,
			expectedError:    "",
		},
		{
			description:      "Social - Sub Type Exchange Specific Boundary + 1",
			givenContextType: 2, // Social constant
			givenSubType:     501,
			expectedError:    "",
		},
		{
			description:      "Social - Invalid Context Type",
			givenContextType: 3,  // Not social constant
			givenSubType:     20, // Social range is currently 20-22
			expectedError:    "request.imp[4].native.request.context is 3, but contextsubtype is 20. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Product - Sub Type Just Below Range",
			givenContextType: 3,  // Product constant
			givenSubType:     29, // Product range is currently 30-32
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Product - Sub Type In Range",
			givenContextType: 3,  // Product constant
			givenSubType:     30, // Product range is currently 30-32
			expectedError:    "",
		},
		{
			description:      "Product - Sub Type In Range - Context Type Exchange Specific Boundary",
			givenContextType: 500,
			givenSubType:     30, // Product range is currently 30-32
			expectedError:    "",
		},
		{
			description:      "Product - Sub Type In Range - Context Type Exchange Specific Boundary + 1",
			givenContextType: 501,
			givenSubType:     30, // Product range is currently 30-32
			expectedError:    "",
		},
		{
			description:      "Product - Sub Type Just Above Range",
			givenContextType: 3,  // Product constant
			givenSubType:     33, // Product range is currently 30-32
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Product - Sub Type Exchange Specific Boundary",
			givenContextType: 3, // Product constant
			givenSubType:     500,
			expectedError:    "",
		},
		{
			description:      "Product - Sub Type Exchange Specific Boundary + 1",
			givenContextType: 3, // Product constant
			givenSubType:     501,
			expectedError:    "",
		},
		{
			description:      "Product - Invalid Context Type",
			givenContextType: 1,  // Not product constant
			givenSubType:     30, // Product range is currently 30-32
			expectedError:    "request.imp[4].native.request.context is 1, but contextsubtype is 30. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
	}

	for _, test := range testCases {
		err := validateNativeContextTypes(test.givenContextType, test.givenSubType, impIndex)
		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}
	}
}

func TestValidateNativePlacementType(t *testing.T) {
	impIndex := 4

	testCases := []struct {
		description        string
		givenPlacementType native1.PlacementType
		expectedError      string
	}{
		{
			description:        "Not Specified",
			givenPlacementType: 0,
			expectedError:      "",
		},
		{
			description:        "Known Value",
			givenPlacementType: 1, // Range is currently 1-4
			expectedError:      "",
		},
		{
			description:        "Exchange Specific - Boundary",
			givenPlacementType: 500,
			expectedError:      "",
		},
		{
			description:        "Exchange Specific - Boundary + 1",
			givenPlacementType: 501,
			expectedError:      "",
		},
		{
			description:        "Negative",
			givenPlacementType: -1,
			expectedError:      "request.imp[4].native.request.plcmttype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40",
		},
		{
			description:        "Just Above Range",
			givenPlacementType: 5, // Range is currently 1-4
			expectedError:      "request.imp[4].native.request.plcmttype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40",
		},
	}

	for _, test := range testCases {
		err := validateNativePlacementType(test.givenPlacementType, impIndex)
		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}
	}
}

func TestValidateNativeEventTracker(t *testing.T) {
	impIndex := 4
	eventIndex := 8

	testCases := []struct {
		description   string
		givenEvent    nativeRequests.EventTracker
		expectedError string
	}{
		{
			description: "Valid",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{1},
			},
			expectedError: "",
		},
		{
			description: "Event - Exchange Specific - Boundary",
			givenEvent: nativeRequests.EventTracker{
				Event:   500,
				Methods: []native1.EventTrackingMethod{1},
			},
			expectedError: "",
		},
		{
			description: "Event - Exchange Specific - Boundary + 1",
			givenEvent: nativeRequests.EventTracker{
				Event:   501,
				Methods: []native1.EventTrackingMethod{1},
			},
			expectedError: "",
		},
		{
			description: "Event - Negative",
			givenEvent: nativeRequests.EventTracker{
				Event:   -1,
				Methods: []native1.EventTrackingMethod{1},
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].event is invalid. See section 7.6: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
		{
			description: "Event - Just Above Range",
			givenEvent: nativeRequests.EventTracker{
				Event:   5, // Range is currently 1-4
				Methods: []native1.EventTrackingMethod{1},
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].event is invalid. See section 7.6: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
		{
			description: "Methods - Many Valid",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{1, 2},
			},
			expectedError: "",
		},
		{
			description: "Methods - Empty",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{},
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].method is required. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
		{
			description: "Methods - Exchange Specific - Boundary",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{500},
			},
			expectedError: "",
		},
		{
			description: "Methods - Exchange Specific - Boundary + 1",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{501},
			},
			expectedError: "",
		},
		{
			description: "Methods - Negative",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{-1},
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].methods[0] is invalid. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
		{
			description: "Methods - Just Above Range",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{3}, // Known values are currently 1-2
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].methods[0] is invalid. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
		{
			description: "Methods - Mixed Valid + Invalid",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{1, -1},
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].methods[1] is invalid. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
	}

	for _, test := range testCases {
		err := validateNativeEventTracker(test.givenEvent, impIndex, eventIndex)
		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}
	}
}

func TestValidateNativeAssetData(t *testing.T) {
	impIndex := 4
	assetIndex := 8

	testCases := []struct {
		description   string
		givenData     nativeRequests.Data
		expectedError string
	}{
		{
			description:   "Valid",
			givenData:     nativeRequests.Data{Type: 1},
			expectedError: "",
		},
		{
			description:   "Exchange Specific - Boundary",
			givenData:     nativeRequests.Data{Type: 500},
			expectedError: "",
		},
		{
			description:   "Exchange Specific - Boundary + 1",
			givenData:     nativeRequests.Data{Type: 501},
			expectedError: "",
		},
		{
			description:   "Not Specified",
			givenData:     nativeRequests.Data{},
			expectedError: "request.imp[4].native.request.assets[8].data.type is invalid. See section 7.4: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40",
		},
		{
			description:   "Negative",
			givenData:     nativeRequests.Data{Type: -1},
			expectedError: "request.imp[4].native.request.assets[8].data.type is invalid. See section 7.4: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40",
		},
		{
			description:   "Just Above Range",
			givenData:     nativeRequests.Data{Type: 13}, // Range is currently 1-12
			expectedError: "request.imp[4].native.request.assets[8].data.type is invalid. See section 7.4: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40",
		},
	}

	for _, test := range testCases {
		err := validateNativeAssetData(&test.givenData, impIndex, assetIndex)
		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}
	}
}

// func TestValidateImpExt(t *testing.T) {
// 	type testCase struct {
// 		description    string
// 		impExt         json.RawMessage
// 		expectedImpExt string
// 		expectedErrs   []error
// 	}
// 	testGroups := []struct {
// 		description string
// 		testCases   []testCase
// 	}{
// 		{
// 			"Empty",
// 			[]testCase{
// 				{
// 					description:    "Empty",
// 					impExt:         nil,
// 					expectedImpExt: "",
// 					expectedErrs:   []error{errors.New("request.imp[0].ext is required")},
// 				},
// 			},
// 		},
// 		{
// 			"Unknown bidder tests",
// 			[]testCase{
// 				{
// 					description:    "Unknown Bidder only",
// 					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555}}`),
// 					expectedImpExt: `{"unknownbidder":{"placement_id":555}}`,
// 					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
// 				},
// 				{
// 					description:    "Unknown Prebid Ext Bidder only",
// 					impExt:         json.RawMessage(`{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}}}`,
// 					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
// 				},
// 				{
// 					description:    "Unknown Prebid Ext Bidder + First Party Data Context",
// 					impExt:         json.RawMessage(`{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
// 					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
// 				},
// 				{
// 					description:    "Unknown Bidder + First Party Data Context",
// 					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555} ,"context":{"data":{"keywords":"prebid server example"}}}`),
// 					expectedImpExt: `{"unknownbidder":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`,
// 					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
// 				},
// 				{
// 					description:    "Unknown Bidder + Disabled Bidder",
// 					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555},"disabledbidder":{"foo":"bar"}}`),
// 					expectedImpExt: `{"unknownbidder":{"placement_id":555},"disabledbidder":{"foo":"bar"}}`,
// 					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
// 				},
// 				{
// 					description:    "Unknown Bidder + Disabled Prebid Ext Bidder",
// 					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555},"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`),
// 					expectedImpExt: `{"unknownbidder":{"placement_id":555},"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`,
// 					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
// 				},
// 			},
// 		},
// 		{
// 			"Disabled bidder tests",
// 			[]testCase{
// 				{
// 					description:    "Disabled Bidder",
// 					impExt:         json.RawMessage(`{"disabledbidder":{"foo":"bar"}}`),
// 					expectedImpExt: `{"disabledbidder":{"foo":"bar"}}`,
// 					expectedErrs: []error{
// 						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
// 						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
// 					},
// 					// if only bidder(s) found in request.imp[x].ext.{biddername} or request.imp[x].ext.prebid.bidder.{biddername} are disabled, return error
// 				},
// 				{
// 					description:    "Disabled Prebid Ext Bidder",
// 					impExt:         json.RawMessage(`{"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`,
// 					expectedErrs: []error{
// 						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
// 						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
// 					},
// 				},
// 				{
// 					description:    "Disabled Bidder + First Party Data Context",
// 					impExt:         json.RawMessage(`{"disabledbidder":{"foo":"bar"},"context":{"data":{"keywords":"prebid server example"}}}`),
// 					expectedImpExt: `{"disabledbidder":{"foo":"bar"},"context":{"data":{"keywords":"prebid server example"}}}`,
// 					expectedErrs: []error{
// 						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
// 						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
// 					},
// 				},
// 				{
// 					description:    "Disabled Prebid Ext Bidder + First Party Data Context",
// 					impExt:         json.RawMessage(`{"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}},"context":{"data":{"keywords":"prebid server example"}}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}},"context":{"data":{"keywords":"prebid server example"}}}`,
// 					expectedErrs: []error{
// 						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
// 						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
// 					},
// 				},
// 			},
// 		},
// 		{
// 			"First Party only",
// 			[]testCase{
// 				{
// 					description:    "First Party Data Context",
// 					impExt:         json.RawMessage(`{"context":{"data":{"keywords":"prebid server example"}}}`),
// 					expectedImpExt: `{"context":{"data":{"keywords":"prebid server example"}}}`,
// 					expectedErrs: []error{
// 						errors.New("request.imp[0].ext.prebid.bidder must contain at least one bidder"),
// 					},
// 				},
// 			},
// 		},
// 		{
// 			"Valid bidder tests",
// 			[]testCase{
// 				{
// 					description:    "Valid bidder root ext",
// 					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}}`,
// 					expectedErrs:   []error{},
// 				},
// 				{
// 					description:    "Valid bidder in prebid field",
// 					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}}`,
// 					expectedErrs:   []error{},
// 				},
// 				{
// 					description:    "Valid Bidder + First Party Data Context",
// 					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
// 					expectedErrs:   []error{},
// 				},
// 				{
// 					description:    "Valid Prebid Ext Bidder + First Party Data Context",
// 					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555}}} ,"context":{"data":{"keywords":"prebid server example"}}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
// 					expectedErrs:   []error{},
// 				},
// 				{
// 					description:    "Valid Bidder + Unknown Bidder",
// 					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"unknownbidder":{"placement_id":555}}`),
// 					expectedImpExt: `{"appnexus":{"placement_id":555},"unknownbidder":{"placement_id":555}}`,
// 					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
// 				},
// 				{
// 					description:    "Valid Bidder + Disabled Bidder",
// 					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}}`,
// 					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
// 				},
// 				{
// 					description:    "Valid Bidder + Disabled Bidder + First Party Data Context",
// 					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"context":{"data":{"keywords":"prebid server example"}}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
// 					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
// 				},
// 				{
// 					description:    "Valid Bidder + Disabled Bidder + Unknown Bidder + First Party Data Context",
// 					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"unknownbidder":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`),
// 					expectedImpExt: `{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"unknownbidder":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`,
// 					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
// 				},
// 				{
// 					description:    "Valid Prebid Ext Bidder + Disabled Bidder Ext",
// 					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"}}}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id": 555}}}}`,
// 					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
// 				},
// 				{
// 					description:    "Valid Prebid Ext Bidder + Disabled Ext Bidder + First Party Data Context",
// 					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"}}},"context":{"data":{"keywords":"prebid server example"}}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id": 555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
// 					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
// 				},
// 				{
// 					description:    "Valid Prebid Ext Bidder + Disabled Ext Bidder + Unknown Ext + First Party Data Context",
// 					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"unknownbidder":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`),
// 					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"unknownbidder":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
// 					expectedErrs:   []error{errors.New("request.imp[0].ext.prebid.bidder contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
// 				},
// 			},
// 		},
// 	}

// 	deps := &endpointDeps{
// 		fakeUUIDGenerator{},
// 		&nobidExchange{},
// 		mockBidderParamValidator{},
// 		&mockStoredReqFetcher{},
// 		empty_fetcher.EmptyFetcher{},
// 		empty_fetcher.EmptyFetcher{},
// 		&config.Configuration{MaxRequestSize: int64(8096)},
// 		&metricsConfig.NilMetricsEngine{},
// 		analyticsBuild.New(&config.Analytics{}),
// 		map[string]string{"disabledbidder": "The bidder 'disabledbidder' has been disabled."},
// 		false,
// 		[]byte{},
// 		openrtb_ext.BuildBidderMap(),
// 		nil,
// 		nil,
// 		hardcodedResponseIPValidator{response: true},
// 		empty_fetcher.EmptyFetcher{},
// 		hooks.EmptyPlanBuilder{},
// 		nil,
// 		openrtb_ext.NormalizeBidderName,
// 	}

// 	for _, group := range testGroups {
// 		for _, test := range group.testCases {
// 			t.Run(test.description, func(t *testing.T) {
// 				imp := &openrtb2.Imp{Ext: test.impExt}
// 				impWrapper := &openrtb_ext.ImpWrapper{Imp: imp}

// 				errs := deps.validateImpExt(impWrapper, nil, 0, false, nil)

// 				assert.NoError(t, impWrapper.RebuildImp(), test.description+":rebuild_imp")

// 				if len(test.expectedImpExt) > 0 {
// 					assert.JSONEq(t, test.expectedImpExt, string(imp.Ext), "imp.ext JSON does not match expected. Test: %s. %s\n", group.description, test.description)
// 				} else {
// 					assert.Empty(t, imp.Ext, "imp.ext expected to be empty but was: %s. Test: %s. %s\n", string(imp.Ext), group.description, test.description)
// 				}
// 				assert.Equal(t, test.expectedErrs, errs, "errs slice does not match expected. Test: %s. %s\n", group.description, test.description)
// 			})
// 		}
// 	}
// }
