package ctv

/*
func (o *AdPodGenerator) getUniqueBids(responseCh chan<- *highestCombination, durationSequence []int) {
	data := [][]*Bid{}
	combinations := []int{}

	for index, duration := range durationSequence {
		data[index] = o.buckets[duration][:]
	}

	responseCh <- findUniqueCombinations(data[:], *o.adpod.IABCategoryExclusionPercent, *o.adpod.AdvertiserExclusionPercent)
}
*/

// Todo: this function is still returning (B3 B4) and (B4 B3), need to work on it
// func findUniqueCombinations(arr [][]Bid) ([][]Bid) {
func findUniqueCombinationsOld(arr [][]*Bid, maxCategoryScore, maxDomainScore int) *highestCombination {
	// number of arrays
	n := len(arr)
	//  to keep track of next element in each of the n arrays
	indices := make([]int, n)
	// indices is initialized with all zeros

	// maintain highest price combination
	var ehc *highestCombination
	var rc FilterReasonCode
	next := n - 1
	hc := &highestCombination{price: 0}
	for true {

		row := []*Bid{}
		// We do not want the same bid to appear twice in a combination
		bidsInRow := make(map[string]bool)
		good := true

		for i := 0; i < n; i++ {
			if _, present := bidsInRow[arr[i][indices[i]].ID]; !present {
				row = append(row, arr[i][indices[i]])
				bidsInRow[arr[i][indices[i]].ID] = true
			} else {
				good = false
				break
			}
		}

		if good {
			// output = append(output, row)
			// give a call for exclusion checking here only
			ehc, next, rc = evaluateOld(row, maxCategoryScore, maxDomainScore)
			if nil != ehc {
				if nil == hc || hc.price < ehc.price {
					hc = ehc
				} else {
					// if you see current combination price lower than the highest one then break the loop
					return hc
				}
			} else {
				arr[next][indices[next]].FilterReasonCode = rc
			}
		}

		// find the rightmost array that has more
		// elements left after the current element
		// in that array
		if -1 == next {
			next = n - 1
		}

		for next >= 0 && (indices[next]+1 >= len(arr[next])) {
			next--
		}

		// no such array is found so no more combinations left
		if next < 0 {
			// return output
			return nil
		}

		// if found move to next element in that array
		indices[next]++

		// for all arrays to the right of this
		// array current index again points to
		// first element
		for i := next + 1; i < n; i++ {
			indices[i] = 0
		}
	}
	// return output
	return hc
}

func evaluateOld(bids []*Bid, maxCategoryScore, maxDomainScore int) (*highestCombination, int, FilterReasonCode) {

	hbc := &highestCombination{
		bids:          bids,
		price:         0,
		categoryScore: make(map[string]int),
		domainScore:   make(map[string]int),
	}

	totalBids := len(bids)

	for index, bid := range bids {
		hbc.price = hbc.price + bid.Price

		for _, cat := range bid.Cat {
			hbc.categoryScore[cat]++
			if (hbc.categoryScore[cat] * 100 / totalBids) > maxCategoryScore {
				return nil, index, CTVRCCategoryExclusion
			}
		}

		for _, domain := range bid.ADomain {
			hbc.domainScore[domain]++
			if (hbc.domainScore[domain] * 100 / totalBids) > maxDomainScore {
				return nil, index, CTVRCDomainExclusion
			}
		}
	}

	return hbc, -1, CTVRCWinningBid
}
