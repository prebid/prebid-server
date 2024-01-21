package openrtb_ext

import (
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestSeatNonBidsAdd(t *testing.T) {
	type fields struct {
		seatNonBidsMap map[string][]NonBid
	}
	type args struct {
		nonBidParam NonBidParams
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
			args:   args{},
			want:   nil,
		},
		{
			name:   "nil-seatNonBidsMap-with-bid-object",
			fields: fields{seatNonBidsMap: nil},
			args:   args{nonBidParam: NonBidParams{Bid: &openrtb2.Bid{}, Seat: "bidder1"}},
			want:   sampleSeatNonBidMap("bidder1", 1),
		},
		{
			name:   "multiple-nonbids-for-same-seat",
			fields: fields{seatNonBidsMap: sampleSeatNonBidMap("bidder2", 1)},
			args:   args{nonBidParam: NonBidParams{Bid: &openrtb2.Bid{}, Seat: "bidder2"}},
			want:   sampleSeatNonBidMap("bidder2", 2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snb := &NonBidCollection{
				seatNonBidsMap: tt.fields.seatNonBidsMap,
			}
			snb.AddBid(tt.args.nonBidParam)
			assert.Equalf(t, tt.want, snb.seatNonBidsMap, "expected seatNonBidsMap not nil")
		})
	}
}

func TestSeatNonBidsGet(t *testing.T) {
	type fields struct {
		snb *NonBidCollection
	}
	tests := []struct {
		name   string
		fields fields
		want   []SeatNonBid
	}{
		{
			name:   "get-seat-nonbids",
			fields: fields{&NonBidCollection{sampleSeatNonBidMap("bidder1", 2)}},
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
			Ext: ExtNonBid{Prebid: ExtNonBidPrebid{Bid: ExtNonBidPrebidBid{}}},
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
			Ext: ExtNonBid{Prebid: ExtNonBidPrebid{Bid: ExtNonBidPrebidBid{}}},
		})
	}
	seatNonBids = append(seatNonBids, seatNonBid)
	return seatNonBids
}

func TestSeatNonBidsMerge(t *testing.T) {
	type target struct {
		snb *NonBidCollection
	}
	tests := []struct {
		name   string
		fields target
		input  NonBidCollection
		want   *NonBidCollection
	}{
		{
			name:   "target-NonBidCollection-is-nil",
			fields: target{nil},
			want:   nil,
		},
		{
			name:   "input-NonBidCollection-contains-nil-map",
			fields: target{&NonBidCollection{}},
			input:  NonBidCollection{seatNonBidsMap: nil},
			want:   &NonBidCollection{},
		},
		{
			name:   "input-NonBidCollection-contains-empty-nonBids",
			fields: target{&NonBidCollection{}},
			input:  NonBidCollection{seatNonBidsMap: make(map[string][]NonBid)},
			want:   &NonBidCollection{},
		},
		{
			name:   "append-nonbids-in-empty-target-NonBidCollection",
			fields: target{&NonBidCollection{}},
			input: NonBidCollection{
				seatNonBidsMap: sampleSeatNonBidMap("pubmatic", 1),
			},
			want: &NonBidCollection{
				seatNonBidsMap: sampleSeatNonBidMap("pubmatic", 1),
			},
		},
		{
			name: "merge-multiple-nonbids-in-non-empty-target-NonBidCollection",
			fields: target{&NonBidCollection{
				seatNonBidsMap: sampleSeatNonBidMap("pubmatic", 1),
			}},
			input: NonBidCollection{
				seatNonBidsMap: sampleSeatNonBidMap("pubmatic", 1),
			},
			want: &NonBidCollection{
				seatNonBidsMap: sampleSeatNonBidMap("pubmatic", 2),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.snb.Append(tt.input)
			assert.Equal(t, tt.want, tt.fields.snb, "incorrect NonBidCollection generated by Append")
		})
	}
}