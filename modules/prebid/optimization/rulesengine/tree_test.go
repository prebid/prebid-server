package rulesengine

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	structs "github.com/prebid/prebid-server/v3/modules/prebid/optimization/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

func TestExecuteRules(t *testing.T) {

	rules := BuildTestRules(t)
	rw := BuildTestRequestWrapper()
	changeSet, err := Execute(&rules, rw)
	assert.NoError(t, err, "unexpected error")
	assert.NotEmptyf(t, changeSet, "change set is empty")
}

func TestExecuteRulesFullConfig(t *testing.T) {

	var conf structs.ModelGroup
	err := jsonutil.Unmarshal(GetConf(), &conf)
	assert.NoError(t, err)

	rules, err := BuildRulesTree(conf)
	assert.NoError(t, err)
	rw := BuildTestRequestWrapper()
	changeSet, err := Execute(rules, rw)
	assert.NoError(t, err, "unexpected error")
	assert.NotEmptyf(t, changeSet, "change set is empty")
}

func BuildTestRules(t *testing.T) Tree {
	devCountryFunc, err := NewDeviceCountry(json.RawMessage(`["USA"]`))
	assert.NoError(t, err, "unexpected error")
	resFunctTrue, err := NewIncludeBidders(json.RawMessage(`[{"SeatNonBid": 123}]`))
	assert.NoError(t, err, "unexpected error")
	resFunctFalse, err := NewExcludeBidders(json.RawMessage(`[{"SeatNonBid": 456}]`))
	assert.NoError(t, err, "unexpected error")

	rules := Tree{
		Root: &Node{
			SchemaFunction: devCountryFunc,
			Children: map[string]*Node{
				"true":  {ResultFunctions: []ResultFunction{resFunctTrue}},
				"false": {ResultFunctions: []ResultFunction{resFunctFalse}},
			},
		},
	}
	return rules
}

func BuildTestRequestWrapper() *openrtb_ext.RequestWrapper {
	rw := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Device: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Country: "USA",
					Region:  "us-east",
				},
			},
		},
	}
	extPrebid := &openrtb_ext.ExtRequestPrebid{Channel: &openrtb_ext.ExtRequestPrebidChannel{Name: "amp"}}
	reqExt, _ := rw.GetRequestExt()
	reqExt.SetPrebid(extPrebid)

	return rw
}

func GetConf() json.RawMessage {

	return json.RawMessage(`
 {
     "schema": [
     {
       "function": "deviceCountry",
       "args": ["USA"]
     },
     {
       "function": "dataCenters",
       "args": ["us-east", "us-west"]
     },
     {
       "function": "channel"
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
