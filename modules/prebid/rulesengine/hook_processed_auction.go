package rulesengine

import (
	"fmt"

	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/randomutil"
)

type RequestWrapper = openrtb_ext.RequestWrapper
type ModelGroup = cacheModelGroup[RequestWrapper, ProcessedAuctionHookResult]

type ProcessedAuctionHookResult struct {
	HookResult   hs.HookResult[hs.ProcessedAuctionRequestPayload]
	BiddersToAdd []string
}

func handleProcessedAuctionHook(ruleSets []cacheRuleSet[openrtb_ext.RequestWrapper, ProcessedAuctionHookResult], payload hs.ProcessedAuctionRequestPayload) (hs.HookResult[hs.ProcessedAuctionRequestPayload], error) {

	result := hs.HookResult[hs.ProcessedAuctionRequestPayload]{
		ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
	}

	auctionHookRes := ProcessedAuctionHookResult{
		HookResult:   result,
		BiddersToAdd: make([]string, 0),
	}

	for _, ruleSet := range ruleSets {
		selectedGroup, err := selectModelGroup(ruleSet.modelGroups, randomutil.RandomNumberGenerator{})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to select model group: %s", err))
			continue
		}

		if err := selectedGroup.tree.Run(payload.Request, &auctionHookRes); err != nil {
			//TODO: classify errors as warnings or errors
			result.Errors = append(result.Errors, err.Error())
		}

		if len(auctionHookRes.BiddersToAdd) > 0 {
			auctionHookRes.HookResult.ChangeSet.ProcessedAuctionRequest().Bidders().Add(auctionHookRes.BiddersToAdd)
		}
	}

	return auctionHookRes.HookResult, nil
}

func selectModelGroup(modelGroups []cacheModelGroup[openrtb_ext.RequestWrapper, ProcessedAuctionHookResult], rg randomutil.RandomGenerator) (ModelGroup, error) {
	if len(modelGroups) == 0 {
		return ModelGroup{}, fmt.Errorf("no model groups available")
	}

	if len(modelGroups) == 1 {
		return modelGroups[0], nil
	}

	// Create cumulative weight distribution
	totalWeight := 0
	cumulativeWeights := make([]int, len(modelGroups))

	for i, group := range modelGroups {
		weight := 100
		if group.weight > 0 {
			weight = group.weight
		}
		totalWeight += weight
		cumulativeWeights[i] = totalWeight
	}

	randomValue := rg.Intn(totalWeight) + 1

	// Find the model group corresponding to the random value
	for i, threshold := range cumulativeWeights {
		if randomValue <= threshold {
			return modelGroups[i], nil
		}
	}

	return modelGroups[0], nil
}
