package ccpa

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Policy represents the CCPA regulatory information from an OpenRTB bid request.
type Policy struct {
	Consent       string
	NoSaleBidders []string
}

// ReadFromRequest extracts the CCPA regulatory information from an OpenRTB bid request.
func ReadFromRequest(req *openrtb.BidRequest) (Policy, error) {
	var consent string
	var noSaleBidders []string

	if req == nil {
		return Policy{}, nil
	}

	// Read consent from request.regs.ext
	if req.Regs != nil && len(req.Regs.Ext) > 0 {
		var ext openrtb_ext.ExtRegs
		if err := json.Unmarshal(req.Regs.Ext, &ext); err != nil {
			return Policy{}, fmt.Errorf("error reading request.regs.ext: %s", err)
		}
		consent = ext.USPrivacy
	}

	// Read no sale bidders from request.ext.prebid
	if len(req.Ext) > 0 {
		var ext openrtb_ext.ExtRequest
		if err := json.Unmarshal(req.Ext, &ext); err != nil {
			return Policy{}, fmt.Errorf("error reading request.ext.prebid: %s", err)
		}
		noSaleBidders = ext.Prebid.NoSale
	}

	return Policy{consent, noSaleBidders}, nil
}

// Write mutates an OpenRTB bid request with the CCPA regulatory information.
func (p Policy) Write(req *openrtb.BidRequest) error {
	if req == nil {
		return nil
	}

	regs, err := buildRegs(p.Consent, req.Regs)
	if err != nil {
		return err
	}
	ext, err := buildExt(p.NoSaleBidders, req.Ext)
	if err != nil {
		return err
	}

	req.Regs = regs
	req.Ext = ext
	return nil
}

func buildRegs(consent string, regs *openrtb.Regs) (*openrtb.Regs, error) {
	if consent == "" {
		return buildRegsClear(regs)
	}
	return buildRegsWrite(consent, regs)
}

func buildRegsClear(regs *openrtb.Regs) (*openrtb.Regs, error) {
	if regs == nil || len(regs.Ext) == 0 {
		return regs, nil
	}

	var extMap map[string]interface{}
	if err := json.Unmarshal(regs.Ext, &extMap); err != nil {
		return nil, err
	}

	delete(extMap, "us_privacy")

	// Remove entire ext if it's now empty
	if len(extMap) == 0 {
		regsResult := *regs
		regsResult.Ext = nil
		return &regsResult, nil
	}

	// Marshal ext if there are still other fields
	var regsResult openrtb.Regs
	ext, err := json.Marshal(extMap)
	if err == nil {
		regsResult = *regs
		regsResult.Ext = ext
	}
	return &regsResult, err
}

func buildRegsWrite(consent string, regs *openrtb.Regs) (*openrtb.Regs, error) {
	if regs == nil {
		return marshalRegsExt(openrtb.Regs{}, openrtb_ext.ExtRegs{USPrivacy: consent})
	}

	if regs.Ext == nil {
		return marshalRegsExt(*regs, openrtb_ext.ExtRegs{USPrivacy: consent})
	}

	var extMap map[string]interface{}
	if err := json.Unmarshal(regs.Ext, &extMap); err != nil {
		return nil, err
	}

	extMap["us_privacy"] = consent
	return marshalRegsExt(*regs, extMap)
}

func marshalRegsExt(regs openrtb.Regs, ext interface{}) (*openrtb.Regs, error) {
	extJSON, err := json.Marshal(ext)
	if err == nil {
		regs.Ext = extJSON
	}
	return &regs, err
}

func buildExt(noSaleBidders []string, ext json.RawMessage) (json.RawMessage, error) {
	if len(noSaleBidders) == 0 {
		return buildExtClear(ext)
	}
	return buildExtWrite(noSaleBidders, ext)
}

func buildExtClear(ext json.RawMessage) (json.RawMessage, error) {
	if len(ext) == 0 {
		return ext, nil
	}

	var extMap map[string]interface{}
	if err := json.Unmarshal(ext, &extMap); err != nil {
		return nil, err
	}

	prebidExt, exists := extMap["prebid"]
	if !exists {
		return ext, nil
	}

	// Verify prebid is an object
	prebidExtMap, ok := prebidExt.(map[string]interface{})
	if !ok {
		return nil, errors.New("request.ext.prebid is not a json object")
	}

	// Remove no sale member
	delete(prebidExtMap, "nosale")
	if len(prebidExtMap) == 0 {
		delete(extMap, "prebid")
	}

	// Remove entire ext if it's empty
	if len(extMap) == 0 {
		return nil, nil
	}

	return json.Marshal(extMap)
}

func buildExtWrite(noSaleBidders []string, ext json.RawMessage) (json.RawMessage, error) {
	if len(ext) == 0 {
		return json.Marshal(openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{NoSale: noSaleBidders}})
	}

	var extMap map[string]interface{}
	if err := json.Unmarshal(ext, &extMap); err != nil {
		return nil, err
	}

	var prebidExt map[string]interface{}
	if prebidExtInterface, exists := extMap["prebid"]; exists {
		// Reference Existing Prebid Ext Map
		if prebidExtMap, ok := prebidExtInterface.(map[string]interface{}); ok {
			prebidExt = prebidExtMap
		} else {
			return nil, errors.New("request.ext.prebid is not a json object")
		}
	} else {
		// Create New Empty Prebid Ext Map
		prebidExt = make(map[string]interface{})
		extMap["prebid"] = prebidExt
	}

	prebidExt["nosale"] = noSaleBidders
	return json.Marshal(extMap)
}
