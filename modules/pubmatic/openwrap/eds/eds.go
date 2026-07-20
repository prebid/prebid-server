package eds

import (
	"encoding/json"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
)

const bidderParamsEdsKey = "eds"

// ResolveEds extracts device.ext.eds and app.ext.eds from signal first, then the main request.
func ResolveEds(signal, request *openrtb2.BidRequest) models.ResolvedEds {
	if resolved := resolveEdsFromRequest(signal); !resolved.IsEmpty() {
		return resolved
	}
	return resolveEdsFromRequest(request)
}

// BuildPubmaticEdsBidderParams places resolved EDS under ext.prebid.bidderparams.{bidder}.eds
// for PubMatic (and PubMatic-core aliases). Signal ext.eds takes priority over the main request.
func BuildPubmaticEdsBidderParams(bidderParams json.RawMessage, signal, request *openrtb2.BidRequest, bidderCodes ...string) (json.RawMessage, models.ResolvedEds, error) {
	resolved := ResolveEds(signal, request)
	if resolved.IsEmpty() || len(bidderCodes) == 0 {
		return bidderParams, resolved, nil
	}

	updated, err := injectIntoBidderParams(bidderParams, resolved, bidderCodes...)
	return updated, resolved, err
}

// StripFromRequest removes device.ext.eds and app.ext.eds from the shared bid request
// so other bidders do not receive PubMatic-only EDS.
func StripFromRequest(req *openrtb2.BidRequest) {
	if req == nil {
		return
	}

	if req.Device != nil {
		req.Device.Ext = nilIfEmptyExt(jsonparser.Delete(req.Device.Ext, "eds"))
	}
	if req.App != nil {
		req.App.Ext = nilIfEmptyExt(jsonparser.Delete(req.App.Ext, "eds"))
	}
}

// StripFromDeviceCtx removes cached device.ext.eds so profile/device enrichment does not
// write EDS back onto the shared request after StripFromRequest.
func StripFromDeviceCtx(dvc *models.DeviceCtx) {
	if dvc == nil || dvc.Ext == nil {
		return
	}
	dvc.Ext.DeleteEds()
}

func resolveEdsFromRequest(req *openrtb2.BidRequest) models.ResolvedEds {
	if req == nil {
		return models.ResolvedEds{}
	}

	resolved := models.ResolvedEds{}
	if req.Device != nil {
		resolved.Device = nestedObject(req.Device.Ext, "eds")
	}
	if req.App != nil {
		resolved.App = nestedObject(req.App.Ext, "eds")
	}
	return resolved
}

func injectIntoBidderParams(bidderParams json.RawMessage, resolved models.ResolvedEds, bidderCodes ...string) (json.RawMessage, error) {
	edsPayload, err := buildEdsPayload(resolved)
	if err != nil {
		return bidderParams, err
	}

	result := cloneJSONBytes(bidderParams)
	if len(result) == 0 {
		result = []byte(`{}`)
	} else if _, _, _, err := jsonparser.Get(result); err != nil {
		return bidderParams, err
	}

	edsPayload = cloneJSONBytes(edsPayload)
	for _, code := range bidderCodes {
		result, err = jsonparser.Set(result, edsPayload, code, bidderParamsEdsKey)
		if err != nil {
			return bidderParams, err
		}
	}

	return result, nil
}

func buildEdsPayload(resolved models.ResolvedEds) (json.RawMessage, error) {
	payload := []byte(`{}`)
	var err error

	if len(resolved.Device) > 0 {
		payload, err = jsonparser.Set(payload, cloneJSONBytes(resolved.Device), "device")
		if err != nil {
			return nil, err
		}
	}
	if len(resolved.App) > 0 {
		payload, err = jsonparser.Set(payload, cloneJSONBytes(resolved.App), "app")
		if err != nil {
			return nil, err
		}
	}

	return payload, nil
}

func cloneJSONBytes(b json.RawMessage) json.RawMessage {
	if len(b) == 0 {
		return b
	}
	copied := make([]byte, len(b))
	copy(copied, b)
	return copied
}

// nestedObject returns a deep copy of the raw JSON value at key when it is a non-empty object.
func nestedObject(ext []byte, key string) []byte {
	if len(ext) == 0 {
		return nil
	}

	value, dataType, _, err := jsonparser.Get(ext, key)
	if err != nil || dataType != jsonparser.Object || len(value) <= 2 {
		return nil
	}

	return cloneJSONBytes(value)
}

func nilIfEmptyExt(ext []byte) []byte {
	if len(ext) == 0 {
		return nil
	}

	_, dataType, _, err := jsonparser.Get(ext)
	if err != nil || dataType != jsonparser.Object {
		return nil
	}

	hasKeys := false
	if err := jsonparser.ObjectEach(ext, func(_ []byte, _ []byte, _ jsonparser.ValueType, _ int) error {
		hasKeys = true
		return nil
	}); err != nil || !hasKeys {
		return nil
	}

	return ext
}
