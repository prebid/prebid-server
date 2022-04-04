package exchange

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRemoveImpsWithStoredResponses(t *testing.T) {
	bidRespId1 := json.RawMessage(`{"id": "resp_id1"}`)
	testCases := []struct {
		description  string
		reqIn        AuctionRequest
		expectedImps []openrtb2.Imp
	}{
		{
			description: "request with imps and stored bid response for this imp",
			reqIn: AuctionRequest{
				BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
					{ID: "imp-id1"},
				}},
				StoredBidResponses: map[string]map[string]json.RawMessage{
					"imp-id1": {"appnexus": bidRespId1},
				},
			},
			expectedImps: nil,
		},
		{
			description: "request with imps and stored bid response for one of these imp",
			reqIn: AuctionRequest{
				BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
					{ID: "imp-id1"},
					{ID: "imp-id2"},
				}},
				StoredBidResponses: map[string]map[string]json.RawMessage{
					"imp-id1": {"appnexus": bidRespId1},
				},
			},
			expectedImps: []openrtb2.Imp{
				{
					ID: "imp-id2",
				},
			},
		},
		{
			description: "request with imps and stored bid response for both of these imp",
			reqIn: AuctionRequest{
				BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
					{ID: "imp-id1"},
					{ID: "imp-id2"},
				}},
				StoredBidResponses: map[string]map[string]json.RawMessage{
					"imp-id1": {"appnexus": bidRespId1},
					"imp-id2": {"appnexus": bidRespId1},
				},
			},
			expectedImps: nil,
		},
		{
			description: "request with imps and no stored bid responses",
			reqIn: AuctionRequest{
				BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
					{ID: "imp-id1"},
					{ID: "imp-id2"},
				}},
				StoredBidResponses: nil,
			},
			expectedImps: []openrtb2.Imp{
				{ID: "imp-id1"},
				{ID: "imp-id2"},
			},
		},
	}
	for _, testCase := range testCases {
		request := testCase.reqIn
		removeImpsWithStoredResponses(request)
		assert.Equal(t, testCase.expectedImps, request.BidRequest.Imp, "incorrect Impressions for testCase %s", testCase.description)
	}
}

func TestBuildStoredBidResponses(t *testing.T) {
	bidRespId1 := json.RawMessage(`{"id": "resp_id1"}`)
	bidRespId2 := json.RawMessage(`{"id": "resp_id2"}`)
	bidRespId3 := json.RawMessage(`{"id": "resp_id3"}`)
	testCases := []struct {
		description    string
		reqIn          AuctionRequest
		expectedResult map[openrtb_ext.BidderName]BidderRequest
	}{
		{
			description: "request with one imp and stored response for this imp with one bidder",
			reqIn: AuctionRequest{
				BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
					{ID: "imp-id1"},
				}},
				StoredBidResponses: map[string]map[string]json.RawMessage{
					"imp-id1": {"bidderA": bidRespId1},
				},
			},
			expectedResult: map[openrtb_ext.BidderName]BidderRequest{
				"bidderA": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id1"},
					}},
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId1,
					},
				},
			},
		},
		{
			description: "request with one imp and stored response for this imp with two bidders",
			reqIn: AuctionRequest{
				BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
					{ID: "imp-id1"},
				}},
				StoredBidResponses: map[string]map[string]json.RawMessage{
					"imp-id1": {"bidderA": bidRespId1, "bidderB": bidRespId2},
				},
			},
			expectedResult: map[openrtb_ext.BidderName]BidderRequest{
				"bidderA": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id1"},
					}},
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId1,
					},
				},
				"bidderB": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id1"},
					}},
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId2,
					},
				},
			},
		},
		{
			description: "request with two imps and stored response for this imp with two bidders",
			reqIn: AuctionRequest{
				BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
					{ID: "imp-id1"},
					{ID: "imp-id2"},
				}},
				StoredBidResponses: map[string]map[string]json.RawMessage{
					"imp-id1": {"bidderA": bidRespId1},
					"imp-id2": {"bidderB": bidRespId2},
				},
			},
			expectedResult: map[openrtb_ext.BidderName]BidderRequest{
				"bidderA": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id1"},
						{ID: "imp-id2"},
					}},
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId1,
					},
				},
				"bidderB": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id1"},
						{ID: "imp-id2"},
					}},
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id2": bidRespId2,
					},
				},
			},
		},

		{
			description: "request with three imps and stored response for these imps with two bidders",
			reqIn: AuctionRequest{
				BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
					{ID: "imp-id1"},
					{ID: "imp-id2"},
					{ID: "imp-id3"},
				}},
				StoredBidResponses: map[string]map[string]json.RawMessage{
					"imp-id1": {"bidderA": bidRespId1},
					"imp-id2": {"bidderB": bidRespId2},
					"imp-id3": {"bidderA": bidRespId3},
				},
			},
			expectedResult: map[openrtb_ext.BidderName]BidderRequest{
				"bidderA": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id1"},
						{ID: "imp-id2"},
						{ID: "imp-id3"},
					}},
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id1": bidRespId1,
						"imp-id3": bidRespId3,
					},
				},
				"bidderB": {
					BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
						{ID: "imp-id1"},
						{ID: "imp-id2"},
						{ID: "imp-id3"},
					}},
					BidderStoredResponses: map[string]json.RawMessage{
						"imp-id2": bidRespId2,
					},
				},
			},
		},
	}
	aliases := map[string]string{"bidderA": "bidderA", "bidderB": "bidderB"}

	for _, testCase := range testCases {
		request := testCase.reqIn
		result := buildStoredBidResponses(request, aliases)
		for expectedBidderName, expectedReq := range result {
			assert.Equal(t, testCase.expectedResult[expectedBidderName].BidderStoredResponses, expectedReq.BidderStoredResponses, "incorrect stored responses for testCase %s", testCase.description)
			assert.Equal(t, testCase.expectedResult[expectedBidderName].BidRequest, expectedReq.BidRequest, "incorrect bid request for testCase %s", testCase.description)
		}
	}
}

func TestPrepareStoredResponse(t *testing.T) {
	result := prepareStoredResponse("imp_id1", json.RawMessage(`{"id": "resp_id1"}`))
	assert.Equal(t, []byte("imp_id1"), result.request.Body, "incorrect request body")
	assert.Equal(t, []byte(`{"id": "resp_id1"}`), result.response.Body, "incorrect response body")
}
