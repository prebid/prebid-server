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

// RawPolicy represents the user provided CCPA regulation values.
type RawPolicy struct {
	Value         string
	NoSaleBidders []string
}

// ParsedPolicy represents the parsed and validated CCPA regulation values.
type ParsedPolicy struct {
	OptOutSaleYes         bool
	NoSaleAllBidders      bool
	NoSaleSpecificBidders map[string]struct{}
}

// ReadPolicy extracts the CCPA regulation policy from an OpenRTB request.
func ReadPolicy(req *openrtb.BidRequest) (Policy, error) {
	policy := Policy{}

	var value string
	if req.Regs != nil && len(req.Regs.Ext) > 0 {
		var ext openrtb_ext.ExtRegs
		if err := json.Unmarshal(req.Regs.Ext, &ext); err != nil {
			return RawPolicy{}, err
		}
		value = ext.USPrivacy
	}

	var noSaleBidders []string
	if len(req.Ext) > 0 {
		var ext openrtb_ext.ExtRequest
		if err := json.Unmarshal(req.Ext, &ext); err != nil {
			return RawPolicy{}, err
		}
		noSaleBidders = ext.Prebid.NoSale
	}

	result := RawPolicy{{
		Value: value,
		NoSaleBidders: noSaleBidders,
	}
	return result, nil
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

// Validate returns an error if the CCPA policy does not adhere to the IAB spec or the NoSale list is invalid.
func (p Policy) Validate(bidders []string) error {
	if err := ValidateConsent(p.Value); err != nil {
		return fmt.Errorf("request.regs.ext.us_privacy %s", err.Error())
	}

	if err := ValidateNoSaleBidders(p.NoSaleBidders); err != nil {
		return fmt.Errorf("request.ext.prebid.nosale %s", err.Error())
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

func ValidateNoSaleBidders(noSaleBidders []string, bidders map[string]openrtb_ext.BidderName, aliases map[string]string) error {
	if len(noSaleBidders) == 1 && noSaleBidders[0] == allBidders {
		return nil
	}

	for _, bidder := range noSaleBidders {
		if !validBidders.cotains[bidder] {
			return fmt.Errorf("unrecognized bidder '%s'", bidder)
		}
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
