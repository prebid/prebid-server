package ctv

import (
	"context"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
)

/********************* AdPodGenerator Functions *********************/

//IAdPodGenerator interface for generating AdPod from Ads
type IAdPodGenerator interface {
	GetAdPodBids() []*Bid
	GetFilterReasonCode() map[string]int
}

type evaluation struct {
	bids          []*Bid
	sum           float64
	categoryScore map[string]int
	domainScore   map[string]int
}

type highestCombination struct {
	bids []*Bid
	sum  float64
}

//AdPodGenerator AdPodGenerator
type AdPodGenerator struct {
	IAdPodGenerator
	buckets       BidsBuckets
	comb          ICombination
	filterReasons map[string]int
	adpod         *openrtb_ext.VideoAdPod
}

//NewAdPodGenerator will generate adpod based on configuration
func NewAdPodGenerator(buckets BidsBuckets, comb ICombination, adpod *openrtb_ext.VideoAdPod) *AdPodGenerator {
	return &AdPodGenerator{
		buckets:       buckets,
		comb:          comb,
		adpod:         adpod,
		filterReasons: make(map[string]int),
	}
}

//GetAdPodBids will return Adpod based on configurations
func (o *AdPodGenerator) GetAdPodBids() []*Bid {

	var maxResult *highestCombination
	isTimedOutORReceivedAllResponses := false
	responseCount := 0
	totalRequest := 0
	maxRequests := 5
	responseCh := make(chan *highestCombination, maxRequests)

	timeout := 50 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for totalRequest < maxRequests {
		durations := o.comb.Get()
		if len(durations) == 0 {
			break
		}

		totalRequest++
		go o.getUniqueBids(responseCh, durations)
	}

	for !isTimedOutORReceivedAllResponses {
		select {
		case <-ctx.Done():
			isTimedOutORReceivedAllResponses = true
		case hbc := <-responseCh:
			responseCount++
			if nil != hbc && (nil == maxResult || maxResult.sum < hbc.sum) {
				maxResult = hbc
			}
			if responseCount == totalRequest {
				isTimedOutORReceivedAllResponses = true
			}
		}
	}

	go cleanupResponseChannel(responseCh, totalRequest-responseCount)

	if nil == maxResult {
		return nil
	}
	return maxResult.bids
}

func cleanupResponseChannel(responseCh <-chan *highestCombination, responseCount int) {
	for responseCount > 0 {
		<-responseCh
		responseCount--
	}
}

func (o *AdPodGenerator) getUniqueBids(responseCh chan<- *highestCombination, durationSequence []int) {
	combinationsInputArray := [][]*Bid{}
	for index, duration := range durationSequence {
		combinationsInputArray[index] = o.buckets[duration][:]
	}

	responseCh <- findUniqueCombinations(combinationsInputArray, *o.adpod.IABCategoryExclusionPercent, *o.adpod.AdvertiserExclusionPercent)
}

// Todo: this function is still returning (B3 B4) and (B4 B3), need to work on it
// func findUniqueCombinations(arr [][]Bid) ([][]Bid) {
func findUniqueCombinations(arr [][]*Bid, maxCategoryScore, maxDomainScore int) *highestCombination {
	// number of arrays
	n := len(arr)
	//  to keep track of next element in each of the n arrays
	indices := make([]int, n)
	// indices is initialized with all zeros

	// output := [][]Bid{}

	// maintain highest sum combination
	hc := &highestCombination{sum: 0}
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
			e := getEvaluation(row)
			// fmt.Println(e)
			if e.isOk(maxCategoryScore, maxDomainScore) {
				if hc.sum < e.sum {
					hc.bids = e.bids
					hc.sum = e.sum
				} else {
					// if you see current combination sum lower than the highest one then break the loop
					return hc
				}
			}
		}

		// find the rightmost array that has more
		// elements left after the current element
		// in that array
		next := n - 1
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

func getEvaluation(bids []*Bid) *evaluation {

	eval := &evaluation{
		bids:          bids,
		sum:           0,
		categoryScore: make(map[string]int),
		domainScore:   make(map[string]int),
	}

	for _, bid := range bids {

		eval.sum = eval.sum + bid.Price
		for _, cat := range bid.Cat {
			if _, present := eval.categoryScore[cat]; !present {
				eval.categoryScore[cat] = 1
			} else {
				eval.categoryScore[cat] = eval.categoryScore[cat] + 1
			}
		}

		l := len(eval.bids)
		for i := range eval.categoryScore {
			eval.categoryScore[i] = (eval.categoryScore[i] * 100 / l)
		}

		for _, domain := range bid.ADomain {
			if _, present := eval.domainScore[domain]; !present {
				eval.domainScore[domain] = 1
			} else {
				eval.domainScore[domain] = eval.domainScore[domain] + 1
			}
		}

		l2 := len(eval.bids)
		for i := range eval.domainScore {
			eval.domainScore[i] = (eval.domainScore[i] * 100 / l2)
		}
	}

	return eval
}

func (e *evaluation) isOk(maxCategoryScore, maxDomainScore int) bool {

	// if we find any CategoryScore above maxCategoryScore then we return false
	for _, score := range e.categoryScore {
		if maxCategoryScore < score {
			return false
		}
	}

	// if we find any DomainScore above maxDomainScore then we return false
	for _, score := range e.domainScore {
		if maxDomainScore < score {
			return false
		}
	}

	return true
}

func (o *AdPodGenerator) GetFilterReasonCode() map[string]int {
	return o.filterReasons
}
