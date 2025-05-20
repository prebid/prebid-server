package wurfl_devicedetection

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/tidwall/sjson"
)

const (
	wurflHeaderCtxKey = "wurfl_header"
)

// declare conformity with  hookstage.RawActionRequest interface
var (
	_ hookstage.RawAuctionRequest = Module{}
	_ hookstage.Entrypoint        = Module{}
)

// payloadPublisherIDPaths specifies the possible paths in the Request payload JSON
// where the publisher ID can be defined.
var payloadPublisherIDPaths = [][]string{
	{"site", "publisher", "id"},
	{"app", "publisher", "id"},
	{"dooh", "publisher", "id"},
}

func Builder(configRaw json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	cfg, err := newConfig(configRaw)
	if err != nil {
		return nil, err
	}

	we, err := newWurflEngine(cfg)
	if err != nil {
		return nil, err
	}

	m := Module{
		we:      we,
		extCaps: cfg.ExtCaps,
	}
	if len(cfg.AllowedPublisherIDs) > 0 {
		m.allowedPublisherIDs = make(map[string]struct{}, len(cfg.AllowedPublisherIDs))
		for _, v := range cfg.AllowedPublisherIDs {
			m.allowedPublisherIDs[v] = struct{}{}
		}
	}
	return m, nil
}

// Module must implement at least 1 hook interface.
type Module struct {
	we                  wurflDeviceDetection
	allowedPublisherIDs map[string]struct{}
	extCaps             bool
}

// HandleEntrypointHook implements hookstage.Entrypoint.
func (m Module) HandleEntrypointHook(ctx context.Context, invocationCtx hookstage.ModuleInvocationContext, payload hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	result := hookstage.HookResult[hookstage.EntrypointPayload]{}
	if !m.isPublisherAllowed(payload.Body) {
		return result, hookexecution.NewFailure("publisher not allowed")
	}
	header := map[string]string{}
	if payload.Request != nil {
		for k := range payload.Request.Header {
			header[k] = payload.Request.Header.Get(k)
		}
	}
	moduleContext := make(hookstage.ModuleContext)
	moduleContext[wurflHeaderCtxKey] = header
	result.ModuleContext = moduleContext

	return result, nil
}

// HandleRawAuctioneHook implements hookstage.RawAuctionRequest.
func (m Module) HandleRawAuctionHook(
	ctx context.Context,
	invocationCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawAuctionRequestPayload,
) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	result := hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}

	if invocationCtx.ModuleContext == nil {
		// The module context has not be inizialized in the entrypoint hook.
		// This could be due to a not allowed publisher or an error.
		// Return the payload as is
		return result, hookexecution.NewFailure("module context has not been inizialized in the entrypoint hook")
	}

	if isWURFLEnrichedRequest(payload) {
		return result, nil
	}

	rawHeaders := invocationCtx.ModuleContext[wurflHeaderCtxKey].(map[string]string)
	result.ChangeSet.AddMutation(func(payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
		ortb2Device, err := getOrtb2Device(payload)
		if err != nil {
			return payload, hookexecution.NewFailure("could not get ortb2.Device from payload %s", err)
		}

		headers := makeHeaders(ortb2Device, rawHeaders)

		wd, err := m.we.DeviceDetection(headers)
		if err != nil {
			return payload, hookexecution.NewFailure("could not perform WURFL device detection %s", err)
		}

		we := &wurflEnricher{
			WurflData: wd,
			ExtCaps:   m.extCaps,
		}
		we.EnrichDevice(&ortb2Device)

		updatedPayload, err := sjson.SetBytes(payload, "device", ortb2Device)
		if err != nil {
			return payload, hookexecution.NewFailure("could not update ortb2.Device payload %s", err)
		}
		return updatedPayload, nil
	},
		hookstage.MutationUpdate,
		"device",
	)
	return result, nil
}

// isPublisherAllowed verifies whether the publisher ID from a request payload is allowed to use this module.
// It checks against a list of authorized IDs, searching for the publisher ID in the site, app, or DOOH fields.
func (m Module) isPublisherAllowed(payload []byte) bool {
	if m.allowedPublisherIDs == nil {
		return true
	}
	var publisherID string
	jsonparser.EachKey(payload, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
		if err != nil {
			return
		}
		if vt != jsonparser.String {
			return
		}
		publisherID = string(value)
	}, payloadPublisherIDPaths...)
	if publisherID == "" {
		return false
	}
	_, found := m.allowedPublisherIDs[publisherID]
	return found
}

// getOrbt2Device extracts the openrtb2.Device from the bid request body.
func getOrtb2Device(payload []byte) (openrtb2.Device, error) {
	device := openrtb2.Device{}
	b, t, _, err := jsonparser.Get(payload, "device")
	if err != nil {
		return device, err
	}
	if t != jsonparser.Object {
		return device, fmt.Errorf("expecting Object, got %s", t)
	}
	err = json.Unmarshal(b, &device)
	return device, err
}

// isWURFLEnrichedRequest returns true if the payload request has been already
// enriched with WURFL data like requests from Prebid.js with WURFL RTD module
func isWURFLEnrichedRequest(payload []byte) bool {
	_, _, _, err := jsonparser.Get(payload, "device", "ext", ortb2WurflExtKey)
	return err == nil
}
