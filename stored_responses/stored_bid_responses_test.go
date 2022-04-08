package stored_responses

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRemoveImpsWithStoredResponses(t *testing.T) {
	bidRespId1 := json.RawMessage(`{"id": "resp_id1"}`)
	testCases := []struct {
		description        string
		reqIn              *openrtb2.BidRequest
		storedBidResponses map[string]map[string]json.RawMessage
		expectedImps       []openrtb2.Imp
	}{
		{
			description: "request with imps and stored bid response for this imp",
			reqIn: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
				{ID: "imp-id1"},
			}},
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"appnexus": bidRespId1},
			},
			expectedImps: nil,
		},
		{
			description: "request with imps and stored bid response for one of these imp",
			reqIn: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
				{ID: "imp-id1"},
				{ID: "imp-id2"},
			}},
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"appnexus": bidRespId1},
			},
			expectedImps: []openrtb2.Imp{
				{
					ID: "imp-id2",
				},
			},
		},
		{
			description: "request with imps and stored bid response for both of these imp",
			reqIn: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
				{ID: "imp-id1"},
				{ID: "imp-id2"},
			}},
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"appnexus": bidRespId1},
				"imp-id2": {"appnexus": bidRespId1},
			},
			expectedImps: nil,
		},
		{
			description: "request with imps and no stored bid responses",
			reqIn: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
				{ID: "imp-id1"},
				{ID: "imp-id2"},
			}},
			storedBidResponses: nil,

			expectedImps: []openrtb2.Imp{
				{ID: "imp-id1"},
				{ID: "imp-id2"},
			},
		},
	}
	for _, testCase := range testCases {
		request := testCase.reqIn
		sr := StoredBidResponses{StoredBidResponses: testCase.storedBidResponses}
		sr.removeImpsWithStoredResponses(request)
		assert.Equal(t, testCase.expectedImps, request.Imp, "incorrect Impressions for testCase %s", testCase.description)
	}
}

func TestBuildStoredBidResponses(t *testing.T) {
	bidRespId1 := json.RawMessage(`{"id": "resp_id1"}`)
	bidRespId2 := json.RawMessage(`{"id": "resp_id2"}`)
	bidRespId3 := json.RawMessage(`{"id": "resp_id3"}`)
	testCases := []struct {
		description        string
		reqIn              *openrtb2.BidRequest
		storedBidResponses map[string]map[string]json.RawMessage
		expectedResult     BidderImpsWithBidResponses
	}{
		{
			description: "request with one imp and stored response for this imp with one bidder",
			reqIn: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
				{ID: "imp-id1"},
			}},
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"bidderA": bidRespId1},
			},
			expectedResult: BidderImpsWithBidResponses{
				"bidderA": {
					"imp-id1": bidRespId1,
				},
			},
		},
		{
			description: "request with one imp and stored response for this imp with two bidders",
			reqIn: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
				{ID: "imp-id1"},
			}},
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"bidderA": bidRespId1, "bidderB": bidRespId2},
			},

			expectedResult: BidderImpsWithBidResponses{
				"bidderA": {
					"imp-id1": bidRespId1,
				},
				"bidderB": {
					"imp-id1": bidRespId2,
				},
			},
		},
		{
			description: "request with two imps and stored response for this imp with two bidders",
			reqIn: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
				{ID: "imp-id1"},
				{ID: "imp-id2"},
			}},
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"bidderA": bidRespId1},
				"imp-id2": {"bidderB": bidRespId2},
			},

			expectedResult: BidderImpsWithBidResponses{
				"bidderA": {
					"imp-id1": bidRespId1,
				},
				"bidderB": {
					"imp-id2": bidRespId2,
				},
			},
		},

		{
			description: "request with three imps and stored response for these imps with two bidders",
			reqIn: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
				{ID: "imp-id1"},
				{ID: "imp-id2"},
				{ID: "imp-id3"},
			}},
			storedBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"bidderA": bidRespId1},
				"imp-id2": {"bidderB": bidRespId2},
				"imp-id3": {"bidderA": bidRespId3},
			},

			expectedResult: BidderImpsWithBidResponses{
				"bidderA": {
					"imp-id1": bidRespId1,
					"imp-id3": bidRespId3,
				},
				"bidderB": {
					"imp-id2": bidRespId2,
				},
			},
		},
	}
	//aliases := map[string]string{"bidderA": "bidderA", "bidderB": "bidderB"}

	for _, testCase := range testCases {

		sr := StoredBidResponses{StoredBidResponses: testCase.storedBidResponses}
		sr.buildStoredResp()
		for expectedBidderName := range testCase.expectedResult {
			assert.Equal(t, testCase.expectedResult[expectedBidderName], sr.BidderToImpToResponses[expectedBidderName], "incorrect stored responses for testCase %s", testCase.description)
		}
	}
}
