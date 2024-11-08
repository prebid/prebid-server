package devicedetection

import (
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
)

// handleAuctionEntryPointRequestHook is a hookstage.HookFunc that is used to handle the auction entrypoint request hook.
func handleAuctionEntryPointRequestHook(cfg config, payload hookstage.EntrypointPayload, deviceDetector deviceDetector, evidenceExtractor evidenceExtractor, accountValidator accountValidator) (result hookstage.HookResult[hookstage.EntrypointPayload], err error) {
	// if account/domain is not allowed, return failure
	if !accountValidator.isAllowed(cfg, payload.Body) {
		return hookstage.HookResult[hookstage.EntrypointPayload]{}, hookexecution.NewFailure("account not allowed")
	}
	// fetch evidence from headers and sua
	evidenceFromHeaders := evidenceExtractor.fromHeaders(payload.Request, deviceDetector.getSupportedHeaders())
	evidenceFromSua := evidenceExtractor.fromSuaPayload(payload.Body)

	// create a Module context and set the evidence from headers, evidence from sua and dd enabled flag
	moduleContext := make(hookstage.ModuleContext)
	moduleContext[evidenceFromHeadersCtxKey] = evidenceFromHeaders
	moduleContext[evidenceFromSuaCtxKey] = evidenceFromSua
	moduleContext[ddEnabledCtxKey] = true

	return hookstage.HookResult[hookstage.EntrypointPayload]{
		ModuleContext: moduleContext,
	}, nil
}
