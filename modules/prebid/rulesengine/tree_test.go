package rulesengine

import (
	"encoding/json"
	"github.com/prebid/prebid-server/v3/usersync"
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

	result := hs.HookResult[hs.ProcessedAuctionRequestPayload]{
		ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
	}
	payload := hs.ProcessedAuctionRequestPayload{Request: rw, Usersyncs: &usersync.Cookie{}}
	err := rules.Run(&payload, &result)
	assert.NoError(t, err, "unexpected error")
	assert.NotEmptyf(t, result.ChangeSet, "change set is empty")
	assert.Len(t, result.ChangeSet.Mutations(), 1)
	assert.Equal(t, hs.MutationUpdate, result.ChangeSet.Mutations()[0].Type())
}

func BuildTestRules(t *testing.T) rules.Tree[hs.ProcessedAuctionRequestPayload, hs.HookResult[hs.ProcessedAuctionRequestPayload]] {
	devCountryFunc, errDevCountry := rules.NewDeviceCountryIn(json.RawMessage(`{"countries": ["USA"]}`))
	assert.NoError(t, errDevCountry, "unexpected error in NewDeviceCountryIn")
	resFuncTrue, errNewIncludeBidders := NewIncludeBidders(json.RawMessage(`{"bidders": ["bidderA"]}`))
	assert.NoError(t, errNewIncludeBidders, "unexpected error in NewIncludeBidders")
	resFuncFalse, errNewExcludeBidders := NewExcludeBidders(json.RawMessage(`{"bidders": ["bidderB"]}`))
	assert.NoError(t, errNewExcludeBidders, "unexpected error in NewExcludeBidders")

	rules := rules.Tree[hs.ProcessedAuctionRequestPayload, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
		Root: &rules.Node[hs.ProcessedAuctionRequestPayload, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
			SchemaFunction: devCountryFunc,
			Children: map[string]*rules.Node[hs.ProcessedAuctionRequestPayload, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
				"true": {
					ResultFunctions: []rules.ResultFunction[hs.ProcessedAuctionRequestPayload, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{resFuncTrue},
				},
				"false": {
					ResultFunctions: []rules.ResultFunction[hs.ProcessedAuctionRequestPayload, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{resFuncFalse},
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
