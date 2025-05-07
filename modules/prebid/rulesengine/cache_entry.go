package rulesengine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
)

type hash = string
type stage = string

type cacheEntry struct {
	timestamp    time.Time
	hashedConfig hash
	ruleSetsForProcessedAuctionRequestStage []cacheRuleSet[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]
}
type cacheRuleSet[T1 any, T2 any] struct {
	name        string
	modelGroups []cacheModelGroup[T1, T2]
}
type cacheModelGroup[T1 any, T2 any] struct {
	weight       int
	version      string
	analyticsKey string
	defaults     []rules.ResultFunction[T2] // can this be set on the tree somehow instead?
	tree         rules.Tree[T1, T2]
}

// NewCacheEntry creates a new cache object for the given configuration
// It builds the tree structures for the rule sets for the processed auction request stage
// and stores them in the cache object
func NewCacheEntry(cfg *config.PbRulesEngine, cfgRaw *json.RawMessage) (cacheEntry, error) {
	newCacheObj := cacheEntry{
		timestamp:    time.Now(),
		hashedConfig: hashConfig(cfgRaw),
	}

	for _, ruleSet := range cfg.RuleSets {
		if ruleSet.Stage != "processed_auction" {
			// TODO: log error / metric --> stage not supported
			continue
		}
		crs, err := createCacheRuleSet(&ruleSet)
		if err != nil {
			// TODO: log error / metric --> 
			continue
		}

		newCacheObj.ruleSetsForProcessedAuctionRequestStage = append(newCacheObj.ruleSetsForProcessedAuctionRequestStage, crs)
	}

	return newCacheObj, nil
}

// createCacheRuleSet creates a new cache rule set for the given configuration
// It builds the tree structures for the model groups and stores them in the cache rule set
func createCacheRuleSet(cfg *config.RuleSet) (cacheRuleSet[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]], error) {
	crs := cacheRuleSet[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{
		name:        cfg.Name,
		modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{},
	}

	for _, modelGroup := range cfg.ModelGroups {
		tree, err := rules.NewTree[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]](
			&treeBuilder[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{
				Config: modelGroup,
				SchemaFuncFactory: rules.NewRequestSchemaFunction,
				ResultFuncFactory: NewProcessedAuctionRequestResultFunction,
			},
		)
		if err != nil {
			return crs, err
		}

		cmg := cacheModelGroup[openrtb_ext.RequestWrapper, hs.ChangeSet[hs.ProcessedAuctionRequestPayload]]{
			weight:       modelGroup.Weight,
			version:      modelGroup.Version,
			analyticsKey: modelGroup.AnalyticsKey,
			tree:         *tree,
		}
		crs.modelGroups = append(crs.modelGroups, cmg)
	}

	return crs, nil
}

// hashConfig generates a hash of the JSON configuration
// This is used to determine if the configuration has changed and if the trees need to be rebuilt
// The hash is a SHA256 hash of the JSON configuration and is stored as a string
func hashConfig(cfg *json.RawMessage) (hash) {
	if cfg == nil {
		return ""
	}
	newHash := sha256.Sum256(*cfg)
	return hex.EncodeToString(newHash[:])
}