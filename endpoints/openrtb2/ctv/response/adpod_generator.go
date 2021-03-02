package response

import (
	"fmt"
	"sync"
	"time"

	"github.com/PubMatic-OpenWrap/openrtb"
	"github.com/PubMatic-OpenWrap/prebid-server/endpoints/openrtb2/ctv/combination"
	"github.com/PubMatic-OpenWrap/prebid-server/endpoints/openrtb2/ctv/constant"
	"github.com/PubMatic-OpenWrap/prebid-server/endpoints/openrtb2/ctv/types"
	"github.com/PubMatic-OpenWrap/prebid-server/endpoints/openrtb2/ctv/util"
	"github.com/PubMatic-OpenWrap/prebid-server/metrics"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
)

/********************* AdPodGenerator Functions *********************/

//IAdPodGenerator interface for generating AdPod from Ads
type IAdPodGenerator interface {
	GetAdPodBids() *types.AdPodBid
}
type filteredBid struct {
	bid        *types.Bid
	reasonCode constant.FilterReasonCode
}
type highestCombination struct {
	bids              []*types.Bid
	bidIDs            []string
	durations         []int
	price             float64
	categoryScore     map[string]int
	domainScore       map[string]int
	filteredBids      map[string]*filteredBid
	timeTakenCompExcl time.Duration // time taken by comp excl
	timeTakenCombGen  time.Duration // time taken by combination generator
	nDealBids         int
}

//AdPodGenerator AdPodGenerator
type AdPodGenerator struct {
	IAdPodGenerator
	request  *openrtb.BidRequest
	impIndex int
	buckets  types.BidsBuckets
	comb     combination.ICombination
	adpod    *openrtb_ext.VideoAdPod
	met      metrics.MetricsEngine
}

//NewAdPodGenerator will generate adpod based on configuration
func NewAdPodGenerator(request *openrtb.BidRequest, impIndex int, buckets types.BidsBuckets, comb combination.ICombination, adpod *openrtb_ext.VideoAdPod, met metrics.MetricsEngine) *AdPodGenerator {
	return &AdPodGenerator{
		request:  request,
		impIndex: impIndex,
		buckets:  buckets,
		comb:     comb,
		adpod:    adpod,
		met:      met,
	}
}

//GetAdPodBids will return Adpod based on configurations
func (o *AdPodGenerator) GetAdPodBids() *types.AdPodBid {
	defer util.TimeTrack(time.Now(), fmt.Sprintf("Tid:%v ImpId:%v adpodgenerator", o.request.ID, o.request.Imp[o.impIndex].ID))

	results := o.getAdPodBids(10 * time.Millisecond)
	adpodBid := o.getMaxAdPodBid(results)

	return adpodBid
}

func (o *AdPodGenerator) cleanup(wg *sync.WaitGroup, responseCh chan *highestCombination) {
	defer func() {
		close(responseCh)
		for extra := range responseCh {
			if nil != extra {
				util.Logf("Tid:%v ImpId:%v Delayed Response Durations:%v Bids:%v", o.request.ID, o.request.Imp[o.impIndex].ID, extra.durations, extra.bidIDs)
			}
		}
	}()
	wg.Wait()
}

func (o *AdPodGenerator) getAdPodBids(timeout time.Duration) []*highestCombination {
	start := time.Now()
	defer util.TimeTrack(start, fmt.Sprintf("Tid:%v ImpId:%v getAdPodBids", o.request.ID, o.request.Imp[o.impIndex].ID))

	maxRoutines := 3
	isTimedOutORReceivedAllResponses := false
	results := []*highestCombination{}
	responseCh := make(chan *highestCombination, maxRoutines)
	wg := new(sync.WaitGroup) // ensures each step generating impressions is finished
	lock := sync.Mutex{}
	ticker := time.NewTicker(timeout)

	combinationCount := 0
	for i := 0; i < maxRoutines; i++ {
		wg.Add(1)
		go func() {
			for !isTimedOutORReceivedAllResponses {
				combGenStartTime := time.Now()
				lock.Lock()
				durations := o.comb.Get()
				lock.Unlock()
				combGenElapsedTime := time.Since(combGenStartTime)

				if len(durations) == 0 {
					break
				}
				hbc := o.getUniqueBids(durations)
				hbc.timeTakenCombGen = combGenElapsedTime
				responseCh <- hbc
				util.Logf("Tid:%v GetUniqueBids Durations:%v Price:%v DealBids:%v Time:%v Bids:%v", o.request.ID, hbc.durations[:], hbc.price, hbc.nDealBids, hbc.timeTakenCompExcl, hbc.bidIDs[:])
			}
			wg.Done()
		}()
	}

	// ensure impressions channel is closed
	// when all go routines are executed
	go o.cleanup(wg, responseCh)

	totalTimeByCombGen := int64(0)
	totalTimeByCompExcl := int64(0)
	for !isTimedOutORReceivedAllResponses {
		select {
		case hbc, ok := <-responseCh:

			if false == ok {
				isTimedOutORReceivedAllResponses = true
				break
			}
			if nil != hbc {
				combinationCount++
				totalTimeByCombGen += int64(hbc.timeTakenCombGen)
				totalTimeByCompExcl += int64(hbc.timeTakenCompExcl)
				results = append(results, hbc)
			}
		case <-ticker.C:
			isTimedOutORReceivedAllResponses = true
			util.Logf("Tid:%v ImpId:%v GetAdPodBids Timeout Reached %v", o.request.ID, o.request.Imp[o.impIndex].ID, timeout)
		}
	}

	defer ticker.Stop()

	labels := metrics.PodLabels{
		AlgorithmName:    string(constant.CombinationGeneratorV1),
		NoOfCombinations: new(int),
	}
	*labels.NoOfCombinations = combinationCount
	o.met.RecordPodCombGenTime(labels, time.Duration(totalTimeByCombGen))

	compExclLabels := metrics.PodLabels{
		AlgorithmName:    string(constant.CompetitiveExclusionV1),
		NoOfResponseBids: new(int),
	}
	*compExclLabels.NoOfResponseBids = 0
	for _, ads := range o.buckets {
		*compExclLabels.NoOfResponseBids += len(ads)
	}
	o.met.RecordPodCompititveExclusionTime(compExclLabels, time.Duration(totalTimeByCompExcl))

	return results[:]
}

func (o *AdPodGenerator) getMaxAdPodBid(results []*highestCombination) *types.AdPodBid {
	if 0 == len(results) {
		util.Logf("Tid:%v ImpId:%v NoBid", o.request.ID, o.request.Imp[o.impIndex].ID)
		return nil
	}

	//Get Max Response
	var maxResult *highestCombination
	for _, result := range results {
		for _, rc := range result.filteredBids {
			if constant.CTVRCDidNotGetChance == rc.bid.FilterReasonCode {
				rc.bid.FilterReasonCode = rc.reasonCode
			}
		}
		if len(result.bidIDs) == 0 {
			continue
		}

		if nil == maxResult ||
			(maxResult.nDealBids < result.nDealBids) ||
			(maxResult.nDealBids == result.nDealBids && maxResult.price < result.price) {
			maxResult = result
		}
	}

	if nil == maxResult {
		util.Logf("Tid:%v ImpId:%v All Combination Filtered in Ad Exclusion", o.request.ID, o.request.Imp[o.impIndex].ID)
		return nil
	}

	adpodBid := &types.AdPodBid{
		Bids:    maxResult.bids[:],
		Price:   maxResult.price,
		ADomain: make([]string, 0),
		Cat:     make([]string, 0),
	}

	//Get Unique Domains
	for domain := range maxResult.domainScore {
		adpodBid.ADomain = append(adpodBid.ADomain, domain)
	}

	//Get Unique Categories
	for cat := range maxResult.categoryScore {
		adpodBid.Cat = append(adpodBid.Cat, cat)
	}

	util.Logf("Tid:%v ImpId:%v Selected Durations:%v Price:%v Bids:%v", o.request.ID, o.request.Imp[o.impIndex].ID, maxResult.durations[:], maxResult.price, maxResult.bidIDs[:])

	return adpodBid
}

func (o *AdPodGenerator) getUniqueBids(durationSequence []int) *highestCombination {
	startTime := time.Now()
	data := [][]*types.Bid{}
	combinations := []int{}

	defer util.TimeTrack(startTime, fmt.Sprintf("Tid:%v ImpId:%v getUniqueBids:%v", o.request.ID, o.request.Imp[o.impIndex].ID, durationSequence))

	uniqueDuration := 0
	for index, duration := range durationSequence {
		if 0 != index && durationSequence[index-1] == duration {
			combinations[uniqueDuration-1]++
			continue
		}
		data = append(data, o.buckets[duration][:])
		combinations = append(combinations, 1)
		uniqueDuration++
	}
	hbc := findUniqueCombinations(data[:], combinations[:], *o.adpod.IABCategoryExclusionPercent, *o.adpod.AdvertiserExclusionPercent)
	hbc.durations = durationSequence[:]
	hbc.timeTakenCompExcl = time.Since(startTime)

	return hbc
}

func findUniqueCombinations(data [][]*types.Bid, combination []int, maxCategoryScore, maxDomainScore int) *highestCombination {
	// number of arrays
	n := len(combination)
	totalBids := 0
	//  to keep track of next element in each of the n arrays
	// indices is initialized
	indices := make([][]int, len(combination))
	for i := 0; i < len(combination); i++ {
		indices[i] = make([]int, combination[i])
		for j := 0; j < combination[i]; j++ {
			indices[i][j] = j
			totalBids++
		}
	}

	hc := &highestCombination{}
	var ehc *highestCombination
	var rc constant.FilterReasonCode
	inext, jnext := n-1, 0
	filterBids := map[string]*filteredBid{}

	// maintain highest price combination
	for true {

		ehc, inext, jnext, rc = evaluate(data[:], indices[:], totalBids, maxCategoryScore, maxDomainScore)
		if nil != ehc {
			if nil == hc || (hc.nDealBids == ehc.nDealBids && hc.price < ehc.price) || (hc.nDealBids < ehc.nDealBids) {
				hc = ehc
			} else {
				// if you see current combination price lower than the highest one then break the loop
				break
			}
		} else {
			//Filtered Bid
			for i := 0; i <= inext; i++ {
				for j := 0; j < combination[i] && !(i == inext && j > jnext); j++ {
					bid := data[i][indices[i][j]]
					if _, ok := filterBids[bid.ID]; !ok {
						filterBids[bid.ID] = &filteredBid{bid: bid, reasonCode: rc}
					}
				}
			}
		}

		if -1 == inext {
			inext, jnext = n-1, 0
		}

		// find the rightmost array that has more
		// elements left after the current element
		// in that array
		inext, jnext := n-1, 0

		for inext >= 0 {
			jnext = len(indices[inext]) - 1
			for jnext >= 0 && (indices[inext][jnext]+1 > (len(data[inext]) - len(indices[inext]) + jnext)) {
				jnext--
			}
			if jnext >= 0 {
				break
			}
			inext--
		}

		// no such array is found so no more combinations left
		if inext < 0 {
			break
		}

		// if found move to next element in that array
		indices[inext][jnext]++

		// for all arrays to the right of this
		// array current index again points to
		// first element
		jnext++
		for i := inext; i < len(combination); i++ {
			for j := jnext; j < combination[i]; j++ {
				if i == inext {
					indices[i][j] = indices[i][j-1] + 1
				} else {
					indices[i][j] = j
				}
			}
			jnext = 0
		}
	}

	//setting filteredBids
	if nil != filterBids {
		hc.filteredBids = filterBids
	}
	return hc
}

func evaluate(bids [][]*types.Bid, indices [][]int, totalBids int, maxCategoryScore, maxDomainScore int) (*highestCombination, int, int, constant.FilterReasonCode) {

	hbc := &highestCombination{
		bids:          make([]*types.Bid, totalBids),
		bidIDs:        make([]string, totalBids),
		price:         0,
		categoryScore: make(map[string]int),
		domainScore:   make(map[string]int),
		nDealBids:     0,
	}
	pos := 0

	for inext := range indices {
		for jnext := range indices[inext] {
			bid := bids[inext][indices[inext][jnext]]

			hbc.bids[pos] = bid
			hbc.bidIDs[pos] = bid.ID
			pos++

			//nDealBids
			if bid.DealTierSatisfied {
				hbc.nDealBids++
			}

			//Price
			hbc.price = hbc.price + bid.Price

			//Categories
			for _, cat := range bid.Cat {
				hbc.categoryScore[cat]++
				if hbc.categoryScore[cat] > 1 && (hbc.categoryScore[cat]*100/totalBids) > maxCategoryScore {
					return nil, inext, jnext, constant.CTVRCCategoryExclusion
				}
			}

			//Domain
			for _, domain := range bid.ADomain {
				hbc.domainScore[domain]++
				if hbc.domainScore[domain] > 1 && (hbc.domainScore[domain]*100/totalBids) > maxDomainScore {
					return nil, inext, jnext, constant.CTVRCDomainExclusion
				}
			}
		}
	}

	return hbc, -1, -1, constant.CTVRCWinningBid
}
