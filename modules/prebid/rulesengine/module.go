package rulesengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/util/timeutil"
)

const fiveMinutes = time.Duration(300) * time.Second

// Builder configures the rules engine module initiating an in-memory cache and kicking
// off a go routine that builds tree structures that represent rule sets optimized for finding
// a rule to applies for a given request.
func Builder(_ json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	schemaValidator, err := config.CreateSchemaValidator(config.RulesEngineSchemaFilePath)
	if err != nil {
		return nil, err
	}

	tm := treeManager{
		done:            make(chan struct{}),
		requests:        make(chan buildInstruction),
		schemaValidator: schemaValidator,
		monitor:         &treeManagerLogger{},
	}
	c := NewCache()

	go tm.Run(c)

	return Module{
		Cache:       c,
		TreeManager: &tm,
	}, nil
}

// Module represents the rules engine module
type Module struct {
	Cache       cacher
	TreeManager *treeManager
}

// HandleProcessedAuctionHook updates field on openrtb2.BidRequest.
// Fields are updated only if request satisfies conditions provided by the module config.
func (m Module) HandleProcessedAuctionHook(
	_ context.Context,
	miCtx hs.ModuleInvocationContext,
	payload hs.ProcessedAuctionRequestPayload,
) (hs.HookResult[hs.ProcessedAuctionRequestPayload], error) {
	result := hs.HookResult[hs.ProcessedAuctionRequestPayload]{}

	// AccountConfig will either be an account-specific config or the default account config
	// AccountConfig only contains the config block for this module
	if len(miCtx.AccountConfig) == 0 {
		return result, nil
	}

	co := m.Cache.Get(miCtx.AccountID)

	// cache miss
	if co == nil {
		bi := buildInstruction{
			accountID: miCtx.AccountID,
			config:    &miCtx.AccountConfig,
		}
		m.TreeManager.requests <- bi

		// TODO: return with reject or no reject, possible config option
		return hs.HookResult[hs.ProcessedAuctionRequestPayload]{
			Message: "skipped, loading rules engine account configuration for future requests",
		}, nil
	}
	// cache hit
	if rebuildTrees(co, &miCtx.AccountConfig) {
		bi := buildInstruction{
			accountID: miCtx.AccountID,
			config:    &miCtx.AccountConfig,
		}
		m.TreeManager.requests <- bi
	}

	if !co.enabled {
		return hs.HookResult[hs.ProcessedAuctionRequestPayload]{
			Message: "skipped, rules engine is disabled for this account",
		}, nil
	}

	ruleSets := co.ruleSetsForProcessedAuctionRequestStage

	return handleProcessedAuctionHook(ruleSets, payload)
}

// Shutdown signals the module to stop processing and waits for the tree manager to finish
// processing any remaining build instructions in the channel.
func (m Module) Shutdown() {
	m.TreeManager.Shutdown()
	<-m.TreeManager.done
}

// rebuildTrees returns true if the trees for this account need to be rebuilt; false otherwise
func rebuildTrees(co *cacheEntry, jsonConfig *json.RawMessage) bool {
	if !expired(&timeutil.RealTime{}, co.timestamp) {
		return false
	}
	return configChanged(co.hashedConfig, jsonConfig)
}

// expired returns true if the refresh time has expired; false otherwise
func expired(t timeutil.Time, ts time.Time) bool {
	currentTime := t.Now().UTC()

	delta := currentTime.Sub(ts.UTC())
	if delta.Seconds() > fiveMinutes.Seconds() {
		return true
	}
	return false
}

// configChanged hashes the raw JSON config comparing it with the old hash returning
// true with the new hash if the hashes are different and false otherwise
func configChanged(oldHash hash, data *json.RawMessage) bool {
	if data == nil {
		return false
	}
	newHash := sha256.Sum256(*data)
	hashStr := hash(hex.EncodeToString(newHash[:]))

	if hashStr != oldHash {
		return true
	}
	return false
}
