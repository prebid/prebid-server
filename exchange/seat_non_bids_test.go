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
		name         string
		builder      SeatNonBidBuilder
		impIds       []string
		nonBidReason NonBidReason
		seat         string
		expected     SeatNonBidBuilder
	}{
		{
			name:         "impIds nil",
			builder:      SeatNonBidBuilder{},
			impIds:       nil,
			nonBidReason: NonBidReason(1),
			seat:         "seat1",
			expected:     SeatNonBidBuilder{},
		},
		{
			name:         "empty impIds",
			builder:      SeatNonBidBuilder{},
			impIds:       []string{},
			nonBidReason: NonBidReason(1),
			seat:         "seat1",
			expected:     SeatNonBidBuilder{},
		},
		{
			name:         "single impId",
			builder:      SeatNonBidBuilder{},
			impIds:       []string{"imp1"},
			nonBidReason: ErrorGeneral,
			seat:         "seat1",
			expected: SeatNonBidBuilder{
				"seat1": {openrtb_ext.NonBid{ImpId: "imp1", StatusCode: int(ErrorGeneral)}},
			},
		},
		{
			name:         "multiple impIds",
			builder:      SeatNonBidBuilder{},
			impIds:       []string{"imp1", "imp2"},
			nonBidReason: ErrorGeneral,
			seat:         "seat1",
			expected: SeatNonBidBuilder{
				"seat1": {
					openrtb_ext.NonBid{ImpId: "imp1", StatusCode: int(ErrorGeneral)},
					openrtb_ext.NonBid{ImpId: "imp2", StatusCode: int(ErrorGeneral)},
				},
			},
		},
		{
			name: "append to existing seat",
			builder: SeatNonBidBuilder{
				"seat1": {openrtb_ext.NonBid{ImpId: "imp1", StatusCode: int(ErrorTimeout)}},
			},
			impIds:       []string{"imp2"},
			nonBidReason: ErrorGeneral,
			seat:         "seat1",
			expected: SeatNonBidBuilder{
				"seat1": {
					openrtb_ext.NonBid{ImpId: "imp1", StatusCode: int(ErrorTimeout)},
					openrtb_ext.NonBid{ImpId: "imp2", StatusCode: int(ErrorGeneral)},
				},
			},
		},
		{
			name: "append to new seat",
			builder: SeatNonBidBuilder{
				"seat1": {openrtb_ext.NonBid{ImpId: "imp1", StatusCode: int(ErrorTimeout)}},
			},
			impIds:       []string{"imp2"},
			nonBidReason: ErrorGeneral,
			seat:         "seat2",
			expected: SeatNonBidBuilder{
				"seat1": {
					openrtb_ext.NonBid{ImpId: "imp1", StatusCode: int(ErrorTimeout)},
				},
				"seat2": {
					openrtb_ext.NonBid{ImpId: "imp2", StatusCode: int(ErrorGeneral)},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.builder.rejectImps(tt.impIds, tt.nonBidReason, tt.seat)
			assert.Equal(t, len(tt.builder), len(tt.expected))
			for seat := range tt.expected {
				assert.ElementsMatch(t, tt.expected[seat], tt.builder[seat])
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
