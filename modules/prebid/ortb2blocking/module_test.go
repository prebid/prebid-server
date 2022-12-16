package ortb2blocking

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v17/adcom1"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/hooks/hookanalytics"
	"github.com/prebid/prebid-server/hooks/hookexecution"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/stretchr/testify/assert"
)

var config = json.RawMessage(`
{
  "attributes": {
    "badv": {
      "enforce_blocks": true,
      "block_unknown_adomain": true,
      "blocked_adomain": [
        "a.com",
        "b.com",
        "c.com"
      ],
      "allowed_adomain_for_deals": [
        "z.com",
        "x.com"
      ],
      "action_overrides": {
        "blocked_adomain": [
          {
            "conditions": {
              "bidders": [
                "appnexus",
                "rubicon"
              ],
              "media_types": [
                "video"
              ]
            },
            "override": [
              "a.com",
              "b.com"
            ]
          },
          {
            "conditions": {
              "bidders": [
                "rubicon"
              ]
            },
            "override": [
              "a.com",
              "b.com",
              "c.com",
              "d.com",
              "e.com"
            ]
          }
        ],
        "block_unknown_adomain": [
          {
            "conditions": {
              "bidders": [
                "appnexus"
              ],
              "media_types": [
                "video"
              ]
            },
            "override": true
          }
        ],
        "allowed_adomain_for_deals": [
          {
            "conditions": {
              "deal_ids": [
                "12345678"
              ]
            },
            "override": [
              "a.com"
            ]
          }
        ]
      }
    },
    "bcat": {
      "enforce_blocks": false,
      "block_unknown_adv_cat": false,
      "category_taxonomy": 6,
      "blocked_adv_cat": [
        "IAB-1",
        "IAB-2"
      ],
      "allowed_adv_cat_for_deals": [
        "IAB-1"
      ],
      "action_overrides": {
        "blocked_adv_cat": [
          {
            "conditions": {
              "media_types": [
                "video"
              ]
            },
            "override": [
              "IAB-1",
              "IAB-2",
              "IAB-3",
              "IAB-4"
            ]
          }
        ],
        "enforce_blocks": [
          {
            "conditions": {
              "bidders": [
                "appnexus"
              ]
            },
            "override": true
          }
        ],
        "block_unknown_adv_cat": [
          {
            "conditions": {
              "media_types": [
                "video"
              ]
            },
            "override": true
          }
        ],
        "allowed_adv_cat_for_deals": [
          {
            "conditions": {
              "deal_ids": [
                "1111111"
              ]
            },
            "override": [
              "IAB-1"
            ]
          }
        ]
      }
    },
    "bapp": {
      "enforce_blocks": false,
      "blocked_app": [
        "app1",
        "app2"
      ],
      "action_overrides": {
        "blocked_app": [
          {
            "conditions": {
              "bidders": [
                "appnexus"
              ]
            },
            "override": [
              "app3"
            ]
          }
        ]
      }
    },
    "btype": {
      "blocked_banner_type": [
        3,
        4
      ],
      "action_overrides": {
        "blocked_banner_type": [
          {
            "conditions": {
              "bidders": [
                "appnexus"
              ]
            },
            "override": [
              3,
              4,
              5
            ]
          }
        ]
      }
    },
    "battr": {
      "enforce_blocks": false,
      "blocked_banner_attr": [
        1,
        8,
        9,
        10
      ],
      "action_overrides": {
        "enforce_blocks": [
          {
            "conditions": {
              "bidders": [
                "appnexus"
              ]
            },
            "override": true
          }
        ]
      }
    }
  }
}`)

const bidder string = "appnexus"

const bAdvA string = "a.com"
const bAdvB string = "b.com"
const bAdvC string = "c.com"

const bApp1 string = "app1"
const bApp2 string = "app2"
const bApp3 string = "app3"

const bCat1 string = "IAB-1"
const bCat2 string = "IAB-2"
const bCat3 string = "IAB-3"
const bCat4 string = "IAB-4"
const bCat5 string = "IAB-5"

const catTax adcom1.CategoryTaxonomy = 6

const bType3 openrtb2.BannerAdType = 3
const bType4 openrtb2.BannerAdType = 4
const bType5 openrtb2.BannerAdType = 5

const bAttr1 adcom1.CreativeAttribute = 1
const bAttr8 adcom1.CreativeAttribute = 8
const bAttr9 adcom1.CreativeAttribute = 9
const bAttr10 adcom1.CreativeAttribute = 10

const impID1 = "Some-impid-1"
const impID2 = "Some-impid-2"

func TestHandleBidderRequestHook(t *testing.T) {
	testCases := []struct {
		description        string
		bidder             string
		config             json.RawMessage
		bidRequest         *openrtb2.BidRequest
		expectedBidRequest *openrtb2.BidRequest
		expectedHookResult hookstage.HookResult[hookstage.BidderRequestPayload]
	}{
		{
			description: "Payload changed after successful BidderRequest hook execution",
			bidder:      bidder,
			config:      config,
			bidRequest:  &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}}},
			expectedBidRequest: &openrtb2.BidRequest{
				BAdv:   []string{bAdvA, bAdvB},
				BApp:   []string{bApp3},
				BCat:   []string{bCat1, bCat2, bCat3, bCat4},
				CatTax: catTax,
				Imp: []openrtb2.Imp{
					{
						ID: "ImpID1",
						Banner: &openrtb2.Banner{
							BType: []openrtb2.BannerAdType{bType3, bType4, bType5},
							// default field override is used if no ActionOverrides defined for field
							BAttr: []adcom1.CreativeAttribute{bAttr1, bAttr8, bAttr9, bAttr10},
						},
						Video: &openrtb2.Video{},
					},
				},
			},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{
				ModuleContext: map[string]interface{}{
					bidder: blockingAttributes{
						badv:   []string{bAdvA, bAdvB},
						bapp:   []string{bApp3},
						bcat:   []string{bCat1, bCat2, bCat3, bCat4},
						btype:  map[string][]int{"ImpID1": toInt([]openrtb2.BannerAdType{bType3, bType4, bType5})},
						battr:  map[string][]int{"ImpID1": toInt([]adcom1.CreativeAttribute{bAttr1, bAttr8, bAttr9, bAttr10})},
						cattax: catTax,
					},
				},
			},
		},
		{
			description: "bidrequest fields are not updated if config empty",
			bidder:      bidder,
			config:      json.RawMessage(`{}`),
			bidRequest: &openrtb2.BidRequest{
				BAdv: []string{bAdvA, bAdvC},
				Imp:  []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}},
			},
			expectedBidRequest: &openrtb2.BidRequest{
				// field values preserved if config doesn't provide values for this field
				BAdv: []string{bAdvA, bAdvC},
				Imp:  []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}},
			},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{
				ModuleContext: map[string]interface{}{bidder: blockingAttributes{
					btype: map[string][]int{},
					battr: map[string][]int{},
				}},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			payload := hookstage.BidderRequestPayload{Bidder: test.bidder, BidRequest: test.bidRequest}

			result, err := Builder(nil, nil)
			assert.NoError(t, err, "Failed to build module.")

			module, ok := result.(Module)
			assert.True(t, ok, "Failed to cast module type.")

			hookResult, err := module.HandleBidderRequestHook(
				context.Background(),
				hookstage.ModuleInvocationContext{
					AccountConfig: test.config,
					Endpoint:      hookexecution.EndpointAuction,
					ModuleContext: map[string]interface{}{},
				},
				payload,
			)
			assert.NoError(t, err, "Hook execution failed.")

			// test mutations separately
			for _, mut := range hookResult.ChangeSet.Mutations() {
				_, err := mut.Apply(payload)
				assert.NoError(t, err)
			}
			assert.Equal(t, test.expectedBidRequest, payload.BidRequest, "Invalid BidRequest after executing BidderRequestHook.")

			// reset ChangeSet not to break hookResult assertion, we validated ChangeSet separately
			hookResult.ChangeSet = hookstage.ChangeSet[hookstage.BidderRequestPayload]{}
			assert.Equal(t, test.expectedHookResult, hookResult, "Invalid hook execution result.")
		})
	}
}

func TestHandleRawBidderResponseHook(t *testing.T) {
	testCases := []struct {
		description        string
		payload            hookstage.RawBidderResponsePayload
		config             json.RawMessage
		moduleCtx          hookstage.ModuleContext
		expectedBids       []*adapters.TypedBid
		expectedHookResult hookstage.HookResult[hookstage.RawBidderResponsePayload]
		expectedError      error
	}{
		{
			description: "Payload not changed when empty account config and empty module contexts are provided. Analytic tags have successful records",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ADomain: []string{"foo"}, ImpID: impID1},
				},
			}},
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ADomain: []string{"foo"}, ImpID: impID1},
				},
			},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "Catch error if wrong data has been passed from previous hook. Payload not changed",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ADomain: []string{"foo"}, ImpID: impID1},
				},
			}},
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ADomain: []string{"foo"}, ImpID: impID1},
				},
			},
			moduleCtx:     map[string]interface{}{bidder: "boo"},
			expectedError: hookexecution.NewFailure("could not cast blocking attributes for bidder `appnexus`, module context has incorrect data"),
		},
		{
			description: "Bid blocked by badv attribute check. Payload updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ADomain: []string{"forbidden_domain"}, ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"badv":{"enforce_blocks": true}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			},
			moduleCtx: map[string]interface{}{bidder: blockingAttributes{badv: []string{"forbidden_domain"}}},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status: hookanalytics.ResultStatusBlock,
									Values: map[string]interface{}{
										attributesAnalyticKey: []string{"badv"},
										badvAnalyticKey:       []string{"forbidden_domain"},
									},
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
				DebugMessages: []string{"Bid 1 from bidder appnexus has been rejected, failed checks: [badv]"},
			},
		},
		{
			description: "Bid not blocked because blocking conditions for current bidder do not exist. Payload not updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ADomain: []string{"forbidden_domain"}, ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"badv":{"enforce_blocks": true}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ADomain: []string{"forbidden_domain"}, ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			},
			moduleCtx: map[string]interface{}{"other-bidder": blockingAttributes{badv: []string{"forbidden_domain"}}},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "Bid not blocked because enforce blocking is disabled by account config. Payload not updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ADomain: []string{"forbidden_domain"}, ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"badv":{"enforce_blocks": false}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ADomain: []string{"forbidden_domain"}, ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			},
			moduleCtx: map[string]interface{}{bidder: blockingAttributes{badv: []string{"forbidden_domain"}}},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "Bid not blocked because enforce blocking overridden for given bidder. Payload not updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ADomain: []string{"forbidden_domain"}, ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"badv":{"enforce_blocks": true, "action_overrides": {"enforce_blocks": [{"conditions": {"bidders": ["appnexus"]}, "override": false}]}}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ADomain: []string{"forbidden_domain"}, ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			},
			moduleCtx: map[string]interface{}{bidder: blockingAttributes{badv: []string{"forbidden_domain"}}},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "Bid blocked by badv attribute check (block unknown attributes). Payload updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"badv":{"enforce_blocks": true, "block_unknown_adomain": true}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			},
			moduleCtx: map[string]interface{}{bidder: blockingAttributes{}},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status: hookanalytics.ResultStatusBlock,
									Values: map[string]interface{}{
										attributesAnalyticKey: []string{"badv"},
										badvAnalyticKey:       []string(nil),
									},
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
				DebugMessages: []string{"Bid 1 from bidder appnexus has been rejected, failed checks: [badv]"},
			},
		},
		{
			description: "Bid not blocked because block unknown overridden for given bidder. Payload not updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"badv":{"enforce_blocks": true, "block_unknown_adomain": true, "action_overrides": {"block_unknown_adomain": [{"conditions": {"bidders": ["appnexus"]}, "override": false}]}}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ImpID: impID2},
				},
			},
			moduleCtx: map[string]interface{}{bidder: blockingAttributes{}},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "Bid not blocked due to deal exception. Payload not updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ADomain: []string{"forbidden_domain"}, ImpID: impID1, DealID: "acceptDealID"},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"badv":{"enforce_blocks": true, "action_overrides": {"allowed_adomain_for_deals": [{"conditions": {"deal_ids": ["acceptDealID"]}, "override": ["forbidden_domain"]}]}}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ADomain: []string{"forbidden_domain"}, ImpID: impID1, DealID: "acceptDealID"},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", ADomain: []string{"good_domain"}, ImpID: impID2},
				},
			},
			moduleCtx: map[string]interface{}{bidder: blockingAttributes{badv: []string{"forbidden_domain"}}},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "Expect error if deal_ids not defined in config override conditions. Analytics should have error status tag",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ADomain: []string{"forbidden_domain"}, DealID: "acceptDealID"},
				},
			}},
			config: json.RawMessage(`{"attributes": {"badv": {"enforce_blocks": true, "action_overrides": {"allowed_adomain_for_deals": [{"conditions": {}}]}}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", ADomain: []string{"forbidden_domain"}, DealID: "acceptDealID"},
				},
			},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusError,
						},
					},
				},
			},
			expectedError: hookexecution.NewFailure("failed to process badv block checking: failed to get override for badv.allowed_adomain_for_deals: conditions field in account configuration must contain deal_ids"),
		},
		{
			description: "Bid blocked by bcat attribute check. Payload updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", Cat: []string{"fishing"}, ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", Cat: []string{"moto"}, ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"bcat":{"enforce_blocks": true}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "2", Cat: []string{"moto"}, ImpID: impID2},
				},
			},
			moduleCtx: map[string]interface{}{bidder: blockingAttributes{bcat: []string{"fishing"}}},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status: hookanalytics.ResultStatusBlock,
									Values: map[string]interface{}{
										attributesAnalyticKey: []string{"bcat"},
										"bcat":                []string{"fishing"},
									},
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
				DebugMessages: []string{"Bid 1 from bidder appnexus has been rejected, failed checks: [bcat]"},
			},
		},
		{
			description: "Bid blocked by cattax attribute check. Payload updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", CatTax: 1, ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", CatTax: 2, ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"bcat":{"enforce_blocks": true}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "2", CatTax: 2, ImpID: impID2},
				},
			},
			moduleCtx: map[string]interface{}{bidder: blockingAttributes{cattax: 2}},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status: hookanalytics.ResultStatusBlock,
									Values: map[string]interface{}{
										attributesAnalyticKey: []string{"cattax"},
										cattaxAnalyticKey:     []adcom1.CategoryTaxonomy{1},
									},
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
				DebugMessages: []string{"Bid 1 from bidder appnexus has been rejected, failed checks: [cattax]"},
			},
		},
		{
			description: "Bid blocked by cattax attribute check (the default value used if no blocking attribute passed). Payload updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", CatTax: 1, ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", CatTax: 2, ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"bcat":{"enforce_blocks": true}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", CatTax: 1, ImpID: impID1},
				},
			},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status: hookanalytics.ResultStatusBlock,
									Values: map[string]interface{}{
										attributesAnalyticKey: []string{"cattax"},
										cattaxAnalyticKey:     []adcom1.CategoryTaxonomy{2},
									},
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
				DebugMessages: []string{"Bid 2 from bidder appnexus has been rejected, failed checks: [cattax]"},
			},
		},
		{
			description: "Bid blocked by bapp attribute check. Payload updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", Bundle: "forbidden_bundle", ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", Bundle: "allowed_bundle", ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"bapp":{"enforce_blocks": true}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "2", Bundle: "allowed_bundle", ImpID: impID2},
				},
			},
			moduleCtx: map[string]interface{}{bidder: blockingAttributes{bapp: []string{"forbidden_bundle"}}},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status: hookanalytics.ResultStatusBlock,
									Values: map[string]interface{}{
										attributesAnalyticKey: []string{"bapp"},
										bappAnalyticKey:       []string{"forbidden_bundle"},
									},
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
				DebugMessages: []string{"Bid 1 from bidder appnexus has been rejected, failed checks: [bapp]"},
			},
		},
		{
			description: "Bid blocked by battr attribute check. Payload updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "1", Attr: []adcom1.CreativeAttribute{1}, ImpID: impID1},
				},
				{
					Bid: &openrtb2.Bid{ID: "2", Attr: []adcom1.CreativeAttribute{2}, ImpID: impID2},
				},
			}},
			config: json.RawMessage(`{"attributes":{"battr":{"enforce_blocks": true}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{ID: "2", Attr: []adcom1.CreativeAttribute{2}, ImpID: impID2},
				},
			},
			moduleCtx: map[string]interface{}{bidder: blockingAttributes{battr: map[string][]int{impID1: {1}}}},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status: hookanalytics.ResultStatusBlock,
									Values: map[string]interface{}{
										attributesAnalyticKey: []string{"battr"},
										battrAnalyticKey:      []int{1},
									},
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
				DebugMessages: []string{"Bid 1 from bidder appnexus has been rejected, failed checks: [battr]"},
			},
		},
		{
			description: "Bid blocked by multiple attribute check. Payload updated. Analytic tags successfully collected",
			payload: hookstage.RawBidderResponsePayload{Bidder: bidder, Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:      "1",
						ADomain: []string{"forbidden_domain"},
						Cat:     []string{"fishing"},
						CatTax:  1,
						Bundle:  "forbidden_bundle",
						Attr:    []adcom1.CreativeAttribute{1},
						ImpID:   impID1,
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:      "2",
						ADomain: []string{"allowed_domain"},
						Cat:     []string{"moto"},
						CatTax:  2,
						Bundle:  "allowed_bundle",
						Attr:    []adcom1.CreativeAttribute{2},
						ImpID:   impID2,
					},
				},
			}},
			config: json.RawMessage(`{"attributes":{"badv":{"enforce_blocks": true}, "bcat":{"enforce_blocks": true}, "cattax":{"enforce_blocks": true}, "bapp":{"enforce_blocks": true}, "battr":{"enforce_blocks": true}}}`),
			expectedBids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:      "2",
						ADomain: []string{"allowed_domain"},
						Cat:     []string{"moto"},
						CatTax:  2,
						Bundle:  "allowed_bundle",
						Attr:    []adcom1.CreativeAttribute{2},
						ImpID:   impID2,
					},
				},
			},
			moduleCtx: map[string]interface{}{bidder: blockingAttributes{
				badv:   []string{"forbidden_domain"},
				bcat:   []string{"fishing"},
				cattax: 2,
				bapp:   []string{"forbidden_bundle"},
				battr:  map[string][]int{impID1: {1}}},
			},
			expectedHookResult: hookstage.HookResult[hookstage.RawBidderResponsePayload]{
				AnalyticsTags: hookanalytics.Analytics{
					Activities: []hookanalytics.Activity{
						{
							Name:   enforceBlockingTag,
							Status: hookanalytics.ActivityStatusSuccess,
							Results: []hookanalytics.Result{
								{
									Status: hookanalytics.ResultStatusBlock,
									Values: map[string]interface{}{
										attributesAnalyticKey: []string{"badv", "bcat", "cattax", "bapp", "battr"},
										badvAnalyticKey:       []string{"forbidden_domain"},
										cattaxAnalyticKey:     []adcom1.CategoryTaxonomy{1},
										bappAnalyticKey:       []string{"forbidden_bundle"},
										battrAnalyticKey:      []int{1},
									},
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID1}},
								},
								{
									Status:    hookanalytics.ResultStatusAllow,
									AppliedTo: hookanalytics.AppliedTo{Bidders: []string{bidder}, ImpIds: []string{impID2}},
								},
							},
						},
					},
				},
				DebugMessages: []string{"Bid 1 from bidder appnexus has been rejected, failed checks: [badv bcat cattax bapp battr]"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			result, err := Builder(nil, nil)
			assert.NoError(t, err, "Failed to build module.")

			module, ok := result.(Module)
			assert.True(t, ok, "Failed to cast module type.")

			hookResult, err := module.HandleRawBidderResponseHook(
				context.Background(),
				hookstage.ModuleInvocationContext{
					AccountConfig: test.config,
					Endpoint:      hookexecution.EndpointAuction,
					ModuleContext: test.moduleCtx,
				},
				test.payload,
			)
			assert.Equal(t, test.expectedError, err, "Invalid hook execution error.")

			// test mutations separately
			for _, mut := range hookResult.ChangeSet.Mutations() {
				newPayload, err := mut.Apply(test.payload)
				assert.NoError(t, err)
				test.payload = newPayload
			}
			assert.Equal(t, test.expectedBids, test.payload.Bids, "Invalid Bids returned after executing RawBidderResponse hook.")

			// reset ChangeSet not to break hookResult assertion, we validated ChangeSet separately
			hookResult.ChangeSet = hookstage.ChangeSet[hookstage.RawBidderResponsePayload]{}
			assert.Equal(t, test.expectedHookResult, hookResult, "Invalid hook execution result.")
		})
	}
}
