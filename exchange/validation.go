package exchange

import (
	"fmt"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type bidResponseWrapper struct {
	adapterBids  *pbsOrtbSeatBid
	adapterExtra *seatResponseExtra
	bidder       openrtb_ext.BidderName
}

// validateBids will run some validation checks on the returned bids and excise any invalid bids
func (brw *bidResponseWrapper) validateBids() (err []error) {
	// Exit early if there is nothing to do.
	if brw.adapterBids == nil || len(brw.adapterBids.bids) == 0 {
		return
	}
	err = make([]error, 0, len(brw.adapterBids.bids))
	validBids := make([]*pbsOrtbBid, 0, len(brw.adapterBids.bids))
	for _, bid := range brw.adapterBids.bids {
		if ok, berr := validateBid(bid); ok {
			validBids = append(validBids, bid)
		} else {
			err = append(err, berr)
		}
	}
	if len(validBids) != len(brw.adapterBids.bids) {
		// If all bids are valid, the two slices should be equal. Otherwise replace the list of bids with the valid bids.
		brw.adapterBids.bids = validBids
	}
	return err
}

// validateBid will run the supplied bid through validation checks and return true if it passes, false otherwise.
func validateBid(bid *pbsOrtbBid) (bool, error) {
	if bid.bid == nil {
		return false, fmt.Errorf("Empty bid object submitted.")
	}
	// These are the three required fields for bids
	if bid.bid.ID == "" || bid.bid.ImpID == "" || bid.bid.Price == 0.0 {
		return false, fmt.Errorf("Bid \"%s\" missing required field (id, impid, price)", bid.bid.ID)
	}
	// Check creative ID
	if bid.bid.CrID == "" {
		return false, fmt.Errorf("Bid \"%s\" missing creative ID", bid.bid.ID)
	}
	return true, nil
}
