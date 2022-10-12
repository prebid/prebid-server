package hooks

import (
	"context"
	"net/http"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/hooks/stages"
	"github.com/prebid/prebid-server/modules/foobar/config"
)

type CheckBodyRawAuctionHook struct {
	client *http.Client
	cfg    config.Config
}

func (h CheckBodyRawAuctionHook) Call(
	_ context.Context,
	_ invocation.Context,
	request stages.BidRequest,
) (invocation.HookResult[stages.BidRequest], error) {
	if v, err := jsonparser.GetString(request, "attribute"); err == nil && v == "value" && h.cfg.AllowReject {
		return invocation.HookResult[stages.BidRequest]{Reject: true}, nil
	}
	return invocation.HookResult[stages.BidRequest]{}, nil
}

func NewCheckBodyRawAuctionHook(client *http.Client, cfg config.Config) CheckBodyRawAuctionHook {
	return CheckBodyRawAuctionHook{client: client, cfg: cfg}
}
