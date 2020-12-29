package response

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/PubMatic-OpenWrap/openrtb"
	"github.com/PubMatic-OpenWrap/prebid-server/endpoints/openrtb2/ctv/types"
)

func Test_findUniqueCombinations(t *testing.T) {
	type args struct {
		data             [][]*types.Bid
		combination      []int
		maxCategoryScore int
		maxDomainScore   int
	}
	tests := []struct {
		name string
		args args
		want *highestCombination
	}{
		{
			name: "sample",
			args: args{
				data: [][]*types.Bid{
					{
						{
							Bid:               &openrtb.Bid{ID: "3-ed72b572-ba62-4220-abba-c19c0bf6346b", Price: 6.339115524232314},
							DealTierSatisfied: true,
						},
						{
							Bid:               &openrtb.Bid{ID: "4-ed72b572-ba62-4220-abba-c19c0bf6346b", Price: 3.532468782358357},
							DealTierSatisfied: true,
						},
						{
							Bid:               &openrtb.Bid{ID: "7-VIDEO12-89A1-41F1-8708-978FD3C0912A", Price: 5},
							DealTierSatisfied: false,
						},
						{
							Bid:               &openrtb.Bid{ID: "8-VIDEO12-89A1-41F1-8708-978FD3C0912A", Price: 5},
							DealTierSatisfied: false,
						},
					}, //20

					{
						{
							Bid:               &openrtb.Bid{ID: "2-ed72b572-ba62-4220-abba-c19c0bf6346b", Price: 3.4502433547413878},
							DealTierSatisfied: true,
						},
						{
							Bid:               &openrtb.Bid{ID: "1-ed72b572-ba62-4220-abba-c19c0bf6346b", Price: 3.329644588311827},
							DealTierSatisfied: true,
						},
						{
							Bid:               &openrtb.Bid{ID: "5-VIDEO12-89A1-41F1-8708-978FD3C0912A", Price: 5},
							DealTierSatisfied: false,
						},
						{
							Bid:               &openrtb.Bid{ID: "6-VIDEO12-89A1-41F1-8708-978FD3C0912A", Price: 5},
							DealTierSatisfied: false,
						},
					}, //25
				},

				combination:      []int{2, 2},
				maxCategoryScore: 100,
				maxDomainScore:   100,
			},
			want: &highestCombination{
				bidIDs:    []string{"3-ed72b572-ba62-4220-abba-c19c0bf6346b", "4-ed72b572-ba62-4220-abba-c19c0bf6346b", "2-ed72b572-ba62-4220-abba-c19c0bf6346b", "1-ed72b572-ba62-4220-abba-c19c0bf6346b"},
				price:     16.651472249643884,
				nDealBids: 4,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findUniqueCombinations(tt.args.data, tt.args.combination, tt.args.maxCategoryScore, tt.args.maxDomainScore)
			assert.Equal(t, tt.want.bidIDs, got.bidIDs, "bidIDs")
			assert.Equal(t, tt.want.nDealBids, got.nDealBids, "nDealBids")
			assert.Equal(t, tt.want.price, got.price, "price")
		})
	}
}
