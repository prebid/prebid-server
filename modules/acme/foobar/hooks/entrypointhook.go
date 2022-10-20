package hooks

import (
	"context"
	"fmt"

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
	_ *invocation.ModuleContext,
	payload hookstage.EntrypointPayload,
	debug bool,
) (invocation.HookResult[hookstage.EntrypointPayload], error) {
	if payload.Request.URL.Query().Get(h.cfg.Attributes.Name) != "" && h.cfg.AllowReject {
		resp := invocation.HookResult[hookstage.EntrypointPayload]{Reject: true}
		if debug {
			resp.DebugMessages = []string{fmt.Sprintf("`Name` attr in query: %s", h.cfg.Attributes.Name)}
		}
		return invocation.HookResult[hookstage.EntrypointPayload]{Reject: true}, nil
	}
	return invocation.HookResult[hookstage.EntrypointPayload]{}, nil
}
