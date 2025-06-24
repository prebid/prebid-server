package openrtb_ext

func mspBidderNames() []BidderName {
	return []BidderName{
		BidderMspGoogle,
		BidderMspNova,
		BidderMspNovaAlpha,
		BidderMspNovaBeta,
		BidderMspNovaGamma,
		BidderMspFbAlpha,
		BidderMspFbBeta,
		BidderMspFbGamma,
		BidderMspMoloco,
		BidderMspMolocoNative,
	}
}

func MspAllBidderNames() []BidderName {
	core := CoreBidderNames()
	return append(core, mspBidderNames()...)
}
