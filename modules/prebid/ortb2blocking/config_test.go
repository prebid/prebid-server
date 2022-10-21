package ortb2blocking

import (
	"testing"

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
	  "action_overrides": [
		{
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
	  ]
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
	  "action_overrides": [
		{
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
	  ]
	},
	"bapp": {
	  "enforce_blocks": false,
	  "blocked_app": [
		"app1",
		"app2"
	  ],
	  "action_overrides": [
		{
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
	  ]
	},
	"btype": {
	  "blocked_banner_type": [
		3,
		4
	  ],
	  "action_overrides": [
		{
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
	  ]
	},
	"battr": {
	  "enforce_blocks": false,
	  "blocked_banner_attr": [
		1,
		8,
		9,
		10
	  ],
	  "action_overrides": [
		{
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
	  ]
	}
  }
}
`)

func TestNewConfig(t *testing.T) {
	c, err := newConfig(fullConfig)
	require.NoError(t, err)

	// todo: complete all tests

	assert.Equal(t, []int{3, 4, 5}, c.Attributes.Btype.ActionOverrides[0].BlockedBannerType[0].Override.Ids)
	assert.True(t, c.Attributes.Battr.ActionOverrides[0].EnforceBlocks[0].Override.IsOn)
}
