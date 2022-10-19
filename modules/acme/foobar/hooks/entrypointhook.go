package hooks

import (
	"context"

	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/modules/acme/foobar/config"
)

func NewEntrypointHook(cfg config.Config) EntrypointHook {
	return EntrypointHook{cfg}
}

type EntrypointHook struct {
	cfg config.Config
}

func (h EntrypointHook) Handle(
	_ context.Context,
	_ invocation.Context,
	payload hookstage.EntrypointPayload,
) (invocation.HookResult[hookstage.EntrypointPayload], error) {
	if payload.Request.URL.Query().Get(h.cfg.Attributes.Name) != "" && h.cfg.AllowReject {
		return invocation.HookResult[hookstage.EntrypointPayload]{Reject: true}, nil
	}
	return invocation.HookResult[hookstage.EntrypointPayload]{}, nil
}
