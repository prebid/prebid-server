package ctv

import "sort"

func GetDurationWiseBidsBucket(bids []*Bid) BidsBuckets {
	result := BidsBuckets{}

	for i, bid := range bids {
		result[bid.Duration] = append(result[bid.Duration], bids[i])
	}

	for k, v := range result {
		sort.Slice(v[:], func(i, j int) bool { return v[i].Price > v[j].Price })
		result[k] = v
	}

	return result
} 