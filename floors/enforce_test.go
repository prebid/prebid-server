package floors

import (
	"encoding/json"
	"errors"
	"sort"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
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
		name       string
		bidRequest *openrtb2.BidRequest
		want       bool
	}{
		{
			bidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
			},
			want: false,
		},
		{
			bidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1234", BidFloor: 10, BidFloorCur: "USD", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidImpBidFloorPresent(tt.bidRequest)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestEnforceFloorToBids(t *testing.T) {

	imp1_bid_1_2 := entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}}
	imp2_bid_1_5 := entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "some-bid-2", Price: 1.5, DealID: "deal_Id", ImpID: "some-impression-id-2"}}
	imp1_bid_0_5 := entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.5, ImpID: "some-impression-id-1"}}
	imp2_bid_2_2 := entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "some-bid-12", Price: 2.2, ImpID: "some-impression-id-2"}}

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
							&imp1_bid_1_2,
							&imp2_bid_1_5,
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							&imp1_bid_0_5,
							&imp2_bid_2_2,
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
						&imp1_bid_1_2,
						&imp2_bid_1_5,
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						&imp2_bid_2_2,
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
						&imp1_bid_0_5,
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
							&imp1_bid_1_2,
							&imp2_bid_1_5,
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							&imp1_bid_0_5,
							&imp2_bid_2_2,
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
						&imp1_bid_1_2,
						&imp2_bid_1_5,
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						&imp2_bid_2_2,
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
						&imp1_bid_0_5,
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
							&imp1_bid_1_2,
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
			expErrs:         []error{errors.New("error in rate conversion from = EUR to USD with bidder pubmatic for impression id some-impression-id-1 and bid id some-bid-1 error = currency conversion not supported")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seatbids, errs, rejBids := enforceFloorToBids(tt.args.bidRequestWrapper, tt.args.seatBids, tt.args.conversions, tt.args.enforceDealFloors)
			assert.Equal(t, tt.expEligibleBids, seatbids, tt.name)
			assert.Equal(t, tt.expErrs, errs)
			assert.Equal(t, tt.expRejectedBids, rejBids)
		})
	}
}

func TestEnforceFloors(t *testing.T) {
	imp1_bid_deal_1_2 := entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, DealID: "deal_Id_1", ImpID: "some-impression-id-1"}}
	imp1_bid_deal_0_5 := entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 0.5, DealID: "deal_Id_3", ImpID: "some-impression-id-1"}}
	imp1_bid_1_2 := entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "some-bid-1", Price: 1.2, ImpID: "some-impression-id-1"}}
	imp2_bid_4_5 := entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "some-bid-11", Price: 4.5, ImpID: "some-impression-id-1"}}

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
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":false},"enabled":false,"skipped":false}}}`),
					},
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							&imp2_bid_4_5,
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
						&imp2_bid_4_5,
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
			},
			expErrs:         []error{errors.New("Floors enforcement is disabled at account or in the request")},
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
							&imp1_bid_1_2,
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
						&imp1_bid_1_2,
					},
					Seat:     "pubmatic",
					Currency: "USD",
				},
			},
			expErrs:         []error{errors.New("Floors enforcement is disabled at account or in the request")},
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
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true,"skipped":false}}}`),
					},
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids:     []*entities.PbsOrtbBid{&imp1_bid_deal_1_2},
						Seat:     "pubmatic",
						Currency: "USD",
					},
					"appnexus": {
						Bids:     []*entities.PbsOrtbBid{&imp1_bid_deal_0_5},
						Seat:     "appnexus",
						Currency: "USD",
					},
				},
				conversions:    convert{},
				priceFloorsCfg: config.AccountPriceFloors{Enabled: true, EnforceFloorsRate: 100, EnforceDealFloors: true},
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
					Bids:     []*entities.PbsOrtbBid{&imp1_bid_deal_1_2},
				},
				{
					Seat:     "appnexus",
					Currency: "USD",
					Bids:     []*entities.PbsOrtbBid{&imp1_bid_deal_0_5},
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
							&imp1_bid_deal_1_2,
						},
						Seat:     "pubmatic",
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							&imp2_bid_4_5,
						},
						Seat:     "appnexus",
						Currency: "USD",
					},
				},
				conversions:    convert{},
				priceFloorsCfg: config.AccountPriceFloors{Enabled: true, EnforceFloorsRate: 100, EnforceDealFloors: false},
			},
			expEligibleBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						&imp1_bid_deal_1_2,
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
					Bids:     []*entities.PbsOrtbBid{&imp2_bid_4_5},
				},
			},
			expErrs: []error{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actEligibleBids, actErrs, actRejecteBids := EnforceFloors(tt.args.bidRequestWrapper, tt.args.seatBids, config.Account{PriceFloors: tt.args.priceFloorsCfg}, tt.args.conversions)
			assert.Equal(t, tt.expErrs, actErrs, tt.name)
			assert.Equal(t, tt.expEligibleBids, actEligibleBids, tt.name)

			sort.Slice(tt.expRejectedBids, func(i, j int) bool {
				return tt.expRejectedBids[i].Seat < tt.expRejectedBids[i].Seat
			})
			sort.Slice(actRejecteBids, func(i, j int) bool {
				return actRejecteBids[i].Seat < actRejecteBids[i].Seat
			})
			assert.Equal(t, tt.expRejectedBids, actRejecteBids, tt.name)

		})
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
		expBidFloor *openrtb_ext.ExtBidFloors
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
			expBidFloor: &openrtb_ext.ExtBidFloors{
				FloorValue:    10,
				FloorCurrency: "USD",
			},
		},
		{
			name: "Valid prebid extension in imp.ext",
			args: args{
				reqImp: &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250}, Ext: []byte(`{"prebid":{"floors":{"floorrule":"test|123|xyz","floorrulevalue":5.5,"floorvalue":5.5}}}`)}},
				bid: &entities.PbsOrtbBid{
					Bid: &openrtb2.Bid{
						Price: 10.10,
						AdM:   "Adm",
					},
				},
				floorCurrency: "USD",
			},
			expBidFloor: &openrtb_ext.ExtBidFloors{
				FloorRule:      "test|123|xyz",
				FloorRuleValue: 5.5,
				FloorValue:     5.5,
				FloorCurrency:  "USD",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateBidExtWithFloors(tt.args.reqImp, tt.args.bid, tt.args.floorCurrency)
			assert.Equal(t, tt.expBidFloor, tt.args.bid.BidFloors, tt.name)
		})
	}
}

func TestIsPriceFloorsEnforcementDisabledForRequest(t *testing.T) {

	tests := []struct {
		name              string
		bidRequestWrapper *openrtb_ext.RequestWrapper
		want              bool
	}{
		{
			name: "Req.ext not provided",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: false,
		},
		{
			name: "Req.ext provided EnforcePBS = false",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":false,"floordeals":true},"enabled":true}}}`),
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: true,
		},
		{
			name: "Req.ext provided EnforcePBS = true",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true}}}`),
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPriceFloorsEnforcementDisabled(tt.bidRequestWrapper)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestIsFloorsSignallingSkipped(t *testing.T) {

	tests := []struct {
		name              string
		bidRequestWrapper *openrtb_ext.RequestWrapper
		want              bool
	}{
		{
			name: "Req.ext not provided",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: false,
		},
		{
			name: "Req.ext provided Skipped = true",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true,"skipped":true}}}`),
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: true,
		},
		{
			name: "Req.ext provided Skipped = false",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true,"skipped":false}}}`),
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFloorsSignallingSkipped(tt.bidRequestWrapper)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestGetEnforceRateRequest(t *testing.T) {

	tests := []struct {
		name              string
		bidRequestWrapper *openrtb_ext.RequestWrapper
		want              int
	}{
		{
			name: "Req.ext not provided",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: 0,
		},
		{
			name: "Req.ext.prebid.floors provided with EnforceRate = 0",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":0},"enabled":true,"skipped":false}}}`),
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: 0,
		},
		{
			name: "Req.ext.prebid.floors provided with EnforceRate = 50",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":50},"enabled":true,"skipped":true}}}`),
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: 50,
		},
		{
			name: "Req.ext.prebid.floors provided with EnforceRate = 100",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100},"enabled":true,"skipped":true}}}`),
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getEnforceRateRequest(tt.bidRequestWrapper)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestGetEnforceDealsFlag(t *testing.T) {

	tests := []struct {
		name              string
		bidRequestWrapper *openrtb_ext.RequestWrapper
		want              bool
	}{
		{
			name: "Req.ext not provided",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: false,
		},
		{
			name: "Req.ext.prebid.floors provided,  enforceDeals not provided",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"enforcerate":0},"enabled":true,"skipped":false}}}`),
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: false,
		},
		{
			name: "Req.ext.prebid.floors provided with enforceDeals = false",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":false,"enforcerate":0},"enabled":true,"skipped":false}}}`),
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: false,
		},
		{
			name: "Req.ext.prebid.floors provided with enforceDeals = true",
			bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
				bw := openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{
							{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
						},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":50},"enabled":true,"skipped":true}}}`),
					},
				}
				bw.RebuildRequest()
				return &bw
			}(),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getEnforceDealsFlag(tt.bidRequestWrapper)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestIsSatisfiedByEnforceRate(t *testing.T) {
	type args struct {
		bidRequestWrapper *openrtb_ext.RequestWrapper
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
				bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
					bw := openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{
								{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
							},
							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":50},"enabled":true}}}`),
						},
					}
					bw.RebuildRequest()
					return &bw
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
				bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
					bw := openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{
								{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
							},
							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100},"enabled":true}}}`),
						},
					}
					bw.RebuildRequest()
					return &bw
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
				bidRequestWrapper: func() *openrtb_ext.RequestWrapper {
					bw := openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{
								{ID: "some-impression-id-1", BidFloor: 1.01, BidFloorCur: "USD"},
							},
							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"EUR","data":{"currency":"USD","floorsschemaversion":"2","modelgroups":[{"modelweight":40,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":21.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100},"enabled":true}}}`),
						},
					}
					bw.RebuildRequest()
					return &bw
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
		t.Run(tt.name, func(t *testing.T) {
			got := isSatisfiedByEnforceRate(tt.args.bidRequestWrapper, tt.args.configEnforceRate, tt.args.f)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}
