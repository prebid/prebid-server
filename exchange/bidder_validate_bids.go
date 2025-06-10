package exchange

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/experiment/adscert"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	goCurrency "golang.org/x/text/currency"
)

// addValidatedBidderMiddleware returns a bidder that removes invalid bids from the argument bidder's response.
// These will be converted into errors instead.
//
// The goal here is to make sure that the response contains Bids which are valid given the initial Request,
// so that Publishers can trust the Bids they get from Prebid Server.
func addValidatedBidderMiddleware(bidder AdaptedBidder) AdaptedBidder {
	return &validatedBidder{
		bidder: bidder,
	}
}

type validatedBidder struct {
	bidder AdaptedBidder
}

func (v *validatedBidder) requestBid(ctx context.Context, bidderRequest BidderRequest, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo, adsCertSigner adscert.Signer, bidRequestOptions bidRequestOptions, alternateBidderCodes openrtb_ext.ExtAlternateBidderCodes, hookExecutor hookexecution.StageExecutor, ruleToAdjustments openrtb_ext.AdjustmentsByDealID) ([]*entities.PbsOrtbSeatBid, extraBidderRespInfo, []error) {
	seatBids, extraBidderRespInfo, errs := v.bidder.requestBid(ctx, bidderRequest, conversions, reqInfo, adsCertSigner, bidRequestOptions, alternateBidderCodes, hookExecutor, ruleToAdjustments)
	for _, seatBid := range seatBids {
		if validationErrors := removeInvalidBids(bidderRequest.BidRequest, seatBid, bidRequestOptions.responseDebugAllowed); len(validationErrors) > 0 {
			errs = append(errs, validationErrors...)
		}
	}
	return seatBids, extraBidderRespInfo, errs
}

// validateBids will run some validation checks on the returned bids and excise any invalid bids
func removeInvalidBids(request *openrtb2.BidRequest, seatBid *entities.PbsOrtbSeatBid, debug bool) []error {
	// Exit early if there is nothing to do.
	if seatBid == nil || len(seatBid.Bids) == 0 {
		return nil
	}

	// By design, default currency is USD.
	if cerr := validateCurrency(request.Cur, seatBid.Currency); cerr != nil {
		seatBid.Bids = nil
		return []error{cerr}
	}

	errs := make([]error, 0, len(seatBid.Bids))
	validBids := make([]*entities.PbsOrtbBid, 0, len(seatBid.Bids))
	for _, bid := range seatBid.Bids {
		if ok, err := validateBid(bid, debug); ok {
			validBids = append(validBids, bid)
		} else if err != nil {
			errs = append(errs, err)
		}
	}
	seatBid.Bids = validBids
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
func validateBid(bid *entities.PbsOrtbBid, debug bool) (bool, error) {
	if bid.Bid == nil {
		return false, errors.New("Empty bid object submitted.")
	}

	if bid.Bid.ID == "" {
		return false, errors.New("Bid missing required field 'id'")
	}
	if bid.Bid.ImpID == "" {
		return false, fmt.Errorf("Bid \"%s\" missing required field 'impid'", bid.Bid.ID)
	}
	if bid.Bid.Price < 0.0 {
		if debug {
			return false, fmt.Errorf("Bid \"%s\" does not contain a positive (or zero if there is a deal) 'price'", bid.Bid.ID)
		}
		return false, nil
	}
	if bid.Bid.Price == 0.0 && bid.Bid.DealID == "" {
		if debug {
			return false, fmt.Errorf("Bid \"%s\" does not contain positive 'price' which is required since there is no deal set for this bid", bid.Bid.ID)
		}
		return false, nil
	}
	if bid.Bid.CrID == "" {
		return false, fmt.Errorf("Bid \"%s\" missing creative ID", bid.Bid.ID)
	}

	return true, nil
}
