package exchange

import (
	"context"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestAllValidBids(t *testing.T) {
	var bidder adaptedBidder = addValidatedBidderMiddleware(&mockAdaptedBidder{
		bidResponse: &pbsOrtbSeatBid{
			bids: []*pbsOrtbBid{
				{
					bid: &openrtb2.Bid{
						ID:    "one-bid",
						ImpID: "thisImp",
						Price: 0.45,
						CrID:  "thisCreative",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:    "thatBid",
						ImpID: "thatImp",
						Price: 0.40,
						CrID:  "thatCreative",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:    "123",
						ImpID: "456",
						Price: 0.44,
						CrID:  "789",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:     "zeroPriceBid",
						ImpID:  "444",
						Price:  0.00,
						CrID:   "555",
						DealID: "777",
					},
				},
			},
		},
	})
	seatBid, errs := bidder.requestBid(context.Background(), &openrtb2.BidRequest{}, openrtb_ext.BidderAppnexus, 1.0, currency.NewConstantRates(), &adapters.ExtraRequestInfo{}, true, false)
	assert.Len(t, seatBid.bids, 4)
	assert.Len(t, errs, 0)
}

func TestAllBadBids(t *testing.T) {
	bidder := addValidatedBidderMiddleware(&mockAdaptedBidder{
		bidResponse: &pbsOrtbSeatBid{
			bids: []*pbsOrtbBid{
				{
					bid: &openrtb2.Bid{
						ID:    "one-bid",
						Price: 0.45,
						CrID:  "thisCreative",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:    "thatBid",
						ImpID: "thatImp",
						CrID:  "thatCreative",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:    "123",
						ImpID: "456",
						Price: 0.44,
					},
				},
				{
					bid: &openrtb2.Bid{
						ImpID: "456",
						Price: 0.44,
						CrID:  "blah",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:     "zeroPriceBidNoDeal",
						ImpID:  "444",
						Price:  0.00,
						CrID:   "555",
						DealID: "",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:    "negativePrice",
						ImpID: "999",
						Price: -0.10,
						CrID:  "888",
					},
				},
				{},
			},
		},
	})
	seatBid, errs := bidder.requestBid(context.Background(), &openrtb2.BidRequest{}, openrtb_ext.BidderAppnexus, 1.0, currency.NewConstantRates(), &adapters.ExtraRequestInfo{}, true, false)
	assert.Len(t, seatBid.bids, 0)
	assert.Len(t, errs, 7)
}

func TestMixedBids(t *testing.T) {
	bidder := addValidatedBidderMiddleware(&mockAdaptedBidder{
		bidResponse: &pbsOrtbSeatBid{
			bids: []*pbsOrtbBid{
				{
					bid: &openrtb2.Bid{
						ID:    "one-bid",
						ImpID: "thisImp",
						Price: 0.45,
						CrID:  "thisCreative",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:    "thatBid",
						ImpID: "thatImp",
						CrID:  "thatCreative",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:    "123",
						ImpID: "456",
						Price: 0.44,
						CrID:  "789",
					},
				},
				{
					bid: &openrtb2.Bid{
						ImpID: "456",
						Price: 0.44,
						CrID:  "blah",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:     "zeroPriceBid",
						ImpID:  "444",
						Price:  0.00,
						CrID:   "555",
						DealID: "777",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:     "zeroPriceBidNoDeal",
						ImpID:  "444",
						Price:  0.00,
						CrID:   "555",
						DealID: "",
					},
				},
				{
					bid: &openrtb2.Bid{
						ID:    "negativePrice",
						ImpID: "999",
						Price: -0.10,
						CrID:  "888",
					},
				},
				{},
			},
		},
	})
	seatBid, errs := bidder.requestBid(context.Background(), &openrtb2.BidRequest{}, openrtb_ext.BidderAppnexus, 1.0, currency.NewConstantRates(), &adapters.ExtraRequestInfo{}, true, false)
	assert.Len(t, seatBid.bids, 3)
	assert.Len(t, errs, 5)
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
		bids := []*pbsOrtbBid{
			{
				bid: &openrtb2.Bid{
					ID:    "one-bid",
					ImpID: "thisImp",
					Price: 0.45,
					CrID:  "thisCreative",
				},
			},
			{
				bid: &openrtb2.Bid{
					ID:    "thatBid",
					ImpID: "thatImp",
					Price: 0.44,
					CrID:  "thatCreative",
				},
			},
		}
		bidder := addValidatedBidderMiddleware(&mockAdaptedBidder{
			bidResponse: &pbsOrtbSeatBid{
				currency: tc.brpCur,
				bids:     bids,
			},
		})

		expectedValidBids := len(bids)
		expectedErrs := 0

		if tc.expectedValidBid != true {
			// If currency mistmatch, we should have one error
			expectedErrs = 1
			expectedValidBids = 0
		}

		request := &openrtb2.BidRequest{
			Cur: tc.brqCur,
		}

		seatBid, errs := bidder.requestBid(context.Background(), request, openrtb_ext.BidderAppnexus, 1.0, currency.NewConstantRates(), &adapters.ExtraRequestInfo{}, true, false)
		assert.Len(t, seatBid.bids, expectedValidBids)
		assert.Len(t, errs, expectedErrs)
	}
}

type mockAdaptedBidder struct {
	bidResponse   *pbsOrtbSeatBid
	errorResponse []error
}

func (b *mockAdaptedBidder) requestBid(ctx context.Context, request *openrtb2.BidRequest, name openrtb_ext.BidderName, bidAdjustment float64, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo, accountDebugAllowed, headerDebugAllowed bool) (*pbsOrtbSeatBid, []error) {
	return b.bidResponse, b.errorResponse
}
