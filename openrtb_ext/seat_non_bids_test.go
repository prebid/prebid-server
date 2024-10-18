package openrtb_ext

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestNewNonBid(t *testing.T) {
	tests := []struct {
		name           string
		bidParams      NonBidParams
		expectedNonBid NonBid
	}{
		{
			name:           "nil-bid-present-in-bidparams",
			bidParams:      NonBidParams{Bid: nil},
			expectedNonBid: NonBid{},
		},
		{
			name:           "non-nil-bid-present-in-bidparams",
			bidParams:      NonBidParams{Bid: &openrtb2.Bid{ImpID: "imp1"}, NonBidReason: 100},
			expectedNonBid: NonBid{ImpId: "imp1", StatusCode: 100},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nonBid := NewNonBid(tt.bidParams)
			assert.Equal(t, tt.expectedNonBid, nonBid, "found incorrect nonBid")
		})
	}
}

func TestSeatNonBidsAdd(t *testing.T) {
	type fields struct {
		seatNonBidsMap SeatNonBidBuilder
	}
	type args struct {
		nonbid NonBid
		seat   string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string][]NonBid
	}{
		{
			name:   "nil-seatNonBidsMap",
			fields: fields{seatNonBidsMap: nil},
			args: args{
				nonbid: NonBid{},
				seat:   "bidder1",
			},
			want: sampleSeatNonBidMap("bidder1", 1),
		},
		{
			name:   "non-nil-seatNonBidsMap",
			fields: fields{seatNonBidsMap: nil},
			args: args{

				nonbid: NonBid{},
				seat:   "bidder1",
			},
			want: sampleSeatNonBidMap("bidder1", 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snb := tt.fields.seatNonBidsMap
			snb.AddBid(tt.args.nonbid, tt.args.seat)
			assert.Equalf(t, tt.want, snb, "found incorrect seatNonBidsMap")
		})
	}
}

func TestSeatNonBidsGet(t *testing.T) {
	type fields struct {
		snb SeatNonBidBuilder
	}
	tests := []struct {
		name   string
		fields fields
		want   []SeatNonBid
	}{
		{
			name:   "get-seat-nonbids",
			fields: fields{sampleSeatNonBidMap("bidder1", 2)},
			want:   sampleSeatBids("bidder1", 2),
		},
		{
			name:   "nil-seat-nonbids",
			fields: fields{nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fields.snb.Get(); !assert.Equal(t, tt.want, got) {
				t.Errorf("seatNonBids.get() = %v, want %v", got, tt.want)
			}
		})
	}
}

var sampleSeatNonBidMap = func(seat string, nonBidCount int) map[string][]NonBid {
	nonBids := make([]NonBid, 0)
	for i := 0; i < nonBidCount; i++ {
		nonBids = append(nonBids, NonBid{
			Ext: &ExtNonBid{Prebid: ExtNonBidPrebid{Bid: ExtNonBidPrebidBid{}}},
		})
	}
	return map[string][]NonBid{
		seat: nonBids,
	}
}

var sampleSeatBids = func(seat string, nonBidCount int) []SeatNonBid {
	seatNonBids := make([]SeatNonBid, 0)
	seatNonBid := SeatNonBid{
		Seat:   seat,
		NonBid: make([]NonBid, 0),
	}
	for i := 0; i < nonBidCount; i++ {
		seatNonBid.NonBid = append(seatNonBid.NonBid, NonBid{
			Ext: &ExtNonBid{Prebid: ExtNonBidPrebid{Bid: ExtNonBidPrebidBid{}}},
		})
	}
	seatNonBids = append(seatNonBids, seatNonBid)
	return seatNonBids
}

func TestSeatNonBidsMerge(t *testing.T) {

	tests := []struct {
		name  string
		snb   SeatNonBidBuilder
		input SeatNonBidBuilder
		want  SeatNonBidBuilder
	}{
		{
			name: "target-SeatNonBidBuilder-is-nil",
			snb:  nil,
			want: nil,
		},
		{
			name:  "input-SeatNonBidBuilder-contains-nil-map",
			snb:   SeatNonBidBuilder{},
			input: nil,
			want:  SeatNonBidBuilder{},
		},
		{
			name:  "input-SeatNonBidBuilder-contains-empty-nonBids",
			snb:   SeatNonBidBuilder{},
			input: SeatNonBidBuilder{},
			want:  SeatNonBidBuilder{},
		},
		{
			name:  "append-nonbids-in-empty-target-SeatNonBidBuilder",
			snb:   SeatNonBidBuilder{},
			input: sampleSeatNonBidMap("pubmatic", 1),
			want:  sampleSeatNonBidMap("pubmatic", 1),
		},
		{
			name:  "merge-multiple-nonbids-in-non-empty-target-SeatNonBidBuilder",
			snb:   sampleSeatNonBidMap("pubmatic", 1),
			input: sampleSeatNonBidMap("pubmatic", 1),
			want:  sampleSeatNonBidMap("pubmatic", 2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.snb.Append(tt.input)
			assert.Equal(t, tt.want, tt.snb, "incorrect SeatNonBidBuilder generated by Append")
		})
	}
}
