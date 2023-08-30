package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
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
			description:    "imp.ext - no prebid but with other params",
			impExt:         json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "anyPrefix"}, "placementId": 12345}, "tid": "1234"}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "imp.ext.prebid - nil",
			impExt:         json.RawMessage(`{"prebid": null}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "imp.ext.prebid - empty",
			impExt:         json.RawMessage(`{"prebid": {}}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "imp.ext.prebid - no bidder but with other params",
			impExt:         json.RawMessage(`{"prebid": {"supportdeals": true}}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "imp.ext.prebid.bidder - one",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "anyPrefix"}, "placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "anyPrefix", MinDealTier: 5}},
		},
		{
			description:    "imp.ext.prebid.bidder - one with other params",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "anyPrefix"}, "placementId": 12345}}, "supportdeals": true}, "tid": "1234"}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "anyPrefix", MinDealTier: 5}},
		},
		{
			description:    "imp.ext.prebid.bidder - multiple",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "appnexusPrefix"}, "placementId": 12345}, "rubicon": {"dealTier": {"minDealTier": 8, "prefix": "rubiconPrefix"}, "placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "appnexusPrefix", MinDealTier: 5}, BidderRubicon: {Prefix: "rubiconPrefix", MinDealTier: 8}},
		},
		{
			description:    "imp.ext.prebid.bidder - one without deal tier",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:   "imp.ext.prebid.bidder - error",
			impExt:        json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": "wrong type", "placementId": 12345}}}}`),
			expectedError: "json: cannot unmarshal string into Go struct field .prebid.bidder.dealTier of type openrtb_ext.DealTier",
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
