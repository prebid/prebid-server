package cfg

import "github.com/prebid/prebid-server/openrtb_ext"

var Rubicon Cache = Cache{
	Url:          "https://www.magnite.com/skadnetworkids.json",
	Bidder:       openrtb_ext.BidderRubicon,
	BidderSKANID: "4468km3ulz.skadnetwork", // rubicon doesn't have a default skadnetwork id, so we're using the first id in their list (https://www.magnite.com/skadnetworkids.json) as a default
}
