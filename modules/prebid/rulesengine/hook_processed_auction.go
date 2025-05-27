package rulesengine

import (
	"fmt"
	"math/rand"

	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type RequestWrapper = openrtb_ext.RequestWrapper
type ProcessedAuctionChangeSet = hs.ChangeSet[hs.ProcessedAuctionRequestPayload]
type ModelGroup = cacheModelGroup[RequestWrapper, ProcessedAuctionChangeSet]

func handleProcessedAuctionHook(
	ruleSets []cacheRuleSet[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]],
	payload hs.ProcessedAuctionRequestPayload) (hs.HookResult[hs.ProcessedAuctionRequestPayload], error) {

	result := hs.HookResult[hs.ProcessedAuctionRequestPayload]{
		ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
	}

	for _, ruleSet := range ruleSets {
		selectedGroup, err := selectModelGroup(ruleSet.modelGroups)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to select model group: %s", err))
			continue
		}

		if err := selectedGroup.tree.Run(payload.Request, &result.ChangeSet); err != nil {
			//TODO: classify errors as warnings or errors
			result.Errors = append(result.Errors, err.Error())
		}
	}

	return result, nil
}

func selectModelGroup(modelGroups []ModelGroup) (ModelGroup, error) {
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

    randomValue := rand.Intn(totalWeight) + 1

    // Find the model group corresponding to the random value
    for i, threshold := range cumulativeWeights {
        if randomValue <= threshold {
            return modelGroups[i], nil
        }
    }

    return modelGroups[0], nil
}