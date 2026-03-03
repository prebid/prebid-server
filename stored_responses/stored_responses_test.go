package stored_responses

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestBuildStoredBidResponses(t *testing.T) {
	bidRespId1 := json.RawMessage(`{"id": "resp_id1"}`)
	bidRespId2 := json.RawMessage(`{"id": "resp_id2"}`)
	bidRespId3 := json.RawMessage(`{"id": "resp_id3"}`)
	testCases := []struct {
		description        string
		storedBidResponses ImpBidderStoredResp
		expectedResult     BidderImpsWithBidResponses
	}{
		{
			description: "one imp and stored response for this imp with one bidder",
			storedBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidRespId1},
			},
			expectedResult: BidderImpsWithBidResponses{
				"bidderA": {
					"imp-id1": bidRespId1,
				},
			},
		},
		{
			description: "one imp and stored response for this imp with two bidders",
			storedBidResponses: ImpBidderStoredResp{
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
			description: "two imps and stored response for this imp with two bidders",
			storedBidResponses: ImpBidderStoredResp{
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
			description: "three imps and stored response for these imps with two bidders",
			storedBidResponses: ImpBidderStoredResp{
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
		{
			description:        "empty stored responses",
			storedBidResponses: ImpBidderStoredResp{},
			expectedResult:     BidderImpsWithBidResponses{},
		},
	}

	for _, testCase := range testCases {

		bidderToImpToResponses := buildStoredResp(testCase.storedBidResponses)
		for expectedBidderName := range testCase.expectedResult {
			assert.Equal(t, testCase.expectedResult[expectedBidderName], bidderToImpToResponses[expectedBidderName], "incorrect stored responses for testCase %s", testCase.description)
		}
	}
}

func TestProcessStoredAuctionAndBidResponsesErrors(t *testing.T) {
	testCases := []struct {
		description       string
		request           openrtb2.BidRequest
		expectedErrorList []error
	}{
		{
			description: "Invalid stored auction response format: empty stored Auction Response Id",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
    			      "prebid": {
    			        "storedauctionresponse": {}
    			      }}`)},
				},
			},
			expectedErrorList: []error{errors.New("request.imp[0] has ext.prebid.storedauctionresponse specified, but \"id\" field is missing ")},
		},
		{
			description: "Invalid stored bid response format: empty storedbidresponse.bidder",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
    			      "prebid": {
    			        "storedbidresponse": [
							{ "id": "123abc"}]
    			      }}`)},
				},
			},
			expectedErrorList: []error{errors.New("request.imp[0] has ext.prebid.storedbidresponse specified, but \"id\" or/and \"bidder\" fields are missing ")},
		},
		{
			description: "Invalid stored bid response format: empty storedbidresponse.id",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
    			      "prebid": {
    			        "storedbidresponse": [
							{ "bidder": "testbidder"}]
    			      }}`)},
				},
			},
			expectedErrorList: []error{errors.New("request.imp[0] has ext.prebid.storedbidresponse specified, but \"id\" or/and \"bidder\" fields are missing ")},
		},
		{
			description: "Invalid stored auction response format: empty stored Auction Response Id in second imp",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
    			      "prebid": {
    			        "storedauctionresponse": {
							"id":"123"
    			        }
    			      }}`)},
					{ID: "imp-id2",
						Ext: json.RawMessage(`{
    			      "prebid": {
    			         "storedauctionresponse": {
							"id":""
    			        }
    			      }}`)},
				},
			},
			expectedErrorList: []error{errors.New("request.imp[1] has ext.prebid.storedauctionresponse specified, but \"id\" field is missing ")},
		},
		{
			description: "Invalid stored bid response format: empty stored bid Response Id in second imp",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
    			      "prebid": {
    			         "storedbidresponse": [
                             {"bidder":"testBidder", "id": "123abc"}
                        ]
    			      }}`)},
					{ID: "imp-id2",
						Ext: json.RawMessage(`{
    			      "prebid": {
    			         "storedbidresponse": [
                             {"bidder":"testBidder", "id": ""}
                        ]
    			      }}`)},
				},
			},
			expectedErrorList: []error{errors.New("request.imp[1] has ext.prebid.storedbidresponse specified, but \"id\" or/and \"bidder\" fields are missing ")},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			rw := &openrtb_ext.RequestWrapper{BidRequest: &test.request}
			_, _, _, errorList := ProcessStoredResponses(context.TODO(), rw, nil)
			assert.Equalf(t, test.expectedErrorList, errorList, "Error doesn't match: %s\n", test.description)
		})
	}

}

func TestProcessStoredAuctionAndBidResponses(t *testing.T) {
	bidStoredResp1 := json.RawMessage(`[{"bid": [{"id": "bid_id1"],"seat": "bidderA"}]`)
	bidStoredResp2 := json.RawMessage(`[{"bid": [{"id": "bid_id2"],"seat": "bidderB"}]`)
	bidStoredResp3 := json.RawMessage(`[{"bid": [{"id": "bid_id3"],"seat": "bidderA"}]`)
	mockStoredResponses := map[string]json.RawMessage{
		"1": bidStoredResp1,
		"2": bidStoredResp2,
		"3": bidStoredResp3,
	}
	fetcher := &mockStoredBidResponseFetcher{mockStoredResponses}

	testCases := []struct {
		description                    string
		request                        openrtb2.BidRequest
		expectedStoredAuctionResponses ImpsWithBidResponses
		expectedStoredBidResponses     ImpBidderStoredResp
		expectedBidderImpReplaceImpID  BidderImpReplaceImpID
	}{
		{
			description: "No stored responses",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
    			      "prebid": {}
    			    }`)},
				},
			},
			expectedStoredAuctionResponses: nil,
			expectedStoredBidResponses:     nil,
			expectedBidderImpReplaceImpID:  nil,
		},
		{
			description: "Stored auction response one imp",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "1"
                    		}
                		}
            		}`)},
				},
			},
			expectedStoredAuctionResponses: ImpsWithBidResponses{
				"imp-id1": bidStoredResp1,
			},
			expectedStoredBidResponses:    ImpBidderStoredResp{},
			expectedBidderImpReplaceImpID: BidderImpReplaceImpID{},
		},
		{
			description: "Stored bid response one imp",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"bidderA": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "1"}
                    		]
                		}
            		}`)},
				},
			},
			expectedStoredAuctionResponses: ImpsWithBidResponses{},
			expectedStoredBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidStoredResp1},
			},
			expectedBidderImpReplaceImpID: BidderImpReplaceImpID{
				"bidderA": map[string]bool{"imp-id1": true},
			},
		},
		{
			description: "Stored bid responses two bidders one imp",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"bidderA": {
							"placementId": 123
                		},
						"bidderB": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "1", "replaceimpid": true},
                        		{"bidder":"bidderB", "id": "2", "replaceimpid": false}
                    		]
                		}
            		}`)},
				},
			},
			expectedStoredAuctionResponses: ImpsWithBidResponses{},
			expectedStoredBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidStoredResp1, "bidderB": bidStoredResp2},
			},
			expectedBidderImpReplaceImpID: BidderImpReplaceImpID{
				"bidderA": map[string]bool{"imp-id1": true},
				"bidderB": map[string]bool{"imp-id1": false},
			},
		},
		{
			description: "Stored bid responses two same mixed case bidders one imp",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"bidderA": {
							"placementId": 123
                		},
						"BIDDERa": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "1", "replaceimpid": true},
                        		{"bidder":"bidderB", "id": "2", "replaceimpid": false}
                    		]
                		}
            		}`)},
				},
			},
			expectedStoredAuctionResponses: ImpsWithBidResponses{},
			expectedStoredBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidStoredResp1, "BIDDERa": bidStoredResp1},
			},
			expectedBidderImpReplaceImpID: BidderImpReplaceImpID{
				"BIDDERa": map[string]bool{"imp-id1": true},
				"bidderA": map[string]bool{"imp-id1": true},
			},
		},
		{
			description: "Stored bid responses 3 same mixed case bidders in imp.ext and imp.ext.prebid.bidders one imp",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"bidderA": {
							"placementId": 123
                		},
						"BIDDERa": {
							"placementId": 123
                		},
                		"prebid": {
							"bidder": {
                        		"BiddeRa": {
                            		"placementId": 12883451
                        		}
                    		},
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "1", "replaceimpid": true},
                        		{"bidder":"bidderB", "id": "2", "replaceimpid": false}
                    		]
                		}
            		}`)},
				},
			},
			expectedStoredAuctionResponses: ImpsWithBidResponses{},
			expectedStoredBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidStoredResp1, "BIDDERa": bidStoredResp1, "BiddeRa": bidStoredResp1},
			},
			expectedBidderImpReplaceImpID: BidderImpReplaceImpID{
				"BIDDERa": map[string]bool{"imp-id1": true},
				"bidderA": map[string]bool{"imp-id1": true},
				"BiddeRa": map[string]bool{"imp-id1": true},
			},
		},
		{
			description: "Stored bid responses 3 same mixed case bidders in imp.ext and imp.ext.prebid.bidders one imp, duplicated stored response",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"bidderA": {
							"placementId": 123
                		},
						"BIDDERa": {
							"placementId": 123
                		},
                		"prebid": {
							"bidder": {
                        		"BiddeRa": {
                            		"placementId": 12883451
                        		}
                    		},
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "1", "replaceimpid": true},
                        		{"bidder":"bidderA", "id": "2", "replaceimpid": true},
                        		{"bidder":"bidderB", "id": "2", "replaceimpid": false}
                    		]
                		}
            		}`)},
				},
			},
			expectedStoredAuctionResponses: ImpsWithBidResponses{},
			expectedStoredBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidStoredResp1, "BIDDERa": bidStoredResp1, "BiddeRa": bidStoredResp1},
			},
			expectedBidderImpReplaceImpID: BidderImpReplaceImpID{
				"BIDDERa": map[string]bool{"imp-id1": true},
				"bidderA": map[string]bool{"imp-id1": true},
				"BiddeRa": map[string]bool{"imp-id1": true},
			},
		},
		{
			//This is not a valid scenario for real auction request, added for testing purposes
			description: "Stored auction and bid responses one imp",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"bidderA": {
							"placementId": 123
                		},
						"bidderB": {
							"placementId": 123
                		},
                		"prebid": {
							"storedauctionresponse": {
                        		"id": "1"
                    		},
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "1"},
                        		{"bidder":"bidderB", "id": "2"}
                    		]
                		}
            		}`)},
				},
			},
			expectedStoredAuctionResponses: ImpsWithBidResponses{
				"imp-id1": bidStoredResp1,
			},
			expectedStoredBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidStoredResp1, "bidderB": bidStoredResp2},
			},
			expectedBidderImpReplaceImpID: BidderImpReplaceImpID{
				"bidderA": map[string]bool{"imp-id1": true},
				"bidderB": map[string]bool{"imp-id1": true},
			},
		},
		{
			description: "Stored auction response three imps",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "1"
                    		}
                		}
            		}`)},
					{ID: "imp-id2",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "2"
                    		}
                		}
            		}`)},
					{
						ID: "imp-id3",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "3"
                    		}
                		}
            		}`),
					},
				},
			},
			expectedStoredAuctionResponses: ImpsWithBidResponses{
				"imp-id1": bidStoredResp1,
				"imp-id2": bidStoredResp2,
				"imp-id3": bidStoredResp3,
			},
			expectedStoredBidResponses:    ImpBidderStoredResp{},
			expectedBidderImpReplaceImpID: BidderImpReplaceImpID{},
		},
		{
			description: "Stored auction response three imps duplicated stored auction response",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "1"
                    		}
                		}
            		}`)},
					{ID: "imp-id2",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "2"
                    		}
                		}
            		}`)},
					{
						ID: "imp-id3",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "2"
                    		}
                		}
            		}`),
					},
				},
			},
			expectedStoredAuctionResponses: ImpsWithBidResponses{
				"imp-id1": bidStoredResp1,
				"imp-id2": bidStoredResp2,
				"imp-id3": bidStoredResp2,
			},
			expectedStoredBidResponses:    ImpBidderStoredResp{},
			expectedBidderImpReplaceImpID: BidderImpReplaceImpID{},
		},
		{
			description: "Stored bid responses two bidders two imp",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"bidderA": {
							"placementId": 123
                		},
						"bidderB": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "1", "replaceimpid": false},
                        		{"bidder":"bidderB", "id": "2"}
                    		]
                		}
            		}`)},
					{ID: "imp-id2",
						Ext: json.RawMessage(`{
                		"bidderA": {
							"placementId": 123
                		},
						"bidderB": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "3"},
                        		{"bidder":"bidderB", "id": "2", "replaceimpid": false}
                    		]
                		}
            		}`)},
				},
			},
			expectedStoredAuctionResponses: ImpsWithBidResponses{},
			expectedStoredBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidStoredResp1, "bidderB": bidStoredResp2},
				"imp-id2": {"bidderA": bidStoredResp3, "bidderB": bidStoredResp2},
			},
			expectedBidderImpReplaceImpID: BidderImpReplaceImpID{
				"bidderA": map[string]bool{"imp-id1": false, "imp-id2": true},
				"bidderB": map[string]bool{"imp-id1": true, "imp-id2": false},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			rw := openrtb_ext.RequestWrapper{BidRequest: &test.request}
			storedAuctionResponses, storedBidResponses, bidderImpReplaceImpId, errorList := ProcessStoredResponses(context.TODO(), &rw, fetcher)
			assert.Equal(t, test.expectedStoredAuctionResponses, storedAuctionResponses)
			assert.Equal(t, test.expectedStoredBidResponses, storedBidResponses)
			assert.Equal(t, test.expectedBidderImpReplaceImpID, bidderImpReplaceImpId)
			assert.Nil(t, errorList, "Error should be nil")
		})
	}

}

func TestProcessStoredResponsesNotFoundResponse(t *testing.T) {
	bidStoredResp1 := json.RawMessage(`[{"bid": [{"id": "bid_id1"],"seat": "bidderA"}]`)
	bidStoredResp2 := json.RawMessage(`[{"bid": [{"id": "bid_id2"],"seat": "bidderB"}]`)
	mockStoredResponses := map[string]json.RawMessage{
		"1": bidStoredResp1,
		"2": bidStoredResp2,
		"3": nil,
		"4": nil,
	}
	fetcher := &mockStoredBidResponseFetcher{mockStoredResponses}

	testCases := []struct {
		description    string
		request        openrtb2.BidRequest
		expectedErrors []error
	}{
		{
			description: "Stored bid response with nil data, one bidder one imp",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"bidderB": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderB", "id": "3"}
                    		]
                		}
            		}`)},
				},
			},
			expectedErrors: []error{
				errors.New("failed to fetch stored bid response for impId = imp-id1, bidder = bidderB and storedBidResponse id = 3"),
			},
		},
		{
			description: "Stored bid response with nil data, one bidder, two imps, one with correct stored response",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"bidderB": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderB", "id": "1"}
                    		]
                		}
            		}`)},
					{ID: "imp-id2",
						Ext: json.RawMessage(`{
                		"bidderB": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderB", "id": "3"}
                    		]
                		}
            		}`)},
				},
			},
			expectedErrors: []error{
				errors.New("failed to fetch stored bid response for impId = imp-id2, bidder = bidderB and storedBidResponse id = 3"),
			},
		},
		{
			description: "Stored bid response with nil data, one bidder, two imps, both with correct stored response",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"bidderB": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderB", "id": "4"}
                    		]
                		}
            		}`)},
					{ID: "imp-id2",
						Ext: json.RawMessage(`{
                		"bidderB": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderB", "id": "3"}
                    		]
                		}
            		}`)},
				},
			},
			expectedErrors: []error{
				errors.New("failed to fetch stored bid response for impId = imp-id1, bidder = bidderB and storedBidResponse id = 4"),
				errors.New("failed to fetch stored bid response for impId = imp-id2, bidder = bidderB and storedBidResponse id = 3"),
			},
		},
		{
			description: "Stored auction response with nil data and one imp",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "4"
                    		}
                		}
            		}`)},
				},
			},
			expectedErrors: []error{
				errors.New("failed to fetch stored auction response for impId = imp-id1 and storedAuctionResponse id = 4"),
			},
		},
		{
			description: "Stored auction response with nil data, and two imps with nil responses",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "4"
                    		}
                		}
            		}`)},
					{ID: "imp-id2",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "3"
                    		}
                		}
            		}`)},
				},
			},
			expectedErrors: []error{
				errors.New("failed to fetch stored auction response for impId = imp-id1 and storedAuctionResponse id = 4"),
				errors.New("failed to fetch stored auction response for impId = imp-id2 and storedAuctionResponse id = 3"),
			},
		},
		{
			description: "Stored auction response with nil data, two imps, one with nil responses",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-id1",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "2"
                    		}
                		}
            		}`)},
					{ID: "imp-id2",
						Ext: json.RawMessage(`{
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "3"
                    		}
                		}
            		}`)},
				},
			},
			expectedErrors: []error{
				errors.New("failed to fetch stored auction response for impId = imp-id2 and storedAuctionResponse id = 3"),
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			rw := openrtb_ext.RequestWrapper{BidRequest: &test.request}
			_, _, _, errorList := ProcessStoredResponses(context.TODO(), &rw, fetcher)
			for _, err := range test.expectedErrors {
				assert.Contains(t, errorList, err)
			}
		})
	}
}

func TestFlipMap(t *testing.T) {
	testCases := []struct {
		description              string
		inImpBidderReplaceImpID  ImpBidderReplaceImpID
		outBidderImpReplaceImpID BidderImpReplaceImpID
	}{
		{
			description:              "Empty ImpBidderReplaceImpID",
			inImpBidderReplaceImpID:  ImpBidderReplaceImpID{},
			outBidderImpReplaceImpID: BidderImpReplaceImpID{},
		},
		{
			description:              "Nil ImpBidderReplaceImpID",
			inImpBidderReplaceImpID:  nil,
			outBidderImpReplaceImpID: BidderImpReplaceImpID{},
		},
		{
			description:              "ImpBidderReplaceImpID has a one element map with single element",
			inImpBidderReplaceImpID:  ImpBidderReplaceImpID{"imp-id": {"bidderA": true}},
			outBidderImpReplaceImpID: BidderImpReplaceImpID{"bidderA": {"imp-id": true}},
		},
		{
			description:              "ImpBidderReplaceImpID has a one element map with multiple elements",
			inImpBidderReplaceImpID:  ImpBidderReplaceImpID{"imp-id": {"bidderA": true, "bidderB": false}},
			outBidderImpReplaceImpID: BidderImpReplaceImpID{"bidderA": {"imp-id": true}, "bidderB": {"imp-id": false}},
		},
		{
			description: "ImpBidderReplaceImpID has multiple elements map with single element",
			inImpBidderReplaceImpID: ImpBidderReplaceImpID{
				"imp-id1": {"bidderA": true},
				"imp-id2": {"bidderB": false}},
			outBidderImpReplaceImpID: BidderImpReplaceImpID{
				"bidderA": {"imp-id1": true},
				"bidderB": {"imp-id2": false}},
		},
		{
			description: "ImpBidderReplaceImpID has multiple elements map with multiple elements",
			inImpBidderReplaceImpID: ImpBidderReplaceImpID{
				"imp-id1": {"bidderA": true, "bidderB": false, "bidderC": false, "bidderD": true},
				"imp-id2": {"bidderA": false, "bidderB": false, "bidderC": true, "bidderD": true},
				"imp-id3": {"bidderA": false, "bidderB": true, "bidderC": true, "bidderD": false}},
			outBidderImpReplaceImpID: BidderImpReplaceImpID{
				"bidderA": {"imp-id1": true, "imp-id2": false, "imp-id3": false},
				"bidderB": {"imp-id1": false, "imp-id2": false, "imp-id3": true},
				"bidderC": {"imp-id1": false, "imp-id2": true, "imp-id3": true},
				"bidderD": {"imp-id1": true, "imp-id2": true, "imp-id3": false}},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			actualResult := flipMap(test.inImpBidderReplaceImpID)
			assert.Equal(t, test.outBidderImpReplaceImpID, actualResult)
		})
	}
}

type mockStoredBidResponseFetcher struct {
	data map[string]json.RawMessage
}

func (cf *mockStoredBidResponseFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return nil, nil, nil
}

func (cf *mockStoredBidResponseFetcher) FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	return cf.data, nil
}
