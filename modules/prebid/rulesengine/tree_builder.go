package rulesengine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// buildInstruction specifies the criteria needed to build the tree structures for an account
type buildInstruction struct {
	accountID string
	config    *json.RawMessage
}

// treeBuilder represents the component that generates trees
type treeBuilder struct {
	requests chan buildInstruction
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
			// if validation fails
			//	record/return error
			//	record metric?
			// for each rule set
			// 	for each module group
			// 		build tree
			// 		if build tree fails (most likely due to schema/result func param type errors)
			// 			record/return error
			// 			record metric?
			ruleSets := make(map[stage][]cacheRuleSet, 0)

			newHash := sha256.Sum256(*req.config)
			hashedConfig := hash(hex.EncodeToString(newHash[:]))

			newCacheObj := cacheObject{
				timestamp:    time.Now(),
				hashedConfig: hashedConfig,
				ruleSets:     ruleSets,
			}
			c.Set(req.accountID, newCacheObj)
		}
		// case TODO: do we need to handle some shutdown signal?
	}
}
