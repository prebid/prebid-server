package rulesengine

import (
	"context"
	"encoding/json"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

func Builder(_ json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	// TODO: is there any host-level config that needs to be attached to the module?
	return Module{}, nil
}

type Module struct{
	Cache cacher
	// Cache map[string]CacheObject // string = account id
}

// HandleProcessedAuctionHook updates field on openrtb2.BidRequest.
// Fields are updated only if request satisfies conditions provided by the module config.
func (m Module) HandleProcessedAuctionHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	result := hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}
	
	// can the module be defined in the host config? how would that be provided?
	// if there is no account-specific config but there is a default config, would that be passed in?
	// will also need to check miCtx.AccountID
	if len(miCtx.AccountConfig) == 0 {
		return result, nil
	}

	// if the cache contains an entry with account ID
		// tree = cache[accountID]
	// else
		// unmarshal account config to structs --> newConfig()
		// validate account config --> validateConfig()
		// if validation fails
			// return
		// for each module group
			// build tree for account
			// if build tree fails (most likely due to schema/result func param type errors)
				// return
		// cache module groups with trees
		// pass module groups to handleProcessedAuctionHook

	root := Node{}
	cacheModelGroups := []cacheModelGroup{{root: root}}

	return handleProcessedAuctionHook(cacheModelGroups, payload)
}
