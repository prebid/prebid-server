package exit_point_demo

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

type (
	Module struct{}
)

func Builder(_ json.RawMessage, _ moduledeps.ModuleDeps) (any, error) {
	return Module{}, nil
}

func (m Module) HandleExitPointHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	payload hookstage.ExitPointPayload,
) (hookstage.HookResult[hookstage.ExitPointPayload], error) {
	moduleContext := make(hookstage.ModuleContext)
	result := hookstage.HookResult[hookstage.ExitPointPayload]{
		ModuleContext: moduleContext,
	}

	// this is example where custom type struct is returned
	result.ChangeSet.AddMutation(func(payload hookstage.ExitPointPayload) (hookstage.ExitPointPayload, error) {
		payload.HTTPHeaders = http.Header{
			"Content-Type": []string{"application/javascript", "text/html"},
			"Accept":       []string{"application/json", "text/html"},
		}
		payload.Response = nil
		return payload, nil
	}, hookstage.MutationUpdate)
	//
	//// this is example where custom type struct is returned
	//result.ChangeSet.AddMutation(func(payload hookstage.ExitPointPayload) (hookstage.ExitPointPayload, error) {
	//	payload.HTTPHeaders = http.Header{
	//		"Content-Type": []string{"application/javascript", "text/html"},
	//		"Accept":       []string{"application/json", "text/html"},
	//	}
	//	payload.Response = struct {
	//		Imp     []openrtb2.Imp     `json:"imp,omitempty"`
	//		BidID   string             `json:"bidid,omitempty"`
	//		SeatBid []openrtb2.SeatBid `json:"seatbid,omitempty"`
	//	}{
	//		Imp:     payload.BidRequest.Imp,
	//		BidID:   payload.BidResponse.BidID,
	//		SeatBid: payload.BidResponse.SeatBid,
	//	}
	//	return payload, nil
	//}, hookstage.MutationUpdate)

	// this is example where json is returned
	//result.ChangeSet.AddMutation(func(payload hookstage.ExitPointPayload) (hookstage.ExitPointPayload, error) {
	//	payload.HTTPHeaders = http.Header{
	//		"Content-Type": []string{"application/javascript", "text/html"},
	//		"Accept":       []string{"application/json", "text/html"},
	//	}
	//	var j json.RawMessage
	//	j, _ = sjson.SetBytes(j, "imp", "value")
	//	j, _ = sjson.SetBytes(j, "bidid", 123)
	//	j, _ = sjson.SetBytes(j, "seatbid", true)
	//	payload.Response = j
	//	return payload, nil
	//}, hookstage.MutationUpdate)

	return result, nil
}
