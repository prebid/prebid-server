package ccpa

import (
	"errors"
	"fmt"

	"github.com/prebid/prebid-server/v3/errortypes"
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

// ValidateConsent returns true if the consent string is empty or valid per the IAB CCPA spec.
func ValidateConsent(consent string) bool {
	_, err := parseConsent(consent)
	return err == nil
}

// ParsedPolicy represents parsed and validated CCPA regulatory information. Use this struct
// to make enforcement decisions.
type ParsedPolicy struct {
	consentSpecified      bool
	consentOptOutSale     bool
	noSaleForAllBidders   bool
	noSaleSpecificBidders map[string]struct{}
}

// Parse returns a parsed and validated ParsedPolicy intended for use in enforcement decisions.
func (p Policy) Parse(validBidders map[string]struct{}) (ParsedPolicy, error) {
	consentOptOut, err := parseConsent(p.Consent)
	if err != nil {
		msg := fmt.Sprintf("request.regs.ext.us_privacy %s", err.Error())
		return ParsedPolicy{}, &errortypes.Warning{
			Message:     msg,
			WarningCode: errortypes.InvalidPrivacyConsentWarningCode,
		}
	}

	noSaleForAllBidders, noSaleSpecificBidders, err := parseNoSaleBidders(p.NoSaleBidders, validBidders)
	if err != nil {
		return ParsedPolicy{}, fmt.Errorf("request.ext.prebid.nosale is invalid: %s", err.Error())
	}

	return ParsedPolicy{
		consentSpecified:      p.Consent != "",
		consentOptOutSale:     consentOptOut,
		noSaleForAllBidders:   noSaleForAllBidders,
		noSaleSpecificBidders: noSaleSpecificBidders,
	}, nil
}

func parseConsent(consent string) (consentOptOutSale bool, err error) {
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
	noSaleSpecificBidders = make(map[string]struct{})

	if len(noSaleBidders) == 1 && noSaleBidders[0] == allBiddersMarker {
		noSaleForAllBidders = true
		return
	}

	for _, bidder := range noSaleBidders {
		if bidder == allBiddersMarker {
			err = errors.New("can only specify all bidders if no other bidders are provided")
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

// CanEnforce returns true when consent is specifically provided by the publisher, as opposed to an empty string.
func (p ParsedPolicy) CanEnforce() bool {
	return p.consentSpecified
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
	return !p.isNoSaleForBidder(bidder) && p.consentOptOutSale
}
