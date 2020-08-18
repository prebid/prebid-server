package ccpa

import (
	"encoding/json"
	"errors"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Policy represents the CCPA regulatory information from the OpenRTB bid request.
type Policy struct {
	Consent       string
	NoSaleBidders []string
}

// ReadPolicy extracts the CCPA regulatory information from the OpenRTB bid request.
func ReadPolicy(req *openrtb.BidRequest) (Policy, error) {
	var consent string
	var noSaleBidders []string

	if req == nil {
		return Policy{}, nil
	}

	// Read consent from request.regs.ext
	if req.Regs != nil && len(req.Regs.Ext) > 0 {
		var ext openrtb_ext.ExtRegs
		if err := json.Unmarshal(req.Regs.Ext, &ext); err != nil {
			return Policy{}, err
		}
		consent = ext.USPrivacy
	}

	// Read no sale bidders from request.ext.prebid
	if len(req.Ext) > 0 {
		var ext openrtb_ext.ExtRequest
		if err := json.Unmarshal(req.Ext, &ext); err != nil {
			return Policy{}, err
		}
		noSaleBidders = ext.Prebid.NoSale
	}

	return Policy{consent, noSaleBidders}, nil
}

// Write mutates an OpenRTB bid request with the CCPA regulatory information.
func (p Policy) Write(req *openrtb.BidRequest) (err error) {
	if req == nil {
		return
	}

	regs, err := buildRegs(p.Consent, req.Regs)
	if err != nil {
		return
	}
	ext, err := buildExt(p.NoSaleBidders, req.Ext)
	if err != nil {
		return
	}

	req.Regs = regs
	req.Ext = ext

	return
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
	err := json.Unmarshal(regs.Ext, &extMap)
	if err == nil {
		regsCopy := *regs

		// Remove CCPA consent
		delete(extMap, "us_privacy")

		// Remove entire ext if it's empty
		if len(extMap) == 0 {
			regsCopy.Ext = nil
			return &regsCopy, nil
		}

		ext, err := json.Marshal(extMap)
		if err != nil {
			return nil, err
		}

		regsCopy.Ext = ext
		return &regsCopy, nil
	}
	return nil, err
}

func buildRegsWrite(consent string, regs *openrtb.Regs) (*openrtb.Regs, error) {
	var regsCopy openrtb.Regs

	if regs == nil {
		regsCopy = openrtb.Regs{}
	} else {
		regsCopy = *regs
	}

	if regsCopy.Ext == nil {
		ext, err := json.Marshal(openrtb_ext.ExtRegs{USPrivacy: consent})
		if err != nil {
			return nil, err
		}

		regsCopy.Ext = ext
		return &regsCopy, nil
	}

	var extMap map[string]interface{}
	err := json.Unmarshal(regsCopy.Ext, &extMap)
	if err == nil {
		// Set CCPA consent
		extMap["us_privacy"] = consent

		ext, err := json.Marshal(extMap)
		if err != nil {
			return nil, err
		}

		regsCopy.Ext = ext
		return &regsCopy, nil
	}

	return nil, err
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
	err := json.Unmarshal(ext, &extMap)
	if err == nil {
		prebidExt, exists := extMap["prebid"]

		// If there's no prebid, there's nothing to do
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
	return nil, err
}

func buildExtWrite(noSaleBidders []string, ext json.RawMessage) (json.RawMessage, error) {
	if len(ext) == 0 {
		return json.Marshal(openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{NoSale: noSaleBidders}})
	}

	var extMap map[string]interface{}
	err := json.Unmarshal(ext, &extMap)
	if err == nil {
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
	return nil, err
}
