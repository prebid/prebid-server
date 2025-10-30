package rulesengine

import (
	"fmt"

	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/util/randomutil"
)

type ModelGroup = cacheModelGroup[hs.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult]

type ProcessedAuctionHookResult struct {
	HookResult     hs.HookResult[hs.ProcessedAuctionRequestPayload]
	AllowedBidders map[string]struct{}
}

func handleProcessedAuctionHook(
	ruleSets []cacheRuleSet[hs.ProcessedAuctionRequestPayload, ProcessedAuctionHookResult],
	payload hs.ProcessedAuctionRequestPayload) (hs.HookResult[hs.ProcessedAuctionRequestPayload], error) {

	result := ProcessedAuctionHookResult{
		HookResult: hs.HookResult[hs.ProcessedAuctionRequestPayload]{
			ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
		},
		AllowedBidders: make(map[string]struct{}),
	}

	for _, ruleSet := range ruleSets {
		selectedGroup, err := selectModelGroup(ruleSet.modelGroups, randomutil.RandomNumberGenerator{})
		if err != nil {
			result.HookResult.Errors = append(result.HookResult.Errors, fmt.Sprintf("failed to select model group: %s", err))
			continue
		}

		if err = selectedGroup.tree.Run(&payload, &result); err != nil {
			//TODO: classify errors as warnings or errors
			result.HookResult.Errors = append(result.HookResult.Errors, err.Error())
		}

		if len(result.AllowedBidders) > 0 {
			result.HookResult.ChangeSet.ProcessedAuctionRequest().Bidders().Add(result.AllowedBidders)
		}
	}

	return result.HookResult, nil
}

func selectModelGroup(modelGroups []ModelGroup, rg randomutil.RandomGenerator) (ModelGroup, error) {
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
