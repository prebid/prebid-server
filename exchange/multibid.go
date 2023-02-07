package exchange

import (
	"fmt"

	"github.com/prebid/prebid-server/openrtb_ext"
)

const DefaultBidLimit = 1
const MaxBidLimit = 9

type ExtMultiBidMap map[string]*openrtb_ext.ExtMultiBid

// Validate and add multiBid value
func (mb ExtMultiBidMap) Add(multiBid *openrtb_ext.ExtMultiBid) []error {
	errs := make([]error, 0)

	// If maxbids is not specified, ignore whole block and add warning when in debug mode
	if multiBid.MaxBids == nil {
		errs = append(errs, fmt.Errorf("maxBid not defined %v", multiBid))
		return errs
	}

	// Min and default is 1
	if *multiBid.MaxBids < DefaultBidLimit {
		errs = append(errs, fmt.Errorf("using default maxBid minimum %d limit %v", DefaultBidLimit, multiBid))
		*multiBid.MaxBids = DefaultBidLimit
	}

	// Max 9
	if *multiBid.MaxBids > MaxBidLimit {
		errs = append(errs, fmt.Errorf("using default maxBid maximum %d limit %v", MaxBidLimit, multiBid))
		*multiBid.MaxBids = MaxBidLimit
	}

	// Prefer Bidder over []Bidders
	if multiBid.Bidder != "" {
		if _, ok := mb[multiBid.Bidder]; ok {
			errs = append(errs, fmt.Errorf("multiBid already defined for %s, ignoring this instance %v", multiBid.Bidder, multiBid))
			return errs
		}

		if multiBid.Bidders != nil {
			errs = append(errs, fmt.Errorf("ignoring bidders from %v", multiBid))
			multiBid.Bidders = nil
		}
		mb[multiBid.Bidder] = multiBid
	} else if len(multiBid.Bidders) > 0 {
		for _, bidder := range multiBid.Bidders {
			if _, ok := mb[bidder]; ok {
				errs = append(errs, fmt.Errorf("multiBid already defined for %s, ignoring this instance %v", bidder, multiBid))
				continue
			}

			if multiBid.TargetBidderCodePrefix != "" {
				errs = append(errs, fmt.Errorf("ignoring targetbiddercodeprefix for %v", multiBid))
				multiBid.TargetBidderCodePrefix = ""
			}
			mb[bidder] = multiBid
		}
	} else {
		errs = append(errs, fmt.Errorf("bidder(s) not specified %v", multiBid))
	}
	return errs
}

// Get multi-bid limit for this bidder
func (mb *ExtMultiBidMap) GetMaxBids(bidder string) int {
	if maxBid, ok := (*mb)[bidder]; ok {
		return *maxBid.MaxBids
	}
	return DefaultBidLimit
}
