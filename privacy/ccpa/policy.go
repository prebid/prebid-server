package ccpa

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Policy represents the CCPA regulation for an OpenRTB bid request.
type Policy struct {
	Value         string
	NoSaleBidders []string
}

// ReadPolicy extracts the CCPA regulation policy from an OpenRTB regs ext.
func ReadPolicy(req *openrtb.BidRequest) (Policy, error) {
	policy := Policy{}

	if req != nil && req.Regs != nil && len(req.Regs.Ext) > 0 {
		var ext openrtb_ext.ExtRegs
		if err := json.Unmarshal(req.Regs.Ext, &ext); err != nil {
			return Policy{}, err
		}
		policy.Value = ext.USPrivacy
	}

	if req != nil && len(req.Ext) > 0 {
		var ext openrtb_ext.ExtRequest
		if err := json.Unmarshal(req.Ext, &ext); err != nil {
			return Policy{}, err
		}
		policy.NoSaleBidders = ext.Prebid.NoSale
	}

	return policy, nil
}

// Write mutates an OpenRTB bid request with the context of the CCPA policy.
func (p Policy) Write(req *openrtb.BidRequest) error {
	var err error

	err = p.writeRegsExt(req)
	if err != nil {
		return err
	}

	err = p.writeExt(req)
	if err != nil {
		return err
	}

	return nil
}

func (p Policy) writeRegsExt(req *openrtb.BidRequest) error {
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

func (p Policy) writeExt(req *openrtb.BidRequest) error {
	if len(p.NoSaleBidders) == 0 {
		return nil
	}

	var ext openrtb_ext.ExtRequest
	if len(req.Ext) == 0 {
		ext = openrtb_ext.ExtRequest{}
	} else {
		err := json.Unmarshal(req.Ext, &ext)
		if err != nil {
			return err
		}
	}

	ext.Prebid.NoSale = p.NoSaleBidders
	extJSON, err := json.Marshal(ext)
	if err != nil {
		return err
	}
	req.Ext = extJSON

	return nil
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
func (p Policy) ShouldEnforce(bidder string) bool {
	if err := p.Validate(); err != nil {
		return false
	}

	for _, b := range p.NoSaleBidders {
		if b == "*" || strings.EqualFold(b, bidder) {
			return false
		}
	}

	return p.Value != "" && p.Value[2] == 'Y'
}
