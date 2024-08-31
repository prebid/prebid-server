package exchange

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/exchange/entities"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestRejectBid(t *testing.T) {
	type fields struct {
		seatNonBidsMap SeatNonBidBuilder
	}
	type args struct {
		bid          *entities.PbsOrtbBid
		nonBidReason int
		seat         string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   SeatNonBidBuilder
	}{
		{
			name:   "nil-seatNonBidsMap",
			fields: fields{seatNonBidsMap: nil},
			args:   args{},
			want:   nil,
		},
		{
			name:   "nil-seatNonBidsMap-with-bid-object",
			fields: fields{seatNonBidsMap: nil},
			args:   args{bid: &entities.PbsOrtbBid{Bid: &openrtb2.Bid{}}, seat: "bidder1"},
			want:   nil,
		},
		{
			name:   "multiple-nonbids-for-same-seat",
			fields: fields{seatNonBidsMap: sampleSeatNonBidMap("bidder2", 1)},
			args:   args{bid: &entities.PbsOrtbBid{Bid: &openrtb2.Bid{}}, seat: "bidder2"},
			want:   sampleSeatNonBidMap("bidder2", 2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snb := tt.fields.seatNonBidsMap
			snb.rejectBid(tt.args.bid, tt.args.nonBidReason, tt.args.seat)
			assert.Equalf(t, tt.want, snb, "expected seatNonBidsMap not nil")
		})
	}
}

func TestSeatNonBidsSlice(t *testing.T) {
	type fields struct {
		snb SeatNonBidBuilder
	}
	tests := []struct {
		name   string
		fields fields
		want   []openrtb_ext.SeatNonBid
	}{
		{
			name:   "get-seat-nonbids",
			fields: fields{sampleSeatNonBidMap("bidder1", 2)},
			want:   sampleSeatBids("bidder1", 2),
		},
		{
			name:   "nil-seat-nonbids",
			fields: fields{snb: SeatNonBidBuilder{}},
			want:   []openrtb_ext.SeatNonBid{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fields.snb.Slice(); !assert.Equal(t, tt.want, got) {
				t.Errorf("seatNonBids.get() = %v, want %v", got, tt.want)
			}
		})
	}
}

var sampleSeatNonBidMap = func(seat string, nonBidCount int) SeatNonBidBuilder {
	nonBids := make([]openrtb_ext.NonBid, 0)
	for i := 0; i < nonBidCount; i++ {
		nonBids = append(nonBids, openrtb_ext.NonBid{
			Ext: &openrtb_ext.NonBidExt{Prebid: openrtb_ext.ExtResponseNonBidPrebid{Bid: openrtb_ext.NonBidObject{}}},
		})
	}
	return SeatNonBidBuilder{
		seat: nonBids,
	}
}

var sampleSeatBids = func(seat string, nonBidCount int) []openrtb_ext.SeatNonBid {
	seatNonBids := make([]openrtb_ext.SeatNonBid, 0)
	seatNonBid := openrtb_ext.SeatNonBid{
		Seat:   seat,
		NonBid: make([]openrtb_ext.NonBid, 0),
	}
	for i := 0; i < nonBidCount; i++ {
		seatNonBid.NonBid = append(seatNonBid.NonBid, openrtb_ext.NonBid{
			Ext: &openrtb_ext.NonBidExt{Prebid: openrtb_ext.ExtResponseNonBidPrebid{Bid: openrtb_ext.NonBidObject{}}},
		})
	}
	seatNonBids = append(seatNonBids, seatNonBid)
	return seatNonBids
}
