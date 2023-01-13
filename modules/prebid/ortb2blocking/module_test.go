package ortb2blocking

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/openrtb/v17/adcom1"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/hooks/hookexecution"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/modules/moduledeps"
	"github.com/stretchr/testify/assert"
)

var testConfig = json.RawMessage(`
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
          },
          {
            "conditions": {
              "bidders": [
                "appnexus"
              ],
			  "media_types": [
				"video"
			  ]
            },
            "override": [
              "app1"
            ]
          },
          {
            "conditions": {
			  "media_types": [
				"video"
			  ]
            },
            "override": [
              "app1"
            ]
          }
        ]
      }
    },
    "btype": {
      "blocked_banner_type": [],
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
          },
          {
            "conditions": {
              "bidders": [
                "appnexus"
              ],
			  "media_types": [
				"video"
			  ]
            },
            "override": [
              3
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
const bAdvD string = "d.com"
const bAdvE string = "e.com"

const bApp1 string = "app1"
const bApp2 string = "app2"
const bApp3 string = "app3"

const bCat1 string = "IAB-1"
const bCat2 string = "IAB-2"
const bCat3 string = "IAB-3"
const bCat4 string = "IAB-4"

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
		expectedError      error
	}{
		{
			description: "Payload changed after successful BidderRequest hook execution",
			bidder:      bidder,
			config:      testConfig,
			bidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "ImpID1",
						Audio:  &openrtb2.Audio{},
						Banner: &openrtb2.Banner{},
						Native: &openrtb2.Native{},
						Video:  &openrtb2.Video{},
					},
					{
						ID:    "ImpID2",
						Audio: &openrtb2.Audio{},
					},
				},
			},
			expectedBidRequest: &openrtb2.BidRequest{
				BAdv: []string{bAdvA, bAdvB},
				// BApp contains only first specific override (other overrides ignored)
				BApp:   []string{bApp3},
				BCat:   []string{bCat1, bCat2, bCat3, bCat4},
				CatTax: catTax,
				Imp: []openrtb2.Imp{
					{
						ID:    "ImpID1",
						Audio: &openrtb2.Audio{},
						Banner: &openrtb2.Banner{
							BType: []openrtb2.BannerAdType{bType3, bType4, bType5},
							// default field override is used if no ActionOverrides defined for field
							BAttr: []adcom1.CreativeAttribute{bAttr1, bAttr8, bAttr9, bAttr10},
						},
						Native: &openrtb2.Native{},
						Video:  &openrtb2.Video{},
					},
					{
						ID:    "ImpID2",
						Audio: &openrtb2.Audio{},
						Banner: &openrtb2.Banner{
							BType: []openrtb2.BannerAdType{bType3, bType4, bType5},
							// default field override is used if no ActionOverrides defined for field
							BAttr: []adcom1.CreativeAttribute{bAttr1, bAttr8, bAttr9, bAttr10},
						},
					},
				},
			},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{
				ModuleContext: map[string]interface{}{
					bidder: blockingAttributes{
						bAdv: []string{bAdvA, bAdvB},
						bApp: []string{bApp3},
						bCat: []string{bCat1, bCat2, bCat3, bCat4},
						bType: map[string][]int{
							"ImpID1": toInt([]openrtb2.BannerAdType{bType3, bType4, bType5}),
							"ImpID2": toInt([]openrtb2.BannerAdType{bType3, bType4, bType5}),
						},
						bAttr: map[string][]int{
							"ImpID1": toInt([]adcom1.CreativeAttribute{bAttr1, bAttr8, bAttr9, bAttr10}),
							"ImpID2": toInt([]adcom1.CreativeAttribute{bAttr1, bAttr8, bAttr9, bAttr10}),
						},
						catTax: catTax,
					},
				},
				Warnings: []string{
					// multiple warnings may be added (per condition)
					"More than one condition matches request. Bidder: appnexus, request media types: audio, banner, native, video",
					"More than one condition matches request. Bidder: appnexus, request media types: audio, banner, native, video",
				},
			},
			expectedError: nil,
		},
		{
			description: "Payload changed after successful BidderRequest hook execution for default config",
			bidder:      bidder,
			config:      json.RawMessage(`{"attributes": {"badv": {"enforce_blocks": true, "block_unknown_adomain": true, "blocked_adomain": ["a.com","b.com","c.com"]}}}`),
			bidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "ImpID1",
						Audio: &openrtb2.Audio{},
					},
				},
			},
			expectedBidRequest: &openrtb2.BidRequest{
				BAdv: []string{bAdvA, bAdvB, bAdvC},
				Imp: []openrtb2.Imp{
					{
						ID:    "ImpID1",
						Audio: &openrtb2.Audio{},
					},
				},
			},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{
				ModuleContext: map[string]interface{}{
					bidder: blockingAttributes{
						bAdv:  []string{bAdvA, bAdvB, bAdvC},
						bType: map[string][]int{},
						bAttr: map[string][]int{},
					},
				},
			},
			expectedError: nil,
		},
		{
			description: "BidderRequest attributes not updated if they already present in BidderRequest",
			bidder:      bidder,
			config:      testConfig,
			bidRequest: &openrtb2.BidRequest{
				BAdv:   []string{"existing.com"},
				BApp:   []string{"existingApp", "existingApp2"},
				BCat:   []string{"Existing-IAB-1", "Existing-IAB-2"},
				CatTax: adcom1.CatTaxIABContent10,
				Imp: []openrtb2.Imp{
					{
						ID:    "ImpID1",
						Audio: &openrtb2.Audio{},
						Banner: &openrtb2.Banner{
							BType: []openrtb2.BannerAdType{openrtb2.BannerAdTypeXHTMLTextAd},
							BAttr: []adcom1.CreativeAttribute{adcom1.AttrSurvey},
						},
						Native: &openrtb2.Native{},
						Video:  &openrtb2.Video{},
					},
				},
			},
			expectedBidRequest: &openrtb2.BidRequest{
				BAdv:   []string{"existing.com"},
				BApp:   []string{"existingApp", "existingApp2"},
				BCat:   []string{"Existing-IAB-1", "Existing-IAB-2"},
				CatTax: adcom1.CatTaxIABContent10,
				Imp: []openrtb2.Imp{
					{
						ID:    "ImpID1",
						Audio: &openrtb2.Audio{},
						Banner: &openrtb2.Banner{
							BType: []openrtb2.BannerAdType{openrtb2.BannerAdTypeXHTMLTextAd},
							BAttr: []adcom1.CreativeAttribute{adcom1.AttrSurvey},
						},
						Native: &openrtb2.Native{},
						Video:  &openrtb2.Video{},
					},
				},
			},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{
				ModuleContext: map[string]interface{}{bidder: blockingAttributes{
					bType: map[string][]int{},
					bAttr: map[string][]int{},
				}},
			},
			expectedError: nil,
		},
		{
			description: "BidRequest fields are not updated if config empty",
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
					bType: map[string][]int{},
					bAttr: map[string][]int{},
				}},
			},
			expectedError: nil,
		},
		{
			description: "Expect empty hook execution result if empty module config provided",
			bidder:      bidder,
			config:      json.RawMessage(""),
			bidRequest: &openrtb2.BidRequest{
				BAdv: []string{bAdvA, bAdvC},
				Imp:  []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}},
			},
			expectedBidRequest: &openrtb2.BidRequest{
				BAdv: []string{bAdvA, bAdvC},
				Imp:  []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}},
			},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{},
			expectedError:      nil,
		},
		{
			description:        "Expect error on config parsing failure",
			bidder:             bidder,
			config:             json.RawMessage("..."),
			bidRequest:         &openrtb2.BidRequest{},
			expectedBidRequest: &openrtb2.BidRequest{},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{},
			expectedError:      errors.New("failed to parse config: invalid character '.' looking for beginning of value"),
		},
		{
			description:        "Expect error if nil BidRequest provided",
			bidder:             bidder,
			config:             testConfig,
			bidRequest:         nil,
			expectedBidRequest: nil,
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{},
			expectedError:      hookexecution.NewFailure("empty BidRequest provided"),
		},
		{
			description:        "Expect baadv error if bidders and media_types not defined in config conditions",
			bidder:             bidder,
			config:             json.RawMessage(`{"attributes": {"badv": {"action_overrides": {"blocked_adomain": [{"conditions": {}}]}}}}`),
			bidRequest:         &openrtb2.BidRequest{},
			expectedBidRequest: &openrtb2.BidRequest{},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{},
			expectedError:      hookexecution.NewFailure("failed to update badv field: failed to get override for badv.blocked_adomain: bidders and media_types absent from conditions, at least one of the fields must be present"),
		},
		{
			description:        "Expect bapp error if bidders and media_types not defined in config conditions",
			bidder:             bidder,
			config:             json.RawMessage(`{"attributes": {"bapp": {"action_overrides": {"blocked_app": [{"conditions": {}}]}}}}`),
			bidRequest:         &openrtb2.BidRequest{},
			expectedBidRequest: &openrtb2.BidRequest{},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{},
			expectedError:      hookexecution.NewFailure("failed to update bapp field: failed to get override for bapp.blocked_app: bidders and media_types absent from conditions, at least one of the fields must be present"),
		},
		{
			description:        "Expect bcat error if bidders and media_types not defined in config conditions",
			bidder:             bidder,
			config:             json.RawMessage(`{"attributes": {"bcat": {"action_overrides": {"blocked_adv_cat": [{"conditions": {}}]}}}}`),
			bidRequest:         &openrtb2.BidRequest{},
			expectedBidRequest: &openrtb2.BidRequest{},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{},
			expectedError:      hookexecution.NewFailure("failed to update bcat field: failed to get override for bcat.blocked_adv_cat: bidders and media_types absent from conditions, at least one of the fields must be present"),
		},
		{
			description:        "Expect btype error if bidders and media_types not defined in config conditions",
			bidder:             bidder,
			config:             json.RawMessage(`{"attributes": {"btype": {"action_overrides": {"blocked_banner_type": [{"conditions": {}}]}}}}`),
			bidRequest:         &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}}},
			expectedBidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}}},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{},
			expectedError:      hookexecution.NewFailure("failed to update btype field: failed to get override for imp.*.banner.btype: bidders and media_types absent from conditions, at least one of the fields must be present"),
		},
		{
			description:        "Expect battr error if bidders and media_types not defined in config conditions",
			bidder:             bidder,
			config:             json.RawMessage(`{"attributes": {"battr": {"action_overrides": {"blocked_banner_attr": [{"conditions": {}}]}}}}`),
			bidRequest:         &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}}},
			expectedBidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}}},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{},
			expectedError:      hookexecution.NewFailure("failed to update battr field: failed to get override for imp.*.banner.battr: bidders and media_types absent from conditions, at least one of the fields must be present"),
		},
		{
			description:        "Expect error if override.names empty in config conditions",
			bidder:             bidder,
			config:             json.RawMessage(`{"attributes": {"badv": {"action_overrides": {"blocked_adomain": [{"conditions": {"bidders": ["appnexus"]}, "override": {}}]}}}}`),
			bidRequest:         &openrtb2.BidRequest{},
			expectedBidRequest: &openrtb2.BidRequest{},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{},
			expectedError:      hookexecution.NewFailure("failed to update badv field: failed to get override for badv.blocked_adomain: empty override field"),
		},
		{
			description:        "Expect error if override.ids empty in config conditions",
			bidder:             bidder,
			config:             json.RawMessage(`{"attributes": {"battr": {"action_overrides": {"blocked_banner_attr": [{"conditions": {"bidders": ["appnexus"]}, "override": {}}]}}}}`),
			bidRequest:         &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}}},
			expectedBidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpID1", Video: &openrtb2.Video{}}}},
			expectedHookResult: hookstage.HookResult[hookstage.BidderRequestPayload]{},
			expectedError:      hookexecution.NewFailure("failed to update battr field: failed to get override for imp.*.banner.battr: empty override field"),
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			payload := hookstage.BidderRequestPayload{Bidder: test.bidder, BidRequest: test.bidRequest}

			result, err := Builder(nil, moduledeps.ModuleDeps{})
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
			assert.Equal(t, test.expectedError, err, "Invalid hook execution error.")

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
