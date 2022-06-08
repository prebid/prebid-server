package exchange

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/currency"
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

func TestEnforceFloorToBids(t *testing.T) {

	type args struct {
		bidRequest        *openrtb2.BidRequest
		seatBids          map[openrtb_ext.BidderName]*pbsOrtbSeatBid
		conversions       currency.Conversions
		enforceDealFloors bool
	}
	tests := []struct {
		name  string
		args  args
		want  map[openrtb_ext.BidderName]*pbsOrtbSeatBid
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
				seatBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
					"pubmatic": {
						bids: []*pbsOrtbBid{
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-1",
									Price: 1.2,
									ImpID: "some-impression-id-1",
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-2",
									Price: 1.5,
									ImpID: "some-impression-id-2",
								},
							},
						},
						currency: "USD",
					},
					"appnexus": {
						bids: []*pbsOrtbBid{
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-11",
									Price: 0.5,
									ImpID: "some-impression-id-1",
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-12",
									Price: 2.2,
									ImpID: "some-impression-id-2",
								},
							},
						},
						currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"pubmatic": {
					bids: []*pbsOrtbBid{
						{
							bid: &openrtb2.Bid{
								ID:    "some-bid-1",
								Price: 1.2,
								ImpID: "some-impression-id-1",
							},
						},
					},
					currency: "USD",
				},
				"appnexus": {
					bids: []*pbsOrtbBid{
						{
							bid: &openrtb2.Bid{
								ID:    "some-bid-12",
								Price: 2.2,
								ImpID: "some-impression-id-2",
							},
						},
					},
					currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.500000 is less than bidFloor value 1.010000 for impression id some-impression-id-1 bidder appnexus", "bid rejected [bid ID: some-bid-2] reason: bid price value 1.500000 is less than bidFloor value 2.010000 for impression id some-impression-id-2 bidder pubmatic"},
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
				seatBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
					"pubmatic": {
						bids: []*pbsOrtbBid{
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-1",
									Price: 1.2,
									ImpID: "some-impression-id-1",
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-2",
									Price: 1.5,
									ImpID: "some-impression-id-2",
								},
							},
						},
						currency: "USD",
					},
					"appnexus": {
						bids: []*pbsOrtbBid{
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-11",
									Price: 0.5,
									ImpID: "some-impression-id-1",
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-12",
									Price: 2.2,
									ImpID: "some-impression-id-2",
								},
							},
						},
						currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"pubmatic": {
					bids: []*pbsOrtbBid{
						{
							bid: &openrtb2.Bid{
								ID:    "some-bid-1",
								Price: 1.2,
								ImpID: "some-impression-id-1",
							},
						},
						{
							bid: &openrtb2.Bid{
								ID:    "some-bid-2",
								Price: 1.5,
								ImpID: "some-impression-id-2",
							},
						},
					},
					currency: "USD",
				},
				"appnexus": {
					bids: []*pbsOrtbBid{
						{
							bid: &openrtb2.Bid{
								ID:    "some-bid-12",
								Price: 2.2,
								ImpID: "some-impression-id-2",
							},
						},
					},
					currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 38.795000 is less than bidFloor value 60.000000 for impression id some-impression-id-1 bidder appnexus"},
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
				seatBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
					"pubmatic": {
						bids: []*pbsOrtbBid{
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-1",
									Price: 1.2,
									ImpID: "some-impression-id-1",
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-2",
									Price: 1.5,
									ImpID: "some-impression-id-2",
								},
							},
						},
						currency: "USD",
					},
					"appnexus": {
						bids: []*pbsOrtbBid{
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-11",
									Price: 0.5,
									ImpID: "some-impression-id-1",
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-12",
									Price: 2.2,
									ImpID: "some-impression-id-2",
								},
							},
						},
						currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"pubmatic": {
					bids: []*pbsOrtbBid{
						{
							bid: &openrtb2.Bid{
								ID:    "some-bid-1",
								Price: 1.2,
								ImpID: "some-impression-id-1",
							},
						},
						{
							bid: &openrtb2.Bid{
								ID:    "some-bid-2",
								Price: 1.5,
								ImpID: "some-impression-id-2",
							},
						},
					},
					currency: "USD",
				},
				"appnexus": {
					bids: []*pbsOrtbBid{
						{
							bid: &openrtb2.Bid{
								ID:    "some-bid-12",
								Price: 2.2,
								ImpID: "some-impression-id-2",
							},
						},
					},
					currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 38.795000 is less than bidFloor value 60.000000 for impression id some-impression-id-1 bidder appnexus"},
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
				seatBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
					"pubmatic": {
						bids: []*pbsOrtbBid{
							{
								bid: &openrtb2.Bid{
									ID:     "some-bid-1",
									Price:  1.2,
									ImpID:  "some-impression-id-1",
									DealID: "1",
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:     "some-bid-2",
									Price:  1.5,
									ImpID:  "some-impression-id-2",
									DealID: "2",
								},
							},
						},
						currency: "USD",
					},
					"appnexus": {
						bids: []*pbsOrtbBid{
							{
								bid: &openrtb2.Bid{
									ID:     "some-bid-11",
									Price:  0.5,
									ImpID:  "some-impression-id-1",
									DealID: "3",
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:     "some-bid-12",
									Price:  2.2,
									ImpID:  "some-impression-id-2",
									DealID: "4",
								},
							},
						},
						currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"pubmatic": {
					bids: []*pbsOrtbBid{
						{
							bid: &openrtb2.Bid{
								ID:     "some-bid-1",
								Price:  1.2,
								ImpID:  "some-impression-id-1",
								DealID: "1",
							},
						},
						{
							bid: &openrtb2.Bid{
								ID:     "some-bid-2",
								Price:  1.5,
								ImpID:  "some-impression-id-2",
								DealID: "2",
							},
						},
					},
					currency: "USD",
				},
				"appnexus": {
					bids: []*pbsOrtbBid{
						{
							bid: &openrtb2.Bid{
								ID:     "some-bid-12",
								Price:  2.2,
								ImpID:  "some-impression-id-2",
								DealID: "4",
							},
						},
					},
					currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 38.795000 is less than bidFloor value 60.000000 for impression id some-impression-id-1 bidder appnexus"},
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
				seatBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
					"pubmatic": {
						bids: []*pbsOrtbBid{
							{
								bid: &openrtb2.Bid{
									ID:     "some-bid-1",
									Price:  1.2,
									ImpID:  "some-impression-id-1",
									DealID: "1",
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:     "some-bid-2",
									Price:  1.5,
									ImpID:  "some-impression-id-2",
									DealID: "2",
								},
							},
						},
						currency: "USD",
					},
					"appnexus": {
						bids: []*pbsOrtbBid{
							{
								bid: &openrtb2.Bid{
									ID:     "some-bid-11",
									Price:  0.5,
									ImpID:  "some-impression-id-1",
									DealID: "3",
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:     "some-bid-12",
									Price:  2.2,
									ImpID:  "some-impression-id-2",
									DealID: "4",
								},
							},
						},
						currency: "USD",
					},
				},
				conversions:       currency.Conversions(convert{}),
				enforceDealFloors: false,
			},
			want: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"pubmatic": {
					bids: []*pbsOrtbBid{
						{
							bid: &openrtb2.Bid{
								ID:     "some-bid-1",
								Price:  1.2,
								ImpID:  "some-impression-id-1",
								DealID: "1",
							},
						},
						{
							bid: &openrtb2.Bid{
								ID:     "some-bid-2",
								Price:  1.5,
								ImpID:  "some-impression-id-2",
								DealID: "2",
							},
						},
					},
					currency: "USD",
				},
				"appnexus": {
					bids: []*pbsOrtbBid{
						{
							bid: &openrtb2.Bid{
								ID:     "some-bid-11",
								Price:  0.5,
								ImpID:  "some-impression-id-1",
								DealID: "3",
							},
						},
						{
							bid: &openrtb2.Bid{
								ID:     "some-bid-12",
								Price:  2.2,
								ImpID:  "some-impression-id-2",
								DealID: "4",
							},
						},
					},
					currency: "USD",
				},
			},
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := EnforceFloorToBids(tt.args.bidRequest, tt.args.seatBids, tt.args.conversions, tt.args.enforceDealFloors)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EnforceFloorToBids() got = %v, want %v", got, tt.want)
			}
			sort.Strings(got1)
			assert.Equal(t, tt.want1, got1)
		})
	}
}
