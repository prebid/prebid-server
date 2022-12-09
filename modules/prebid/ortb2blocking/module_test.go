package ortb2blocking

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v17/adcom1"
	"github.com/prebid/openrtb/v17/openrtb2"
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

type numeric interface {
	openrtb2.BannerAdType | adcom1.CreativeAttribute
}

func toInt[T numeric](values []T) []int {
	ints := make([]int, len(values))
	for i := range values {
		ints[i] = int(values[i])
	}
	return ints
}
