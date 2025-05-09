package rulesengine

import (
	"encoding/json"
	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildTreeFullConfig(t *testing.T) {

	var modelGroup config.ModelGroup
	err := jsonutil.Unmarshal(GetConf(), &modelGroup)
	assert.NoError(t, err)

	builder := &treeBuilder[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{
		Config:            modelGroup,
		SchemaFuncFactory: rules.NewRequestSchemaFunction,
		ResultFuncFactory: NewProcessedAuctionRequestResultFunction,
	}
	tree := rules.Tree[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{
		Root: &rules.Node[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{}}

	err = builder.Build(&tree)
	assert.NoError(t, err)
}

func GetConf() json.RawMessage {

	return json.RawMessage(`
 {
     "schema": [
     {
       "function": "deviceCountry",
       "args": []
     },
 	 {
       "function": "deviceCountryIn",
       "args": ["USA", "UKR"]
     },
     {
       "function": "dataCenters",
       "args": ["us-east", "us-west"]
     },
     {
       "function": "channel"
     }
   ],
    "default": [
        {
           "function": "logATag",
           "args": {"analyticsValue": "default-allow"}
        },
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
