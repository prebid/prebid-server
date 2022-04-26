package stored_responses

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRemoveImpsWithStoredResponses(t *testing.T) {
	bidRespId1 := json.RawMessage(`{"id": "resp_id1"}`)
	testCases := []struct {
		description        string
		reqIn              *openrtb2.BidRequest
		storedBidResponses ImpBidderStoredResp
		expectedImps       []openrtb2.Imp
	}{
		{
			description: "request with imps and stored bid response for this imp",
			reqIn: &openrtb2.BidRequest{Imp: []openrtb2.Imp{
				{ID: "imp-id1"},
			}},
			storedBidResponses: ImpBidderStoredResp{
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
			storedBidResponses: ImpBidderStoredResp{
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
			storedBidResponses: ImpBidderStoredResp{
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
		removeImpsWithStoredResponses(request, testCase.storedBidResponses)
		assert.Equal(t, testCase.expectedImps, request.Imp, "incorrect Impressions for testCase %s", testCase.description)
	}
}

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
	bidderMap := map[string]openrtb_ext.BidderName{"testBidder": "testBidder"}

	testCases := []struct {
		description       string
		requestJson       []byte
		expectedErrorList []error
	}{
		{
			description: "Invalid stored auction response format: empty stored Auction Response Id",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
    			      "prebid": {
    			        "storedauctionresponse": {
    			        }
    			      }
    			    }
    			  }
    			]}`),
			expectedErrorList: []error{errors.New("request.imp[0] has ext.prebid.storedauctionresponse specified, but \"id\" field is missing ")},
		},
		{
			description: "Invalid stored bid response format: empty storedbidresponse.bidder",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
    			      "prebid": {
    			        "storedbidresponse": [
							{ "id": "123abc"}]
    			      }
    			    }
    			  }
    			]}`),
			expectedErrorList: []error{errors.New("request.imp[0] has ext.prebid.storedbidresponse specified, but \"id\" or/and \"bidder\" fields are missing ")},
		},
		{
			description: "Invalid stored bid response format: empty storedbidresponse.id",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
    			      "prebid": {
    			        "storedbidresponse": [
							{ "bidder": "testbidder"}]
    			      }
    			    }
    			  }
    			]}`),
			expectedErrorList: []error{errors.New("request.imp[0] has ext.prebid.storedbidresponse specified, but \"id\" or/and \"bidder\" fields are missing ")},
		},
		{
			description: "Invalid stored bid response: storedbidresponse.bidder not found",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
    			      "prebid": {
    			        "storedbidresponse": [
							{ "bidder": "testBidder123", "id": "123abc"}]
    			      }
    			    }
    			  }
    			]}`),
			expectedErrorList: []error{errors.New("request.imp[impId: imp-id1].ext contains unknown bidder: testBidder123. Did you forget an alias in request.ext.prebid.aliases?")},
		},
		{
			description: "Invalid stored auction response format: empty stored Auction Response Id in second imp",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
    			      "prebid": {
    			        "storedauctionresponse": {
							"id":"123"
    			        }
    			      }
    			    }
    			  },
			      {
    			    "id": "imp-id2",
    			    "ext": {
    			      "prebid": {
    			        "storedauctionresponse": {
							"id":""
    			        }
    			      }
    			    }
    			  }
    			]}`),
			expectedErrorList: []error{errors.New("request.imp[1] has ext.prebid.storedauctionresponse specified, but \"id\" field is missing ")},
		},
		{
			description: "Invalid stored bid response format: empty stored bid Response Id in second imp",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
    			      "prebid": {
    			        "storedbidresponse": [
                             {"bidder":"testBidder", "id": "123abc"}
                        ]
    			      }
    			    }
    			  },
			      {
    			    "id": "imp-id2",
    			    "ext": {
    			      "prebid": {
    			        "storedbidresponse": [
                             {"bidder":"testBidder", "id": ""}
                        ]
    			      }
    			    }
    			  }
    			]}`),
			expectedErrorList: []error{errors.New("request.imp[1] has ext.prebid.storedbidresponse specified, but \"id\" or/and \"bidder\" fields are missing ")},
		},
	}

	for _, test := range testCases {
		_, _, errorList := ProcessStoredResponses(nil, test.requestJson, nil, bidderMap)
		assert.Equalf(t, test.expectedErrorList, errorList, "Error doesn't match: %s\n", test.description)
	}

}

func TestProcessStoredAuctionAndBidResponses(t *testing.T) {
	bidderMap := map[string]openrtb_ext.BidderName{"bidderA": "bidderA", "bidderB": "bidderB"}
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
		requestJson                    []byte
		expectedStoredAuctionResponses ImpsWithBidResponses
		expectedStoredBidResponses     ImpBidderStoredResp
	}{
		{
			description: "No stored responses",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
    			      "prebid": {
    			        
    			      }
    			    }
    			  }
    			]}`),
			expectedStoredAuctionResponses: nil,
			expectedStoredBidResponses:     nil,
		},
		{
			description: "Stored auction response one imp",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "1"
                    		}
                		}
            		}
    			  }
    			]}`),
			expectedStoredAuctionResponses: ImpsWithBidResponses{
				"imp-id1": bidStoredResp1,
			},
			expectedStoredBidResponses: ImpBidderStoredResp{},
		},
		{
			description: "Stored bid response one imp",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "1"}
                    		]
                		}
            		}
    			  }
    			]}`),
			expectedStoredAuctionResponses: ImpsWithBidResponses{},
			expectedStoredBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidStoredResp1},
			},
		},
		{
			description: "Stored bid responses two bidders one imp",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "1"},
                        		{"bidder":"bidderB", "id": "2"}
                    		]
                		}
            		}
    			  }
    			]}`),
			expectedStoredAuctionResponses: ImpsWithBidResponses{},
			expectedStoredBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidStoredResp1, "bidderB": bidStoredResp2},
			},
		},
		{
			//This is not a valid scenario for real auction request, added for testing purposes
			description: "Stored auction and bid responses one imp",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
                		"appnexus": {
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
            		}
    			  }
    			]}`),
			expectedStoredAuctionResponses: ImpsWithBidResponses{
				"imp-id1": bidStoredResp1,
			},
			expectedStoredBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidStoredResp1, "bidderB": bidStoredResp2},
			},
		},
		{
			description: "Stored auction response three imps",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "1"
                    		}
                		}
            		}
    			  },
					{
    			    "id": "imp-id2",
    			    "ext": {
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "2"
                    		}
                		}
            		}
    			  },
					{
    			    "id": "imp-id3",
    			    "ext": {
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "3"
                    		}
                		}
            		}
    			  }
    			]}`),
			expectedStoredAuctionResponses: ImpsWithBidResponses{
				"imp-id1": bidStoredResp1,
				"imp-id2": bidStoredResp2,
				"imp-id3": bidStoredResp3,
			},
			expectedStoredBidResponses: ImpBidderStoredResp{},
		},
		{
			description: "Stored auction response three imps duplicated stored auction response",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "1"
                    		}
                		}
            		}
    			  },
					{
    			    "id": "imp-id2",
    			    "ext": {
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "2"
                    		}
                		}
            		}
    			  },
					{
    			    "id": "imp-id3",
    			    "ext": {
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedauctionresponse": {
                        		"id": "2"
                    		}
                		}
            		}
    			  }
    			]}`),
			expectedStoredAuctionResponses: ImpsWithBidResponses{
				"imp-id1": bidStoredResp1,
				"imp-id2": bidStoredResp2,
				"imp-id3": bidStoredResp2,
			},
			expectedStoredBidResponses: ImpBidderStoredResp{},
		},
		{
			description: "Stored bid responses two bidders two imp",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "1"},
                        		{"bidder":"bidderB", "id": "2"}
                    		]
                		}
            		}
    			  },
					{
    			    "id": "imp-id2",
    			    "ext": {
                		"appnexus": {
							"placementId": 123
                		},
                		"prebid": {
                    		"storedbidresponse": [
                        		{"bidder":"bidderA", "id": "3"},
                        		{"bidder":"bidderB", "id": "2"}
                    		]
                		}
            		}
    			  }
    			]}`),
			expectedStoredAuctionResponses: ImpsWithBidResponses{},
			expectedStoredBidResponses: ImpBidderStoredResp{
				"imp-id1": {"bidderA": bidStoredResp1, "bidderB": bidStoredResp2},
				"imp-id2": {"bidderA": bidStoredResp3, "bidderB": bidStoredResp2},
			},
		},
	}

	for _, test := range testCases {
		storedAuctionResponses, storedBidResponses, errorList := ProcessStoredResponses(nil, test.requestJson, fetcher, bidderMap)
		assert.Equal(t, test.expectedStoredAuctionResponses, storedAuctionResponses, "storedAuctionResponses doesn't match: %s\n", test.description)
		assert.Equalf(t, test.expectedStoredBidResponses, storedBidResponses, "storedBidResponses doesn't match: %s\n", test.description)
		assert.Nil(t, errorList, "Error should be nil")
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
