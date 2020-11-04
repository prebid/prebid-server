package util

import (
	"fmt"
	"testing"

	"github.com/PubMatic-OpenWrap/openrtb"

	"github.com/PubMatic-OpenWrap/prebid-server/endpoints/openrtb2/ctv/types"
	"github.com/stretchr/testify/assert"
)

func TestDecodeImpressionID(t *testing.T) {
	type args struct {
		id string
	}
	type want struct {
		id  string
		seq int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "TC1",
			args: args{id: "impid"},
			want: want{id: "impid", seq: 0},
		},
		{
			name: "TC2",
			args: args{id: "impid_1"},
			want: want{id: "impid", seq: 1},
		},
		{
			name: "TC1",
			args: args{id: "impid_1_2"},
			want: want{id: "impid_1", seq: 2},
		},
		{
			name: "TC1",
			args: args{id: "impid_1_x"},
			want: want{id: "impid_1_x", seq: 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, seq := DecodeImpressionID(tt.args.id)
			assert.Equal(t, tt.want.id, id)
			assert.Equal(t, tt.want.seq, seq)
		})
	}
}

func TestSortByDealPriority(t *testing.T) {

	type testbid struct {
		id        string
		price     float64
		isDealBid bool
	}

	testcases := []struct {
		scenario              string
		bids                  []testbid
		expectedBidIDOrdering []string
	}{
		/* tests based on truth table */
		{
			scenario: "all_deal_bids_do_price_based_sort",
			bids: []testbid{
				{id: "DB_$5", price: 5.0, isDealBid: true},   // Deal bid with low price
				{id: "DB_$10", price: 10.0, isDealBid: true}, // Deal bid with high price
			},
			expectedBidIDOrdering: []string{"DB_$10", "DB_$5"}, // sort by price among deal bids
		},
		{
			scenario: "normal_and_deal_bid_mix_case_1",
			bids: []testbid{
				{id: "DB_$15", price: 15.0, isDealBid: true}, // Deal bid with low price
				{id: "B_$30", price: 30.0, isDealBid: false}, // Normal bid with high price
			},
			expectedBidIDOrdering: []string{"DB_$15", "B_$30"}, // no sort expected. Deal bid is already 1st in order
		},
		{
			scenario: "normal_and_deal_bid_mix_case_2", // deal bids are not at start position in order
			bids: []testbid{
				{id: "B_$30", price: 30.0, isDealBid: false}, // Normal bid with high price
				{id: "DB_$15", price: 15.0, isDealBid: true}, // Deal bid with low price
			},
			expectedBidIDOrdering: []string{"DB_$15", "B_$30"}, // sort based on deal bid
		},
		{
			scenario: "all_normal_bids_sort_by_price_case_1",
			bids: []testbid{
				{id: "B_$5", price: 5.0, isDealBid: false},
				{id: "B_$10", price: 10.0, isDealBid: false},
			},
			expectedBidIDOrdering: []string{"B_$10", "B_$5"}, // sort by price
		},
		{
			scenario: "all_normal_bids_sort_by_price_case_2", // already sorted by highest price
			bids: []testbid{
				{id: "B_$10", price: 10.0, isDealBid: false},
				{id: "B_$5", price: 5.0, isDealBid: false},
			},
			expectedBidIDOrdering: []string{"B_$10", "B_$5"}, // no sort required as already sorted
		},
		/* use cases */
		{
			scenario: "deal_bids_with_same_price",
			bids: []testbid{
				{id: "DB2_$10", price: 10.0, isDealBid: true},
				{id: "DB1_$10", price: 10.0, isDealBid: true},
			},
			expectedBidIDOrdering: []string{"DB2_$10", "DB1_$10"}, // no sort expected
		},
		/* more than 2 Bids testcases */
		{
			scenario: "4_bids_with_first_and_last_are_deal_bids",
			bids: []testbid{
				{id: "DB_$15", price: 15.0, isDealBid: true}, // deal bid with low CPM than another bid
				{id: "B_$40", price: 40.0, isDealBid: false}, // normal bid with highest CPM
				{id: "B_$3", price: 3.0, isDealBid: false},
				{id: "DB_$20", price: 20.0, isDealBid: true}, // deal bid with high cpm than another deal bid
			},
			expectedBidIDOrdering: []string{"DB_$20", "DB_$15", "B_$40", "B_$3"},
		},
		{
			scenario: "deal_bids_and_normal_bids_with_same_price",
			bids: []testbid{
				{id: "B1_$7", price: 7.0, isDealBid: false},
				{id: "DB2_$7", price: 7.0, isDealBid: true},
				{id: "B3_$7", price: 7.0, isDealBid: false},
				{id: "DB1_$7", price: 7.0, isDealBid: true},
				{id: "B2_$7", price: 7.0, isDealBid: false},
			},
			expectedBidIDOrdering: []string{"DB2_$7", "DB1_$7", "B1_$7", "B3_$7", "B2_$7"}, // no sort expected
		},
	}

	newBid := func(bid testbid) *types.Bid {
		return &types.Bid{
			Bid: &openrtb.Bid{
				ID:    bid.id,
				Price: bid.price,
				//Ext:   json.RawMessage(`{"prebid":{ "dealTierSatisfied" : ` + bid.isDealBid + ` }}`),
			},
			DealTierSatisfied: bid.isDealBid,
		}
	}

	for _, test := range testcases {
		// if test.scenario != "deal_bids_and_normal_bids_with_same_price" {
		// 	continue
		// }
		fmt.Println("Scenario : ", test.scenario)
		bids := []*types.Bid{}
		for _, bid := range test.bids {
			bids = append(bids, newBid(bid))
		}
		for _, bid := range bids {
			fmt.Println(bid.ID, ",", bid.Price, ",", bid.DealTierSatisfied)
		}
		sortBids(bids[:])
		fmt.Println("After sort")
		actual := []string{}
		for _, bid := range bids {
			fmt.Println(bid.ID, ",", bid.Price, ", ", bid.DealTierSatisfied)
			actual = append(actual, bid.ID)
		}
		assert.Equal(t, test.expectedBidIDOrdering, actual, test.scenario+" failed")
		fmt.Println("")
	}
}
