package ortb2blocking

import (
	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
)

const enforceBlockingTag = "enforce_blocking"

const (
	attributesAnalyticKey = "attributes"
	badvAnalyticKey       = "adomain"
	cattaxAnalyticKey     = "bcat"
	bappAnalyticKey       = "bundle"
	battrAnalyticKey      = "attr"
)

// ortb2blocking module has only 1 activity: `enforce_blocking` which will be used in further result processing
func newEnforceBlockingTags() hookanalytics.Analytics {
	return hookanalytics.Analytics{
		Activities: []hookanalytics.Activity{
			{
				Name:   enforceBlockingTag,
				Status: hookanalytics.ActivityStatusSuccess,
			},
		},
	}
}

func addFailedStatusTag(result *hookstage.HookResult[hookstage.RawBidderResponsePayload]) {
	result.AnalyticsTags.Activities[0].Status = hookanalytics.ActivityStatusError
}

func addAllowedAnalyticTag(result *hookstage.HookResult[hookstage.RawBidderResponsePayload], bidder, ImpID string) {
	newAllowedResult := hookanalytics.Result{
		Status: hookanalytics.ResultStatusAllow,
		AppliedTo: hookanalytics.AppliedTo{
			Bidder: bidder,
			ImpIds: []string{ImpID},
		},
	}

	result.AnalyticsTags.Activities[0].Results = append(result.AnalyticsTags.Activities[0].Results, newAllowedResult)
}

func addBlockedAnalyticTag(
	result *hookstage.HookResult[hookstage.RawBidderResponsePayload],
	bidder, ImpID string,
	failedAttributes []string,
	data map[string]interface{},
) {
	values := make(map[string]interface{})

	values[attributesAnalyticKey] = failedAttributes
	for _, attribute := range [5]string{
		"badv",
		"bcat",
		"cattax",
		"bapp",
		"battr",
	} {
		if _, ok := data[attribute]; ok {
			analyticKey := getAnalyticKeyForAttribute(attribute)
			values[analyticKey] = data[attribute]
		}
	}

	newBlockedResult := hookanalytics.Result{
		Status: hookanalytics.ResultStatusBlock,
		Values: values,
		AppliedTo: hookanalytics.AppliedTo{
			Bidder: bidder,
			ImpIds: []string{ImpID},
		},
	}

	result.AnalyticsTags.Activities[0].Results = append(result.AnalyticsTags.Activities[0].Results, newBlockedResult)
}

// most of the attributes have their own representation for an analytic key
func getAnalyticKeyForAttribute(attribute string) string {
	switch attribute {
	case "badv":
		return badvAnalyticKey
	case "cattax":
		return cattaxAnalyticKey
	case "bapp":
		return bappAnalyticKey
	case "battr":
		return battrAnalyticKey
	default:
		return attribute
	}
}
