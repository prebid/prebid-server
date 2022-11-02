package hooks

import (
	"context"
	"net/http"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/modules/acme/foobar/config"
)

type RawAuctionHook struct {
	client *http.Client
	cfg    config.Config
}

func (h RawAuctionHook) Handle(
	_ context.Context,
	_ hookstage.InvocationContext,
	request hookstage.RawAuctionPayload,
) (hookstage.HookResult[hookstage.RawAuctionPayload], error) {
	if v, err := jsonparser.GetString(request, "attribute"); err == nil && v == "value" && h.cfg.AllowReject {
		return hookstage.HookResult[hookstage.RawAuctionPayload]{Reject: true}, nil
	}
	return hookstage.HookResult[hookstage.RawAuctionPayload]{}, nil
}

func NewRawAuctionHook(client *http.Client, cfg config.Config) RawAuctionHook {
	return RawAuctionHook{client: client, cfg: cfg}
}
