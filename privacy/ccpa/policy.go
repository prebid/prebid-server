package ccpa

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	ccpaVersion1      = '1'
	ccpaNo            = 'N'
	ccpaYes           = 'Y'
	ccpaNotApplicable = '-'
)

const (
	indexVersion                = 0
	indexExplicitNotice         = 1
	indexOptOutSale             = 2
	indexLSPACoveredTransaction = 3
)

const allBidders = "*"

// Policy represents the CCPA regulation for an OpenRTB bid request.
type Policy struct {
	Value         string
	NoSaleBidders []string
}

// ReadPolicy extracts the CCPA regulation policy from an OpenRTB regs ext.
func ReadPolicy(req *openrtb.BidRequest) (Policy, error) {
	policy := Policy{}

	if req == nil {
		return policy, nil
	}

	if req.Regs != nil && len(req.Regs.Ext) > 0 {
		var ext openrtb_ext.ExtRegs
		if err := json.Unmarshal(req.Regs.Ext, &ext); err != nil {
			return policy, err
		}
		policy.Value = ext.USPrivacy
	}

	if len(req.Ext) > 0 {
		var ext openrtb_ext.ExtRequest

		// Errors with reading the NoSaleBidders list shouldn't block enforcement of CCPA, so take a
		// 'best effort' approach here and ignore problems unmarshalling the Prebid extension. Failure
		// here is very unlikely due to request validation happening early in the auction endpoint.
		if err := json.Unmarshal(req.Ext, &ext); err == nil {
			policy.NoSaleBidders = ext.Prebid.NoSale
		}
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
	if len(p.Value) == 0 {
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

func (p Policy) writeExt(req *openrtb.BidRequest) error {
	if len(p.NoSaleBidders) == 0 {
		return nil
	}

	if len(req.Ext) == 0 {
		ext := openrtb_ext.ExtRequest{}
		ext.Prebid.NoSale = p.NoSaleBidders

		extJSON, err := json.Marshal(ext)
		if err == nil {
			req.Ext = extJSON
		}

		return err
	}

	var extMap map[string]interface{}
	if err := json.Unmarshal(req.Ext, &extMap); err != nil {
		return err
	}

	var extMapPrebid map[string]interface{}
	if v, exists := extMap["prebid"]; !exists {
		extMapPrebid = make(map[string]interface{})
		extMap["prebid"] = extMapPrebid
	} else {
		vCasted, ok := v.(map[string]interface{})
		if !ok {
			return errors.New("invalid data type for ext.prebid")
		}
		extMapPrebid = vCasted
	}

	extMapPrebid["nosale"] = p.NoSaleBidders

	extJSON, err := json.Marshal(extMap)
	if err == nil {
		req.Ext = extJSON
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

	if consent[indexVersion] != ccpaVersion1 {
		return errors.New("must specify version 1")
	}

	var c byte

	c = consent[indexExplicitNotice]
	if c != ccpaNo && c != ccpaYes && c != ccpaNotApplicable {
		return errors.New("must specify 'N', 'Y', or '-' for the explicit notice")
	}

	c = consent[indexOptOutSale]
	if c != ccpaNo && c != ccpaYes && c != ccpaNotApplicable {
		return errors.New("must specify 'N', 'Y', or '-' for the opt-out sale")
	}

	c = consent[indexLSPACoveredTransaction]
	if c != ccpaNo && c != ccpaYes && c != ccpaNotApplicable {
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
		if b == allBidders || strings.EqualFold(b, bidder) {
			return false
		}
	}

	return p.Value != "" && p.Value[indexOptOutSale] == ccpaYes
}
