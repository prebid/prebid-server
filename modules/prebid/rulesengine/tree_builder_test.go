package rulesengine

import (
	"encoding/json"
	"testing"

	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

func TestBuildTreeFullConfig(t *testing.T) {

	var modelGroup config.ModelGroup
	err := jsonutil.Unmarshal(GetConf(), &modelGroup)
	assert.NoError(t, err)

	builder := &treeBuilder[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
		Config:            modelGroup,
		SchemaFuncFactory: rules.NewRequestSchemaFunction,
		ResultFuncFactory: NewProcessedAuctionRequestResultFunction,
	}
	tree := rules.Tree[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
		Root: &rules.Node[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
	}

	err = builder.Build(&tree)
	assert.NoError(t, err, "tree builder error not expected")

	assert.Equal(t, 1, len(tree.DefaultFunctions), "default functions count mismatch")

	assert.Equal(t, ExcludeBiddersName, tree.DefaultFunctions[0].Name(), "default function name mismatch")

	assert.Equal(t, rules.DeviceCountryIn, tree.Root.SchemaFunction.Name(), "schema function name mismatch")
	assert.Empty(t, tree.Root.ResultFunctions, "root result functions should be empty")
	assert.Equal(t, 2, len(tree.Root.Children), "wrong number of children")

	assert.Equal(t, 2, len(tree.Root.Children["true"].Children), "wrong number of children for 'true' node")
	assert.Equal(t, rules.DataCenterIn, tree.Root.Children["true"].SchemaFunction.Name(), "wrong schema function name on 'true' node")

	assert.Equal(t, 1, len(tree.Root.Children["false"].Children), "wrong number of children for 'false' node")
	assert.Equal(t, rules.DataCenterIn, tree.Root.Children["false"].SchemaFunction.Name(), "wrong schema function name on 'false' node")

	assert.Equal(t, 1, len(tree.Root.Children["true"].Children["true"].Children))
	assert.Equal(t, rules.Channel, tree.Root.Children["true"].Children["true"].SchemaFunction.Name())

	assert.Equal(t, 1, len(tree.Root.Children["true"].Children["false"].Children))
	assert.Equal(t, rules.Channel, tree.Root.Children["true"].Children["false"].SchemaFunction.Name())

	assert.Equal(t, 1, len(tree.Root.Children["false"].Children["false"].Children))
	assert.Equal(t, rules.Channel, tree.Root.Children["false"].Children["false"].SchemaFunction.Name())

	assert.Equal(t, 1, len(tree.Root.Children["true"].Children["true"].Children["amp"].ResultFunctions))
	assert.Equal(t, ExcludeBiddersName, tree.Root.Children["true"].Children["true"].Children["amp"].ResultFunctions[0].Name())

	assert.Equal(t, 1, len(tree.Root.Children["true"].Children["false"].Children["web"].ResultFunctions))
	assert.Equal(t, ExcludeBiddersName, tree.Root.Children["true"].Children["false"].Children["web"].ResultFunctions[0].Name())

	assert.Equal(t, 1, len(tree.Root.Children["false"].Children["false"].Children["*"].ResultFunctions))
	assert.Equal(t, IncludeBiddersName, tree.Root.Children["false"].Children["false"].Children["*"].ResultFunctions[0].Name())
}

func GetConf() json.RawMessage {

	return json.RawMessage(`
 {
     "schema": [
     {
       "function": "deviceCountryIn",
	   "args": {"countries": ["USA", "UKR"]}
     },
     {
       "function": "dataCenterIn",
	   "args": {"datacenters": ["us-east", "us-west"]}
     },
     {
       "function": "channel"
     }
   ],
    "default": [
        {
           "function": "excludeBidders",
           "args": [{
               "bidders": ["bidderA"],
			   "seatNonBid": 111
           }]
        }
    ],

   "rules": [
     {
       "conditions": ["true", "true", "amp"],
       "results": [
         {
           "function": "excludeBidders",
           "args": [
             {
               "bidders": ["bidderA"],
			   "seatNonBid": 111
             }
           ]
         }
       ]
     },
     {
       "conditions": ["true", "false","web"],
       "results": [
         {
           "function": "excludeBidders",
           "args": [
             {
               "bidders": ["bidderB"],
               "seatNonBid": 222
             }
           ]
         }
       ]
     },
     {
       "conditions": ["false", "false", "*"],
       "results": [
         {
           "function": "includeBidders",
           "args": [
             {
               "bidders": ["bidderC"],
               "seatNonBid": 333
             }
           ]
         }
       ]
     }
   ]
 }`)

}
