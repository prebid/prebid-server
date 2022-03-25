package exchange

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/openrtb_ext"
	goCurrency "golang.org/x/text/currency"
)

// addValidatedBidderMiddleware returns a bidder that removes invalid bids from the argument bidder's response.
// These will be converted into errors instead.
//
// The goal here is to make sure that the response contains Bids which are valid given the initial Request,
// so that Publishers can trust the Bids they get from Prebid Server.
func addValidatedBidderMiddleware(bidder adaptedBidder) adaptedBidder {
	return &validatedBidder{
		bidder: bidder,
	}
}

type validatedBidder struct {
	bidder adaptedBidder
}

func (v *validatedBidder) requestBid(ctx context.Context, request *openrtb2.BidRequest, name openrtb_ext.BidderName, bidAdjustment float64, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo, accountDebugAllowed, headerDebugAllowed bool) (*pbsOrtbSeatBid, []error) {
	seatBid, errs := v.bidder.requestBid(ctx, request, name, bidAdjustment, conversions, reqInfo, accountDebugAllowed, headerDebugAllowed)
	if validationErrors := removeInvalidBids(request, seatBid); len(validationErrors) > 0 {
		errs = append(errs, validationErrors...)
	}
	return seatBid, errs
}

// validateBids will run some validation checks on the returned bids and excise any invalid bids
func removeInvalidBids(request *openrtb2.BidRequest, seatBid *pbsOrtbSeatBid) []error {
	// Exit early if there is nothing to do.
	if seatBid == nil || len(seatBid.bids) == 0 {
		return nil
	}

	// By design, default currency is USD.
	if cerr := validateCurrency(request.Cur, seatBid.currency); cerr != nil {
		seatBid.bids = nil
		return []error{cerr}
	}

	errs := make([]error, 0, len(seatBid.bids))
	validBids := make([]*pbsOrtbBid, 0, len(seatBid.bids))
	for _, bid := range seatBid.bids {
		if ok, berr := validateBid(bid); ok {
			validBids = append(validBids, bid)
		} else {
			errs = append(errs, berr)
		}
	}
	seatBid.bids = validBids
	return errs
}

// validateCurrency will run currency validation checks and return true if it passes, false otherwise.
func validateCurrency(requestAllowedCurrencies []string, bidCurrency string) error {
	// Default currency is `USD` by design.
	defaultCurrency := "USD"
	// Make sure bid currency is a valid ISO currency code
	if bidCurrency == "" {
		// If bid currency is not set, then consider it's default currency.
		bidCurrency = defaultCurrency
	}
	currencyUnit, cerr := goCurrency.ParseISO(bidCurrency)
	if cerr != nil {
		return cerr
	}
	// Make sure the bid currency is allowed from bid request via `cur` field.
	// If `cur` field array from bid request is empty, then consider it accepts the default currency.
	currencyAllowed := false
	if len(requestAllowedCurrencies) == 0 {
		requestAllowedCurrencies = []string{defaultCurrency}
	}
	for _, allowedCurrency := range requestAllowedCurrencies {
		if strings.ToUpper(allowedCurrency) == currencyUnit.String() {
			currencyAllowed = true
			break
		}
	}
	if !currencyAllowed {
		return fmt.Errorf(
			"Bid currency is not allowed. Was '%s', wants: ['%s']",
			currencyUnit.String(),
			strings.Join(requestAllowedCurrencies, "', '"),
		)
	}

	return nil
}

// validateBid will run the supplied bid through validation checks and return true if it passes, false otherwise.
func validateBid(bid *pbsOrtbBid) (bool, error) {
	if bid.bid == nil {
		return false, errors.New("Empty bid object submitted.")
	}

	if bid.bid.ID == "" {
		return false, errors.New("Bid missing required field 'id'")
	}
	if bid.bid.ImpID == "" {
		return false, fmt.Errorf("Bid \"%s\" missing required field 'impid'", bid.bid.ID)
	}
	if bid.bid.Price < 0.0 {
		return false, fmt.Errorf("Bid \"%s\" does not contain a positive (or zero if there is a deal) 'price'", bid.bid.ID)
	}
	if bid.bid.Price == 0.0 && bid.bid.DealID == "" {
		return false, fmt.Errorf("Bid \"%s\" does not contain positive 'price' which is required since there is no deal set for this bid", bid.bid.ID)
	}
	if bid.bid.CrID == "" {
		return false, fmt.Errorf("Bid \"%s\" missing creative ID", bid.bid.ID)
	}

	return true, nil
}
