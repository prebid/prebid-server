package ccpa

import (
	"errors"
	"fmt"
)

const (
	ccpaVersion1      = '1'
	ccpaYes           = 'Y'
	ccpaNo            = 'N'
	ccpaNotApplicable = '-'
)

const (
	indexVersion                = 0
	indexExplicitNotice         = 1
	indexOptOutSale             = 2
	indexLSPACoveredTransaction = 3
)

const allBiddersMarker = "*"

// ParsedPolicy represents parsed and validated CCPA regulatory information from the OpenRTB bid request.
type ParsedPolicy struct {
	Policy
	isValid               bool
	consentOptOut         bool
	noSaleForAllBidders   bool
	noSaleSpecificBidders map[string]struct{}
}

// Parse returns a parsed and validated ParsedPolicy which can be used for enforcement checks.
func (p Policy) Parse(validBidders map[string]struct{}) (ParsedPolicy, error) {
	consentOptOut, err := parseConsent(p.Consent)
	if err != nil {
		return ParsedPolicy{isValid: false}, fmt.Errorf("request.regs.ext.us_privacy is invalid. %s", err.Error())
	}

	noSaleForAllBidders, noSaleSpecificBidders, err := parseNoSaleBidders(p.NoSaleBidders, validBidders)
	if err != nil {
		return ParsedPolicy{isValid: false}, fmt.Errorf("request.ext.prebid.nosale is invalid. %s", err.Error())
	}

	return ParsedPolicy{
		Policy:                p,
		isValid:               true,
		consentOptOut:         consentOptOut,
		noSaleForAllBidders:   noSaleForAllBidders,
		noSaleSpecificBidders: noSaleSpecificBidders,
	}, nil
}

func parseConsent(consent string) (consentOptOut bool, err error) {
	if consent == "" {
		return false, nil
	}

	if len(consent) != 4 {
		return false, errors.New("must contain 4 characters")
	}

	if consent[indexVersion] != ccpaVersion1 {
		return false, errors.New("must specify version 1")
	}

	var c byte

	c = consent[indexExplicitNotice]
	if c != ccpaNo && c != ccpaYes && c != ccpaNotApplicable {
		return false, errors.New("must specify 'N', 'Y', or '-' for the explicit notice")
	}

	c = consent[indexOptOutSale]
	if c != ccpaNo && c != ccpaYes && c != ccpaNotApplicable {
		return false, errors.New("must specify 'N', 'Y', or '-' for the opt-out sale")
	}

	c = consent[indexLSPACoveredTransaction]
	if c != ccpaNo && c != ccpaYes && c != ccpaNotApplicable {
		return false, errors.New("must specify 'N', 'Y', or '-' for the limited service provider agreement")
	}

	return consent[indexOptOutSale] == ccpaYes, nil
}

func parseNoSaleBidders(noSaleBidders []string, validBidders map[string]struct{}) (noSaleForAllBidders bool, noSaleSpecificBidders map[string]struct{}, err error) {
	if len(noSaleBidders) == 1 && noSaleBidders[0] == allBiddersMarker {
		noSaleForAllBidders = true
		return
	}

	for _, bidder := range noSaleBidders {
		if bidder == allBiddersMarker {
			err = errors.New("err")
			return
		}

		if _, exists := validBidders[bidder]; exists {
			noSaleSpecificBidders[bidder] = struct{}{}
		} else {
			err = fmt.Errorf("unrecognized bidder '%s'", bidder)
			return
		}
	}

	return
}

func (p ParsedPolicy) isNoSaleForBidder(bidder string) bool {
	if p.noSaleForAllBidders {
		return true
	}

	_, exists := p.noSaleSpecificBidders[bidder]
	return exists
}

// ShouldEnforce returns true when the opt-out signal is explicitly detected.
func (p ParsedPolicy) ShouldEnforce(bidder string) bool {
	if !p.isValid {
		return false
	}

	if p.isNoSaleForBidder(bidder) {
		return false
	}

	return p.consentOptOut
}
