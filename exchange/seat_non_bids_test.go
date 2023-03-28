package exchange

import (
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestSeatNonBidsAdd(t *testing.T) {
	type fields struct {
		seatNonBidsMap map[string][]openrtb_ext.NonBid
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
		want   map[string][]openrtb_ext.NonBid
	}{
		{
			name:   "nil seatNonBidsMap",
			fields: fields{seatNonBidsMap: nil},
			args:   args{},
			want:   nil,
		},
		{
			name:   "nil seatNonBidsMap with bid object",
			fields: fields{seatNonBidsMap: nil},
			args:   args{bid: &entities.PbsOrtbBid{Bid: &openrtb2.Bid{}}, seat: "bidder1"},
			want:   sampleSeatNonBidMap("bidder1", 1),
		},
		{
			name:   "multiple nonbids for same seat",
			fields: fields{seatNonBidsMap: sampleSeatNonBidMap("bidder2", 1)},
			args:   args{bid: &entities.PbsOrtbBid{Bid: &openrtb2.Bid{}}, seat: "bidder2"},
			want:   sampleSeatNonBidMap("bidder2", 2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snb := &seatNonBids{
				seatNonBidsMap: tt.fields.seatNonBidsMap,
			}
			snb.add(tt.args.bid, tt.args.nonBidReason, tt.args.seat)
			assert.Equalf(t, tt.want, snb.seatNonBidsMap, "expected seatNonBidsMap not nil")
		})
	}
}

func TestSeatNonBidsGet(t *testing.T) {
	type fields struct {
		seatNonBidsMap map[string][]openrtb_ext.NonBid
	}
	tests := []struct {
		name   string
		fields fields
		want   []openrtb_ext.SeatNonBid
	}{
		{
			name:   "get seat bids",
			fields: fields{sampleSeatNonBidMap("bidder1", 2)},
			want:   sampleSeatBids("bidder1", 2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snb := &seatNonBids{
				seatNonBidsMap: tt.fields.seatNonBidsMap,
			}
			if got := snb.get(); !assert.Equal(t, tt.want, got) {
				t.Errorf("seatNonBids.get() = %v, want %v", got, tt.want)
			}
		})
	}
}

var sampleSeatNonBidMap = func(seat string, nonBidCount int) map[string][]openrtb_ext.NonBid {
	nonBids := make([]openrtb_ext.NonBid, 0)
	for i := 0; i < nonBidCount; i++ {
		nonBids = append(nonBids, openrtb_ext.NonBid{
			Ext: openrtb_ext.NonBidExt{Prebid: openrtb_ext.Prebid{Bid: openrtb2.Bid{}}},
		})
	}
	return map[string][]openrtb_ext.NonBid{
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
			Ext: openrtb_ext.NonBidExt{Prebid: openrtb_ext.Prebid{Bid: openrtb2.Bid{}}},
		})
	}
	seatNonBids = append(seatNonBids, seatNonBid)
	return seatNonBids
}
