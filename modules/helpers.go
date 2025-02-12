package modules

import (
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
)

var moduleReplacer = strings.NewReplacer(".", "_", "-", "_")

func createModuleStageNamesCollection(modules map[string]interface{}) (map[string][]string, error) {
	moduleStageNameCollector := make(map[string][]string)
	var added bool

	for id, hook := range modules {
		if _, ok := hook.(hookstage.Entrypoint); ok {
			added = true
			stageName := hooks.StageEntrypoint.String()
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, stageName)
		}

		if _, ok := hook.(hookstage.RawAuctionRequest); ok {
			added = true
			stageName := hooks.StageRawAuctionRequest.String()
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, stageName)
		}

		if _, ok := hook.(hookstage.ProcessedAuctionRequest); ok {
			added = true
			stageName := hooks.StageProcessedAuctionRequest.String()
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, stageName)
		}

		if _, ok := hook.(hookstage.BidderRequest); ok {
			added = true
			stageName := hooks.StageBidderRequest.String()
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, stageName)
		}

		if _, ok := hook.(hookstage.RawBidderResponse); ok {
			added = true
			stageName := hooks.StageRawBidderResponse.String()
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, stageName)
		}

		if _, ok := hook.(hookstage.AllProcessedBidResponses); ok {
			added = true
			stageName := hooks.StageAllProcessedBidResponses.String()
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, stageName)
		}

		if _, ok := hook.(hookstage.AuctionResponse); ok {
			added = true
			stageName := hooks.StageAuctionResponse.String()
			moduleStageNameCollector = addModuleStageName(moduleStageNameCollector, id, stageName)
		}

		if !added {
			return nil, fmt.Errorf(`hook "%s" does not implement any supported hook interface`, id)
		}
	}

	return moduleStageNameCollector, nil
}

func addModuleStageName(moduleStageNameCollector map[string][]string, id string, stage string) map[string][]string {
	str := moduleReplacer.Replace(id)
	moduleStageNameCollector[str] = append(moduleStageNameCollector[str], stage)

	return moduleStageNameCollector
}
