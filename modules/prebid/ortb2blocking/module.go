package ortb2blocking

import (
	"context"
	"encoding/json"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

func Builder(_ json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	return Module{}, nil
}

type Module struct{}

// HandleBidderRequestHook updates blocking fields on the openrtb2.BidRequest.
// Fields are updated only if request satisfies conditions provided by the module config.
func (m Module) HandleBidderRequestHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.BidderRequestPayload,
) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	result := hookstage.HookResult[hookstage.BidderRequestPayload]{}
	if len(miCtx.AccountConfig) == 0 {
		return result, nil
	}

	cfg, err := newConfig(miCtx.AccountConfig)
	if err != nil {
		return result, err
	}

	return handleBidderRequestHook(cfg, payload)
}

// HandleRawBidderResponseHook rejects bids for a specific bidder if they fail the attribute check.
func (m Module) HandleRawBidderResponseHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawBidderResponsePayload,
) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	result := hookstage.HookResult[hookstage.RawBidderResponsePayload]{}
	var cfg config
	if len(miCtx.AccountConfig) != 0 {
		ncfg, err := newConfig(miCtx.AccountConfig)
		if err != nil {
			return result, err
		}
		cfg = ncfg
	}

	return handleRawBidderResponseHook(cfg, payload, miCtx.ModuleContext)
}

type blockingAttributes struct {
	bAdv   []string
	bApp   []string
	bCat   []string
	bType  map[string][]int
	bAttr  map[string][]int
	catTax adcom1.CategoryTaxonomy
}
