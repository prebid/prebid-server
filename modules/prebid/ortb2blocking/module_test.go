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

func TestHandleBidderRequestHook(t *testing.T) {
	bidRequest := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}}}
	expectedBidRequest := &openrtb2.BidRequest{
		BAdv:   []string{"a.com", "b.com"},
		BApp:   []string{"app3"},
		BCat:   []string{"IAB-1", "IAB-2", "IAB-3", "IAB-4"},
		CatTax: adcom1.CategoryTaxonomy(6),
		Imp: []openrtb2.Imp{
			{
				ID: "ImpID1",
				Banner: &openrtb2.Banner{
					BType: []openrtb2.BannerAdType{3, 4, 5},
					BAttr: []adcom1.CreativeAttribute{1, 8, 9, 10},
				},
				Video: &openrtb2.Video{},
			},
		},
	}

	payload := hookstage.BidderRequestPayload{Bidder: "appnexus", BidRequest: bidRequest}

	result, err := Builder(nil, nil)
	assert.NoError(t, err, "Failed to build module.")

	module, ok := result.(Module)
	assert.True(t, ok, "Failed to cast module type.")

	hookResult, err := module.HandleBidderRequestHook(
		context.Background(),
		hookstage.ModuleInvocationContext{AccountConfig: config,
			Endpoint:      hookexecution.EndpointAuction,
			ModuleContext: map[string]interface{}{},
		},
		payload,
	)

	assert.NoError(t, err, "Hook execution failed.")
	assert.False(t, hookResult.Reject, "Reject not expected.")

	// todo: assert hookResult

	// test mutations
	for _, mut := range hookResult.ChangeSet.Mutations() {
		_, err := mut.Apply(payload)
		assert.NoError(t, err)
	}

	assert.Equal(t, expectedBidRequest, payload.BidRequest, "Invalid BidRequest after executing BidderRequestHook.")
}
