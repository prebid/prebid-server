package rulesengine

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/xeipuuv/gojsonschema"
)

// buildInstruction specifies the criteria needed to build the tree structures for an account
type buildInstruction struct {
	accountID string
	config    *json.RawMessage
}

// treeManager represents the component that generates trees
type treeManager struct {
	requests        chan buildInstruction
	schemaValidator *gojsonschema.Schema
}

// Run reads build instructions from a channel, and if the trees for the rule sets for a given account
// need to be rebuilt, it rebuilds them storing them in cache
func (tb *treeManager) Run(c cacher) error {
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

			parsedCfg, err := config.NewConfig(*req.config, tb.schemaValidator)
			if err != nil {
				// TODO: log error / metric
				break
			}

			newCacheObj, err := NewCacheEntry(parsedCfg, req.config)
			if err != nil {
				// TODO: log error / metric
				break
			}

			c.Set(req.accountID, newCacheObj)
		}
		// case TODO: do we need to handle some shutdown signal?
	}
}
