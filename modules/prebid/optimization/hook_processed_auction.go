package optimization

import (
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/prebid/optimization/cache"
)

// at this point, we have well-formed trees (meaning the depth of the tree is the number of schema functions, all leaves are at this depth)
func handleProcessedAuctionHook(modelGroups []cache.CacheModelGroup, payload hookstage.ProcessedAuctionRequestPayload) (result hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], err error) {

	// determine which model group tree to run
	// run tree
	// run default functions if no tree leaf found
	// prepare hook result
	// return hook result

	return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}, nil
}
