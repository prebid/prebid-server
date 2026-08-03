package helpers

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/analytics"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v4/hooks/hookexecution"
	"github.com/stretchr/testify/assert"
)

func TestJsonifyAuctionObject(t *testing.T) {
	ao := &analytics.AuctionObject{
		Status: http.StatusOK,
	}

	_, err := JsonifyAuctionObject(ao, "scopeId")
	assert.NoError(t, err)
}

func TestJsonifyAuctionObjectIncludesRulesEngineWarningAnalytics(t *testing.T) {
	ao := &analytics.AuctionObject{
		Status: http.StatusOK,
		HookExecutionOutcome: []hookexecution.StageOutcome{
			{
				Groups: []hookexecution.GroupOutcome{
					{
						InvocationResults: []hookexecution.HookOutcome{
							{
								HookID: hookexecution.HookID{
									ModuleCode:   "prebid.rulesengine",
									HookImplCode: "rulesengine",
								},
								AnalyticsTags: hookanalytics.Analytics{
									Activities: []hookanalytics.Activity{{
										Name:   "rules_engine_bidder_filtering",
										Status: hookanalytics.ActivityStatusSuccess,
										Results: []hookanalytics.Result{{
											Status: hookanalytics.ResultStatusBlock,
											Values: map[string]interface{}{
												"code":   errortypes.RulesEngineBidderExcludedWarningCode,
												"reason": "excluded_by_rule",
											},
										}},
									}},
								},
							},
						},
					},
				},
			},
		},
	}

	data, err := JsonifyAuctionObject(ao, "scopeId")

	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(data), `"name":"rules_engine_bidder_filtering"`), string(data))
	assert.True(t, strings.Contains(string(data), `"code":`+strconv.Itoa(errortypes.RulesEngineBidderExcludedWarningCode)), string(data))
	assert.True(t, strings.Contains(string(data), `"reason":"excluded_by_rule"`), string(data))
}

func TestJsonifyVideoObject(t *testing.T) {
	vo := &analytics.VideoObject{
		Status: http.StatusOK,
	}

	_, err := JsonifyVideoObject(vo, "scopeId")
	assert.NoError(t, err)
}

func TestJsonifyCookieSync(t *testing.T) {
	cso := &analytics.CookieSyncObject{
		Status:       http.StatusOK,
		BidderStatus: []*analytics.CookieSyncBidder{},
	}

	_, err := JsonifyCookieSync(cso, "scopeId")
	assert.NoError(t, err)
}

func TestJsonifySetUIDObject(t *testing.T) {
	so := &analytics.SetUIDObject{
		Status: http.StatusOK,
		Bidder: "any-bidder",
		UID:    "uid string",
	}

	_, err := JsonifySetUIDObject(so, "scopeId")
	assert.NoError(t, err)
}

func TestJsonifyAmpObject(t *testing.T) {
	ao := &analytics.AmpObject{
		Status:             http.StatusOK,
		Errors:             make([]error, 0),
		AuctionResponse:    &openrtb2.BidResponse{},
		AmpTargetingValues: map[string]string{},
	}

	_, err := JsonifyAmpObject(ao, "scopeId")
	assert.NoError(t, err)
}
