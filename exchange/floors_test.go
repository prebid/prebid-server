package exchange

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
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
			got, got1, _ := enforceFloorToBids(tt.args.bidRequest, tt.args.seatBids, tt.args.conversions, tt.args.enforceDealFloors)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, ErrToString(got1))
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
			name: "Should enforce floors for deals, ext.prebid.floors.enforcement.floorDeals=true and floors enabled = true",
			args: args{
				r: &AuctionRequest{
					BidRequestWrapper: &openrtb_ext.RequestWrapper{
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
					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
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
					Enabled: true,
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
		{
			name: "Should not enforce floors for deals, ext.prebid.floors.enforcement.floorDeals not provided",
			args: args{
				r: &AuctionRequest{
					BidRequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID:          "some-impression-id-1",
								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
								BidFloor:    20.01,
								BidFloorCur: "USD",
							}},
							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true},"enabled":true,"skipped":false}}}`),
						},
					},
					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
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
						},
						currency: "USD",
					},
				},
				floor: config.PriceFloors{
					Enabled: true,
				},
				conversions:        convert{},
				responseDebugAllow: true,
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
					},
					currency: "USD",
				},
				"appnexus": {
					bids:     []*pbsOrtbBid{},
					currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder appnexus"},
		},
		{
			name: "Should not enforce floors for deals, ext.prebid.floors.enforcement.floorDeals=false is set",
			args: args{
				r: &AuctionRequest{
					BidRequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID:          "some-impression-id-1",
								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
								BidFloor:    20.01,
								BidFloorCur: "USD",
							}},
							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true, "floordeals":false},"enabled":true,"skipped":false}}}`),
						},
					},
					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
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
						},
						currency: "USD",
					},
				},
				floor: config.PriceFloors{
					Enabled: true,
				},
				conversions:        convert{},
				responseDebugAllow: true,
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
					},
					currency: "USD",
				},
				"appnexus": {
					bids:     []*pbsOrtbBid{},
					currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder appnexus"},
		},
		{
			name: "Should not enforce floors for deals, ext.prebid.floors.enforcement.floorDeals=true and EnforceDealFloors = false from config",
			args: args{
				r: &AuctionRequest{
					BidRequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID:          "some-impression-id-1",
								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
								BidFloor:    20.01,
								BidFloorCur: "USD",
							}},
							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true, "floordeals":true},"enabled":true,"skipped":false}}}`),
						},
					},
					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: false}},
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
						},
						currency: "USD",
					},
				},
				floor: config.PriceFloors{
					Enabled: true,
				},
				conversions:        convert{},
				responseDebugAllow: true,
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
					},
					currency: "USD",
				},
				"appnexus": {
					bids:     []*pbsOrtbBid{},
					currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 20.0100 USD for impression id some-impression-id-1 bidder appnexus"},
		},
		{
			name: "Should enforce floors when imp.bidfloor provided and req.ext.prebid not provided",
			args: args{
				r: &AuctionRequest{
					BidRequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID:          "some-impression-id-1",
								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
								BidFloor:    5.01,
								BidFloorCur: "USD",
							}},
						},
					},
					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: false}},
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
						},
						currency: "USD",
					},
				},
				floor: config.PriceFloors{
					Enabled: true,
				},
				conversions:        convert{},
				responseDebugAllow: true,
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
					},
					currency: "USD",
				},
				"appnexus": {
					bids:     []*pbsOrtbBid{},
					currency: "USD",
				},
			},
			want1: []string{"bid rejected [bid ID: some-bid-11] reason: bid price value 0.5000 USD is less than bidFloor value 5.0100 USD for impression id some-impression-id-1 bidder appnexus"},
		},
		{
			name: "Should not enforce floors when imp.bidfloor not provided and req.ext.prebid not provided",
			args: args{
				r: &AuctionRequest{
					BidRequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID:     "some-impression-id-1",
								Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
							}},
						},
					},
					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: false}},
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
						},
						currency: "USD",
					},
				},
				floor: config.PriceFloors{
					Enabled: true,
				},
				conversions:        convert{},
				responseDebugAllow: true,
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
					},
					currency: "USD",
				},
			},
			want1: nil,
		},
		{
			name: "Should not enforce floors when  config flag Enabled = false",
			args: args{
				r: &AuctionRequest{
					BidRequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID:          "some-impression-id-1",
								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
								BidFloor:    5.01,
								BidFloorCur: "USD",
							}},
						},
					},
					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: false}},
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
					Enabled: false,
				},
				conversions:        convert{},
				responseDebugAllow: true,
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
			want1: nil,
		},
		{
			name: "Should not enforce floors when req.ext.prebid.floors.enabled = false ",
			args: args{
				r: &AuctionRequest{
					BidRequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID:          "some-impression-id-1",
								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
								BidFloor:    20.01,
								BidFloorCur: "USD",
							}},
							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":false,"skipped":false}}}`),
						},
					},
					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
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
					Enabled: true,
				},
				conversions:        convert{},
				responseDebugAllow: true,
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
			want1: nil,
		},
		{
			name: "Should not enforce floors when req.ext.prebid.floors.enforcement.enforcepbs = false ",
			args: args{
				r: &AuctionRequest{
					BidRequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID:          "some-impression-id-1",
								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
								BidFloor:    20.01,
								BidFloorCur: "USD",
							}},
							Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"data":{"currency":"USD","skiprate":100,"modelgroups":[{"modelversion":"version1","skiprate":10,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20.01,"*|*|www.website1.com":16.01},"default":21}]},"enforcement":{"enforcepbs":false,"floordeals":true},"enabled":true,"skipped":false}}}`),
						},
					},
					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
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
					Enabled: true,
				},
				conversions:        convert{},
				responseDebugAllow: true,
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
			want1: nil,
		},
		{
			name: "Should enforce floors as req.ext.prebid.floors not provided and imp.bidfloor provided",
			args: args{
				r: &AuctionRequest{
					BidRequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID:          "some-impression-id-1",
								Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
								BidFloor:    20.01,
								BidFloorCur: "USD",
							}},
						},
					},
					Account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true, EnforceFloorRate: 100, EnforceDealFloors: true}},
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
						},
						currency: "USD",
					},
				},
				floor: config.PriceFloors{
					Enabled: true,
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
			seatbid, errs, _ := enforceFloors(tt.args.r, tt.args.seatBids, tt.args.floor, tt.args.conversions, tt.args.responseDebugAllow)
			for biderName, seat := range seatbid {
				if len(seat.bids) != len(tt.want[biderName].bids) {
					t.Errorf("enforceFloors() got = %v bids, want %v bids for BidderCode = %v ", len(seat.bids), len(tt.want[biderName].bids), biderName)
				}
			}
			assert.Equal(t, tt.want1, ErrToString(errs))
		})
	}
}
