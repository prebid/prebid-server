package exchange

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestRejectBid(t *testing.T) {
	type fields struct {
		builder SeatNonBidBuilder
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
			name: "nil_builder",
			fields: fields{
				builder: nil,
			},
			args: args{},
			want: nil,
		},
		{
			name: "nil_pbsortbid",
			fields: fields{
				builder: SeatNonBidBuilder{},
			},
			args: args{
				bid: nil,
			},
			want: SeatNonBidBuilder{},
		},
		{
			name: "nil_bid",
			fields: fields{
				builder: SeatNonBidBuilder{},
			},
			args: args{
				bid: &entities.PbsOrtbBid{
					Bid: nil,
				},
			},
			want: SeatNonBidBuilder{},
		},
		{
			name: "append_nonbids_new_seat",
			fields: fields{
				builder: SeatNonBidBuilder{},
			},
			args: args{
				bid: &entities.PbsOrtbBid{
					Bid: &openrtb2.Bid{
						ImpID: "Imp1",
						Price: 10,
					},
				},
				nonBidReason: int(ErrorGeneral),
				seat:         "seat1",
			},
			want: SeatNonBidBuilder{
				"seat1": []openrtb_ext.NonBid{
					{
						ImpId:      "Imp1",
						StatusCode: int(ErrorGeneral),
						Ext: &openrtb_ext.NonBidExt{
							Prebid: openrtb_ext.ExtResponseNonBidPrebid{
								Bid: openrtb_ext.NonBidObject{
									Price: 10,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "append_nonbids_for_different_seat",
			fields: fields{
				builder: SeatNonBidBuilder{
					"seat1": []openrtb_ext.NonBid{
						{
							ImpId:      "Imp1",
							StatusCode: int(ErrorGeneral),
						},
					},
				},
			},
			args: args{
				bid: &entities.PbsOrtbBid{
					Bid: &openrtb2.Bid{
						ImpID: "Imp2",
						Price: 10,
					},
				},
				nonBidReason: int(ErrorGeneral),
				seat:         "seat2",
			},
			want: SeatNonBidBuilder{
				"seat1": []openrtb_ext.NonBid{
					{
						ImpId:      "Imp1",
						StatusCode: int(ErrorGeneral),
					},
				},
				"seat2": []openrtb_ext.NonBid{
					{
						ImpId:      "Imp2",
						StatusCode: int(ErrorGeneral),
						Ext: &openrtb_ext.NonBidExt{
							Prebid: openrtb_ext.ExtResponseNonBidPrebid{
								Bid: openrtb_ext.NonBidObject{
									Price: 10,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "append_nonbids_for_existing_seat",
			fields: fields{
				builder: SeatNonBidBuilder{
					"seat1": []openrtb_ext.NonBid{
						{
							ImpId:      "Imp1",
							StatusCode: int(ErrorGeneral),
						},
					},
				},
			},
			args: args{
				bid: &entities.PbsOrtbBid{
					Bid: &openrtb2.Bid{
						ImpID: "Imp2",
						Price: 10,
					},
				},
				nonBidReason: int(ErrorGeneral),
				seat:         "seat1",
			},
			want: SeatNonBidBuilder{
				"seat1": []openrtb_ext.NonBid{
					{
						ImpId:      "Imp1",
						StatusCode: int(ErrorGeneral),
					},
					{
						ImpId:      "Imp2",
						StatusCode: int(ErrorGeneral),
						Ext: &openrtb_ext.NonBidExt{
							Prebid: openrtb_ext.ExtResponseNonBidPrebid{
								Bid: openrtb_ext.NonBidObject{
									Price: 10,
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snb := tt.fields.builder
			snb.rejectBid(tt.args.bid, tt.args.nonBidReason, tt.args.seat)
			assert.Equal(t, tt.want, snb)
		})
	}
}

func TestAppend(t *testing.T) {
	tests := []struct {
		name     string
		builder  SeatNonBidBuilder
		toAppend []SeatNonBidBuilder
		expected SeatNonBidBuilder
	}{
		{
			name:     "nil_buider",
			builder:  nil,
			toAppend: []SeatNonBidBuilder{{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}}},
			expected: nil,
		},
		{
			name:     "empty_builder",
			builder:  SeatNonBidBuilder{},
			toAppend: []SeatNonBidBuilder{{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}}},
			expected: SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}},
		},
		{
			name:     "append_one_different_seat",
			builder:  SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}},
			toAppend: []SeatNonBidBuilder{{"seat2": []openrtb_ext.NonBid{{ImpId: "imp2"}}}},
			expected: SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}, "seat2": []openrtb_ext.NonBid{{ImpId: "imp2"}}},
		},
		{
			name:     "append_multiple_different_seats",
			builder:  SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}},
			toAppend: []SeatNonBidBuilder{{"seat2": []openrtb_ext.NonBid{{ImpId: "imp2"}}}, {"seat3": []openrtb_ext.NonBid{{ImpId: "imp3"}}}},
			expected: SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}, "seat2": []openrtb_ext.NonBid{{ImpId: "imp2"}}, "seat3": []openrtb_ext.NonBid{{ImpId: "imp3"}}},
		},
		{
			name:     "nil_append",
			builder:  SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}},
			toAppend: nil,
			expected: SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}},
		},
		{
			name:     "empty_append",
			builder:  SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}},
			toAppend: []SeatNonBidBuilder{},
			expected: SeatNonBidBuilder{"seat1": []openrtb_ext.NonBid{{ImpId: "imp1"}}},
		},
		{
			name: "append_multiple_same_seat",
			builder: SeatNonBidBuilder{
				"seat1": []openrtb_ext.NonBid{
					{ImpId: "imp1"},
				},
			},
			toAppend: []SeatNonBidBuilder{
				{
					"seat1": []openrtb_ext.NonBid{
						{ImpId: "imp2"},
					},
				},
			},
			expected: SeatNonBidBuilder{
				"seat1": []openrtb_ext.NonBid{
					{ImpId: "imp1"},
					{ImpId: "imp2"},
				},
			},
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
		{
			name:   "many_imps_appended_to_prepopulated_list_same_seat",
			impIDs: []string{"imp1", "imp2"},
			builder: SeatNonBidBuilder{
				"seat1": []openrtb_ext.NonBid{
					{
						ImpId:      "imp0",
						StatusCode: 300,
					},
				},
			},
			want: SeatNonBidBuilder{
				"seat1": []openrtb_ext.NonBid{
					{
						ImpId:      "imp0",
						StatusCode: 300,
					},
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
