package optimization

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	structs "github.com/prebid/prebid-server/v3/modules/prebid/optimization/config"
	"github.com/prebid/prebid-server/v3/modules/prebid/optimization/rulesengine"

	"github.com/xeipuuv/gojsonschema"
)

// buildInstruction specifies the criteria needed to build the tree structures for an account
type buildInstruction struct {
	accountID string
	config    *json.RawMessage
}

// treeBuilder represents the component that generates trees
type treeBuilder struct {
	requests        chan buildInstruction
	schemaValidator *gojsonschema.Schema
}

// Run reads build instructions from a channel, and if the trees for the rule sets for a given account
// need to be rebuilt, it rebuilds them storing them in cache
func (tb *treeBuilder) Run(c cacher) error {
	for {
		select {
		case req := <-tb.requests:
			if req.config == nil {
				break
			}

			cacheObj := c.Get(req.accountID)
			if cacheObj != nil && !rebuildTrees(cacheObj, req.config) {
				break
			}

			// TODO: validate and build tree here
			// unmarshal account config to structs --> newConfig()
			// validate account config --> validateConfig()
			re, err := structs.NewConfig(*req.config, tb.schemaValidator)
			if err != nil {
				// if validation fails
				//	record/return error
				return err
				//	record metric?
			}

			if re.Enabled == false {
				//	record/return error
				return errors.New("Not enabled error")
				//	record metric?
			}

			ruleSetsPerStage := make(map[stage][]cacheRuleSet, len(re.RuleSets))
			// for each rule set
			// 	for each module group
			for i := range re.RuleSets {
				rs := cacheRuleSet{
					name:        re.RuleSets[i].Name,
					modelGroups: make([]cacheModelGroup, len(re.RuleSets[i].ModelGroups)),
				}
				for j := range re.RuleSets[i].ModelGroups {
					// build tree
					treeToCache, err := rulesengine.BuildRulesTree(re.RuleSets[i].ModelGroups[j])
					// if build tree fails (most likely due to schema/result func param type errors)
					if err != nil {
						// record/return error
						return err
						// record metric?
						// close channel?
					}

					// store built tree
					mg := cacheModelGroup{
						weight:       re.RuleSets[i].ModelGroups[j].Weight,
						version:      re.RuleSets[i].ModelGroups[j].Version,
						analyticsKey: re.RuleSets[i].ModelGroups[j].AnalyticsKey,
						defaults:     make([]rulesengine.ResultFunction, len(re.RuleSets[i].ModelGroups[j].Default)),
						root:         *treeToCache.Root,
					}

					// Create defaults ([]ResultFunction array)
					for k := range re.RuleSets[i].ModelGroups[j].Default {
						resFunc, err := rulesengine.NewResultFunctionFactory(re.RuleSets[i].ModelGroups[j].Default[k].Func, re.RuleSets[i].ModelGroups[j].Default[k].Args)
						if err != nil {
							// record/return error
							return err
							// record metric?
							// close channel?
						}
						mg.defaults = append(mg.defaults, resFunc)
					}

					// append to modelgroups
					rs.modelGroups = append(rs.modelGroups, mg)
				}
				ruleSetsPerStage[stage(re.RuleSets[i].Stage)] = append(ruleSetsPerStage[stage(re.RuleSets[i].Stage)], rs)
			}

			newHash := sha256.Sum256(*req.config)
			hashedConfig := hash(hex.EncodeToString(newHash[:]))

			co := &cacheObject{
				timestamp:    time.Now(),
				hashedConfig: hashedConfig,
				ruleSets:     ruleSetsPerStage,
			}

			c.Set(req.accountID, co)
		}
		// case TODO: do we need to handle some shutdown signal?
	}
}
