package rulesengine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/prebid/prebid-server/v3/hooks"
	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
)

type hash = string

type cacheEntry struct {
	enabled                                 bool
	timestamp                               time.Time
	hashedConfig                            hash
	ruleSetsForProcessedAuctionRequestStage []cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]
}
type cacheRuleSet[T1 any, T2 any] struct {
	name        string
	modelGroups []cacheModelGroup[T1, T2]
}
type cacheModelGroup[T1 any, T2 any] struct {
	weight       int
	version      string
	analyticsKey string
	tree         rules.Tree[T1, T2]
}

// NewCacheEntry creates a new cache object for the given configuration
// It builds the tree structures for the rule sets for the processed auction request stage
// and stores them in the cache object
func NewCacheEntry(cfg *config.PbRulesEngine, cfgRaw *json.RawMessage) (cacheEntry, error) {
	if cfg == nil {
		return cacheEntry{}, errors.New("no rules engine configuration provided")
	}

	idHash := hashConfig(cfgRaw)
	if idHash == "" {
		return cacheEntry{}, errors.New("Can't create identifier hash from empty raw json configuration")
	}

	newCacheObj := cacheEntry{
		enabled:      cfg.Enabled,
		timestamp:    time.Now(),
		hashedConfig: idHash,
	}

	for _, ruleSet := range cfg.RuleSets {
		if ruleSet.Stage != hooks.StageProcessedAuctionRequest {
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
func createCacheRuleSet(cfg *config.RuleSet) (cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]], error) {
	if cfg == nil {
		return cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{}, errors.New("no rules engine configuration provided")
	}

	crs := cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
		name:        cfg.Name,
		modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
	}

	for _, modelGroup := range cfg.ModelGroups {
		tree, err := rules.NewTree[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]](
			&treeBuilder[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
				Config:            modelGroup,
				SchemaFuncFactory: rules.NewRequestSchemaFunction,
				ResultFuncFactory: NewProcessedAuctionRequestResultFunction,
			},
		)
		if err != nil {
			return crs, err
		}

		cmg := cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
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
func hashConfig(cfg *json.RawMessage) hash {
	if cfg == nil || len(*cfg) == 0 {
		return ""
	}
	newHash := sha256.Sum256(*cfg)
	return hex.EncodeToString(newHash[:])
}
