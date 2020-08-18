package ccpa

import (
	"errors"
	"fmt"
	"strings"

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

type ValidatedPolicy struct {
	Policy
	OptOutSaleYes         bool
	NoSaleAllBidders      bool
	NoSaleSpecificBidders map[string]struct{}
}

type ValidationErrors struct {
	Consent       error
	NoSaleBidders error
}

func (p Policy) Validate() (ValidatedPolicy, error) {
	if err := ValidateConsent(p.Value); err != nil {
		return fmt.Errorf("request.regs.ext.us_privacy %s", err.Error())
	}

	if err := ValidateNoSaleBidders(p.NoSaleBidders); err != nil {
		return fmt.Errorf("request.ext.prebid.nosale %s", err.Error())
	}

	return nil
}

// ValidateConsent returns an error if the CCPA consent string does not adhere to the IAB spec.
func ValidateConsent(consent string) (ValidatedPolicy, error) {
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
func (p ValidatedPolicy) ShouldEnforce(bidder string) bool {
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
