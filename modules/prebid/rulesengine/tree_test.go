package rulesengine

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/stretchr/testify/assert"
)

func TestExecuteRulesFullConfig(t *testing.T) {
	rw := BuildTestRequestWrapper()
	rules := BuildTestRules(t)

	hookResult := hs.HookResult[hs.ProcessedAuctionRequestPayload]{
		ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
	}
	result := ProcessedAuctionHookResult{
		HookResult:     hookResult,
		AllowedBidders: make(map[string]struct{}),
	}

	err := rules.Run(&rw, &result)
	assert.NoError(t, err, "unexpected error")
	assert.NotEmptyf(t, result.HookResult.ChangeSet, "change set is empty")
	assert.Len(t, result.HookResult.ChangeSet.Mutations(), 1)
	assert.Equal(t, hs.MutationDelete, result.HookResult.ChangeSet.Mutations()[0].Type())
}

func BuildTestRules(t *testing.T) rules.Tree[hs.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult] {
	devCountryFunc, errDevCountry := rules.NewDeviceCountryIn(json.RawMessage(`{"countries": ["JPN"]}`))
	assert.NoError(t, errDevCountry, "unexpected error in NewDeviceCountryIn")
	resFuncTrue, errNewIncludeBidders := NewIncludeBidders(json.RawMessage(`{"bidders": ["bidderA"]}`))
	assert.NoError(t, errNewIncludeBidders, "unexpected error in NewIncludeBidders")
	resFuncFalse, errNewExcludeBidders := NewExcludeBidders(json.RawMessage(`{"bidders": ["bidderB"]}`))
	assert.NoError(t, errNewExcludeBidders, "unexpected error in NewExcludeBidders")

	rules := rules.Tree[hs.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
		Root: &rules.Node[hs.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
			SchemaFunction: devCountryFunc,
			Children: map[string]*rules.Node[hs.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{
				"true": {
					ResultFunctions: []rules.ResultFunction[hs.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{resFuncTrue},
				},
				"false": {
					ResultFunctions: []rules.ResultFunction[hs.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]{resFuncFalse},
				},
			},
		},
		DefaultFunctions: nil,
	}

	return rules
}

func BuildTestRequestWrapper() hs.ProcessedAuctionRequestPayload {
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

	return hs.ProcessedAuctionRequestPayload{
		Request: rw,
	}
}
