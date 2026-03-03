package ortb2blocking

import (
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fullConfig = []byte(`
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
                "bidderA",
                "bidderB"
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
                "bidderB"
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
                "bidderA"
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
                "bidderA"
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
                "bidderA"
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
                "bidderA"
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
                "bidderA"
              ]
            },
            "override": true
          }
        ]
      }
    }
  }
}
`)

func TestNewConfig(t *testing.T) {
	c, err := newConfig(fullConfig)
	require.NoError(t, err)

	// badv
	assert.True(t, c.Attributes.Badv.EnforceBlocks, "attributes.badv.enforce_blocks")
	assert.True(t, c.Attributes.Badv.BlockUnknownAdomain, "attributes.badv.block_unknown_adomain")
	assert.Equal(t, []string{"a.com", "b.com", "c.com"}, c.Attributes.Badv.BlockedAdomain, "attributes.badv.blocked_adomain")
	assert.Equal(t, []string{"z.com", "x.com"}, c.Attributes.Badv.AllowedAdomainForDeals, "attributes.badv.allowed_adomain_for_deals")

	assert.Empty(t, c.Attributes.Badv.ActionOverrides.EnforceBlocks, "attributes.badv.action_overrides[0].enforce_blocks")

	assert.Equal(t, []string{"bidderA", "bidderB"}, c.Attributes.Badv.ActionOverrides.BlockedAdomain[0].Conditions.Bidders, "attributes.badv.action_overrides[0].blocked_adomain[0].conditions.bidders")
	assert.Equal(t, []string{"video"}, c.Attributes.Badv.ActionOverrides.BlockedAdomain[0].Conditions.MediaTypes, "attributes.badv.action_overrides[0].blocked_adomain[0].conditions.media_types")
	assert.Empty(t, c.Attributes.Badv.ActionOverrides.BlockedAdomain[0].Conditions.DealIds, "attributes.badv.action_overrides[0].blocked_adomain[0].conditions.deal_ids")
	assert.False(t, c.Attributes.Badv.ActionOverrides.BlockedAdomain[0].Override.IsActive, "attributes.badv.action_overrides[0].blocked_adomain[0].override")
	assert.Empty(t, c.Attributes.Badv.ActionOverrides.BlockedAdomain[0].Override.Ids, "attributes.badv.action_overrides[0].blocked_adomain[0].override")
	assert.Equal(t, []string{"a.com", "b.com"}, c.Attributes.Badv.ActionOverrides.BlockedAdomain[0].Override.Names, "attributes.badv.action_overrides[0].blocked_adomain[0].override")

	assert.Equal(t, []string{"bidderB"}, c.Attributes.Badv.ActionOverrides.BlockedAdomain[1].Conditions.Bidders, "attributes.badv.action_overrides[0].blocked_adomain[1].conditions.bidders")
	assert.Empty(t, c.Attributes.Badv.ActionOverrides.BlockedAdomain[1].Conditions.MediaTypes, "attributes.badv.action_overrides[0].blocked_adomain[1].conditions.media_types")
	assert.Empty(t, c.Attributes.Badv.ActionOverrides.BlockedAdomain[1].Conditions.DealIds, "attributes.badv.action_overrides[0].blocked_adomain[1].conditions.deal_ids")
	assert.False(t, c.Attributes.Badv.ActionOverrides.BlockedAdomain[1].Override.IsActive, "attributes.badv.action_overrides[0].blocked_adomain[1].override")
	assert.Empty(t, c.Attributes.Badv.ActionOverrides.BlockedAdomain[1].Override.Ids, "attributes.badv.action_overrides[0].blocked_adomain[1].override")
	assert.Equal(t, []string{"a.com", "b.com", "c.com", "d.com", "e.com"}, c.Attributes.Badv.ActionOverrides.BlockedAdomain[1].Override.Names, "attributes.badv.action_overrides[0].blocked_adomain[1].override")

	assert.Equal(t, []string{"bidderA"}, c.Attributes.Badv.ActionOverrides.BlockUnknownAdomain[0].Conditions.Bidders, "attributes.badv.action_overrides[0].block_unknown_adomain[0].conditions.bidders")
	assert.Equal(t, []string{"video"}, c.Attributes.Badv.ActionOverrides.BlockUnknownAdomain[0].Conditions.MediaTypes, "attributes.badv.action_overrides[0].block_unknown_adomain[0].conditions.media_types")
	assert.True(t, c.Attributes.Badv.ActionOverrides.BlockUnknownAdomain[0].Override.IsActive, "attributes.badv.action_overrides[0].block_unknown_adomain[0].override")
	assert.Empty(t, c.Attributes.Badv.ActionOverrides.BlockUnknownAdomain[0].Override.Ids, "attributes.badv.action_overrides[0].block_unknown_adomain[0].override")
	assert.Empty(t, c.Attributes.Badv.ActionOverrides.BlockUnknownAdomain[0].Override.Names, "attributes.badv.action_overrides[0].block_unknown_adomain[0].override")

	assert.Equal(t, []string{"12345678"}, c.Attributes.Badv.ActionOverrides.AllowedAdomainForDeals[0].Conditions.DealIds, "attributes.badv.action_overrides[0].allowed_adomain_for_deals[0].conditions.deal_ids")
	assert.False(t, c.Attributes.Badv.ActionOverrides.AllowedAdomainForDeals[0].Override.IsActive, "attributes.badv.action_overrides[0].allowed_adomain_for_deals[0].override")
	assert.Empty(t, c.Attributes.Badv.ActionOverrides.AllowedAdomainForDeals[0].Override.Ids, "attributes.badv.action_overrides[0].allowed_adomain_for_deals[0].override")
	assert.Equal(t, []string{"a.com"}, c.Attributes.Badv.ActionOverrides.AllowedAdomainForDeals[0].Override.Names, "attributes.badv.action_overrides[0].allowed_adomain_for_deals[0].override")

	// bcat
	assert.False(t, c.Attributes.Bcat.EnforceBlocks, "attributes.bcat.enforce_blocks")
	assert.False(t, c.Attributes.Bcat.BlockUnknownAdvCat, "attributes.bcat.block_unknown_adv_cat")
	assert.Equal(t, adcom1.CategoryTaxonomy(6), c.Attributes.Bcat.CategoryTaxonomy, "attributes.bcat.category_taxonomy")
	assert.Equal(t, []string{"IAB-1", "IAB-2"}, c.Attributes.Bcat.BlockedAdvCat, "attributes.bcat.blocked_adv_cat")
	assert.Equal(t, []string{"IAB-1"}, c.Attributes.Bcat.AllowedAdvCatForDeals, "attributes.bcat.allowed_adv_cat_for_deals")

	assert.Equal(t, []string{"video"}, c.Attributes.Bcat.ActionOverrides.BlockedAdvCat[0].Conditions.MediaTypes, "attributes.bcat.action_overrides[0].blocked_adv_cat[0].conditions.media_types")
	assert.Empty(t, c.Attributes.Bcat.ActionOverrides.BlockedAdvCat[0].Conditions.Bidders, "attributes.bcat.action_overrides[0].blocked_adv_cat[0].conditions.bidders")
	assert.Empty(t, c.Attributes.Bcat.ActionOverrides.BlockedAdvCat[0].Conditions.DealIds, "attributes.bcat.action_overrides[0].blocked_adv_cat[0].conditions.deal_ids")
	assert.False(t, c.Attributes.Bcat.ActionOverrides.BlockedAdvCat[0].Override.IsActive, "attributes.bcat.action_overrides[0].blocked_adv_cat[0].override")
	assert.Empty(t, c.Attributes.Bcat.ActionOverrides.BlockedAdvCat[0].Override.Ids, "attributes.bcat.action_overrides[0].blocked_adv_cat[0].override")
	assert.Equal(t, []string{"IAB-1", "IAB-2", "IAB-3", "IAB-4"}, c.Attributes.Bcat.ActionOverrides.BlockedAdvCat[0].Override.Names, "attributes.bcat.action_overrides[0].blocked_adv_cat[0].override")

	assert.Equal(t, []string{"bidderA"}, c.Attributes.Bcat.ActionOverrides.EnforceBlocks[0].Conditions.Bidders, "attributes.bcat.action_overrides[0].enforce_blocks[0].conditions.bidders")
	assert.True(t, c.Attributes.Bcat.ActionOverrides.EnforceBlocks[0].Override.IsActive, "attributes.bcat.action_overrides[0].enforce_blocks[0].override")
	assert.Empty(t, c.Attributes.Bcat.ActionOverrides.EnforceBlocks[0].Override.Ids, "attributes.bcat.action_overrides[0].enforce_blocks[0].override")
	assert.Empty(t, c.Attributes.Bcat.ActionOverrides.EnforceBlocks[0].Override.Names, "attributes.bcat.action_overrides[0].enforce_blocks[0].override")

	assert.Equal(t, []string{"video"}, c.Attributes.Bcat.ActionOverrides.BlockUnknownAdvCat[0].Conditions.MediaTypes, "attributes.bcat.action_overrides[0].block_unknown_adv_cat[0].conditions.media_types")
	assert.True(t, c.Attributes.Bcat.ActionOverrides.BlockUnknownAdvCat[0].Override.IsActive, "attributes.bcat.action_overrides[0].block_unknown_adv_cat[0].override")
	assert.Empty(t, c.Attributes.Bcat.ActionOverrides.BlockUnknownAdvCat[0].Override.Ids, "attributes.bcat.action_overrides[0].block_unknown_adv_cat[0].override")
	assert.Empty(t, c.Attributes.Bcat.ActionOverrides.BlockUnknownAdvCat[0].Override.Names, "attributes.bcat.action_overrides[0].block_unknown_adv_cat[0].override")

	assert.Equal(t, []string{"1111111"}, c.Attributes.Bcat.ActionOverrides.AllowedAdvCatForDeals[0].Conditions.DealIds, "attributes.bcat.action_overrides[0].allowed_adv_cat_for_deals[0].conditions.deal_ids")
	assert.False(t, c.Attributes.Bcat.ActionOverrides.AllowedAdvCatForDeals[0].Override.IsActive, "attributes.bcat.action_overrides[0].allowed_adv_cat_for_deals[0].override")
	assert.Empty(t, c.Attributes.Bcat.ActionOverrides.AllowedAdvCatForDeals[0].Override.Ids, "attributes.bcat.action_overrides[0].allowed_adv_cat_for_deals[0].override")
	assert.Equal(t, []string{"IAB-1"}, c.Attributes.Bcat.ActionOverrides.AllowedAdvCatForDeals[0].Override.Names, "attributes.bcat.action_overrides[0].allowed_adv_cat_for_deals[0].override")

	// bapp
	assert.False(t, c.Attributes.Bapp.EnforceBlocks, "attributes.bapp.enforce_blocks")
	assert.Equal(t, []string{"app1", "app2"}, c.Attributes.Bapp.BlockedApp, "attributes.bapp.blocked_app")
	assert.Empty(t, c.Attributes.Bapp.AllowedAppForDeals, "attributes.bapp.allowed_app_for_deals")

	assert.Empty(t, c.Attributes.Bapp.ActionOverrides.AllowedAppForDeals, "attributes.bapp.action_overrides[0].allowed_app_for_deals")
	assert.Empty(t, c.Attributes.Bapp.ActionOverrides.EnforceBlocks, "attributes.bapp.action_overrides[0].enforce_blocks")

	assert.Equal(t, []string{"bidderA"}, c.Attributes.Bapp.ActionOverrides.BlockedApp[0].Conditions.Bidders, "attributes.bapp.action_overrides[0].blocked_app[0].conditions.bidders")
	assert.False(t, c.Attributes.Bapp.ActionOverrides.BlockedApp[0].Override.IsActive, "attributes.bapp.action_overrides[0].blocked_app[0].override")
	assert.Empty(t, c.Attributes.Bapp.ActionOverrides.BlockedApp[0].Override.Ids, "attributes.bapp.action_overrides[0].blocked_app[0].override")
	assert.Equal(t, []string{"app3"}, c.Attributes.Bapp.ActionOverrides.BlockedApp[0].Override.Names, "attributes.bapp.action_overrides[0].blocked_app[0].override")

	// btype
	assert.Equal(t, []int{3, 4}, c.Attributes.Btype.BlockedBannerType, "attributes.btype.blocked_banner_type")

	assert.Equal(t, []string{"bidderA"}, c.Attributes.Btype.ActionOverrides.BlockedBannerType[0].Conditions.Bidders, "attributes.btype.action_overrides[0].blocked_banner_type[0].conditions.bidders")
	assert.Equal(t, []int{3, 4, 5}, c.Attributes.Btype.ActionOverrides.BlockedBannerType[0].Override.Ids, "attributes.btype.action_overrides[0].blocked_banner_type[0].override")
	assert.Empty(t, c.Attributes.Btype.ActionOverrides.BlockedBannerType[0].Override.Names, "attributes.btype.action_overrides[0].blocked_banner_type[0].override")
	assert.False(t, c.Attributes.Btype.ActionOverrides.BlockedBannerType[0].Override.IsActive, "attributes.btype.action_overrides[0].blocked_banner_type[0].override")

	// battr
	assert.Empty(t, c.Attributes.Battr.AllowedBannerAttrForDeals, "attributes.battr.allowed_banner_attr_for_deals")
	assert.False(t, c.Attributes.Battr.EnforceBlocks, "attributes.battr.enforce_blocks")
	assert.Equal(t, []int{1, 8, 9, 10}, c.Attributes.Battr.BlockedBannerAttr, "attributes.battr.blocked_banner_attr")

	assert.Empty(t, c.Attributes.Battr.ActionOverrides.AllowedBannerAttrForDeals, "attributes.battr.action_overrides[0].allowed_banner_attr_for_deals")
	assert.Empty(t, c.Attributes.Battr.ActionOverrides.BlockedBannerAttr, "attributes.battr.action_overrides[0].blocked_banner_attr")

	assert.Equal(t, []string{"bidderA"}, c.Attributes.Battr.ActionOverrides.EnforceBlocks[0].Conditions.Bidders, "attributes.battr.action_overrides[0].enforce_blocks[0].conditions.bidders")
	assert.True(t, c.Attributes.Battr.ActionOverrides.EnforceBlocks[0].Override.IsActive, "attributes.battr.action_overrides[0].enforce_blocks[0].override")
	assert.Empty(t, c.Attributes.Battr.ActionOverrides.EnforceBlocks[0].Override.Ids, "attributes.battr.action_overrides[0].enforce_blocks[0].override")
	assert.Empty(t, c.Attributes.Battr.ActionOverrides.EnforceBlocks[0].Override.Names, "attributes.battr.action_overrides[0].enforce_blocks[0].override")
}

func TestOverride_UnmarshalJSON(t *testing.T) {
	// error on invalid JSON
	override := Override{}
	assert.Error(t, override.UnmarshalJSON([]byte("...")), "Error expected on invalid JSON.")

	// expect IsActive to be initialized correctly
	override = Override{}
	assert.NoError(t, override.UnmarshalJSON([]byte("true")), "Failed to unmarshal bool override.")
	assert.Equal(t, Override{IsActive: true}, override, "Override.IsActive expected to be true.")

	// expect IDs to be initialized correctly
	override = Override{}
	assert.NoError(t, override.UnmarshalJSON([]byte("[1, 2, 3]")), "Failed to unmarshal override with IDs.")
	assert.Equal(t, Override{Ids: []int{1, 2, 3}}, override, "Invalid override.IDs.")

	// expect Names to be initialized correctly
	override = Override{}
	assert.NoError(t, override.UnmarshalJSON([]byte(`["one", "two"]`)), "Failed to unmarshal override with Names.")
	assert.Equal(t, Override{Names: []string{"one", "two"}}, override, "Invalid override.Names.")

	// expect empty override on ignored JSON
	override = Override{}
	assert.NoError(t, override.UnmarshalJSON([]byte(`"string"`)), "Failed to unmarshal override with ignored value.")
	assert.Equal(t, Override{}, override, "Empty override expected.")
}
