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

func TestAppend(t *testing.T) {
	tests := []struct {
		name     string
		builder  SeatNonBidBuilder
		toAppend []SeatNonBidBuilder
		expected SeatNonBidBuilder
	}{
		{
			name:     "nil receiver",
			builder:  nil,
			toAppend: []SeatNonBidBuilder{{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}}},
			expected: nil,
		},
		{
			name:     "empty builder",
			builder:  SeatNonBidBuilder{},
			toAppend: []SeatNonBidBuilder{{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}}},
			expected: SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}},
		},
		{
			name:     "multiple seats",
			builder:  SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}},
			toAppend: []SeatNonBidBuilder{{"seat2": []openrtb_ext.NonBid{{ImpId: "imp2"}}}},
			expected: SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}, "seat2": []openrtb_ext.NonBid{{ImpId: "imp2"}}},
		},
		{
			name:     "multiple appends",
			builder:  SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}},
			toAppend: []SeatNonBidBuilder{{"seat2": []openrtb_ext.NonBid{{ImpId: "imp2"}}}, {"seat3": []openrtb_ext.NonBid{{ImpId: "imp3"}}}},
			expected: SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}, "seat2": []openrtb_ext.NonBid{{ImpId: "imp2"}}, "seat3": []openrtb_ext.NonBid{{ImpId: "imp3"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.builder.append(tt.toAppend...)
			assert.Equal(t, tt.expected, tt.builder)
		})
	}
}

func TestRejectImps(t *testing.T) {
	tests := []struct {
		name    string
		impIDs  []string
		builder SeatNonBidBuilder
		want    SeatNonBidBuilder
	}{
		{
			name:    "nil_imps",
			impIDs:  nil,
			builder: SeatNonBidBuilder{},
			want:    SeatNonBidBuilder{},
		},
		{
			name:    "empty_imps",
			impIDs:  []string{},
			builder: SeatNonBidBuilder{},
			want:    SeatNonBidBuilder{},
		},
		{
			name:    "one_imp",
			impIDs:  []string{"imp1"},
			builder: SeatNonBidBuilder{},
			want: SeatNonBidBuilder{
				"seat1": []openrtb_ext.NonBid{
					{
						ImpId:      "imp1",
						StatusCode: 300,
					},
				},
			},
		},
		{
			name:    "many_imps",
			impIDs:  []string{"imp1", "imp2"},
			builder: SeatNonBidBuilder{},
			want: SeatNonBidBuilder{
				"seat1": []openrtb_ext.NonBid{
					{
						ImpId:      "imp1",
						StatusCode: 300,
					},
					{
						ImpId:      "imp2",
						StatusCode: 300,
					},
				},
			},
		},
		{
			name:   "many_imps_appended_to_prepopulated_list",
			impIDs: []string{"imp1", "imp2"},
			builder: SeatNonBidBuilder{
				"seat0": []openrtb_ext.NonBid{
					{
						ImpId:      "imp0",
						StatusCode: 0,
					},
				},
			},
			want: SeatNonBidBuilder{
				"seat0": []openrtb_ext.NonBid{
					{
						ImpId:      "imp0",
						StatusCode: 0,
					},
				},
				"seat1": []openrtb_ext.NonBid{
					{
						ImpId:      "imp1",
						StatusCode: 300,
					},
					{
						ImpId:      "imp2",
						StatusCode: 300,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.builder.rejectImps(test.impIDs, 300, "seat1")

			assert.Equal(t, len(test.builder), len(test.want))
			for seat := range test.want {
				assert.ElementsMatch(t, test.want[seat], test.builder[seat])
			}
		})
	}
}

func TestSlice(t *testing.T) {
	tests := []struct {
		name    string
		builder SeatNonBidBuilder
		want    []openrtb_ext.SeatNonBid
	}{
		{
			name:    "nil",
			builder: nil,
			want:    []openrtb_ext.SeatNonBid{},
		},
		{
			name:    "empty",
			builder: SeatNonBidBuilder{},
			want:    []openrtb_ext.SeatNonBid{},
		},
		{
			name: "one_no_nonbids",
			builder: SeatNonBidBuilder{
				"a": []openrtb_ext.NonBid{},
			},
			want: []openrtb_ext.SeatNonBid{
				{
					NonBid: []openrtb_ext.NonBid{},
					Seat:   "a",
				},
			},
		},
		{
			name: "one_with_nonbids",
			builder: SeatNonBidBuilder{
				"a": []openrtb_ext.NonBid{
					{
						ImpId:      "imp1",
						StatusCode: 100,
					},
					{
						ImpId:      "imp2",
						StatusCode: 200,
					},
				},
			},
			want: []openrtb_ext.SeatNonBid{
				{
					NonBid: []openrtb_ext.NonBid{
						{
							ImpId:      "imp1",
							StatusCode: 100,
						},
						{
							ImpId:      "imp2",
							StatusCode: 200,
						},
					},
					Seat: "a",
				},
			},
		},
		{
			name: "many_no_nonbids",
			builder: SeatNonBidBuilder{
				"a": []openrtb_ext.NonBid{},
				"b": []openrtb_ext.NonBid{},
				"c": []openrtb_ext.NonBid{},
			},
			want: []openrtb_ext.SeatNonBid{
				{
					NonBid: []openrtb_ext.NonBid{},
					Seat:   "a",
				},
				{
					NonBid: []openrtb_ext.NonBid{},
					Seat:   "b",
				},
				{
					NonBid: []openrtb_ext.NonBid{},
					Seat:   "c",
				},
			},
		},
		{
			name: "many_with_nonbids",
			builder: SeatNonBidBuilder{
				"a": []openrtb_ext.NonBid{
					{
						ImpId:      "imp1",
						StatusCode: 100,
					},
					{
						ImpId:      "imp2",
						StatusCode: 200,
					},
				},
				"b": []openrtb_ext.NonBid{
					{
						ImpId:      "imp3",
						StatusCode: 300,
					},
				},
				"c": []openrtb_ext.NonBid{
					{
						ImpId:      "imp4",
						StatusCode: 400,
					},
					{
						ImpId:      "imp5",
						StatusCode: 500,
					},
				},
			},
			want: []openrtb_ext.SeatNonBid{
				{
					NonBid: []openrtb_ext.NonBid{
						{
							ImpId:      "imp1",
							StatusCode: 100,
						},
						{
							ImpId:      "imp2",
							StatusCode: 200,
						},
					},
					Seat: "a",
				},
				{
					NonBid: []openrtb_ext.NonBid{
						{
							ImpId:      "imp3",
							StatusCode: 300,
						},
					},
					Seat: "b",
				},
				{
					NonBid: []openrtb_ext.NonBid{
						{
							ImpId:      "imp4",
							StatusCode: 400,
						},
						{
							ImpId:      "imp5",
							StatusCode: 500,
						},
					},
					Seat: "c",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.builder.Slice()
			assert.ElementsMatch(t, test.want, result)
		})
	}
}
