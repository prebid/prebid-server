package hooks

import (
	"context"

	"github.com/prebid/prebid-server/hooks/hookstage"
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
	_ *hookstage.ModuleContext,
	payload hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	if payload.Request.URL.Query().Get(h.cfg.Attributes.Name) != "" && h.cfg.AllowReject {
		return hookstage.HookResult[hookstage.EntrypointPayload]{Reject: true}, nil
	}
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
}
