package openrtb_ext

func mspBidderNames() []BidderName {
	return []BidderName{
		BidderMspGoogle,
	}
}

func MspAllBidderNames() []BidderName {
	core := CoreBidderNames()
	return append(core, mspBidderNames()...)
}
