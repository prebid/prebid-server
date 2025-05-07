package rulesengine

import (
	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func handleProcessedAuctionHook(
	ruleSets []cacheRuleSet[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]],
	payload  hs.ProcessedAuctionRequestPayload) (hs.HookResult[hs.ProcessedAuctionRequestPayload], error) {

	result := hs.HookResult[hs.ProcessedAuctionRequestPayload]{
		ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
	}

	for _, ruleSet := range ruleSets {
		for _, modelGroup := range ruleSet.modelGroups {
			err := modelGroup.tree.Run(payload.Request, &result.ChangeSet)
			if err != nil {
				//TODO: classify errors as warnings or errors
				result.Errors = append(result.Errors, err.Error())
			}
		}
	}

	return result, nil
}