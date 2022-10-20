package hooks

import (
	"context"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/modules/foobar/config"
)

func NewValidateHeaderEntrypointHook(cfg config.Config) ValidateHeaderEntrypointHook {
	return ValidateHeaderEntrypointHook{cfg}
}

type ValidateHeaderEntrypointHook struct {
	cfg config.Config
}

func (h ValidateHeaderEntrypointHook) Call(_ context.Context, _ invocation.InvocationContext, payload hookstage.EntrypointPayload) (invocation.HookResult[hookstage.EntrypointPayload], error) {
	if payload.Request.Header.Get("someheader") != "" && h.cfg.AllowReject {
		return invocation.HookResult[hookstage.EntrypointPayload]{Reject: true}, nil
	}
	return invocation.HookResult[hookstage.EntrypointPayload]{}, nil
}

func NewValidateQueryParamEntrypointHook(cfg config.Config) ValidateQueryParamEntrypointHook {
	return ValidateQueryParamEntrypointHook{cfg}
}

type ValidateQueryParamEntrypointHook struct {
	cfg config.Config
}

func (h ValidateQueryParamEntrypointHook) Call(_ context.Context, _ invocation.InvocationContext, payload hookstage.EntrypointPayload) (invocation.HookResult[hookstage.EntrypointPayload], error) {
	if payload.Request.URL.Query().Get("someparam") != "" && h.cfg.AllowReject {
		return invocation.HookResult[hookstage.EntrypointPayload]{Reject: true}, nil
	}
	return invocation.HookResult[hookstage.EntrypointPayload]{}, nil
}
