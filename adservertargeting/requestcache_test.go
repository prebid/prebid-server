package adservertargeting

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestRequestImpCache(t *testing.T) {
	testCases := []struct {
		description     string
		inputRequest    json.RawMessage
		expectedReqSize int
		expectedImpsNum int
		expectedError   bool
	}{
		{
			description:     "valid request with 2 imps",
			inputRequest:    json.RawMessage(reqValid),
			expectedReqSize: 355,
			expectedImpsNum: 2,
			expectedError:   false,
		},
		{
			description:     "invalid request ",
			inputRequest:    json.RawMessage(reqInvalid),
			expectedReqSize: 88,
			expectedImpsNum: 0,
			expectedError:   true,
		},
		{
			description:     "valid request with no imps",
			inputRequest:    json.RawMessage(reqNoImps),
			expectedReqSize: 52,
			expectedImpsNum: 0,
			expectedError:   true,
		},
	}

	for _, test := range testCases {
		reqImpCache := requestCache{resolvedReq: test.inputRequest}

		actualReq := reqImpCache.GetReqJson()
		assert.Len(t, actualReq, test.expectedReqSize, "incorrect request returned")
		actualImps, err := reqImpCache.GetImpsData()
		assert.Len(t, actualImps, test.expectedImpsNum, "incorrect number of impressions returned")

		if test.expectedError {
			assert.Error(t, err, "expected error not returned")
		} else {
			assert.NoError(t, err, "unexpected error returned")
		}
	}
}

func TestBidsCache(t *testing.T) {

	testCases := []struct {
		description      string
		inputBidder      string
		inputBidId       string
		inputBid         openrtb2.Bid
		expectedBidBytes []byte
		expectedError    bool
	}{
		{
			description:      "valid bid not in cache for existing bidder",
			inputBidder:      "bidderA",
			inputBidId:       "bid3",
			inputBid:         openrtb2.Bid{ID: "test_bid3"},
			expectedBidBytes: []byte(`{"id":"test_bid3","impid":"","price":0}`),
			expectedError:    false,
		},
		{
			description:      "valid bid and not existing bidder",
			inputBidder:      "bidderB",
			inputBidId:       "bid1",
			inputBid:         openrtb2.Bid{ID: "test_bid1"},
			expectedBidBytes: []byte(`{"id":"test_bid1","impid":"","price":0}`),
			expectedError:    false,
		},
		{
			description:      "valid bid in cache",
			inputBidder:      "bidderA",
			inputBidId:       "bid2",
			inputBid:         openrtb2.Bid{},
			expectedBidBytes: []byte(`{"bidid":"test_bid2"}`),
			expectedError:    false,
		},
	}

	bCache := bidsCache{bids: map[string]map[string][]byte{
		"bidderA": {
			"bid1": []byte(`{"bidid":"test_bid1"}`),
			"bid2": []byte(`{"bidid":"test_bid2"}`),
		},
	}}

	for _, test := range testCases {
		bidBytes, err := bCache.GetBid(test.inputBidder, test.inputBidId, test.inputBid)
		if test.expectedError {
			assert.Error(t, err, "expected error not returned")
		} else {
			assert.NoError(t, err, "unexpected error returned")
			assert.Equal(t, test.expectedBidBytes, bidBytes, "incorrect bid returned")
		}
	}
}
