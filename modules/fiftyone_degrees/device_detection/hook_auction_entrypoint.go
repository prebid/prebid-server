package device_detection

import (
	"github.com/prebid/prebid-server/v2/hooks/hookexecution"
	"github.com/prebid/prebid-server/v2/hooks/hookstage"
)

// handleAuctionEntryPointRequestHook is a hookstage.HookFunc that is used to handle the auction entrypoint request hook.
func handleAuctionEntryPointRequestHook(cfg Config, payload hookstage.EntrypointPayload, deviceDetector deviceDetector, evidenceExtractor evidenceExtractor, accountValidator accountValidator) (result hookstage.HookResult[hookstage.EntrypointPayload], err error) {
	// if account/domain is not allowed, return failure
	if accountValidator.IsAllowed(cfg, payload.Body) != true {
		return hookstage.HookResult[hookstage.EntrypointPayload]{}, hookexecution.NewFailure("account not allowed")
	}
	// fetch evidence from headers and sua
	evidenceFromHeaders := evidenceExtractor.FromHeaders(payload.Request, deviceDetector.GetSupportedHeaders())
	evidenceFromSua := evidenceExtractor.FromSuaPayload(payload.Request, payload.Body)

	// create a module context and set the evidence from headers, evidence from sua and dd enabled flag
	moduleContext := make(hookstage.ModuleContext)
	moduleContext[EvidenceFromHeadersCtxKey] = evidenceFromHeaders
	moduleContext[EvidenceFromSuaCtxKey] = evidenceFromSua
	moduleContext[DDEnabledCtxKey] = true

	return hookstage.HookResult[hookstage.EntrypointPayload]{
		ModuleContext: moduleContext,
	}, nil
}
