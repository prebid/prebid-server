package gdpr

type AuctionPermissions struct {
	AllowBidRequest bool
	PassGeo         bool
	PassID          bool
}

var AllowAll = AuctionPermissions{
	AllowBidRequest: true,
	PassGeo:         true,
	PassID:          true,
}

var DenyAll = AuctionPermissions{
	AllowBidRequest: false,
	PassGeo:         false,
	PassID:          false,
}

var AllowBidRequestOnly = AuctionPermissions{
	AllowBidRequest: true,
	PassGeo:         false,
	PassID:          false,
}
