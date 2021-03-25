package cfg

import "github.com/prebid/prebid-server/openrtb_ext"

var Pubmatic Cache = Cache{
	Url:          "https://pubmatic.com/skadnetworkids.json",
	Bidder:       openrtb_ext.BidderPubmatic,
	BidderSKANID: "k674qkevps.skadnetwork",
}
