package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestReadDealTiersFromImp(t *testing.T) {
	testCases := []struct {
		description    string
		impExt         json.RawMessage
		expectedResult DealTierBidderMap
		expectedError  string
	}{
		{
			description:    "Nil",
			impExt:         nil,
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "None",
			impExt:         json.RawMessage(``),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "Empty Object",
			impExt:         json.RawMessage(`{}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "imp.ext - with other params",
			impExt:         json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "anyPrefix"}, "placementId": 12345}}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "anyPrefix", MinDealTier: 5}},
		},
		{
			description:    "imp.ext - multiple",
			impExt:         json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "appnexusPrefix"}, "placementId": 12345}, "rubicon": {"dealTier": {"minDealTier": 8, "prefix": "rubiconPrefix"}, "placementId": 12345}}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "appnexusPrefix", MinDealTier: 5}, BidderRubicon: {Prefix: "rubiconPrefix", MinDealTier: 8}},
		},
		{
			description:    "imp.ext - no deal tier",
			impExt:         json.RawMessage(`{"appnexus": {"placementId": 12345}}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:   "imp.ext - error",
			impExt:        json.RawMessage(`{"appnexus": {"dealTier": "wrong type", "placementId": 12345}}`),
			expectedError: "json: cannot unmarshal string into Go struct field .dealTier of type openrtb_ext.DealTier",
		},
		{
			description:    "imp.ext.prebid",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "anyPrefix"}, "placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "anyPrefix", MinDealTier: 5}},
		},
		{
			description:    "imp.ext.prebid- multiple",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "appnexusPrefix"}, "placementId": 12345}, "rubicon": {"dealTier": {"minDealTier": 8, "prefix": "rubiconPrefix"}, "placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "appnexusPrefix", MinDealTier: 5}, BidderRubicon: {Prefix: "rubiconPrefix", MinDealTier: 8}},
		},
		{
			description:    "imp.ext.prebid - no deal tier",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:   "imp.ext.prebid - error",
			impExt:        json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": "wrong type", "placementId": 12345}}}}`),
			expectedError: "json: cannot unmarshal string into Go struct field .prebid.bidder.dealTier of type openrtb_ext.DealTier",
		},
		{
			description:    "imp.ext.prebid wins over imp.ext",
			impExt:         json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "impExt"}, "placementId": 12345}, "prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 8, "prefix": "impExtPrebid"}, "placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "impExtPrebid", MinDealTier: 8}},
		},
		{
			description:    "imp.ext.prebid coexists with imp.ext",
			impExt:         json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "impExt"}, "placementId": 12345}, "prebid": {"bidder": {"rubicon": {"dealTier": {"minDealTier": 8, "prefix": "impExtPrebid"}, "placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "impExt", MinDealTier: 5}, BidderRubicon: {Prefix: "impExtPrebid", MinDealTier: 8}},
		},
	}

	for _, test := range testCases {
		imp := openrtb2.Imp{Ext: test.impExt}

		result, err := ReadDealTiersFromImp(imp)

		assert.Equal(t, test.expectedResult, result, test.description+":result")

		if len(test.expectedError) == 0 {
			assert.NoError(t, err, test.description+":error")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":error")
		}
	}
}
