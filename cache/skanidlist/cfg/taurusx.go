package cfg

import "github.com/prebid/prebid-server/openrtb_ext"

var TaurusX Cache = Cache{
	Url:          "https://www.taurusx.com/skadnetworkids.json",
	Bidder:       openrtb_ext.BidderTaurusX,
	BidderSKANID: "22mmun2rn5.skadnetwork",
}
