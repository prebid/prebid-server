package bt_server

import (
	"context"
	"encoding/json"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/modules/moduledeps"
)

func Builder(config json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	glog.Infof("module config: %s", config)
	return Module{}, nil
}

type Module struct{}

func (m Module) HandleEntryPointHook(
	ctx context.Context,
	invocationCtx hookstage.ModuleInvocationContext,
	payload hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	c := hookstage.ChangeSet[hookstage.EntrypointPayload]{}

	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		payload.Request.Header.Set("xi", "test")
		return payload, nil
	}, hookstage.MutationUpdate)

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}
