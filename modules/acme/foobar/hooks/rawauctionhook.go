package hooks

import (
	"context"
	"net/http"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/hooks/stages"
	"github.com/prebid/prebid-server/modules/acme/foobar/config"
)

type RawAuctionHook struct {
	client *http.Client
	cfg    config.Config
}

func (h RawAuctionHook) Handle(
	_ context.Context,
	_ invocation.Context,
	request stages.BidRequest,
) (invocation.HookResult[stages.BidRequest], error) {
	if v, err := jsonparser.GetString(request, "attribute"); err == nil && v == "value" && h.cfg.AllowReject {
		return invocation.HookResult[stages.BidRequest]{Reject: true}, nil
	}
	return invocation.HookResult[stages.BidRequest]{}, nil
}

func NewRawAuctionHook(client *http.Client, cfg config.Config) RawAuctionHook {
	return RawAuctionHook{client: client, cfg: cfg}
}
