package floors

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

// Builder creates a new floors injector module instance
func Builder(_ json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	return &FloorsInjector{}, nil
}

type FloorsInjector struct{}

func (f *FloorsInjector) HandleRawAuctionHook(
	ctx context.Context,
	moduleCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawAuctionRequestPayload,
) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {

	c := hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	c.AddMutation(
		func(orig hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
			// Parse the incoming OpenRTB request
			var req map[string]interface{}
			if err := json.Unmarshal(orig, &req); err != nil {
				// If unmarshal fails, return original payload (fail open)
				return orig, nil
			}

			// Ensure ext and ext.prebid exist
			ext, ok := req["ext"].(map[string]interface{})
			if !ok {
				ext = make(map[string]interface{})
				req["ext"] = ext
			}
			prebid, ok := ext["prebid"].(map[string]interface{})
			if !ok {
				prebid = make(map[string]interface{})
				ext["prebid"] = prebid
			}

			// Ensure floors exists
			floors, ok := prebid["floors"].(map[string]interface{})
			if !ok {
				floors = make(map[string]interface{})
				prebid["floors"] = floors
			}

			// Inject floorendpoint - this triggers the fetch
			// Don't set 'location' or 'fetchstatus' - Prebid Server sets these after fetch
			floors["floorendpoint"] = map[string]interface{}{
				// "url": "https://floors.atmtd.com/floors.json?siteID=g35tzr",
				"url": "http://localhost:8000/api/domains/floors_test/",
			}
			floors["enabled"] = true
			floors["enforcement"] = map[string]interface{}{
				"enforcerate": 100,  // 0-100, where 100 = always enforce
				"enforcepbs":  true, // enforce in PBS
				"floordeals":  true, // enforce for deals too
			}

			fmt.Println(floors)

			// Marshal back to JSON
			mutated, err := json.Marshal(req)
			if err != nil {
				return orig, nil
			}

			fmt.Println(string(mutated))
			return hookstage.RawAuctionRequestPayload(mutated), nil
		}, hookstage.MutationUpdate, "ext", "prebid", "floors",
	)

	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{
		Reject:    false,
		ChangeSet: c,
	}, nil
}
