package rulesengine

import (
	"encoding/json"
	"fmt"

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
	done            chan struct{}
	requests        chan buildInstruction
	schemaValidator *gojsonschema.Schema
	monitor         RulesEngineObserver
}

// Run reads build instructions from a channel, and if the trees for the rule sets for a given account
// need to be rebuilt, it rebuilds them storing them in cache
func (tm *treeManager) Run(c cacher) error {
	for {
		select {
		case req := <-tm.requests:
			if req.config == nil {
				break
			}

			cacheObj := c.Get(req.accountID)
			if cacheObj != nil && !rebuildTrees(cacheObj, req.config) {
				break
			}

			parsedCfg, err := config.NewConfig(*req.config, tm.schemaValidator)
			if err != nil {
				tm.monitor.logError(fmt.Sprintf("Rules engine error parsing config for account %s: %v", req.accountID, err))
				break
			}
			if !parsedCfg.Enabled {
				c.Delete(req.accountID)
				tm.monitor.logInfo(fmt.Sprintf("Rules engine disabled for account %s", req.accountID))
				break
			}

			newCacheObj, err := NewCacheEntry(parsedCfg, req.config)
			if err != nil {
				tm.monitor.logError(fmt.Sprintf("Rules engine error creating cache entry for account %s: %v", req.accountID, err))
				break
			}

			c.Set(req.accountID, &newCacheObj)

		case <-tm.done:
			tm.monitor.logInfo("Rules engine tree manager shutting down")
			return nil
		}
	}
}

// Shutdown signals the tree manager to stop processing
func (tm *treeManager) Shutdown() {
	close(tm.done)
}
