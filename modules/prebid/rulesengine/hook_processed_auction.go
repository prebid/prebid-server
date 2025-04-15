package rulesengine

import (
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
)

// at this point, we have well-formed trees (meaning the depth of the tree is the number of schema functions, all leaves are at this depth)
func handleProcessedAuctionHook(modelGroups []cacheRuleSet, payload hookstage.ProcessedAuctionRequestPayload) (result hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], err error) {

	// TODO: traverse trees here
	// for each rule set:
	// 	run tree
	// 	run default functions if no tree leaf found
	// prepare hook result
	// return hook result

	return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}, nil
}
