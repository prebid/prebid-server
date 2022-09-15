package exchange

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/config"
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

func ErrToString(Err []error) []string {
	var errString []string
	for _, eachErr := range Err {
		errString = append(errString, eachErr.Error())
	}
	sort.Strings(errString)
	return errString
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
							{
								bid: &openrtb2.Bid{
									ID:    "some-bid-3",
									Price: 1.4,
									ImpID: "some-impression-id-3",
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
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 1.0100 USD for impression id some-impression-id-1 bidder appnexus", "bid rejected [bid ID: some-bid-2] reason: bid price value 1.5000 USD is less than bidFloor value 2.0100 USD for impression id some-impression-id-2 bidder pubmatic"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := enforceFloorToBids(tt.args.bidRequest, tt.args.seatBids, tt.args.conversions, tt.args.enforceDealFloors)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("enforceFloorToBids() got = %v, want %v", got, tt.want)
			}
			assert.Equal(t, tt.want1, ErrToString(got1))
		})
	}
}

func TestEnforceFloorToBidsConversion(t *testing.T) {

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
						currency: "EUR",
					},
				},
				conversions:       convert{},
				enforceDealFloors: true,
			},
			want: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"pubmatic": {
					bids:     []*pbsOrtbBid{},
					currency: "EUR",
				},
			},
			want1: []string{"Error in rate conversion from = EUR to USD with bidder pubmatic for impression id some-impression-id-1 and bid id some-bid-1", "Error in rate conversion from = EUR to USD with bidder pubmatic for impression id some-impression-id-2 and bid id some-bid-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := enforceFloorToBids(tt.args.bidRequest, tt.args.seatBids, tt.args.conversions, tt.args.enforceDealFloors)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, ErrToString(got1))
		})
	}
}

func TestSelectFloorsAndModifyImp(t *testing.T) {
	type args struct {
		r                  *AuctionRequest
		floor              config.PriceFloors
		conversions        currency.Conversions
		responseDebugAllow bool
	}
	tests := []struct {
		name string
		args args
		want []error
	}{
		{
			name: "Should Signal Floors",
			args: args{
				r: func() *AuctionRequest {
					var wrapper openrtb_ext.RequestWrapper
					strReq := `{"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"id":"Some-imp-1","banner":{"format":[{"w":300,"h":250}],"pos":7,"api":[5,6,7]},"displaymanager":"PubMatic_OpenBid_SDK","displaymanagerver":"1.4.0","instl":1,"tagid":"/1234/DMDemo","bidfloor":100,"bidfloorcur":"USD","secure":0,"ext":{"appnexus-1":{"placementId":234234},"appnexus-2":{"placementId":9880618},"pubmatic":{"adSlot":"/1234/DMDemo@300x250","publisherId":"123","wiid":"e643368f-06fe-4493-86a8-36ae2f13286a","wrapper":{"version":1,"profile":123}}}}],"app":{"name":"OpenWrapperSample","bundle":"com.pubmatic.openbid.app","domain":"www.website1.com","storeurl":"https://myurl.com","ver":"1.0","publisher":{"id":"123"}},"device":{"ua":"Mozilla/5.0 (Linux; Android 9; Android SDK built for x86 Build/PSR1.180720.075; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/69.0.3497.100 Mobile Safari/537.36","geo":{"lat":37.421998333333335,"lon":-122.08400000000002,"type":1},"lmt":0,"ip":"192.1.1.1","devicetype":4,"make":"Google","model":"Android SDK built for x86","os":"Android","osv":"9","h":1794,"w":1080,"pxratio":2.625,"js":1,"language":"en","carrier":"Android","mccmnc":"310-260","connectiontype":6,"ifa":"07c387f2-e030-428f-8336-42f682150759"},"user":{},"at":1,"tmax":1891995,"cur":["USD"],"source":{"tid":"95d6643c-3da6-40a2-b9ca-12279393ffbf","ext":{"omidpn":"","omidpv":""}},"ext":{"prebid":{"aliases":{"adg":"adgeneration","andbeyond":"adkernel","appnexus-1":"appnexus","appnexus-2":"appnexus","districtm":"appnexus","districtmDMX":"dmx","pubmatic2":"pubmatic"},"channel":{"name":"app","version":""},"debug":true,"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"includewinners":true,"includebidderkeys":true,"includebrandcategory":null,"includeformat":false,"durationrangesec":null,"preferdeals":false},"bidderparams":{"pubmatic":{"wiid":"e643368f-06fe-4493-86a8-36ae2f13286a"}},"floors":{"floormin":1,"data":{"currency":"USD","skiprate":0,"modelgroups":[{"modelweight":40,"modelversion":"version1","skiprate":0,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":17.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01,"*|300x600|*":13.01,"*|300x600|www.website1.com":12.01,"*|728x90|*":15.01,"*|728x90|www.website1.com":14.01,"banner|*|*":90.01,"banner|*|www.website1.com":80.01,"banner|300x250|*":30.01,"banner|300x250|www.website1.com":20.01,"banner|300x600|*":50.01,"banner|300x600|www.website1.com":40.01,"banner|728x90|*":70.01,"banner|728x90|www.website1.com":60.01},"default":21},{"modelweight":40,"modelversion":"version2","skiprate":0,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":17.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01,"*|300x600|*":13.01,"*|300x600|www.website1.com":12.01,"*|728x90|*":15.01,"*|728x90|www.website1.com":14.01,"banner|*|*":90.01,"banner|*|www.website1.com":80.01,"banner|300x250|*":30.01,"banner|300x250|www.website1.com":20.01,"banner|300x600|*":50.01,"banner|300x600|www.website1.com":40.01,"banner|728x90|*":70.01,"banner|728x90|www.website1.com":60.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true}}}}`
					_ = json.Unmarshal([]byte(strReq), &wrapper)
					ar := AuctionRequest{BidRequestWrapper: &wrapper}
					return &ar
				}(),
				floor: config.PriceFloors{
					Enabled:           true,
					EnforceFloorsRate: 100,
					EnforceDealFloors: true,
				},
				conversions:        convert{},
				responseDebugAllow: true,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectFloorsAndModifyImp(tt.args.r, tt.args.floor, tt.args.conversions, tt.args.responseDebugAllow); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("selectFloorsAndModifyImp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnforceFloors(t *testing.T) {
	type args struct {
		r                  *AuctionRequest
		seatBids           map[openrtb_ext.BidderName]*pbsOrtbSeatBid
		floor              config.PriceFloors
		conversions        currency.Conversions
		responseDebugAllow bool
	}
	tests := []struct {
		name  string
		args  args
		want  map[openrtb_ext.BidderName]*pbsOrtbSeatBid
		want1 []string
	}{
		{
			name: "Should enforce floors",
			args: args{
				r: func() *AuctionRequest {
					var wrapper openrtb_ext.RequestWrapper
					strReq := `{"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"id":"some-impression-id-1","banner":{"format":[{"w":300,"h":250}],"pos":7,"api":[5,6,7]},"displaymanager":"PubMatic_OpenBid_SDK","displaymanagerver":"1.4.0","instl":1,"tagid":"/43743431/DMDemo","bidfloor":20.01,"bidfloorcur":"USD","secure":0,"ext":{"appnexus-1":{"placementId":234234},"appnexus-2":{"placementId":9880618},"pubmatic":{"adSlot":"/43743431/DMDemo@300x250","publisherId":"5890","wiid":"42faaac0-9134-41c2-a283-77f1302d00ac","wrapper":{"version":1,"profile":7255}},"prebid":{"floors":{"floorRule":"banner|300x250|www.website1.com","floorRuleValue":20.01}}}}],"app":{"name":"OpenWrapperSample","bundle":"com.pubmatic.openbid.app","domain":"www.website1.com","storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098?appnexus_banner_fixedbid=1&fixedbid=1","ver":"1.0","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0 (Linux; Android 9; Android SDK built for x86 Build/PSR1.180720.075; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/69.0.3497.100 Mobile Safari/537.36","geo":{"lat":37.421998333333335,"lon":-122.08400000000002,"type":1},"lmt":0,"ip":"192.0.2.1","devicetype":4,"make":"Google","model":"Android SDK built for x86","os":"Android","osv":"9","h":1794,"w":1080,"pxratio":2.625,"js":1,"language":"en","carrier":"Android","mccmnc":"310-260","connectiontype":6,"ifa":"07c387f2-e030-428f-8336-42f682150759"},"user":{},"at":1,"tmax":1891525,"cur":["USD"],"source":{"tid":"95d6643c-3da6-40a2-b9ca-12279393ffbf","ext":{"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"ext":{"prebid":{"aliases":{"adg":"adgeneration","andbeyond":"adkernel","appnexus-1":"appnexus","appnexus-2":"appnexus","districtm":"appnexus","districtmDMX":"dmx","pubmatic2":"pubmatic"},"channel":{"name":"app","version":""},"debug":true,"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"includewinners":true,"includebidderkeys":true,"includebrandcategory":null,"includeformat":false,"durationrangesec":null,"preferdeals":false},"bidderparams":{"pubmatic":{"wiid":"42faaac0-9134-41c2-a283-77f1302d00ac"}},"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelweight":40,"debugweight":75,"modelversion":"version2","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":17.01,"*|*|www.website1.com":16.01,"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01,"*|300x600|*":13.01,"*|300x600|www.website1.com":12.01,"*|728x90|*":15.01,"*|728x90|www.website1.com":14.01,"banner|*|*":90.01,"banner|*|www.website1.com":80.01,"banner|300x250|*":30.01,"banner|300x250|www.website1.com":20.01,"banner|300x600|*":50.01,"banner|300x600|www.website1.com":40.01,"banner|728x90|*":70.01,"banner|728x90|www.website1.com":60.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true,"skipped":false}}}}`
					_ = json.Unmarshal([]byte(strReq), &wrapper)
					ar := AuctionRequest{BidRequestWrapper: &wrapper}
					return &ar
				}(),
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
						},
						currency: "USD",
					},
				},
				floor: config.PriceFloors{
					Enabled:           true,
					EnforceFloorsRate: 100,
					EnforceDealFloors: true,
				},
				conversions:        convert{},
				responseDebugAllow: true,
			},
			want: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"pubmatic": {
					bids:     []*pbsOrtbBid{},
					currency: "USD",
				},
				"appnexus": {
					bids:     []*pbsOrtbBid{},
					currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder appnexus", "bid rejected [bid ID: some-bid-1] reason: bid price value 1.2000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder pubmatic"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := enforceFloors(tt.args.r, tt.args.seatBids, tt.args.floor, tt.args.conversions, tt.args.responseDebugAllow)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("enforceFloors() got = %v, want %v", got, tt.want)
			}
			assert.Equal(t, tt.want1, ErrToString(got1))
		})
	}
}
