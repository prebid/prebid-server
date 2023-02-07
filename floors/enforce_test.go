package floors

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
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

func ErrToString(Err []error) []string {
	var errString []string
	for _, eachErr := range Err {
		errString = append(errString, eachErr.Error())
	}
	sort.Strings(errString)
	return errString
}

func getFalse() *bool {
	b := false
	return &b
}

func getTrue() *bool {
	b := true
	return &b
}

func TestIsImpBidfloorPresentInRequest(t *testing.T) {

	tests := []struct {
		name       string
		bidRequest *openrtb2.BidRequest
		want       bool
	}{
		{
			bidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
			},
			want: false,
		},
		{
			bidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", BidFloor: 10, BidFloorCur: "USD", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsImpBidfloorPresentInRequest(tt.bidRequest); got != tt.want {
				t.Errorf("RequestHasFloors() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestShouldEnforceFloors(t *testing.T) {
	type args struct {
		bidRequest        *openrtb2.BidRequest
		floorExt          *openrtb_ext.PriceFloorRules
		configEnforceRate int
		f                 func(int) int
	}
	tests := []struct {
		name            string
		args            args
		expEnforce      bool
		expReqExtUpdate bool
	}{
		{
			name: "enfocement = true of enforcement object not provided",
			args: args{
				bidRequest: func() *openrtb2.BidRequest {
					r := openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								BidFloor:    2.2,
								BidFloorCur: "USD",
							},
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
						},
					}
					return &r
				}(),
				configEnforceRate: 100,
				f: func(n int) int {
					return n - 1
				},
			},
			expEnforce:      true,
			expReqExtUpdate: true,
		},

		{
			name: "No enfocement of floors when enforcePBS is false",
			args: args{
				bidRequest: func() *openrtb2.BidRequest {
					r := openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								BidFloor:    2.2,
								BidFloorCur: "USD",
							},
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
						},
					}
					return &r
				}(),
				floorExt: &openrtb_ext.PriceFloorRules{
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS: getFalse(),
					},
					Skipped: getFalse(),
				},
				configEnforceRate: 10,
				f: func(n int) int {
					return n
				},
			},
			expEnforce:      false,
			expReqExtUpdate: false,
		},
		{
			name: "No enfocement of floors when enforcePBS is true but enforce rate is low",
			args: args{
				bidRequest: func() *openrtb2.BidRequest {
					r := openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								BidFloor:    2.2,
								BidFloorCur: "USD",
							},
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
						},
					}
					return &r
				}(),
				floorExt: &openrtb_ext.PriceFloorRules{
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS: getTrue(),
					},
					Skipped: getFalse(),
				},
				configEnforceRate: 10,
				f: func(n int) int {
					return n
				},
			},
			expEnforce:      false,
			expReqExtUpdate: true,
		},
		{
			name: "No enfocement of floors when enforcePBS is true but enforce rate is low in incoming request",
			args: args{
				bidRequest: func() *openrtb2.BidRequest {
					r := openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								BidFloor:    2.2,
								BidFloorCur: "USD",
							},
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
						},
					}
					return &r
				}(),
				floorExt: &openrtb_ext.PriceFloorRules{
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS:  getTrue(),
						EnforceRate: 10,
					},
					Skipped: getFalse(),
				},
				configEnforceRate: 100,
				f: func(n int) int {
					return n
				},
			},
			expEnforce:      false,
			expReqExtUpdate: true,
		},
		{
			name: "No Enfocement of floors when skipped is true, non zero value of bidfloor in imp",
			args: args{
				bidRequest: func() *openrtb2.BidRequest {
					r := openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								BidFloor:    2.2,
								BidFloorCur: "USD",
							},
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
						},
					}
					return &r
				}(),
				floorExt: &openrtb_ext.PriceFloorRules{
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS: getTrue(),
					},
					Skipped: getTrue(),
				},
				configEnforceRate: 98,
				f: func(n int) int {
					return n - 5
				},
			},
			expEnforce:      false,
			expReqExtUpdate: false,
		},
		{
			name: "No enfocement of floors when skipped is true, zero value of bidfloor in imp",
			args: args{
				bidRequest: func() *openrtb2.BidRequest {
					r := openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
						},
					}
					return &r
				}(),
				floorExt: &openrtb_ext.PriceFloorRules{
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS: getTrue(),
					},
					Skipped: getTrue(),
				},
				configEnforceRate: 98,
				f: func(n int) int {
					return n - 5
				},
			},
			expEnforce:      false,
			expReqExtUpdate: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldEnforce, updateReq := shouldEnforceFloors(tt.args.bidRequest, tt.args.floorExt, tt.args.configEnforceRate, tt.args.f)
			if shouldEnforce != tt.expEnforce {
				t.Errorf("shouldEnforce = %v, want %v", shouldEnforce, tt.expEnforce)
			}

			if updateReq != tt.expReqExtUpdate {
				t.Errorf("expReqExtUpdate  %v, want %v", updateReq, tt.expReqExtUpdate)
			}
		})
	}
}

func TestEnforceFloorToBids(t *testing.T) {

	type args struct {
		bidRequest        *openrtb2.BidRequest
		seatBids          map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		conversions       currency.Conversions
		enforceDealFloors bool
	}
	tests := []struct {
		name  string
		args  args
		want  map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		want1 []string
	}{
		{
			name: "Bids with same currency",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					ID: "some-request-id",
					Imp: []openrtb2.Imp{
						{
							ID:          "some-impression-id-1",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							Ext:         json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor:    1.01,
							BidFloorCur: "USD",
						},
						{
							ID:          "some-impression-id-2",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 400, H: 350}, {W: 200, H: 600}}},
							Ext:         json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor:    2.01,
							BidFloorCur: "USD",
						},
					},
					Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
					AT:   1,
					TMax: 500,
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-1",
									Price: 1.2,
									ImpID: "some-impression-id-1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-2",
									Price: 1.5,
									ImpID: "some-impression-id-2",
								},
							},
						},
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-11",
									Price: 0.5,
									ImpID: "some-impression-id-1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-12",
									Price: 2.2,
									ImpID: "some-impression-id-2",
								},
							},
						},
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-1",
								Price: 1.2,
								ImpID: "some-impression-id-1",
							},
						},
					},
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-12",
								Price: 2.2,
								ImpID: "some-impression-id-2",
							},
						},
					},
					Currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 1.0100 USD for impression id some-impression-id-1 bidder appnexus", "bid rejected [bid ID: some-bid-2] reason: bid price value 1.5000 USD is less than bidFloor value 2.0100 USD for impression id some-impression-id-2 bidder pubmatic"},
		},
		{
			name: "Bids with different currency",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					ID: "some-request-id",
					Imp: []openrtb2.Imp{
						{
							ID:          "some-impression-id-1",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							Ext:         json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor:    60,
							BidFloorCur: "INR",
						},
						{
							ID:          "some-impression-id-2",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 400, H: 350}, {W: 200, H: 600}}},
							Ext:         json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor:    100,
							BidFloorCur: "INR",
						},
					},
					Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
					AT:   1,
					TMax: 500,
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-1",
									Price: 1.2,
									ImpID: "some-impression-id-1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-2",
									Price: 1.5,
									ImpID: "some-impression-id-2",
								},
							},
						},
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-11",
									Price: 0.5,
									ImpID: "some-impression-id-1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-12",
									Price: 2.2,
									ImpID: "some-impression-id-2",
								},
							},
						},
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-1",
								Price: 1.2,
								ImpID: "some-impression-id-1",
							},
						},
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-2",
								Price: 1.5,
								ImpID: "some-impression-id-2",
							},
						},
					},
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-12",
								Price: 2.2,
								ImpID: "some-impression-id-2",
							},
						},
					},
					Currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 38.7950 INR is less than bidFloor value 60.0000 INR for impression id some-impression-id-1 bidder appnexus"},
		},
		{
			name: "Bids with different currency with enforceDealFloor false",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					ID: "some-request-id",
					Imp: []openrtb2.Imp{
						{
							ID:          "some-impression-id-1",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							Ext:         json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor:    60,
							BidFloorCur: "INR",
						},
						{
							ID:          "some-impression-id-2",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 400, H: 350}, {W: 200, H: 600}}},
							Ext:         json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor:    100,
							BidFloorCur: "INR",
						},
					},
					Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
					AT:   1,
					TMax: 500,
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-1",
									Price: 1.2,
									ImpID: "some-impression-id-1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-2",
									Price: 1.5,
									ImpID: "some-impression-id-2",
								},
							},
						},
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-11",
									Price: 0.5,
									ImpID: "some-impression-id-1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-12",
									Price: 2.2,
									ImpID: "some-impression-id-2",
								},
							},
						},
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-1",
								Price: 1.2,
								ImpID: "some-impression-id-1",
							},
						},
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-2",
								Price: 1.5,
								ImpID: "some-impression-id-2",
							},
						},
					},
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-12",
								Price: 2.2,
								ImpID: "some-impression-id-2",
							},
						},
					},
					Currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 38.7950 INR is less than bidFloor value 60.0000 INR for impression id some-impression-id-1 bidder appnexus"},
		},
		{
			name: "Dealid not empty, enforceDealFloors is true",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					ID: "some-request-id",
					Imp: []openrtb2.Imp{
						{
							ID:          "some-impression-id-1",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							Ext:         json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor:    60,
							BidFloorCur: "INR",
						},
						{
							ID:          "some-impression-id-2",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 400, H: 350}, {W: 200, H: 600}}},
							Ext:         json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor:    100,
							BidFloorCur: "INR",
						},
					},
					Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
					AT:   1,
					TMax: 500,
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:     "some-bid-1",
									Price:  1.2,
									ImpID:  "some-impression-id-1",
									DealID: "1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:     "some-bid-2",
									Price:  1.5,
									ImpID:  "some-impression-id-2",
									DealID: "2",
								},
							},
						},
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:     "some-bid-11",
									Price:  0.5,
									ImpID:  "some-impression-id-1",
									DealID: "3",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:     "some-bid-12",
									Price:  2.2,
									ImpID:  "some-impression-id-2",
									DealID: "4",
								},
							},
						},
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:     "some-bid-1",
								Price:  1.2,
								ImpID:  "some-impression-id-1",
								DealID: "1",
							},
						},
						{
							Bid: &openrtb2.Bid{
								ID:     "some-bid-2",
								Price:  1.5,
								ImpID:  "some-impression-id-2",
								DealID: "2",
							},
						},
					},
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:     "some-bid-12",
								Price:  2.2,
								ImpID:  "some-impression-id-2",
								DealID: "4",
							},
						},
					},
					Currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 38.7950 INR is less than bidFloor value 60.0000 INR for impression id some-impression-id-1 bidder appnexus"},
		},
		{
			name: "Dealid not empty, enforceDealFloors is false",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					ID: "some-request-id",
					Imp: []openrtb2.Imp{
						{
							ID:          "some-impression-id-1",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							Ext:         json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor:    60,
							BidFloorCur: "INR",
						},
						{
							ID:          "some-impression-id-2",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 400, H: 350}, {W: 200, H: 600}}},
							Ext:         json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor:    100,
							BidFloorCur: "INR",
						},
					},
					Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
					AT:   1,
					TMax: 500,
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:     "some-bid-1",
									Price:  1.2,
									ImpID:  "some-impression-id-1",
									DealID: "1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:     "some-bid-2",
									Price:  1.5,
									ImpID:  "some-impression-id-2",
									DealID: "2",
								},
							},
						},
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:     "some-bid-11",
									Price:  0.5,
									ImpID:  "some-impression-id-1",
									DealID: "3",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:     "some-bid-12",
									Price:  2.2,
									ImpID:  "some-impression-id-2",
									DealID: "4",
								},
							},
						},
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: false,
			},
			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:     "some-bid-1",
								Price:  1.2,
								ImpID:  "some-impression-id-1",
								DealID: "1",
							},
						},
						{
							Bid: &openrtb2.Bid{
								ID:     "some-bid-2",
								Price:  1.5,
								ImpID:  "some-impression-id-2",
								DealID: "2",
							},
						},
					},
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:     "some-bid-11",
								Price:  0.5,
								ImpID:  "some-impression-id-1",
								DealID: "3",
							},
						},
						{
							Bid: &openrtb2.Bid{
								ID:     "some-bid-12",
								Price:  2.2,
								ImpID:  "some-impression-id-2",
								DealID: "4",
							},
						},
					},
					Currency: "USD",
				},
			},
			want1: nil,
		},
		{
			name: "Impression does not have currency defined",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					ID:  "some-request-id",
					Cur: []string{"USD"},
					Imp: []openrtb2.Imp{
						{
							ID:       "some-impression-id-1",
							Banner:   &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							Ext:      json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor: 1.01,
						},
						{
							ID:       "some-impression-id-2",
							Banner:   &openrtb2.Banner{Format: []openrtb2.Format{{W: 400, H: 350}, {W: 200, H: 600}}},
							Ext:      json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor: 2.01,
						},
					},
					Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
					AT:   1,
					TMax: 500,
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-1",
									Price: 1.2,
									ImpID: "some-impression-id-1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-2",
									Price: 1.5,
									ImpID: "some-impression-id-2",
								},
							},
						},
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-11",
									Price: 0.5,
									ImpID: "some-impression-id-1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-12",
									Price: 2.2,
									ImpID: "some-impression-id-2",
								},
							},
						},
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-1",
								Price: 1.2,
								ImpID: "some-impression-id-1",
							},
						},
					},
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-12",
								Price: 2.2,
								ImpID: "some-impression-id-2",
							},
						},
					},
					Currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 1.0100 USD for impression id some-impression-id-1 bidder appnexus", "bid rejected [bid ID: some-bid-2] reason: bid price value 1.5000 USD is less than bidFloor value 2.0100 USD for impression id some-impression-id-2 bidder pubmatic"},
		},
		{
			name: "Impression map does not have imp id",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					ID:  "some-request-id",
					Cur: []string{"USD"},
					Imp: []openrtb2.Imp{
						{
							ID:       "some-impression-id-1",
							Banner:   &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							Ext:      json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor: 1.01,
						},
						{
							ID:       "some-impression-id-2",
							Banner:   &openrtb2.Banner{Format: []openrtb2.Format{{W: 400, H: 350}, {W: 200, H: 600}}},
							Ext:      json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor: 2.01,
						},
					},
					Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
					AT:   1,
					TMax: 500,
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-1",
									Price: 1.2,
									ImpID: "some-impression-id-1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-2",
									Price: 1.5,
									ImpID: "some-impression-id-2",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-3",
									Price: 1.4,
									ImpID: "some-impression-id-3",
								},
							},
						},
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-11",
									Price: 0.5,
									ImpID: "some-impression-id-1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-12",
									Price: 2.2,
									ImpID: "some-impression-id-2",
								},
							},
						},
						Currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-1",
								Price: 1.2,
								ImpID: "some-impression-id-1",
							},
						},
					},
					Currency: "USD",
				},
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{
						{
							Bid: &openrtb2.Bid{
								ID:    "some-bid-12",
								Price: 2.2,
								ImpID: "some-impression-id-2",
							},
						},
					},
					Currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 1.0100 USD for impression id some-impression-id-1 bidder appnexus", "bid rejected [bid ID: some-bid-2] reason: bid price value 1.5000 USD is less than bidFloor value 2.0100 USD for impression id some-impression-id-2 bidder pubmatic"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seatbids, errs, _ := enforceFloorToBids(tt.args.bidRequest, tt.args.seatBids, tt.args.conversions, tt.args.enforceDealFloors)
			if !reflect.DeepEqual(seatbids, tt.want) {
				t.Errorf("enforceFloorToBids() got = %v, want %v", seatbids, tt.want)
			}
			assert.Equal(t, tt.want1, ErrToString(errs))
		})
	}
}

func TestEnforceFloorToBidsConversion(t *testing.T) {

	type args struct {
		bidRequest        *openrtb2.BidRequest
		seatBids          map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		conversions       currency.Conversions
		enforceDealFloors bool
	}

	tests := []struct {
		name  string
		args  args
		want  map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		want1 []string
	}{
		{
			name: "Error in currency conversion",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					ID:  "some-request-id",
					Cur: []string{"USD"},
					Imp: []openrtb2.Imp{
						{
							ID:       "some-impression-id-1",
							Banner:   &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							Ext:      json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor: 1.01,
						},
						{
							ID:       "some-impression-id-2",
							Banner:   &openrtb2.Banner{Format: []openrtb2.Format{{W: 400, H: 350}, {W: 200, H: 600}}},
							Ext:      json.RawMessage(`{"appnexus": {"placementId": 1}}`),
							BidFloor: 2.01,
						},
					},
					Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
					AT:   1,
					TMax: 500,
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-1",
									Price: 1.2,
									ImpID: "some-impression-id-1",
								},
							},
							{
								Bid: &openrtb2.Bid{
									ID:    "some-bid-2",
									Price: 1.5,
									ImpID: "some-impression-id-2",
								},
							},
						},
						Currency: "EUR",
					},
				},
				conversions:       convert{},
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids:     []*entities.PbsOrtbBid{},
					Currency: "EUR",
				},
			},
			want1: []string{"Error in rate conversion from = EUR to USD with bidder pubmatic for impression id some-impression-id-1 and bid id some-bid-1", "Error in rate conversion from = EUR to USD with bidder pubmatic for impression id some-impression-id-2 and bid id some-bid-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, _ := enforceFloorToBids(tt.args.bidRequest, tt.args.seatBids, tt.args.conversions, tt.args.enforceDealFloors)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, ErrToString(got1))
		})
	}
}

// func TestEnforceFloors(t *testing.T) {
// 	type args struct {
// 		r                  *AuctionRequest
// 		seatBids           map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
// 		floor              config.PriceFloors
// 		conversions        currency.Conversions
// 		responseDebugAllow bool
// 	}
// 	tests := []struct {
// 		name  string
// 		args  args
// 		want  map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
// 		want1 []string
// 	}{
// 		{
// 			name: "Should enforce floors for deals, ext.prebid.floors.enforcement.floorDeals=true and floors enabled = true",
// 			args: args{
// 				r: &AuctionRequest{
// 					BidRequestWrapper: &openrtb_ext.RequestWrapper{
// 						BidRequest: &openrtb2.BidRequest{
// 							ID: "some-request-id",
// 							Imp: []openrtb2.Imp{{
// 								ID:          "some-impression-id-1",
// 								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
// 								BidFloor:    20.01,
// 								BidFloorCur: "USD",
// 							}},
// 							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true,"skipped":false}}}`),
// 						},
// 					},
// 					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
// 				},
// 				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 					"pubmatic": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-1",
// 									Price:  1.2,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 					"appnexus": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-11",
// 									Price:  0.5,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "3",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 				},
// 				floor: config.PriceFloors{
// 					Enabled: true,
// 				},
// 				conversions:        convert{},
// 				responseDebugAllow: true,
// 			},
// 			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 				"pubmatic": {
// 					Bids:     []*entities.PbsOrtbBid{},
// 					Currency: "USD",
// 				},
// 				"appnexus": {
// 					Bids:     []*entities.PbsOrtbBid{},
// 					Currency: "USD",
// 				},
// 			},
// 			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder appnexus", "bid rejected [bid ID: some-bid-1] reason: bid price value 1.2000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder pubmatic"},
// 		},
// 		{
// 			name: "Should not enforce floors for deals, ext.prebid.floors.enforcement.floorDeals not provided",
// 			args: args{
// 				r: &AuctionRequest{
// 					BidRequestWrapper: &openrtb_ext.RequestWrapper{
// 						BidRequest: &openrtb2.BidRequest{
// 							ID: "some-request-id",
// 							Imp: []openrtb2.Imp{{
// 								ID:          "some-impression-id-1",
// 								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
// 								BidFloor:    20.01,
// 								BidFloorCur: "USD",
// 							}},
// 							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true},"enabled":true,"skipped":false}}}`),
// 						},
// 					},
// 					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
// 				},
// 				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 					"pubmatic": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-1",
// 									Price:  1.2,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 					"appnexus": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:    "some-bid-11",
// 									Price: 0.5,
// 									ImpID: "some-impression-id-1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 				},
// 				floor: config.PriceFloors{
// 					Enabled: true,
// 				},
// 				conversions:        convert{},
// 				responseDebugAllow: true,
// 			},
// 			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 				"pubmatic": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:     "some-bid-1",
// 								Price:  1.2,
// 								ImpID:  "some-impression-id-1",
// 								DealID: "1",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 				"appnexus": {
// 					Bids:     []*entities.PbsOrtbBid{},
// 					Currency: "USD",
// 				},
// 			},
// 			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder appnexus"},
// 		},
// 		{
// 			name: "Should not enforce floors for deals, ext.prebid.floors.enforcement.floorDeals=false is set",
// 			args: args{
// 				r: &AuctionRequest{
// 					BidRequestWrapper: &openrtb_ext.RequestWrapper{
// 						BidRequest: &openrtb2.BidRequest{
// 							ID: "some-request-id",
// 							Imp: []openrtb2.Imp{{
// 								ID:          "some-impression-id-1",
// 								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
// 								BidFloor:    20.01,
// 								BidFloorCur: "USD",
// 							}},
// 							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true, "floordeals":false},"enabled":true,"skipped":false}}}`),
// 						},
// 					},
// 					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
// 				},
// 				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 					"pubmatic": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-1",
// 									Price:  1.2,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 					"appnexus": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:    "some-bid-11",
// 									Price: 0.5,
// 									ImpID: "some-impression-id-1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 				},
// 				floor: config.PriceFloors{
// 					Enabled: true,
// 				},
// 				conversions:        convert{},
// 				responseDebugAllow: true,
// 			},
// 			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 				"pubmatic": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:     "some-bid-1",
// 								Price:  1.2,
// 								ImpID:  "some-impression-id-1",
// 								DealID: "1",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 				"appnexus": {
// 					Bids:     []*entities.PbsOrtbBid{},
// 					Currency: "USD",
// 				},
// 			},
// 			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder appnexus"},
// 		},
// 		{
// 			name: "Should not enforce floors for deals, ext.prebid.floors.enforcement.floorDeals=true and EnforceDealFloors = false from config",
// 			args: args{
// 				r: &AuctionRequest{
// 					BidRequestWrapper: &openrtb_ext.RequestWrapper{
// 						BidRequest: &openrtb2.BidRequest{
// 							ID: "some-request-id",
// 							Imp: []openrtb2.Imp{{
// 								ID:          "some-impression-id-1",
// 								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
// 								BidFloor:    20.01,
// 								BidFloorCur: "USD",
// 							}},
// 							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true, "floordeals":true},"enabled":true,"skipped":false}}}`),
// 						},
// 					},
// 					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: false}},
// 				},
// 				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 					"pubmatic": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-1",
// 									Price:  1.2,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 					"appnexus": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:    "some-bid-11",
// 									Price: 0.5,
// 									ImpID: "some-impression-id-1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 				},
// 				floor: config.PriceFloors{
// 					Enabled: true,
// 				},
// 				conversions:        convert{},
// 				responseDebugAllow: true,
// 			},
// 			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 				"pubmatic": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:     "some-bid-1",
// 								Price:  1.2,
// 								ImpID:  "some-impression-id-1",
// 								DealID: "1",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 				"appnexus": {
// 					Bids:     []*entities.PbsOrtbBid{},
// 					Currency: "USD",
// 				},
// 			},
// 			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder appnexus"},
// 		},
// 		{
// 			name: "Should enforce floors when imp.bidfloor provided and req.ext.prebid not provided",
// 			args: args{
// 				r: &AuctionRequest{
// 					BidRequestWrapper: &openrtb_ext.RequestWrapper{
// 						BidRequest: &openrtb2.BidRequest{
// 							ID: "some-request-id",
// 							Imp: []openrtb2.Imp{{
// 								ID:          "some-impression-id-1",
// 								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
// 								BidFloor:    5.01,
// 								BidFloorCur: "USD",
// 							}},
// 						},
// 					},
// 					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: false}},
// 				},
// 				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 					"pubmatic": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-1",
// 									Price:  1.2,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 					"appnexus": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:    "some-bid-11",
// 									Price: 0.5,
// 									ImpID: "some-impression-id-1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 				},
// 				floor: config.PriceFloors{
// 					Enabled: true,
// 				},
// 				conversions:        convert{},
// 				responseDebugAllow: true,
// 			},
// 			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 				"pubmatic": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:     "some-bid-1",
// 								Price:  1.2,
// 								ImpID:  "some-impression-id-1",
// 								DealID: "1",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 				"appnexus": {
// 					Bids:     []*entities.PbsOrtbBid{},
// 					Currency: "USD",
// 				},
// 			},
// 			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 5.0100 USD for impression id some-impression-id-1 bidder appnexus"},
// 		},
// 		{
// 			name: "Should not enforce floors when imp.bidfloor not provided and req.ext.prebid not provided",
// 			args: args{
// 				r: &AuctionRequest{
// 					BidRequestWrapper: &openrtb_ext.RequestWrapper{
// 						BidRequest: &openrtb2.BidRequest{
// 							ID: "some-request-id",
// 							Imp: []openrtb2.Imp{{
// 								ID:     "some-impression-id-1",
// 								Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
// 							}},
// 						},
// 					},
// 					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: false}},
// 				},
// 				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 					"pubmatic": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-1",
// 									Price:  1.2,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 					"appnexus": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:    "some-bid-11",
// 									Price: 0.5,
// 									ImpID: "some-impression-id-1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 				},
// 				floor: config.PriceFloors{
// 					Enabled: true,
// 				},
// 				conversions:        convert{},
// 				responseDebugAllow: true,
// 			},
// 			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 				"pubmatic": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:     "some-bid-1",
// 								Price:  1.2,
// 								ImpID:  "some-impression-id-1",
// 								DealID: "1",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 				"appnexus": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:    "some-bid-11",
// 								Price: 0.5,
// 								ImpID: "some-impression-id-1",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 			},
// 			want1: nil,
// 		},
// 		{
// 			name: "Should not enforce floors when  config flag Enabled = false",
// 			args: args{
// 				r: &AuctionRequest{
// 					BidRequestWrapper: &openrtb_ext.RequestWrapper{
// 						BidRequest: &openrtb2.BidRequest{
// 							ID: "some-request-id",
// 							Imp: []openrtb2.Imp{{
// 								ID:          "some-impression-id-1",
// 								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
// 								BidFloor:    5.01,
// 								BidFloorCur: "USD",
// 							}},
// 						},
// 					},
// 					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: false}},
// 				},
// 				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 					"pubmatic": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-1",
// 									Price:  1.2,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 					"appnexus": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-11",
// 									Price:  0.5,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "3",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 				},
// 				floor: config.PriceFloors{
// 					Enabled: false,
// 				},
// 				conversions:        convert{},
// 				responseDebugAllow: true,
// 			},
// 			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 				"pubmatic": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:     "some-bid-1",
// 								Price:  1.2,
// 								ImpID:  "some-impression-id-1",
// 								DealID: "1",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 				"appnexus": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:     "some-bid-11",
// 								Price:  0.5,
// 								ImpID:  "some-impression-id-1",
// 								DealID: "3",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 			},
// 			want1: nil,
// 		},
// 		{
// 			name: "Should not enforce floors when req.ext.prebid.floors.enabled = false ",
// 			args: args{
// 				r: &AuctionRequest{
// 					BidRequestWrapper: &openrtb_ext.RequestWrapper{
// 						BidRequest: &openrtb2.BidRequest{
// 							ID: "some-request-id",
// 							Imp: []openrtb2.Imp{{
// 								ID:          "some-impression-id-1",
// 								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
// 								BidFloor:    20.01,
// 								BidFloorCur: "USD",
// 							}},
// 							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":false,"skipped":false}}}`),
// 						},
// 					},
// 					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
// 				},
// 				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 					"pubmatic": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-1",
// 									Price:  1.2,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 					"appnexus": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-11",
// 									Price:  0.5,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "3",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 				},
// 				floor: config.PriceFloors{
// 					Enabled: true,
// 				},
// 				conversions:        convert{},
// 				responseDebugAllow: true,
// 			},
// 			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 				"pubmatic": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:     "some-bid-1",
// 								Price:  1.2,
// 								ImpID:  "some-impression-id-1",
// 								DealID: "1",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 				"appnexus": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:     "some-bid-11",
// 								Price:  0.5,
// 								ImpID:  "some-impression-id-1",
// 								DealID: "3",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 			},
// 			want1: nil,
// 		},
// 		{
// 			name: "Should not enforce floors when req.ext.prebid.floors.enforcement.enforcepbs = false ",
// 			args: args{
// 				r: &AuctionRequest{
// 					BidRequestWrapper: &openrtb_ext.RequestWrapper{
// 						BidRequest: &openrtb2.BidRequest{
// 							ID: "some-request-id",
// 							Imp: []openrtb2.Imp{{
// 								ID:          "some-impression-id-1",
// 								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
// 								BidFloor:    20.01,
// 								BidFloorCur: "USD",
// 							}},
// 							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":false,"floordeals":true},"enabled":true,"skipped":false}}}`),
// 						},
// 					},
// 					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
// 				},
// 				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 					"pubmatic": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-1",
// 									Price:  1.2,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 					"appnexus": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:     "some-bid-11",
// 									Price:  0.5,
// 									ImpID:  "some-impression-id-1",
// 									DealID: "3",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 				},
// 				floor: config.PriceFloors{
// 					Enabled: true,
// 				},
// 				conversions:        convert{},
// 				responseDebugAllow: true,
// 			},
// 			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 				"pubmatic": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:     "some-bid-1",
// 								Price:  1.2,
// 								ImpID:  "some-impression-id-1",
// 								DealID: "1",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 				"appnexus": {
// 					Bids: []*entities.PbsOrtbBid{
// 						{
// 							Bid: &openrtb2.Bid{
// 								ID:     "some-bid-11",
// 								Price:  0.5,
// 								ImpID:  "some-impression-id-1",
// 								DealID: "3",
// 							},
// 						},
// 					},
// 					Currency: "USD",
// 				},
// 			},
// 			want1: nil,
// 		},
// 		{
// 			name: "Should enforce floors as req.ext.prebid.floors not provided and imp.bidfloor provided",
// 			args: args{
// 				r: &AuctionRequest{
// 					BidRequestWrapper: &openrtb_ext.RequestWrapper{
// 						BidRequest: &openrtb2.BidRequest{
// 							ID: "some-request-id",
// 							Imp: []openrtb2.Imp{{
// 								ID:          "some-impression-id-1",
// 								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
// 								BidFloor:    20.01,
// 								BidFloorCur: "USD",
// 							}},
// 						},
// 					},
// 					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
// 				},
// 				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 					"pubmatic": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:    "some-bid-1",
// 									Price: 1.2,
// 									ImpID: "some-impression-id-1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 					"appnexus": {
// 						Bids: []*entities.PbsOrtbBid{
// 							{
// 								Bid: &openrtb2.Bid{
// 									ID:    "some-bid-11",
// 									Price: 0.5,
// 									ImpID: "some-impression-id-1",
// 								},
// 							},
// 						},
// 						Currency: "USD",
// 					},
// 				},
// 				floor: config.PriceFloors{
// 					Enabled: true,
// 				},
// 				conversions:        convert{},
// 				responseDebugAllow: true,
// 			},
// 			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
// 				"pubmatic": {
// 					Bids:     []*entities.PbsOrtbBid{},
// 					Currency: "USD",
// 				},
// 				"appnexus": {
// 					Bids:     []*entities.PbsOrtbBid{},
// 					Currency: "USD",
// 				},
// 			},
// 			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder appnexus", "bid rejected [bid ID: some-bid-1] reason: bid price value 1.2000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder pubmatic"},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			seatbid, errs, _ := EnforceFloors(tt.args.r, tt.args.seatBids, tt.args.conversions)
// 			for biderName, seat := range seatbid {
// 				if len(seat.Bids) != len(tt.want[biderName].Bids) {
// 					t.Errorf("enforceFloors() got = %v bids, want %v bids for BidderCode = %v ", len(seat.Bids), len(tt.want[biderName].Bids), biderName)
// 				}
// 			}
// 			assert.Equal(t, tt.want1, ErrToString(errs))
// 		})
// 	}
// }

func TestEnforceFloors(t *testing.T) {
	type args struct {
		bidRequestWrapper *openrtb_ext.RequestWrapper
		bidRequest        *openrtb2.BidRequest
		seatBids          map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		priceFloorsCfg    config.AccountPriceFloors
		conversions       currency.Conversions
	}
	tests := []struct {
		name  string
		args  args
		want  map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		want1 []error
		want2 []RejectedBid
	}{
		{
			name: "Should enforce floors for deals, ext.prebid.floors.enforcement.floorDeals=true and floors enabled = true",
			args: args{
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "some-request-id",
						Imp: []openrtb2.Imp{{
							ID:          "some-impression-id-1",
							Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							BidFloor:    20.01,
							BidFloorCur: "USD",
						}},
						Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true,"skipped":false}}}`),
					},
				},
				seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:     "some-bid-1",
									Price:  1.2,
									ImpID:  "some-impression-id-1",
									DealID: "1",
								},
							},
						},
						Currency: "USD",
					},
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID:     "some-bid-11",
									Price:  0.5,
									ImpID:  "some-impression-id-1",
									DealID: "3",
								},
							},
						},
						Currency: "USD",
					},
				},
				conversions:    convert{},
				priceFloorsCfg: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true},
			},
			want: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"pubmatic": {
					Bids:     []*entities.PbsOrtbBid{},
					Currency: "USD",
				},
				"appnexus": {
					Bids:     []*entities.PbsOrtbBid{},
					Currency: "USD",
				},
			},
			want1: []error{errors.New("bid rejected [bid ID: some-bid-1] reason: bid price value 1.2000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder pubmatic"),
				errors.New("bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder appnexus"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, _ := EnforceFloors(tt.args.bidRequestWrapper, tt.args.bidRequestWrapper.BidRequest, tt.args.seatBids, tt.args.priceFloorsCfg, tt.args.conversions)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EnforceFloors() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("EnforceFloors() got1 = %v, want %v", got1, tt.want1)
			}

			// if !reflect.DeepEqual(got2, tt.want2) {
			// 	t.Errorf("EnforceFloors() got2 = %v, want %v", got2, tt.want2)
			// }
		})
	}
}
