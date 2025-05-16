package rulesengine

import (
	"encoding/json"
	"github.com/prebid/openrtb/v20/openrtb2"
	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExecuteRulesFullConfig(t *testing.T) {
	rw := BuildTestRequestWrapper()
	rules := BuildTestRules()

	result := hs.HookResult[hs.ProcessedAuctionRequestPayload]{
		ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
	}
	err := rules.Run(rw, &result.ChangeSet)
	assert.NoError(t, err, "unexpected error")
	assert.NotEmptyf(t, result.ChangeSet, "change set is empty")
	assert.Len(t, result.ChangeSet.Mutations(), 1)
	assert.Equal(t, hs.MutationUpdate, result.ChangeSet.Mutations()[0].Type())
}

func BuildTestRules() rules.Tree[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]] {
	devCountryFunc, _ := rules.NewDeviceCountryIn(json.RawMessage(`[["USA"]]`))          // handle err
	resFuncTrue, _ := NewIncludeBidders(json.RawMessage(`[{ "bidders": ["bidderA"]}]`))  //handle err
	resFuncFalse, _ := NewExcludeBidders(json.RawMessage(`[{ "bidders": ["bidderB"]}]`)) //handle err

	rules := rules.Tree[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{
		Root: &rules.Node[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{
			SchemaFunction: devCountryFunc,
			Children: map[string]*rules.Node[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{
				"true": &rules.Node[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{
					ResultFunctions: []rules.ResultFunction[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{resFuncTrue},
				},
				"false": &rules.Node[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{
					ResultFunctions: []rules.ResultFunction[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{resFuncFalse},
				},
			},
		},
		DefaultFunctions: nil,
	}

	return rules
}

func BuildTestRequestWrapper() *openrtb_ext.RequestWrapper {
	rw := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{
				{
					ID: "ImpId1",
					Ext: json.RawMessage(`{"prebid": 
												{"bidder": 
													{
														"bidderA": {"paramA": "valueA"},
														"bidderB": {"paramB": "valueB"}
													}
												}
											}`),
				},
			},

			Device: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Country: "USA",
				},
			},
		},
	}
	extPrebid := &openrtb_ext.ExtRequestPrebid{Channel: &openrtb_ext.ExtRequestPrebidChannel{Name: "amp"}}
	reqExt, _ := rw.GetRequestExt()
	reqExt.SetPrebid(extPrebid)

	return rw
}
