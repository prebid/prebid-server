package ortb2blocking

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/prebid/openrtb/v17/adcom1"
	"github.com/prebid/prebid-server/hooks/hookstage"
)

func Builder(_ json.RawMessage, _ *http.Client) (interface{}, error) {
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

const ctxKeyBlockingAttributes = "blocking_attributes"

type blockingAttributes struct {
	badv   []string
	bapp   []string
	bcat   []string
	btype  map[string][]int
	battr  map[string][]int
	cattax adcom1.CategoryTaxonomy
}
