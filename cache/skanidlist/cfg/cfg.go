package cfg

import "github.com/prebid/prebid-server/openrtb_ext"

type Cache struct {
	Url          string
	Bidder       openrtb_ext.BidderName
	BidderSKANID string
}
