package modules

import (
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
)

func createModuleStageNamesCollection(modules map[string]interface{}) (map[string][]string, error) {
	moduleStageNameCollector := make(map[string][]string)
	var added bool

	for id, hook := range modules {
		if _, ok := hook.(hookstage.Entrypoint); ok {
			added = true
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, hooks.StageEntrypoint)
		}

		if _, ok := hook.(hookstage.RawAuction); ok {
			added = true
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, hooks.StageRawAuction)
		}

		if _, ok := hook.(hookstage.ProcessedAuction); ok {
			added = true
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, hooks.StageProcessedAuction)
		}

		if _, ok := hook.(hookstage.BidderRequest); ok {
			added = true
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, hooks.StageBidRequest)
		}

		if _, ok := hook.(hookstage.RawBidderResponse); ok {
			added = true
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, hooks.StageRawBidResponse)
		}

		if _, ok := hook.(hookstage.AllProcessedBidResponses); ok {
			added = true
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, hooks.StageAllProcessedBidResponses)
		}

		if _, ok := hook.(hookstage.AuctionResponse); ok {
			added = true
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, hooks.StageAuctionResponse)
		}

		if !added {
			return nil, fmt.Errorf(`hook "%s" does not implement any supported hook interface`, id)
		}
	}

	return moduleStageNameCollector, nil
}

func addModuleStageName(moduleStageNameCollector map[string][]string, id string, stage string) map[string][]string {
	str := strings.Replace(id, ".", "-", -1)
	moduleStageNameCollector[str] = append(moduleStageNameCollector[str], stage)

	return moduleStageNameCollector
}
