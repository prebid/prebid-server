package exchange

import (
	"testing"

	"github.com/mxmCherry/openrtb"
)

func TestAllValidBids(t *testing.T) {
	brq := &openrtb.BidRequest{}

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
	assertBids(t, brq, brw, 3, 0)
}

func TestAllBadBids(t *testing.T) {
	brq := &openrtb.BidRequest{}
	bids := make([]*pbsOrtbBid, 5)

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
	bids[2] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			ID:    "123",
			ImpID: "456",
			Price: 0.44,
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
	assertBids(t, brq, brw, 0, 5)
}

func TestMixeddBids(t *testing.T) {
	brq := &openrtb.BidRequest{}

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
	assertBids(t, brq, brw, 2, 3)
}

func TestCurrencyBids(t *testing.T) {
	currencyTestCases := []struct {
		brqCur           []string
		brpCur           string
		defaultCur       string
		expectedValidBid bool
	}{
		// Case bid request and bid response don't specify any currencies.
		// Expected to be valid since both bid request / response will be overridden with default currency (USD).
		{
			brqCur:           []string{},
			brpCur:           "",
			expectedValidBid: true,
		},
		// Case bid request specifies a currency (default one) but bid response doesn't.
		// Expected to be valid since bid response will be overridden with default currency (USD).
		{
			brqCur:           []string{"USD"},
			brpCur:           "",
			expectedValidBid: true,
		},
		// Case bid request specifies more than 1 currency (default one and another one) but bid response doesn't.
		// Expected to be valid since bid response will be overridden with default currency (USD).
		{
			brqCur:           []string{"USD", "EUR"},
			brpCur:           "",
			expectedValidBid: true,
		},
		// Case bid request specifies more than 1 currency (default one and another one) and bid response specifies default currency (USD).
		// Expected to be valid.
		{
			brqCur:           []string{"USD", "EUR"},
			brpCur:           "USD",
			expectedValidBid: true,
		},
		// Case bid request specifies more than 1 currency (default one and another one) and bid response specifies the second currency allowed (not USD).
		// Expected to be valid.
		{
			brqCur:           []string{"USD", "EUR"},
			brpCur:           "EUR",
			expectedValidBid: true,
		},
		// Case bid request specifies only 1 currency which is not the default one.
		// Bid response doesn't specify any currency.
		// Expected to be invalid.
		{
			brqCur:           []string{"JPY"},
			brpCur:           "",
			expectedValidBid: false,
		},
		// Case bid request doesn't specify any currencies.
		// Bid response specifies a currency which is not the default one.
		// Expected to be invalid.
		{
			brqCur:           []string{},
			brpCur:           "JPY",
			expectedValidBid: false,
		},
		// Case bid request specifies a currency.
		// Bid response specifies a currency which is not the one specified in bid request.
		// Expected to be invalid.
		{
			brqCur:           []string{"USD"},
			brpCur:           "EUR",
			expectedValidBid: false,
		},
		// Case bid request specifies several currencies.
		// Bid response specifies a currency which is not the one specified in bid request.
		// Expected to be invalid.
		{
			brqCur:           []string{"USD", "EUR"},
			brpCur:           "JPY",
			expectedValidBid: false,
		},
	}

	for _, tc := range currencyTestCases {

		brq := &openrtb.BidRequest{
			Cur: tc.brqCur,
		}

		bids := make([]*pbsOrtbBid, 2)
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
				Price: 0.44,
				CrID:  "thatCreative",
			},
		}

		brw := &bidResponseWrapper{
			adapterBids: &pbsOrtbSeatBid{
				bids:     bids,
				currency: tc.brpCur,
			},
		}

		expectedValidBids := len(bids)
		expectedErrs := 0

		if tc.expectedValidBid != true {
			// If currency mistmatch, we should have one error
			expectedErrs = 1
			expectedValidBids = 0
		}

		assertBids(t, brq, brw, expectedValidBids, expectedErrs)
	}
}

func assertBids(t *testing.T, brq *openrtb.BidRequest, brw *bidResponseWrapper, ebids int, eerrs int) {
	errs := brw.validateBids(brq)
	if len(errs) != eerrs {
		t.Errorf("Expected %d Errors validating bids, found %d", eerrs, len(errs))
	}
	if len(brw.adapterBids.bids) != ebids {
		t.Errorf("Expected %d bids, found %d bids", ebids, len(brw.adapterBids.bids))
	}
}
