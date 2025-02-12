package floors

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

type convert struct {
}

func (c convert) GetRate(from string, to string) (float64, error) {
	if from == to {
		return 1, nil
	}

	if from == "USD" && to == "INR" {
		return 77.59, nil
	} else if from == "INR" && to == "USD" {
		return 0.013, nil
	}
	return 0, errors.New("currency conversion not supported")
}

func (c convert) GetRates() *map[string]map[string]float64 {
	return &map[string]map[string]float64{}
}

func TestIsValidImpBidfloorPresentInRequest(t *testing.T) {

	tests := []struct {
		name string
		imp  []openrtb2.Imp
		want bool
	}{
		{
			imp:  []openrtb2.Imp{{ID: "1234"}},
			want: false,
		},
		{
			imp:  []openrtb2.Imp{{ID: "1234", BidFloor: 10, BidFloorCur: "USD"}},
			want: true,
		},
	}

	for _, tt := range tests {
		got := isValidImpBidFloorPresent(tt.imp)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestEnforceFloorToBids(t *testing.T) {
	type args struct {
		bidRequestWrapper *openrtb_ext.RequestWrapper
		seatBids          map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		conversions       currency.Conversions
		enforceDealFloors bool
	}
	tests := []struct {
		name            string
		args            args
		expEligibleBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		expErrs         []error
		expRejectedBids []*entities.PbsOrtbSeatBid
	}{
		{
			name: "Floors enforcement disabled using enforcepbs = false",
			args: args{
				bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
					bw := openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{
								{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
								{ID: "some-impression-id-2", BidFloor: 2.01, BidFloorCur: "USD"},
							},
							Ext: json.RawMessage(`{"prebid":{"floors":{"enforcement":{"enforcepbs":false}}}}`),
						},
					}
					bw.RebuildRequest()
					return &bw
				}(),
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
							{Bid: &openrtb2.Bid{ID: "some-bid-2", Price: 1.5, DealID: "deal_Id", ImpID: "some-impression-id-2"}},
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.5, ImpID: "some-impression-id-1"}},
							{Bid: &openrtb2.Bid{ID: "some-bid-12", Price: 2.2, ImpID: "some-impression-id-2"}},
						},
						Seat:     "appnexus",
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: false,
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
						{Bid: &openrtb2.Bid{ID: "some-bid-2", Price: 1.5, DealID: "deal_Id", ImpID: "some-impression-id-2"}},
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.5, ImpID: "some-impression-id-1"}},
						{Bid: &openrtb2.Bid{ID: "some-bid-12", Price: 2.2, ImpID: "some-impression-id-2"}},
					},
					Seat:     "appnexus",
					Currency: "USD",
				},
			},
			expRejectedBids: []*entities.PbsOrtbSeatBid{},
			expErrs:         []error{},
		},
		{
			name: "Bids with price less than bidfloor",
			args: args{
				bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
					bw := openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{
								{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
								{ID: "some-impression-id-2", BidFloor: 2.01, BidFloorCur: "USD"},
							},
						},
					}
					bw.RebuildRequest()
					return &bw
				}(),
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
							{Bid: &openrtb2.Bid{ID: "some-bid-2", Price: 1.5, DealID: "deal_Id", ImpID: "some-impression-id-2"}},
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.5, ImpID: "some-impression-id-1"}},
							{Bid: &openrtb2.Bid{ID: "some-bid-12", Price: 2.2, ImpID: "some-impression-id-2"}},
						},
						Seat:     "appnexus",
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: false,
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
						{Bid: &openrtb2.Bid{ID: "some-bid-2", Price: 1.5, DealID: "deal_Id", ImpID: "some-impression-id-2"}},
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-12", Price: 2.2, ImpID: "some-impression-id-2"}},
					},
					Seat:     "appnexus",
					Currency: "USD",
				},
			},
			expRejectedBids: []*entities.PbsOrtbSeatBid{
				{
					Seat:     "appnexus",
					Currency: "USD",
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.5, ImpID: "some-impression-id-1"}},
					},
				},
			},
			expErrs: []error{},
		},
		{
			name: "Bids with price less than bidfloor with floorsPrecision",
			args: args{
				bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
					bw := openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{
								{ID: "some-impression-id-1", BidFloor: 1, BidFloorCur: "USD"},
								{ID: "some-impression-id-2", BidFloor: 2, BidFloorCur: "USD"},
							},
						},
					}
					bw.RebuildRequest()
					return &bw
				}(),
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 0.998, ImpID: "some-impression-id-1"}},
							{Bid: &openrtb2.Bid{ID: "some-bid-2", Price: 1.5, DealID: "deal_Id", ImpID: "some-impression-id-2"}},
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.8, ImpID: "some-impression-id-1"}},
							{Bid: &openrtb2.Bid{ID: "some-bid-12", Price: 2.2, ImpID: "some-impression-id-2"}},
						},
						Seat:     "appnexus",
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: false,
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 0.998, ImpID: "some-impression-id-1"}},
						{Bid: &openrtb2.Bid{ID: "some-bid-2", Price: 1.5, DealID: "deal_Id", ImpID: "some-impression-id-2"}},
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-12", Price: 2.2, ImpID: "some-impression-id-2"}},
					},
					Seat:     "appnexus",
					Currency: "USD",
				},
			},
			expRejectedBids: []*entities.PbsOrtbSeatBid{
				{
					Seat:     "appnexus",
					Currency: "USD",
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.8, ImpID: "some-impression-id-1"}},
					},
				},
			},
			expErrs: []error{},
		},
		{
			name: "Bids with different currency with enforceDealFloor true",
			args: args{
				bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
					bw := openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{
								{ID: "some-impression-id-1", BidFloor: 60, BidFloorCur: "INR"},
								{ID: "some-impression-id-2", BidFloor: 100, BidFloorCur: "INR"},
							},
						},
					}
					bw.RebuildRequest()
					return &bw
				}(),
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
							{Bid: &openrtb2.Bid{ID: "some-bid-2", Price: 1.5, DealID: "deal_Id", ImpID: "some-impression-id-2"}},
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.5, ImpID: "some-impression-id-1"}},
							{Bid: &openrtb2.Bid{ID: "some-bid-12", Price: 2.2, ImpID: "some-impression-id-2"}},
						},
						Seat:     "appnexus",
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: true,
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
						{Bid: &openrtb2.Bid{ID: "some-bid-2", Price: 1.5, DealID: "deal_Id", ImpID: "some-impression-id-2"}},
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-12", Price: 2.2, ImpID: "some-impression-id-2"}},
					},
					Seat:     "appnexus",
					Currency: "USD",
				},
			},
			expRejectedBids: []*entities.PbsOrtbSeatBid{
				{
					Seat:     "appnexus",
					Currency: "USD",
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.5, ImpID: "some-impression-id-1"}},
					},
				},
			},
			expErrs: []error{},
		},
		{
			name: "Error in currency conversion",
			args: args{
				bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
					bw := openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID:  "some-request-id",
							Cur: []string{"USD"},
							Imp: []openrtb2.Imp{{ID: "some-impression-id-1", BidFloor: 1.01}},
						},
					}
					bw.RebuildRequest()
					return &bw
				}(),
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
						},
						Currency: "EUR",
					},
				},
				conversions:       convert{},
				enforceDealFloors: true,
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids:     []*entities.PbsOrtbBid{},
					Currency: "EUR",
				},
			},
			expRejectedBids: []*entities.PbsOrtbSeatBid{},
			expErrs:         []error{errors.New("error in rate conversion from = EUR to  with bidder pubmatic for impression id some-impression-id-1 and bid id some-bid-1 error = currency conversion not supported")},
		},
		{
			name: "Bids with invalid impression ID",
			args: args{
				bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
					bw := openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{
								{ID: "some-impression-id-2", BidFloor: 2.01, BidFloorCur: "USD"},
							},
						},
					}
					bw.RebuildRequest()
					return &bw
				}(),
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-123"}},
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: false,
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids:     []*entities.PbsOrtbBid{},
					Seat:     "pubmatic",
					Currency: "USD",
				},
			},
			expRejectedBids: []*entities.PbsOrtbSeatBid{},
			expErrs:         []error{},
		},
	}
	for _, tt := range tests {
		seatbids, errs, rejBids := enforceFloorToBids(tt.args.bidRequestWrapper, tt.args.seatBids, tt.args.conversions, tt.args.enforceDealFloors)
		assert.Equal(t, tt.expEligibleBids, seatbids, tt.name)
		assert.Equal(t, tt.expErrs, errs, tt.name)
		assert.Equal(t, tt.expRejectedBids, rejBids, tt.name)
	}
}

func TestEnforce(t *testing.T) {
	type args struct {
		bidRequestWrapper *openrtb_ext.RequestWrapper
		bidRequest        *openrtb2.BidRequest
		seatBids          map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		priceFloorsCfg    config.AccountPriceFloors
		conversions       currency.Conversions
	}
	tests := []struct {
		name            string
		args            args
		expEligibleBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		expErrs         []error
		expRejectedBids []*entities.PbsOrtbSeatBid
	}{
		{
			name: "Error in getting request extension",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{{
							ID:          "some-impression-id-1",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
							BidFloor:    5.01,
							BidFloorCur: "USD",
						}},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":false},"enabled":false,"skipped":false}}`),
					},
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 4.5, ImpID: "some-impression-id-1"}},
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
				},
				conversions:    convert{},
				priceFloorsCfg: config.AccountPriceFloors{Enabled: false, EnforceFloorsRate: 100, EnforceDealFloors: true},
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 4.5, ImpID: "some-impression-id-1"}},
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
			},
			expErrs:         []error{errors.New("Error in getting request extension")},
			expRejectedBids: []*entities.PbsOrtbSeatBid{},
		},
		{
			name: "Should not enforce floors when req.ext.prebid.floors.enabled = false",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{{
							ID:          "some-impression-id-1",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
							BidFloor:    5.01,
							BidFloorCur: "USD",
						}},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":false},"enabled":false}}}`),
					},
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 4.5, ImpID: "some-impression-id-1"}},
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
				},
				conversions:    convert{},
				priceFloorsCfg: config.AccountPriceFloors{Enabled: true, EnforceFloorsRate: 100, EnforceDealFloors: true},
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 4.5, ImpID: "some-impression-id-1"}},
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
			},
			expRejectedBids: []*entities.PbsOrtbSeatBid{},
		},
		{
			name: "Should not enforce floors is disabled in account config",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{{
							ID:          "some-impression-id-1",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							BidFloor:    5.01,
							BidFloorCur: "USD",
						}},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":false},"enabled":true}}}`),
					},
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
				},
				conversions:    convert{},
				priceFloorsCfg: config.AccountPriceFloors{Enabled: false, EnforceFloorsRate: 100, EnforceDealFloors: true},
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
			},
			expRejectedBids: []*entities.PbsOrtbSeatBid{},
		},
		{
			name: "Should not enforce floors when req.ext.prebid.floors.enforcement.enforcepbs = false",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{{
							ID:          "some-impression-id-1",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							BidFloor:    5.01,
							BidFloorCur: "USD",
						}},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":false,"floordeals":false},"enabled":true,"skipped":false}}}`),
					},
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
				},
				conversions:    convert{},
				priceFloorsCfg: config.AccountPriceFloors{Enabled: true, EnforceFloorsRate: 100, EnforceDealFloors: true},
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}, BidFloors: &openrtb_ext.ExtBidPrebidFloors{FloorValue: 5.01, FloorCurrency: "USD"}},
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
			},
			expErrs:         []error{},
			expRejectedBids: []*entities.PbsOrtbSeatBid{},
		},
		{
			name: "Should not enforce floors when req.ext.prebid.floors.skipped = true",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{{
							ID:          "some-impression-id-1",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							BidFloor:    5.01,
							BidFloorCur: "USD",
						}},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":false},"enabled":true,"skipped":true}}}`),
					},
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
				},
				conversions:    convert{},
				priceFloorsCfg: config.AccountPriceFloors{Enabled: true, EnforceFloorsRate: 100, EnforceDealFloors: true},
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}},
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
			},
			expRejectedBids: []*entities.PbsOrtbSeatBid{},
		},
		{
			name: "Should enforce floors for deals, ext.prebid.floors.enforcement.floorDeals = true and floors enabled = true",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{{
							ID: "some-impression-id-1", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}, BidFloor: 20.01, BidFloorCur: "USD",
						}},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true}}}`),
					},
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids:     []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, DealID: "deal_Id_1", ImpID: "some-impression-id-1"}}},
						Seat:     "pubmatic",
						Currency: "USD",
					},
					"appnexus": {
						Bids:     []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.5, DealID: "deal_Id_3", ImpID: "some-impression-id-1"}}},
						Seat:     "appnexus",
						Currency: "USD",
					},
				},
				conversions:    convert{},
				priceFloorsCfg: config.AccountPriceFloors{Enabled: true, EnforceFloorsRate: 0, EnforceDealFloors: true},
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids:     []*entities.PbsOrtbBid{},
					Seat:     "pubmatic",
					Currency: "USD",
				},
				"appnexus": {
					Bids:     []*entities.PbsOrtbBid{},
					Seat:     "appnexus",
					Currency: "USD",
				},
			},
			expRejectedBids: []*entities.PbsOrtbSeatBid{
				{
					Seat:     "pubmatic",
					Currency: "USD",
					Bids:     []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, DealID: "deal_Id_1", ImpID: "some-impression-id-1"}, BidFloors: &openrtb_ext.ExtBidPrebidFloors{FloorCurrency: "USD", FloorValue: 20.01}}},
				},
				{
					Seat:     "appnexus",
					Currency: "USD",
					Bids:     []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.5, DealID: "deal_Id_3", ImpID: "some-impression-id-1"}, BidFloors: &openrtb_ext.ExtBidPrebidFloors{FloorCurrency: "USD", FloorValue: 20.01}}},
				},
			},
			expErrs: []error{},
		},
		{
			name: "Should enforce floors when imp.bidfloor provided and enforcepbs not provided",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{{
							ID: "some-impression-id-1", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}, BidFloor: 5.01, BidFloorCur: "USD",
						}},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1}}}`),
					},
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, DealID: "deal_Id_1", ImpID: "some-impression-id-1"}},
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 4.5, ImpID: "some-impression-id-1"}},
						},
						Seat:     "appnexus",
						Currency: "USD",
					},
				},
				conversions:    convert{},
				priceFloorsCfg: config.AccountPriceFloors{Enabled: true, EnforceFloorsRate: 0, EnforceDealFloors: false},
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, DealID: "deal_Id_1", ImpID: "some-impression-id-1"}, BidFloors: &openrtb_ext.ExtBidPrebidFloors{FloorValue: 5.01, FloorCurrency: "USD"}},
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
				"appnexus": {
					Bids:     []*entities.PbsOrtbBid{},
					Seat:     "appnexus",
					Currency: "USD",
				},
			},
			expRejectedBids: []*entities.PbsOrtbSeatBid{
				{
					Seat:     "appnexus",
					Currency: "USD",
					Bids:     []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 4.5, ImpID: "some-impression-id-1"}, BidFloors: &openrtb_ext.ExtBidPrebidFloors{FloorValue: 5.01, FloorCurrency: "USD"}}},
				},
			},
			expErrs: []error{},
		},
	}
	for _, tt := range tests {
		actEligibleBids, actErrs, actRejecteBids := Enforce(tt.args.bidRequestWrapper, tt.args.seatBids, config.Account{PriceFloors: tt.args.priceFloorsCfg}, tt.args.conversions)
		assert.Equal(t, tt.expErrs, actErrs, tt.name)
		assert.ElementsMatch(t, tt.expRejectedBids, actRejecteBids, tt.name)

		if !reflect.DeepEqual(tt.expEligibleBids, actEligibleBids) {
			assert.Failf(t, "eligible bids don't match", "Expected: %v, Got: %v", tt.expEligibleBids, actEligibleBids)
		}
	}
}

func TestUpdateBidExtWithFloors(t *testing.T) {
	type args struct {
		reqImp        *openrtb_ext.ImpWrapper
		bid           *entities.PbsOrtbBid
		floorCurrency string
	}
	tests := []struct {
		name        string
		expBidFloor *openrtb_ext.ExtBidPrebidFloors
		args        args
	}{
		{
			name: "Empty prebid extension in imp.ext",
			args: args{
				reqImp: &openrtb_ext.ImpWrapper{
					Imp: &openrtb2.Imp{BidFloor: 10, BidFloorCur: "USD"},
				},
				bid: &entities.PbsOrtbBid{
					Bid: &openrtb2.Bid{
						Price: 10.10,
						AdM:   "Adm",
					},
				},
				floorCurrency: "USD",
			},
			expBidFloor: &openrtb_ext.ExtBidPrebidFloors{
				FloorValue:    10,
				FloorCurrency: "USD",
			},
		},
		{
			name: "Valid prebid extension in imp.ext",
			args: args{
				reqImp: &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "1234", Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](300), H: ptrutil.ToPtr[int64](250)}, Ext: []byte(`{"prebid":{"floors":{"floorrule":"test|123|xyz","floorrulevalue":5.5,"floorvalue":5.5}}}`)}},
				bid: &entities.PbsOrtbBid{
					Bid: &openrtb2.Bid{
						Price: 10.10,
						AdM:   "Adm",
					},
				},
				floorCurrency: "USD",
			},
			expBidFloor: &openrtb_ext.ExtBidPrebidFloors{
				FloorRule:      "test|123|xyz",
				FloorRuleValue: 5.5,
				FloorValue:     5.5,
				FloorCurrency:  "USD",
			},
		},
	}
	for _, tt := range tests {
		updateBidExtWithFloors(tt.args.reqImp, tt.args.bid, tt.args.floorCurrency)
		assert.Equal(t, tt.expBidFloor, tt.args.bid.BidFloors, tt.name)
	}
}

func TestIsEnforcementEnabledForRequest(t *testing.T) {
	tests := []struct {
		name   string
		reqExt *openrtb_ext.RequestExt
		want   bool
	}{
		{
			name: "Req.ext not provided",
			reqExt: func() *openrtb_ext.RequestExt {
				return &openrtb_ext.RequestExt{}
			}(),
			want: true,
		},
		{
			name: "Req.ext provided EnforcePBS = false",
			reqExt: func() *openrtb_ext.RequestExt {
				reqExt := openrtb_ext.RequestExt{}
				prebidExt := openrtb_ext.ExtRequestPrebid{
					Floors: &openrtb_ext.PriceFloorRules{
						Enforcement: &openrtb_ext.PriceFloorEnforcement{
							EnforcePBS: ptrutil.ToPtr(false),
						},
					},
				}
				reqExt.SetPrebid(&prebidExt)
				return &reqExt
			}(),
			want: false,
		},
		{
			name: "Req.ext provided EnforcePBS = true",
			reqExt: func() *openrtb_ext.RequestExt {
				reqExt := openrtb_ext.RequestExt{}
				prebidExt := openrtb_ext.ExtRequestPrebid{
					Floors: &openrtb_ext.PriceFloorRules{
						Enforcement: &openrtb_ext.PriceFloorEnforcement{
							EnforcePBS: ptrutil.ToPtr(true),
						},
					},
				}
				reqExt.SetPrebid(&prebidExt)
				return &reqExt
			}(),
			want: true,
		},
	}
	for _, tt := range tests {
		got := isEnforcementEnabled(tt.reqExt)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestIsSignalingSkipped(t *testing.T) {
	tests := []struct {
		name   string
		reqExt *openrtb_ext.RequestExt
		want   bool
	}{
		{
			name: "Req.ext nil",
			reqExt: func() *openrtb_ext.RequestExt {
				return nil
			}(),
			want: false,
		},
		{
			name: "Req.ext provided without prebid ext",
			reqExt: func() *openrtb_ext.RequestExt {
				return &openrtb_ext.RequestExt{}
			}(),
			want: false,
		},
		{
			name: "Req.ext provided without Floors in prebid ext",
			reqExt: func() *openrtb_ext.RequestExt {
				reqExt := openrtb_ext.RequestExt{}
				prebidExt := openrtb_ext.ExtRequestPrebid{}
				reqExt.SetPrebid(&prebidExt)
				return &reqExt
			}(),
			want: false,
		},
		{
			name: "Req.ext provided Skipped = true",
			reqExt: func() *openrtb_ext.RequestExt {
				reqExt := openrtb_ext.RequestExt{}
				prebidExt := openrtb_ext.ExtRequestPrebid{
					Floors: &openrtb_ext.PriceFloorRules{
						Skipped: ptrutil.ToPtr(true),
					},
				}
				reqExt.SetPrebid(&prebidExt)
				return &reqExt
			}(),
			want: true,
		},
		{
			name: "Req.ext provided Skipped = false",
			reqExt: func() *openrtb_ext.RequestExt {
				reqExt := openrtb_ext.RequestExt{}
				prebidExt := openrtb_ext.ExtRequestPrebid{
					Floors: &openrtb_ext.PriceFloorRules{
						Skipped: ptrutil.ToPtr(false),
					},
				}
				reqExt.SetPrebid(&prebidExt)
				return &reqExt
			}(),
			want: false,
		},
	}
	for _, tt := range tests {
		got := isSignalingSkipped(tt.reqExt)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestGetEnforceRateRequest(t *testing.T) {
	tests := []struct {
		name   string
		reqExt *openrtb_ext.RequestExt
		want   int
	}{
		{
			name: "Req.ext not provided",
			reqExt: func() *openrtb_ext.RequestExt {
				return &openrtb_ext.RequestExt{}
			}(),
			want: 0,
		},
		{
			name: "Req.ext.prebid.floors provided with EnforceRate = 0",
			reqExt: func() *openrtb_ext.RequestExt {
				reqExt := openrtb_ext.RequestExt{}
				prebidExt := openrtb_ext.ExtRequestPrebid{
					Floors: &openrtb_ext.PriceFloorRules{
						Enforcement: &openrtb_ext.PriceFloorEnforcement{
							EnforceRate: 0,
						},
					},
				}
				reqExt.SetPrebid(&prebidExt)
				return &reqExt
			}(),
			want: 0,
		},
		{
			name: "Req.ext.prebid.floors provided with EnforceRate = 50",
			reqExt: func() *openrtb_ext.RequestExt {
				reqExt := openrtb_ext.RequestExt{}
				prebidExt := openrtb_ext.ExtRequestPrebid{
					Floors: &openrtb_ext.PriceFloorRules{
						Enforcement: &openrtb_ext.PriceFloorEnforcement{
							EnforceRate: 50,
						},
					},
				}
				reqExt.SetPrebid(&prebidExt)
				return &reqExt
			}(),
			want: 50,
		},
		{
			name: "Req.ext.prebid.floors provided with EnforceRate = 100",
			reqExt: func() *openrtb_ext.RequestExt {
				reqExt := openrtb_ext.RequestExt{}
				prebidExt := openrtb_ext.ExtRequestPrebid{
					Floors: &openrtb_ext.PriceFloorRules{
						Enforcement: &openrtb_ext.PriceFloorEnforcement{
							EnforceRate: 100,
						},
					},
				}
				reqExt.SetPrebid(&prebidExt)
				return &reqExt
			}(),
			want: 100,
		},
	}
	for _, tt := range tests {
		got := getEnforceRateRequest(tt.reqExt)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestGetEnforceDealsFlag(t *testing.T) {
	tests := []struct {
		name   string
		reqExt *openrtb_ext.RequestExt
		want   bool
	}{
		{
			name: "Req.ext not provided",
			reqExt: func() *openrtb_ext.RequestExt {
				return &openrtb_ext.RequestExt{}
			}(),
			want: false,
		},
		{
			name: "Req.ext.prebid.floors provided, enforceDeals not provided",
			reqExt: func() *openrtb_ext.RequestExt {
				reqExt := openrtb_ext.RequestExt{}
				prebidExt := openrtb_ext.ExtRequestPrebid{
					Floors: &openrtb_ext.PriceFloorRules{
						Enforcement: &openrtb_ext.PriceFloorEnforcement{},
					},
				}
				reqExt.SetPrebid(&prebidExt)
				return &reqExt
			}(),
			want: false,
		},
		{
			name: "Req.ext.prebid.floors provided with enforceDeals = false",
			reqExt: func() *openrtb_ext.RequestExt {
				reqExt := openrtb_ext.RequestExt{}
				prebidExt := openrtb_ext.ExtRequestPrebid{
					Floors: &openrtb_ext.PriceFloorRules{
						Enforcement: &openrtb_ext.PriceFloorEnforcement{
							FloorDeals: ptrutil.ToPtr(false),
						},
					},
				}
				reqExt.SetPrebid(&prebidExt)
				return &reqExt
			}(),
			want: false,
		},
		{
			name: "Req.ext.prebid.floors provided with enforceDeals = true",
			reqExt: func() *openrtb_ext.RequestExt {
				reqExt := openrtb_ext.RequestExt{}
				prebidExt := openrtb_ext.ExtRequestPrebid{
					Floors: &openrtb_ext.PriceFloorRules{
						Enforcement: &openrtb_ext.PriceFloorEnforcement{
							FloorDeals: ptrutil.ToPtr(true),
						},
					},
				}
				reqExt.SetPrebid(&prebidExt)
				return &reqExt
			}(),
			want: true,
		},
	}
	for _, tt := range tests {
		got := getEnforceDealsFlag(tt.reqExt)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestIsSatisfiedByEnforceRate(t *testing.T) {
	type args struct {
		reqExt            *openrtb_ext.RequestExt
		configEnforceRate int
		f                 func(int) int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "With EnforceRate = 50",
			args: args{
				reqExt: func() *openrtb_ext.RequestExt {
					reqExt := openrtb_ext.RequestExt{}
					prebidExt := openrtb_ext.ExtRequestPrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforceRate: 50,
								EnforcePBS:  ptrutil.ToPtr(true),
							},
						},
					}
					reqExt.SetPrebid(&prebidExt)
					return &reqExt
				}(),
				configEnforceRate: 100,
				f: func(n int) int {
					return n
				},
			},
			want: false,
		},
		{
			name: "With EnforceRate = 100",
			args: args{
				reqExt: func() *openrtb_ext.RequestExt {
					reqExt := openrtb_ext.RequestExt{}
					prebidExt := openrtb_ext.ExtRequestPrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforceRate: 100,
								EnforcePBS:  ptrutil.ToPtr(true),
							},
						},
					}
					reqExt.SetPrebid(&prebidExt)
					return &reqExt
				}(),
				configEnforceRate: 100,
				f: func(n int) int {
					return n - 1
				},
			},
			want: true,
		},
		{
			name: "With configEnforceRate = 0",
			args: args{
				reqExt: func() *openrtb_ext.RequestExt {
					reqExt := openrtb_ext.RequestExt{}
					prebidExt := openrtb_ext.ExtRequestPrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforceRate: 0,
								EnforcePBS:  ptrutil.ToPtr(true),
							},
						},
					}
					reqExt.SetPrebid(&prebidExt)
					return &reqExt
				}(),
				configEnforceRate: 0,
				f: func(n int) int {
					return n - 1
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		got := isSatisfiedByEnforceRate(tt.args.reqExt, tt.args.configEnforceRate, tt.args.f)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestUpdateEnforcePBS(t *testing.T) {
	type args struct {
		enforceFloors bool
		reqExt        *openrtb_ext.RequestExt
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Enforce PBS is true in request and to be updated = true",
			args: args{
				enforceFloors: true,
				reqExt: func() *openrtb_ext.RequestExt {
					reqExt := openrtb_ext.RequestExt{}
					prebidExt := openrtb_ext.ExtRequestPrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforcePBS: ptrutil.ToPtr(true),
							},
						},
					}
					reqExt.SetPrebid(&prebidExt)
					return &reqExt
				}(),
			},
			want: false,
		},
		{
			name: "Enforce PBS is false in request and to be updated = true",
			args: args{
				enforceFloors: true,
				reqExt: func() *openrtb_ext.RequestExt {
					reqExt := openrtb_ext.RequestExt{}
					prebidExt := openrtb_ext.ExtRequestPrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforcePBS: ptrutil.ToPtr(false),
							},
						},
					}
					reqExt.SetPrebid(&prebidExt)
					return &reqExt
				}(),
			},
			want: false,
		},
		{
			name: "Enforce PBS is true in request and to be updated = false",
			args: args{
				enforceFloors: false,
				reqExt: func() *openrtb_ext.RequestExt {
					reqExt := openrtb_ext.RequestExt{}
					prebidExt := openrtb_ext.ExtRequestPrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforcePBS: ptrutil.ToPtr(true),
							},
						},
					}
					reqExt.SetPrebid(&prebidExt)
					return &reqExt
				}(),
			},
			want: true,
		},
		{
			name: "empty prebid ext and to be updated = false",
			args: args{enforceFloors: false,
				reqExt: func() *openrtb_ext.RequestExt {
					return &openrtb_ext.RequestExt{}
				}(),
			},
			want: true,
		},
		{
			name: "empty prebid ext and to be updated = true",
			args: args{enforceFloors: true,
				reqExt: func() *openrtb_ext.RequestExt {
					reqExt := openrtb_ext.RequestExt{}
					prebidExt := openrtb_ext.ExtRequestPrebid{}
					reqExt.SetPrebid(&prebidExt)
					return &reqExt
				}(),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateEnforcePBS(tt.args.enforceFloors, tt.args.reqExt)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}
