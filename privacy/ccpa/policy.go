package ccpa

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Policy represents the CCPA regulation for an OpenRTB bid request.
type Policy struct {
	Value string
}

// ReadPolicy extracts the CCPA regulation policy from an OpenRTB request.
func ReadPolicy(req *openrtb.BidRequest) (Policy, error) {
	policy := Policy{}

	if req != nil && req.Regs != nil && len(req.Regs.Ext) > 0 {
		var ext openrtb_ext.ExtRegs
		if err := json.Unmarshal(req.Regs.Ext, &ext); err != nil {
			return policy, err
		}
		policy.Value = ext.USPrivacy
	}

	return policy, nil
}

// Write mutates an OpenRTB bid request with the context of the CCPA policy.
func (p Policy) Write(req *openrtb.BidRequest) error {
	if p.Value == "" {
		return clearPolicy(req)
	}

	if req == nil {
		return nil
	}

	if req.Regs == nil {
		req.Regs = &openrtb.Regs{}
	}

	if req.Regs.Ext == nil {
		ext, err := json.Marshal(openrtb_ext.ExtRegs{USPrivacy: p.Value})
		if err == nil {
			req.Regs.Ext = ext
		}
		return err
	}

	var extMap map[string]interface{}
	err := json.Unmarshal(req.Regs.Ext, &extMap)
	if err == nil {
		extMap["us_privacy"] = p.Value
		ext, err := json.Marshal(extMap)
		if err == nil {
			req.Regs.Ext = ext
		}
	}
	return err
}

func clearPolicy(req *openrtb.BidRequest) error {
	if req == nil {
		return nil
	}

	if req.Regs == nil {
		return nil
	}

	if len(req.Regs.Ext) == 0 {
		return nil
	}

	var extMap map[string]interface{}
	err := json.Unmarshal(req.Regs.Ext, &extMap)
	if err == nil {
		delete(extMap, "us_privacy")
		if len(extMap) == 0 {
			req.Regs.Ext = nil
		} else {
			ext, err := json.Marshal(extMap)
			if err == nil {
				req.Regs.Ext = ext
			}
			return err
		}
	}

	return err
}

// Validate returns an error if the CCPA policy does not adhere to the IAB spec.
func (p Policy) Validate() error {
	if err := ValidateConsent(p.Value); err != nil {
		return fmt.Errorf("request.regs.ext.us_privacy %s", err.Error())
	}

	return nil
}

// ValidateConsent returns an error if the CCPA consent string does not adhere to the IAB spec.
func ValidateConsent(consent string) error {
	if consent == "" {
		return nil
	}

	if len(consent) != 4 {
		return errors.New("must contain 4 characters")
	}

	if consent[0] != '1' {
		return errors.New("must specify version 1")
	}

	var c byte

	c = consent[1]
	if c != 'N' && c != 'Y' && c != '-' {
		return errors.New("must specify 'N', 'Y', or '-' for the explicit notice")
	}

	c = consent[2]
	if c != 'N' && c != 'Y' && c != '-' {
		return errors.New("must specify 'N', 'Y', or '-' for the opt-out sale")
	}

	c = consent[3]
	if c != 'N' && c != 'Y' && c != '-' {
		return errors.New("must specify 'N', 'Y', or '-' for the limited service provider agreement")
	}

	return nil
}

// ShouldEnforce returns true when the opt-out signal is explicitly detected.
func (p Policy) ShouldEnforce() bool {
	if err := p.Validate(); err != nil {
		return false
	}

	return p.Value != "" && p.Value[2] == 'Y'
}
