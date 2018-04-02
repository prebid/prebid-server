package exchange

import (
	"testing"

	"github.com/mxmCherry/openrtb"
)

func TestAllValidBids(t *testing.T) {
	bids := make([]*pbsOrtbBid, 3)
	bids[0] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			ID:    "one-bid",
			ImpID: "thisImp",
			Price: 0.45,
			CrID:  "thisCreative",
		},
	}
	bids[1] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			ID:    "thatBid",
			ImpID: "thatImp",
			Price: 0.40,
			CrID:  "thatCreative",
		},
	}
	bids[2] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			ID:    "123",
			ImpID: "456",
			Price: 0.44,
			CrID:  "789",
		},
	}
	brw := &bidResponseWrapper{
		adapterBids: &pbsOrtbSeatBid{
			bids: bids,
		},
	}
	assertBids(t, brw, 3, 0)
}

func TestAllBadBids(t *testing.T) {
	bids := make([]*pbsOrtbBid, 4)
	bids[0] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			ID:    "one-bid",
			Price: 0.45,
			CrID:  "thisCreative",
		},
	}
	bids[1] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			ID:    "thatBid",
			ImpID: "thatImp",
			CrID:  "thatCreative",
		},
	}
	// TODO #427: Add this back in after a breaking change window
	// bids[2] = &pbsOrtbBid{
	// 	bid: &openrtb.Bid{
	// 		ID:    "123",
	// 		ImpID: "456",
	// 		Price: 0.44,
	// 	},
	// }
	bids[2] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			ImpID: "456",
			Price: 0.44,
			CrID:  "blah",
		},
	}
	bids[3] = &pbsOrtbBid{}
	brw := &bidResponseWrapper{
		adapterBids: &pbsOrtbSeatBid{
			bids: bids,
		},
	}
	assertBids(t, brw, 0, 4)
}

func TestMixeddBids(t *testing.T) {
	bids := make([]*pbsOrtbBid, 5)
	bids[0] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			ID:    "one-bid",
			ImpID: "thisImp",
			Price: 0.45,
			CrID:  "thisCreative",
		},
	}
	bids[1] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			ID:    "thatBid",
			ImpID: "thatImp",
			CrID:  "thatCreative",
		},
	}
	bids[2] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			ID:    "123",
			ImpID: "456",
			Price: 0.44,
			CrID:  "789",
		},
	}
	bids[3] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			ImpID: "456",
			Price: 0.44,
			CrID:  "blah",
		},
	}
	bids[4] = &pbsOrtbBid{}
	brw := &bidResponseWrapper{
		adapterBids: &pbsOrtbSeatBid{
			bids: bids,
		},
	}
	assertBids(t, brw, 2, 3)
}

func assertBids(t *testing.T, brw *bidResponseWrapper, ebids int, eerrs int) {
	errs := brw.validateBids()
	if len(errs) != eerrs {
		t.Errorf("Expected %d Errors validating bids, found %d", eerrs, len(errs))
	}
	if len(brw.adapterBids.bids) != ebids {
		t.Errorf("Expected %d bids, found %d bids", ebids, len(brw.adapterBids.bids))
	}

}
