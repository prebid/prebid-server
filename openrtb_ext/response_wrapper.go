package openrtb_ext

import (
	"encoding/json"

	"github.com/prebid/openrtb/v17/openrtb2"
)

// ResponseWrapper wraps the OpenRTB response and unmarshalled extensions
// objects.
// NOTE: Do not set bidResponse.Ext directly while using ResponseWrapper. It will overwrite the contents
// Instead prefer respExt.SetExt() method. Typicall usage
//
//	ext := respExt.GetExt()  // Get Copy ext object
//	ext["mykey"] = []byte(`my_value`)
//	respExt.SetExt(ext)
type ResponseWrapper struct {
	*openrtb2.BidResponse
	responseExt iResponseExt
}

func (rw *ResponseWrapper) rebuildResponseExt() error {
	if rw.responseExt == nil || !rw.responseExt.dirty() {
		return nil
	}
	responseExtJson, err := rw.responseExt.marshal()
	if err != nil {
		return err
	}
	rw.Ext = responseExtJson
	return nil
}

func (rw *ResponseWrapper) GetResponseExt() (iResponseExt, error) {
	if rw.responseExt != nil {
		return rw.responseExt, nil
	}
	rw.responseExt = &ResponseExt{}
	if rw.BidResponse == nil || rw.Ext == nil {
		return rw.responseExt, rw.responseExt.unmarshal(json.RawMessage{})
	}
	return rw.responseExt, rw.responseExt.unmarshal(rw.Ext)
}

func (rw *ResponseWrapper) RebuildResponse() error {
	if rw.BidResponse == nil {
		return nil
	}
	return rw.rebuildResponseExt()
}

// ---------------------------------------------------------------
// ResponseExt provides an interface for response.ext
// ---------------------------------------------------------------
type ResponseExt struct {
	prebid      *ExtResponsePrebid
	prebidDirty bool
	extMap      map[string]json.RawMessage // map version of response.ext
	extMapDirty bool
}
type iResponseExt interface {
	marshal() (json.RawMessage, error)
	unmarshal(json.RawMessage) error
	dirty() bool
}

func (re *ResponseExt) marshal() (json.RawMessage, error) {
	if re.prebidDirty {
		if re.prebid != nil {
			prebidJson, err := json.Marshal(re.prebid)
			if err != nil {
				return nil, err
			}
			if len(prebidJson) > jsonEmptyObjectLength {
				re.extMap[prebidKey] = prebidJson
			} else {
				delete(re.extMap, prebidKey)
			}
		}
		re.prebidDirty = false
	}
	re.extMapDirty = false
	if len(re.extMap) == 0 {
		return nil, nil
	}
	return json.Marshal(re.extMap)
}

func (re *ResponseExt) unmarshal(extJson json.RawMessage) error {
	if len(re.extMap) != 0 || re.dirty() {
		return nil
	}
	if len(extJson) == 0 {
		return nil
	}
	re.extMap = make(map[string]json.RawMessage)
	if err := json.Unmarshal(extJson, &re.extMap); err != nil {
		return err
	}
	prebidJson, hasPrebid := re.extMap[prebidKey]
	if hasPrebid {
		re.prebid = &ExtResponsePrebid{}
		if err := json.Unmarshal(prebidJson, re.prebid); err != nil {
			return err
		}
	}
	return nil
}

func (re *ResponseExt) dirty() bool {
	return re.prebidDirty || re.extMapDirty
}

func (re *ResponseExt) SetPrebid(prebid *ExtResponsePrebid) {
	re.prebid = prebid
	re.prebidDirty = true
}

// SetExt use this method instead of directly setting ResponseWrapper.Ext
func (re *ResponseExt) SetExt(ext map[string]json.RawMessage) {
	re.extMap = ext
	re.extMapDirty = true
}

func (re *ResponseExt) GetExt() map[string]json.RawMessage {
	ext := make(map[string]json.RawMessage)
	for k, v := range re.extMap {
		ext[k] = v
	}
	return ext
}

func (re *ResponseExt) GetPrebid() *ExtResponsePrebid {
	if re == nil || re.prebid == nil {
		return nil
	}
	prebid := *re.prebid
	return &prebid
}
