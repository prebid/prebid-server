package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestReadDealTiersFromImp(t *testing.T) {
	testCases := []struct {
		description       string
		impExt            json.RawMessage
		expectedResult    DealTierBidderMap
		expectedErrorType error
	}{
		{
			description:    "nil",
			impExt:         nil,
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "none",
			impExt:         json.RawMessage(``),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "empty_object",
			impExt:         json.RawMessage(`{}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "imp.ext_no_prebid_but_with_other_params",
			impExt:         json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "anyPrefix"}, "placementId": 12345}, "tid": "1234"}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "imp.ext.prebid_nil",
			impExt:         json.RawMessage(`{"prebid": null}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "imp.ext.prebid_empty",
			impExt:         json.RawMessage(`{"prebid": {}}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "imp.ext.prebid_no bidder but with other params",
			impExt:         json.RawMessage(`{"prebid": {"supportdeals": true}}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:    "imp.ext.prebid.bidder_one",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "anyPrefix"}, "placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "anyPrefix", MinDealTier: 5}},
		},
		{
			description:    "imp.ext.prebid.bidder_one_but_not_found_in_the_adapter_bidder_list",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"unknown": {"dealTier": {"minDealTier": 5, "prefix": "anyPrefix"}, "placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{"unknown": {Prefix: "anyPrefix", MinDealTier: 5}},
		},
		{
			description:    "imp.ext.prebid.bidder_one_but_not_found_in_the_adapter_bidder_list_with_case_insensitive",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"UnKnOwn": {"dealTier": {"minDealTier": 5, "prefix": "anyPrefix"}, "placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{"UnKnOwn": {Prefix: "anyPrefix", MinDealTier: 5}},
		},
		{
			description:    "imp.ext.prebid.bidder_one_but_case_is_different_from_the_adapter_bidder_list",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"APpNExUS": {"dealTier": {"minDealTier": 5, "prefix": "anyPrefix"}, "placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "anyPrefix", MinDealTier: 5}},
		},
		{
			description:    "imp.ext.prebid.bidder_one_with_other_params",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "anyPrefix"}, "placementId": 12345}}, "supportdeals": true}, "tid": "1234"}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "anyPrefix", MinDealTier: 5}},
		},
		{
			description:    "imp.ext.prebid.bidder_multiple",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "appnexusPrefix"}, "placementId": 12345}, "rubicon": {"dealTier": {"minDealTier": 8, "prefix": "rubiconPrefix"}, "placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{BidderAppnexus: {Prefix: "appnexusPrefix", MinDealTier: 5}, BidderRubicon: {Prefix: "rubiconPrefix", MinDealTier: 8}},
		},
		{
			description:    "imp.ext.prebid.bidder_one_without_deal_tier",
			impExt:         json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"placementId": 12345}}}}`),
			expectedResult: DealTierBidderMap{},
		},
		{
			description:       "imp.ext.prebid.bidder_error",
			impExt:            json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": "wrong type", "placementId": 12345}}}}`),
			expectedErrorType: &errortypes.FailedToUnmarshal{},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {

			imp := openrtb2.Imp{Ext: test.impExt}
			result, err := ReadDealTiersFromImp(imp)

			assert.Equal(t, test.expectedResult, result)

			if test.expectedErrorType != nil {
				assert.IsType(t, test.expectedErrorType, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	t.Run("imp.ext.prebid.bidder_dedupe", func(t *testing.T) {
		impExt := json.RawMessage(`{"prebid": {"bidder": {"APPNEXUS": {"dealTier": {"minDealTier": 100}},"APpNExUS": {"dealTier": {"minDealTier": 5}}}}}`)
		imp := openrtb2.Imp{Ext: impExt}
		result, err := ReadDealTiersFromImp(imp)

		assert.Len(t, result, 1)
		assert.NotNil(t, result["appnexus"])
		assert.NoError(t, err)
	})
}
