package openrtb_ext

import "fmt"

const DefaultBidLimit = 1
const MaxBidLimit = 9

func ValidateAndBuildExtMultiBidMap(prebid *ExtRequestPrebid) []error {
	if prebid.Multibid == nil {
		return nil
	}

	var errs []error
	for _, multiBid := range prebid.Multibid {
		errs = append(errs, addMultiBid(prebid.MultibidMap, multiBid)...)
	}

	return errs
}

// Validate and add multiBid
func addMultiBid(multiBidMap map[string]ExtMultiBid, multiBid *ExtMultiBid) []error {
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
		if _, ok := multiBidMap[multiBid.Bidder]; ok {
			errs = append(errs, fmt.Errorf("multiBid already defined for %s, ignoring this instance %v", multiBid.Bidder, multiBid))
			return errs
		}

		if multiBid.Bidders != nil {
			errs = append(errs, fmt.Errorf("ignoring bidders from %v", multiBid))
			multiBid.Bidders = nil
		}
		multiBidMap[multiBid.Bidder] = *multiBid
	} else if len(multiBid.Bidders) > 0 {
		for _, bidder := range multiBid.Bidders {
			if _, ok := multiBidMap[bidder]; ok {
				errs = append(errs, fmt.Errorf("multiBid already defined for %s, ignoring this instance %v", bidder, multiBid))
				continue
			}

			if multiBid.TargetBidderCodePrefix != "" {
				errs = append(errs, fmt.Errorf("ignoring targetbiddercodeprefix for %v", multiBid))
				multiBid.TargetBidderCodePrefix = ""
			}
			multiBidMap[bidder] = *multiBid
		}
	} else {
		errs = append(errs, fmt.Errorf("bidder(s) not specified %v", multiBid))
	}
	return errs
}
