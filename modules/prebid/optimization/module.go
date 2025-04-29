package optimization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/util/timeutil"
)

const fiveMinutes = time.Duration(300) * time.Second

// Builder configures the rules engine module initiating an in-memory cache and kicking
// off a go routine that builds tree structures that represent rule sets optimized for finding
// a rule to applies for a given request.
func Builder(_ json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	tb := treeBuilder{
		requests: make(chan buildInstruction),
	}
	c := cache{}

	go tb.Run(&c)

	return Module{
		Cache:       &c,
		TreeBuilder: &tb,
	}, nil
}

// Module represents the rules engine module
type Module struct {
	Cache       cacher
	TreeBuilder *treeBuilder
}

// HandleProcessedAuctionHook updates field on openrtb2.BidRequest.
// Fields are updated only if request satisfies conditions provided by the module config.
func (m Module) HandleProcessedAuctionHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	result := hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}

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
		m.TreeBuilder.requests <- bi

		// TODO: return with reject or no reject, possible config option
		return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}, nil
	}
	// cache hit
	if rebuildTrees(co, &miCtx.AccountConfig) {
		bi := buildInstruction{
			accountID: miCtx.AccountID,
			config:    &miCtx.AccountConfig,
		}
		m.TreeBuilder.requests <- bi
	}

	ruleSets := co.ruleSets["processed-auction-request"]

	return handleProcessedAuctionHook(ruleSets, payload)
}

// rebuildTrees returns true if the trees for this account need to be rebuilt; false otherwise
func rebuildTrees(co *cacheObject, jsonConfig *json.RawMessage) bool {
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
