package openrtb_ext

import "fmt"

const DefaultBidLimit = 1
const MaxBidLimit = 9

func ValidateAndBuildExtMultiBid(prebid *ExtRequestPrebid) ([]*ExtMultiBid, []error) {
	if prebid == nil || prebid.MultiBid == nil {
		return nil, nil
	}

	var validationErrs, errs []error
	var validatedMultiBids, newMultiBids []*ExtMultiBid //returning slice instead of map to keep the downstream req.Ext payload consistent
	multiBidMap := make(map[string]struct{})            // map is needed temporarily for validate of duplicate entries, etc.
	for _, multiBid := range prebid.MultiBid {
		newMultiBids, errs = addMultiBid(multiBidMap, multiBid)
		if len(errs) != 0 {
			validationErrs = append(validationErrs, errs...)
		}
		if len(newMultiBids) != 0 {
			validatedMultiBids = append(validatedMultiBids, newMultiBids...)
		}
	}

	return validatedMultiBids, validationErrs
}

// Validate and add multiBid
func addMultiBid(multiBidMap map[string]struct{}, multiBid *ExtMultiBid) ([]*ExtMultiBid, []error) {
	errs := make([]error, 0)

	if multiBid.MaxBids == nil {
		errs = append(errs, fmt.Errorf("maxBids not defined for %v", *multiBid))
		return nil, errs
	}

	if *multiBid.MaxBids < DefaultBidLimit {
		errs = append(errs, fmt.Errorf("invalid maxBids value, using minimum %d limit for %v", DefaultBidLimit, *multiBid))
		*multiBid.MaxBids = DefaultBidLimit
	}

	if *multiBid.MaxBids > MaxBidLimit {
		errs = append(errs, fmt.Errorf("invalid maxBids value, using maximum %d limit for %v", MaxBidLimit, *multiBid))
		*multiBid.MaxBids = MaxBidLimit
	}

	var validatedMultiBids []*ExtMultiBid
	if multiBid.Bidder != "" {
		if _, ok := multiBidMap[multiBid.Bidder]; ok {
			errs = append(errs, fmt.Errorf("multiBid already defined for %s, ignoring this instance %v", multiBid.Bidder, *multiBid))
			return nil, errs
		}

		if multiBid.Bidders != nil {
			errs = append(errs, fmt.Errorf("ignoring bidders from %v", *multiBid))
			multiBid.Bidders = nil
		}
		multiBidMap[multiBid.Bidder] = struct{}{}
		validatedMultiBids = append(validatedMultiBids, multiBid)
	} else if len(multiBid.Bidders) > 0 {
		var bidders []string
		for _, bidder := range multiBid.Bidders {
			if _, ok := multiBidMap[bidder]; ok {
				errs = append(errs, fmt.Errorf("multiBid already defined for %s, ignoring this instance %v", bidder, *multiBid))
				continue
			}
			multiBidMap[bidder] = struct{}{}
			bidders = append(bidders, bidder)
		}
		if multiBid.TargetBidderCodePrefix != "" {
			errs = append(errs, fmt.Errorf("ignoring targetbiddercodeprefix for %v", *multiBid))
			multiBid.TargetBidderCodePrefix = ""
		}
		if len(bidders) != 0 {
			validatedMultiBids = append(validatedMultiBids, &ExtMultiBid{
				MaxBids: multiBid.MaxBids,
				Bidders: bidders,
			})
		}
	} else {
		errs = append(errs, fmt.Errorf("bidder(s) not specified for %v", *multiBid))
	}
	return validatedMultiBids, errs
}
