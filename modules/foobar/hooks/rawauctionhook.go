package hooks

import (
	"context"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"net/http"

	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/modules/foobar/config"
)

type CheckBodyRawAuctionHook struct {
	client *http.Client
	cfg    config.Config
}

func (h CheckBodyRawAuctionHook) Call(
	_ context.Context,
	_ invocation.InvocationContext,
	_ hookstage.BidRequest,
) (invocation.HookResult[hookstage.BidRequest], error) {
	// comment for now as this demonstrative module may be removed, it has old implementation
	//if v, err := jsonparser.GetString(request, "attribute"); err == nil && v == "value" && h.cfg.AllowReject {
	//	return invocation.HookResult[hookstage.BidRequest]{Reject: true}, nil
	//}
	return invocation.HookResult[hookstage.BidRequest]{}, nil
}

func NewCheckBodyRawAuctionHook(client *http.Client, cfg config.Config) CheckBodyRawAuctionHook {
	return CheckBodyRawAuctionHook{client: client, cfg: cfg}
}
