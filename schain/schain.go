package schain

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// BidderToPrebidSChains organizes the ORTB 2.5 multiple root schain nodes into a map of schain nodes by bidder
func BidderToPrebidSChains(sChains []*openrtb_ext.ExtRequestPrebidSChain) (map[string]*openrtb2.SupplyChain, error) {
	bidderToSChains := make(map[string]*openrtb2.SupplyChain)

	for _, schainWrapper := range sChains {
		for _, bidder := range schainWrapper.Bidders {
			if _, present := bidderToSChains[bidder]; present {
				return nil, fmt.Errorf("request.ext.prebid.schains contains multiple schains for bidder %s; "+
					"it must contain no more than one per bidder.", bidder)
			} else {
				bidderToSChains[bidder] = &schainWrapper.SChain
			}
		}
	}

	return bidderToSChains, nil
}
