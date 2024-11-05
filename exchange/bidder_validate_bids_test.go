package exchange

import (
	"context"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/experiment/adscert"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestAllValidBids(t *testing.T) {
	var bidder AdaptedBidder = addValidatedBidderMiddleware(&mockAdaptedBidder{
		bidResponse: []*entities.PbsOrtbSeatBid{{
			Bids: []*entities.PbsOrtbBid{
				{
					Bid: &openrtb2.Bid{
						ID:    "one-bid",
						ImpID: "thisImp",
						Price: 0.45,
						CrID:  "thisCreative",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "thatBid",
						ImpID: "thatImp",
						Price: 0.40,
						CrID:  "thatCreative",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "123",
						ImpID: "456",
						Price: 0.44,
						CrID:  "789",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:     "zeroPriceBid",
						ImpID:  "444",
						Price:  0.00,
						CrID:   "555",
						DealID: "777",
					},
				},
			},
		},
		}})
	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{},
		BidderName: openrtb_ext.BidderAppnexus,
	}
	bidAdjustments := map[string]float64{string(openrtb_ext.BidderAppnexus): 1.0}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed: true,
		headerDebugAllowed:  false,
		addCallSignHeader:   false,
		bidAdjustments:      bidAdjustments,
	}
	seatBids, _, errs := bidder.requestBid(context.Background(), bidderReq, currency.NewConstantRates(), &adapters.ExtraRequestInfo{}, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{}, nil)
	assert.Len(t, seatBids, 1)
	assert.Len(t, seatBids[0].Bids, 4)
	assert.Len(t, errs, 0)
}

func TestAllBadBids(t *testing.T) {
	bidder := addValidatedBidderMiddleware(&mockAdaptedBidder{
		bidResponse: []*entities.PbsOrtbSeatBid{{
			Bids: []*entities.PbsOrtbBid{
				{
					Bid: &openrtb2.Bid{
						ID:    "one-bid",
						Price: 0.45,
						CrID:  "thisCreative",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "thatBid",
						ImpID: "thatImp",
						CrID:  "thatCreative",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "123",
						ImpID: "456",
						Price: 0.44,
					},
				},
				{
					Bid: &openrtb2.Bid{
						ImpID: "456",
						Price: 0.44,
						CrID:  "blah",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:     "zeroPriceBidNoDeal",
						ImpID:  "444",
						Price:  0.00,
						CrID:   "555",
						DealID: "",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "negativePrice",
						ImpID: "999",
						Price: -0.10,
						CrID:  "888",
					},
				},
				{},
			},
		},
		}})
	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{},
		BidderName: openrtb_ext.BidderAppnexus,
	}
	bidAdjustments := map[string]float64{string(openrtb_ext.BidderAppnexus): 1.0}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed:  true,
		headerDebugAllowed:   false,
		addCallSignHeader:    false,
		bidAdjustments:       bidAdjustments,
		responseDebugAllowed: true,
	}
	seatBids, _, errs := bidder.requestBid(context.Background(), bidderReq, currency.NewConstantRates(), &adapters.ExtraRequestInfo{}, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{}, nil)
	assert.Len(t, seatBids, 1)
	assert.Len(t, seatBids[0].Bids, 0)
	assert.Len(t, errs, 7)
}

func TestMixedBids(t *testing.T) {
	bidder := addValidatedBidderMiddleware(&mockAdaptedBidder{
		bidResponse: []*entities.PbsOrtbSeatBid{{
			Bids: []*entities.PbsOrtbBid{
				{
					Bid: &openrtb2.Bid{
						ID:    "one-bid",
						ImpID: "thisImp",
						Price: 0.45,
						CrID:  "thisCreative",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "thatBid",
						ImpID: "thatImp",
						CrID:  "thatCreative",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "123",
						ImpID: "456",
						Price: 0.44,
						CrID:  "789",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ImpID: "456",
						Price: 0.44,
						CrID:  "blah",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:     "zeroPriceBid",
						ImpID:  "444",
						Price:  0.00,
						CrID:   "555",
						DealID: "777",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:     "zeroPriceBidNoDeal",
						ImpID:  "444",
						Price:  0.00,
						CrID:   "555",
						DealID: "",
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "negativePrice",
						ImpID: "999",
						Price: -0.10,
						CrID:  "888",
					},
				},
				{},
			},
		},
		}})
	bidderReq := BidderRequest{
		BidRequest: &openrtb2.BidRequest{},
		BidderName: openrtb_ext.BidderAppnexus,
	}
	bidAdjustments := map[string]float64{string(openrtb_ext.BidderAppnexus): 1.0}
	bidReqOptions := bidRequestOptions{
		accountDebugAllowed:  true,
		headerDebugAllowed:   false,
		addCallSignHeader:    false,
		bidAdjustments:       bidAdjustments,
		responseDebugAllowed: false,
	}
	seatBids, _, errs := bidder.requestBid(context.Background(), bidderReq, currency.NewConstantRates(), &adapters.ExtraRequestInfo{}, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{}, nil)
	assert.Len(t, seatBids, 1)
	assert.Len(t, seatBids[0].Bids, 3)
	assert.Len(t, errs, 2)
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
		bids := []*entities.PbsOrtbBid{
			{
				Bid: &openrtb2.Bid{
					ID:    "one-bid",
					ImpID: "thisImp",
					Price: 0.45,
					CrID:  "thisCreative",
				},
			},
			{
				Bid: &openrtb2.Bid{
					ID:    "thatBid",
					ImpID: "thatImp",
					Price: 0.44,
					CrID:  "thatCreative",
				},
			},
		}
		bidder := addValidatedBidderMiddleware(&mockAdaptedBidder{
			bidResponse: []*entities.PbsOrtbSeatBid{{
				Currency: tc.brpCur,
				Bids:     bids,
			},
			}})

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
		bidderRequest := BidderRequest{BidRequest: request, BidderName: openrtb_ext.BidderAppnexus}

		bidAdjustments := map[string]float64{string(openrtb_ext.BidderAppnexus): 1.0}
		bidReqOptions := bidRequestOptions{
			accountDebugAllowed: true,
			headerDebugAllowed:  false,
			addCallSignHeader:   false,
			bidAdjustments:      bidAdjustments,
		}
		seatBids, _, errs := bidder.requestBid(context.Background(), bidderRequest, currency.NewConstantRates(), &adapters.ExtraRequestInfo{}, &adscert.NilSigner{}, bidReqOptions, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{}, nil)
		assert.Len(t, seatBids, 1)
		assert.Len(t, seatBids[0].Bids, expectedValidBids)
		assert.Len(t, errs, expectedErrs)
	}
}

type mockAdaptedBidder struct {
	bidResponse   []*entities.PbsOrtbSeatBid
	extraRespInfo extraBidderRespInfo
	errorResponse []error
}

func (b *mockAdaptedBidder) requestBid(ctx context.Context, bidderRequest BidderRequest, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo, adsCertSigner adscert.Signer, bidRequestMetadata bidRequestOptions, alternateBidderCodes openrtb_ext.ExtAlternateBidderCodes, executor hookexecution.StageExecutor, ruleToAdjustments openrtb_ext.AdjustmentsByDealID) ([]*entities.PbsOrtbSeatBid, extraBidderRespInfo, []error) {
	return b.bidResponse, b.extraRespInfo, b.errorResponse
}
