package ccpa

import (
	"encoding/json"
	"errors"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Policy represents the CCPA regulation for an OpenRTB bid request.
type Policy struct {
	Value string
}

// ReadPolicy extracts the CCPA regulation policy from an OpenRTB regs ext.
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
		return nil
	}

	if req.Regs == nil {
		req.Regs = &openrtb.Regs{}
	}

	if req.Regs.Ext == nil {
		req.Regs.Ext = json.RawMessage(`{"us_privacy":"` + p.Value + `"}`)
		return nil
	}

	var err error
	req.Regs.Ext, err = jsonparser.Set(req.Regs.Ext, []byte(`"`+p.Value+`"`), "us_privacy")
	return err
}

// Validate returns an error if the CCPA regulation value does not adhere to the IAB spec.
func (p Policy) Validate() error {
	if p.Value == "" {
		return nil
	}

	if len(p.Value) != 4 {
		return errors.New("request.regs.ext.us_privacy must contain 4 characters")
	}

	if p.Value[0] != '1' {
		return errors.New("request.regs.ext.us_privacy must specify version 1")
	}

	var c byte

	c = p.Value[1]
	if c != 'N' && c != 'Y' && c != '-' {
		return errors.New("request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the explicit notice")
	}

	c = p.Value[2]
	if c != 'N' && c != 'Y' && c != '-' {
		return errors.New("request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the opt-out sale")
	}

	c = p.Value[3]
	if c != 'N' && c != 'Y' && c != '-' {
		return errors.New("request.regs.ext.us_privacy must specify 'N', 'Y', or '-' for the limited service provider agreement")
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
