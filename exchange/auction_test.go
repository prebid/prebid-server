package exchange

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/prebid_cache_client"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"

	"github.com/stretchr/testify/assert"
)

func TestMakeVASTGiven(t *testing.T) {
	const expect = `<VAST version="3.0"></VAST>`
	bid := &openrtb2.Bid{
		AdM: expect,
	}
	vast := makeVAST(bid)
	assert.Equal(t, expect, vast)
}

func TestMakeVASTNurl(t *testing.T) {
	const url = "http://domain.com/win-notify/1"
	const expect = `<VAST version="3.0"><Ad><Wrapper>` +
		`<AdSystem>prebid.org wrapper</AdSystem>` +
		`<VASTAdTagURI><![CDATA[` + url + `]]></VASTAdTagURI>` +
		`<Impression></Impression><Creatives></Creatives>` +
		`</Wrapper></Ad></VAST>`
	bid := &openrtb2.Bid{
		NURL: url,
	}
	vast := makeVAST(bid)
	assert.Equal(t, expect, vast)
}

func TestBuildCacheString(t *testing.T) {
	testCases := []struct {
		description      string
		debugLog         DebugLog
		expectedDebugLog DebugLog
	}{
		{
			description: "DebugLog strings should have tags and be formatted",
			debugLog: DebugLog{
				Data: DebugData{
					Request:  "test request string",
					Headers:  "test headers string",
					Response: "test response string",
				},
				Regexp: regexp.MustCompile(`[<>]`),
			},
			expectedDebugLog: DebugLog{
				Data: DebugData{
					Request:  "<Request>test request string</Request>",
					Headers:  "<Headers>test headers string</Headers>",
					Response: "<Response>test response string</Response>",
				},
				Regexp: regexp.MustCompile(`[<>]`),
			},
		},
		{
			description: "DebugLog strings should have no < or > characters",
			debugLog: DebugLog{
				Data: DebugData{
					Request:  "<test>test request string</test>",
					Headers:  "test <headers string",
					Response: "test <response> string",
				},
				Regexp: regexp.MustCompile(`[<>]`),
			},
			expectedDebugLog: DebugLog{
				Data: DebugData{
					Request:  "<Request>testtest request string/test</Request>",
					Headers:  "<Headers>test headers string</Headers>",
					Response: "<Response>test response string</Response>",
				},
				Regexp: regexp.MustCompile(`[<>]`),
			},
		},
	}

	for _, test := range testCases {
		test.expectedDebugLog.CacheString = fmt.Sprintf("%s<Log>%s%s%s</Log>", xml.Header, test.expectedDebugLog.Data.Request, test.expectedDebugLog.Data.Headers, test.expectedDebugLog.Data.Response)

		test.debugLog.BuildCacheString()

		assert.Equal(t, test.expectedDebugLog, test.debugLog, test.description)
	}
}

// TestCacheJSON executes tests for all the *.json files in cachetest.
// customcachekey.json test here verifies custom cache key not used for non-vast video
func TestCacheJSON(t *testing.T) {
	for _, dir := range []string{"cachetest", "customcachekeytest", "impcustomcachekeytest", "eventscachetest"} {
		if specFiles, err := os.ReadDir(dir); err == nil {
			for _, specFile := range specFiles {
				fileName := filepath.Join(dir, specFile.Name())
				fileDisplayName := "exchange/" + fileName
				t.Run(fileDisplayName, func(t *testing.T) {
					specData, err := loadCacheSpec(fileName)
					if assert.NoError(t, err, "Failed to load contents of file %s: %v", fileDisplayName, err) {
						runCacheSpec(t, fileDisplayName, specData)
					}
				})
			}
		} else {
			t.Fatalf("Failed to read contents of directory exchange/%s: %v", dir, err)
		}
	}
}

func TestIsDebugOverrideEnabled(t *testing.T) {
	type inTest struct {
		debugHeader string
		configToken string
	}
	type aTest struct {
		desc   string
		in     inTest
		result bool
	}
	testCases := []aTest{
		{
			desc:   "test debug header is empty, config token is empty",
			in:     inTest{debugHeader: "", configToken: ""},
			result: false,
		},
		{
			desc:   "test debug header is present, config token is empty",
			in:     inTest{debugHeader: "TestToken", configToken: ""},
			result: false,
		},
		{
			desc:   "test debug header is empty, config token is present",
			in:     inTest{debugHeader: "", configToken: "TestToken"},
			result: false,
		},
		{
			desc:   "test debug header is present, config token is present, not equal",
			in:     inTest{debugHeader: "TestToken123", configToken: "TestToken"},
			result: false,
		},
		{
			desc:   "test debug header is present, config token is present, equal",
			in:     inTest{debugHeader: "TestToken", configToken: "TestToken"},
			result: true,
		},
		{
			desc:   "test debug header is present, config token is present, not case equal",
			in:     inTest{debugHeader: "TestTokeN", configToken: "TestToken"},
			result: false,
		},
	}

	for _, test := range testCases {
		result := IsDebugOverrideEnabled(test.in.debugHeader, test.in.configToken)
		assert.Equal(t, test.result, result, test.desc)
	}

}

// LoadCacheSpec reads and parses a file as a test case. If something goes wrong, it returns an error.
func loadCacheSpec(filename string) (*cacheSpec, error) {
	specData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %v", filename, err)
	}

	var spec cacheSpec
	if err := jsonutil.UnmarshalValid(specData, &spec); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal JSON from file: %v", err)
	}

	return &spec, nil
}

// runCacheSpec cycles through the bids found in the json test cases and
// finds the highest bid of every Imp, then tests doCache() with resulting auction object
func runCacheSpec(t *testing.T, fileDisplayName string, specData *cacheSpec) {
	var bid *entities.PbsOrtbBid
	winningBidsByImp := make(map[string]*entities.PbsOrtbBid)
	allBidsByBidder := make(map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid)
	roundedPrices := make(map[*entities.PbsOrtbBid]string)
	bidCategory := make(map[string]string)

	// Traverse through the bid list found in the parsed in Json file
	for _, pbsBid := range specData.PbsBids {
		bid = &entities.PbsOrtbBid{
			Bid:     pbsBid.Bid,
			BidType: pbsBid.BidType,
		}
		cpm := bid.Bid.Price

		// Map this bid if it's the highest we've seen from this Imp so far
		wbid, ok := winningBidsByImp[bid.Bid.ImpID]
		if !ok || cpm > wbid.Bid.Price {
			winningBidsByImp[bid.Bid.ImpID] = bid
		}

		// Map this bid if it's the highest we've seen from this bidder so far
		if bidMap, ok := allBidsByBidder[bid.Bid.ImpID]; ok {
			bidMap[pbsBid.Bidder] = append(bidMap[pbsBid.Bidder], bid)
		} else {
			allBidsByBidder[bid.Bid.ImpID] = map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				pbsBid.Bidder: {bid},
			}
		}

		for _, topBidsPerBidder := range allBidsByBidder {
			for _, topBids := range topBidsPerBidder {
				sort.Slice(topBids, func(i, j int) bool {
					return isNewWinningBid(topBids[i].Bid, topBids[j].Bid, true)
				})
			}
		}

		if len(pbsBid.Bid.Cat) == 1 {
			bidCategory[pbsBid.Bid.ID] = pbsBid.Bid.Cat[0]
		}
		roundedPrices[bid] = strconv.FormatFloat(bid.Bid.Price, 'f', 2, 64)
	}

	ctx := context.Background()
	cache := &mockCache{}

	targData := &targetData{
		priceGranularity: openrtb_ext.PriceGranularity{
			Precision: ptrutil.ToPtr(2),
			Ranges: []openrtb_ext.GranularityRange{
				{
					Min:       0,
					Max:       5,
					Increment: 0.05,
				},
				{
					Min:       5,
					Max:       10,
					Increment: 0.1,
				},
				{
					Min:       10,
					Max:       20,
					Increment: 0.5,
				},
			},
		},
		includeWinners:    specData.TargetDataIncludeWinners,
		includeBidderKeys: specData.TargetDataIncludeBidderKeys,
		includeCacheBids:  specData.TargetDataIncludeCacheBids,
		includeCacheVast:  specData.TargetDataIncludeCacheVast,
	}

	testAuction := &auction{
		winningBids:     winningBidsByImp,
		allBidsByBidder: allBidsByBidder,
		roundedPrices:   roundedPrices,
	}
	evTracking := &eventTracking{
		accountID:          "TEST_ACC_ID",
		enabledForAccount:  specData.EventsDataEnabledForAccount,
		enabledForRequest:  specData.EventsDataEnabledForRequest,
		externalURL:        "http://localhost",
		auctionTimestampMs: 1234567890,
	}
	_ = testAuction.doCache(ctx, cache, targData, evTracking, &specData.BidRequest, 60, &specData.DefaultTTLs, bidCategory, &specData.DebugLog)

	if len(specData.ExpectedCacheables) > len(cache.items) {
		t.Errorf("%s:  [CACHE_ERROR] Less elements were cached than expected \n", fileDisplayName)
	} else if len(specData.ExpectedCacheables) < len(cache.items) {
		t.Errorf("%s:  [CACHE_ERROR] More elements were cached than expected \n", fileDisplayName)
	} else { // len(specData.ExpectedCacheables) == len(cache.items)
		// We cached the exact number of elements we expected, now we compare them side by side in n^2
		var matched int = 0
		for i, expectedCacheable := range specData.ExpectedCacheables {
			found := false
			var expectedData interface{}
			if err := jsonutil.UnmarshalValid(expectedCacheable.Data, &expectedData); err != nil {
				t.Fatalf("Failed to decode expectedCacheables[%d].value: %v", i, err)
			}
			if s, ok := expectedData.(string); ok && expectedCacheable.Type == prebid_cache_client.TypeJSON {
				// decode again if we have pre-encoded json string values
				if err := jsonutil.UnmarshalValid([]byte(s), &expectedData); err != nil {
					t.Fatalf("Failed to re-decode expectedCacheables[%d].value :%v", i, err)
				}
			}
			for j, cachedItem := range cache.items {
				var actualData interface{}
				if err := jsonutil.UnmarshalValid(cachedItem.Data, &actualData); err != nil {
					t.Fatalf("Failed to decode actual cache[%d].value: %s", j, err)
				}
				if assert.ObjectsAreEqual(expectedData, actualData) &&
					expectedCacheable.TTLSeconds == cachedItem.TTLSeconds &&
					expectedCacheable.Type == cachedItem.Type &&
					len(expectedCacheable.Key) <= len(cachedItem.Key) &&
					expectedCacheable.Key == cachedItem.Key[:len(expectedCacheable.Key)] {
					found = true
					cache.items = append(cache.items[:j], cache.items[j+1:]...) // remove matched item
					break
				}
			}
			if found {
				matched++
			} else {
				t.Errorf("%s: [CACHE_ERROR] Did not see expected cacheable #%d: type=%s, ttl=%d, value=%s", fileDisplayName, i, expectedCacheable.Type, expectedCacheable.TTLSeconds, string(expectedCacheable.Data))
			}
		}
		if matched != len(specData.ExpectedCacheables) {
			for i, item := range cache.items {
				t.Errorf("%s: [CACHE_ERROR] Got unexpected cached item #%d: type=%s, ttl=%d, value=%s", fileDisplayName, i, item.Type, item.TTLSeconds, string(item.Data))
			}
			t.FailNow()
		}
	}
}

func TestNewAuction(t *testing.T) {
	bid1p077 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp1",
			Price: 0.77,
		},
	}
	bid1p123 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp1",
			Price: 1.23,
		},
	}
	bid1p230 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp1",
			Price: 2.30,
		},
	}
	bid1p088d := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID:  "imp1",
			Price:  0.88,
			DealID: "SpecialDeal",
		},
	}
	bid1p166d := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID:  "imp1",
			Price:  1.66,
			DealID: "BigDeal",
		},
	}
	bid2p123 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp2",
			Price: 1.23,
		},
	}
	bid2p144 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp2",
			Price: 1.44,
		},
	}
	bid2p155 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp2",
			Price: 1.55,
		},
	}
	bid2p166 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp2",
			Price: 1.66,
		},
	}
	tests := []struct {
		description     string
		seatBids        map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		numImps         int
		preferDeals     bool
		expectedAuction auction
	}{
		{
			description: "Basic auction test",
			seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{&bid1p123},
				},
				"rubicon": {
					Bids: []*entities.PbsOrtbBid{&bid1p230},
				},
			},
			numImps:     1,
			preferDeals: false,
			expectedAuction: auction{
				winningBids: map[string]*entities.PbsOrtbBid{
					"imp1": &bid1p230,
				},
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p123},
						"rubicon":  []*entities.PbsOrtbBid{&bid1p230},
					},
				},
			},
		},
		{
			description: "Multi-imp auction",
			seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{&bid1p230, &bid2p123},
				},
				"rubicon": {
					Bids: []*entities.PbsOrtbBid{&bid1p077, &bid2p144},
				},
				"openx": {
					Bids: []*entities.PbsOrtbBid{&bid1p123},
				},
			},
			numImps:     2,
			preferDeals: false,
			expectedAuction: auction{
				winningBids: map[string]*entities.PbsOrtbBid{
					"imp1": &bid1p230,
					"imp2": &bid2p144,
				},
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p230},
						"rubicon":  []*entities.PbsOrtbBid{&bid1p077},
						"openx":    []*entities.PbsOrtbBid{&bid1p123},
					},
					"imp2": {
						"appnexus": []*entities.PbsOrtbBid{&bid2p123},
						"rubicon":  []*entities.PbsOrtbBid{&bid2p144},
					},
				},
			},
		},
		{
			description: "Basic auction with deals, no preference",
			seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{&bid1p123},
				},
				"rubicon": {
					Bids: []*entities.PbsOrtbBid{&bid1p088d},
				},
			},
			numImps:     1,
			preferDeals: false,
			expectedAuction: auction{
				winningBids: map[string]*entities.PbsOrtbBid{
					"imp1": &bid1p123,
				},
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p123},
						"rubicon":  []*entities.PbsOrtbBid{&bid1p088d},
					},
				},
			},
		},
		{
			description: "Basic auction with deals, prefer deals",
			seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{&bid1p123},
				},
				"rubicon": {
					Bids: []*entities.PbsOrtbBid{&bid1p088d},
				},
			},
			numImps:     1,
			preferDeals: true,
			expectedAuction: auction{
				winningBids: map[string]*entities.PbsOrtbBid{
					"imp1": &bid1p088d,
				},
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p123},
						"rubicon":  []*entities.PbsOrtbBid{&bid1p088d},
					},
				},
			},
		},
		{
			description: "Auction with 2 deals",
			seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{&bid1p166d},
				},
				"rubicon": {
					Bids: []*entities.PbsOrtbBid{&bid1p088d},
				},
			},
			numImps:     1,
			preferDeals: true,
			expectedAuction: auction{
				winningBids: map[string]*entities.PbsOrtbBid{
					"imp1": &bid1p166d,
				},
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p166d},
						"rubicon":  []*entities.PbsOrtbBid{&bid1p088d},
					},
				},
			},
		},
		{
			description: "Auction with 3 bids and 2 deals",
			seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{&bid1p166d},
				},
				"rubicon": {
					Bids: []*entities.PbsOrtbBid{&bid1p088d},
				},
				"openx": {
					Bids: []*entities.PbsOrtbBid{&bid1p230},
				},
			},
			numImps:     1,
			preferDeals: true,
			expectedAuction: auction{
				winningBids: map[string]*entities.PbsOrtbBid{
					"imp1": &bid1p166d,
				},
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p166d},
						"rubicon":  []*entities.PbsOrtbBid{&bid1p088d},
						"openx":    []*entities.PbsOrtbBid{&bid1p230},
					},
				},
			},
		},
		{
			description: "Auction with 3 bids and 2 deals - multiple bids under each seatBids",
			seatBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"appnexus": {
					Bids: []*entities.PbsOrtbBid{&bid1p166d, &bid1p077, &bid2p123, &bid2p144},
				},
				"pubmatic": {
					Bids: []*entities.PbsOrtbBid{&bid1p088d, &bid1p123, &bid2p155, &bid2p166},
				},
			},
			numImps:     1,
			preferDeals: true,
			expectedAuction: auction{
				winningBids: map[string]*entities.PbsOrtbBid{
					"imp1": &bid1p166d,
					"imp2": &bid2p166,
				},
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p166d, &bid1p077},
						"pubmatic": []*entities.PbsOrtbBid{&bid1p088d, &bid1p123},
					},
					"imp2": {
						"appnexus": []*entities.PbsOrtbBid{&bid2p123, &bid2p144},
						"pubmatic": []*entities.PbsOrtbBid{&bid2p155, &bid2p166},
					},
				},
			},
		},
	}

	for _, test := range tests {
		auc := newAuction(test.seatBids, test.numImps, test.preferDeals)

		assert.Equal(t, test.expectedAuction, *auc, test.description)
	}

}

func TestValidateAndUpdateMultiBid(t *testing.T) {
	// create new bids for new test cases since the last one changes a few bids. Ex marks bid1p001.Bid = nil
	bid1p001 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp1",
			Price: 0.01,
		},
	}
	bid1p077 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp1",
			Price: 0.77,
		},
	}
	bid1p123 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp1",
			Price: 1.23,
		},
	}
	bid1p088d := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID:  "imp1",
			Price:  0.88,
			DealID: "SpecialDeal",
		},
	}
	bid1p166d := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID:  "imp1",
			Price:  1.66,
			DealID: "BigDeal",
		},
	}
	bid2p123 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp2",
			Price: 1.23,
		},
	}
	bid2p144 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp2",
			Price: 1.44,
		},
	}
	bid2p155 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp2",
			Price: 1.55,
		},
	}
	bid2p166 := entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ImpID: "imp2",
			Price: 1.66,
		},
	}

	type fields struct {
		winningBids     map[string]*entities.PbsOrtbBid
		allBidsByBidder map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid
		roundedPrices   map[*entities.PbsOrtbBid]string
		cacheIds        map[*openrtb2.Bid]string
		vastCacheIds    map[*openrtb2.Bid]string
	}
	type args struct {
		adapterBids            map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		preferDeals            bool
		accountDefaultBidLimit int
	}
	type want struct {
		allBidsByBidder map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid
		adapterBids     map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
	}
	tests := []struct {
		description string
		fields      fields
		args        args
		want        want
	}{
		{
			description: "DefaultBidLimit is 0 (default value)",
			fields: fields{
				winningBids: map[string]*entities.PbsOrtbBid{
					"imp1": &bid1p166d,
					"imp2": &bid2p166,
				},
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p001, &bid1p166d, &bid1p077},
						"pubmatic": []*entities.PbsOrtbBid{&bid1p088d, &bid1p123},
					},
					"imp2": {
						"appnexus": []*entities.PbsOrtbBid{&bid2p123, &bid2p144},
						"pubmatic": []*entities.PbsOrtbBid{&bid2p155, &bid2p166},
					},
				},
			},
			args: args{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{&bid1p001, &bid1p166d, &bid1p077, &bid2p123, &bid2p144},
					},
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{&bid1p088d, &bid1p123, &bid2p155, &bid2p166},
					},
				},
				accountDefaultBidLimit: 0,
				preferDeals:            true,
			},
			want: want{
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p166d, &bid1p077, &bid1p001},
						"pubmatic": []*entities.PbsOrtbBid{&bid1p088d, &bid1p123},
					},
					"imp2": {
						"appnexus": []*entities.PbsOrtbBid{&bid2p144, &bid2p123},
						"pubmatic": []*entities.PbsOrtbBid{&bid2p166, &bid2p155},
					},
				},
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{&bid1p001, &bid1p166d, &bid1p077, &bid2p123, &bid2p144},
					},
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{&bid1p088d, &bid1p123, &bid2p155, &bid2p166},
					},
				},
			},
		},
		{
			description: "Adapters bid count per imp within DefaultBidLimit",
			fields: fields{
				winningBids: map[string]*entities.PbsOrtbBid{
					"imp1": &bid1p166d,
					"imp2": &bid2p166,
				},
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p001, &bid1p166d, &bid1p077},
						"pubmatic": []*entities.PbsOrtbBid{&bid1p088d, &bid1p123},
					},
					"imp2": {
						"appnexus": []*entities.PbsOrtbBid{&bid2p123, &bid2p144},
						"pubmatic": []*entities.PbsOrtbBid{&bid2p155, &bid2p166},
					},
				},
			},
			args: args{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{&bid1p001, &bid1p166d, &bid1p077, &bid2p123, &bid2p144},
					},
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{&bid1p088d, &bid1p123, &bid2p155, &bid2p166},
					},
				},
				accountDefaultBidLimit: 3,
				preferDeals:            true,
			},
			want: want{
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p166d, &bid1p077, &bid1p001},
						"pubmatic": []*entities.PbsOrtbBid{&bid1p088d, &bid1p123},
					},
					"imp2": {
						"appnexus": []*entities.PbsOrtbBid{&bid2p144, &bid2p123},
						"pubmatic": []*entities.PbsOrtbBid{&bid2p166, &bid2p155},
					},
				},
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{&bid1p001, &bid1p166d, &bid1p077, &bid2p123, &bid2p144},
					},
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{&bid1p088d, &bid1p123, &bid2p155, &bid2p166},
					},
				},
			},
		},
		{
			description: "Adapters bid count per imp more than DefaultBidLimit",
			fields: fields{
				winningBids: map[string]*entities.PbsOrtbBid{
					"imp1": &bid1p166d,
					"imp2": &bid2p166,
				},
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p001, &bid1p166d, &bid1p077},
						"pubmatic": []*entities.PbsOrtbBid{&bid1p088d, &bid1p123},
					},
					"imp2": {
						"appnexus": []*entities.PbsOrtbBid{&bid2p123, &bid2p144},
						"pubmatic": []*entities.PbsOrtbBid{&bid2p155, &bid2p166},
					},
				},
			},
			args: args{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{&bid1p001, &bid1p166d, &bid1p077, &bid2p123, &bid2p144},
					},
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{&bid1p088d, &bid1p123, &bid2p155, &bid2p166},
					},
				},
				accountDefaultBidLimit: 2,
				preferDeals:            true,
			},
			want: want{
				allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
					"imp1": {
						"appnexus": []*entities.PbsOrtbBid{&bid1p166d, &bid1p077},
						"pubmatic": []*entities.PbsOrtbBid{&bid1p088d, &bid1p123},
					},
					"imp2": {
						"appnexus": []*entities.PbsOrtbBid{&bid2p144, &bid2p123},
						"pubmatic": []*entities.PbsOrtbBid{&bid2p166, &bid2p155},
					},
				},
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"appnexus": {
						Bids: []*entities.PbsOrtbBid{&bid1p166d, &bid1p077, &bid2p123, &bid2p144},
					},
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{&bid1p088d, &bid1p123, &bid2p155, &bid2p166},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			a := &auction{
				winningBids:     tt.fields.winningBids,
				allBidsByBidder: tt.fields.allBidsByBidder,
				roundedPrices:   tt.fields.roundedPrices,
				cacheIds:        tt.fields.cacheIds,
				vastCacheIds:    tt.fields.vastCacheIds,
			}
			a.validateAndUpdateMultiBid(tt.args.adapterBids, tt.args.preferDeals, tt.args.accountDefaultBidLimit)
			assert.Equal(t, tt.want.allBidsByBidder, tt.fields.allBidsByBidder, tt.description)
			assert.Equal(t, tt.want.adapterBids, tt.args.adapterBids, tt.description)
		})
	}
}

type cacheSpec struct {
	BidRequest                  openrtb2.BidRequest             `json:"bidRequest"`
	PbsBids                     []pbsBid                        `json:"pbsBids"`
	ExpectedCacheables          []prebid_cache_client.Cacheable `json:"expectedCacheables"`
	DefaultTTLs                 config.DefaultTTLs              `json:"defaultTTLs"`
	TargetDataIncludeWinners    bool                            `json:"targetDataIncludeWinners"`
	TargetDataIncludeBidderKeys bool                            `json:"targetDataIncludeBidderKeys"`
	TargetDataIncludeCacheBids  bool                            `json:"targetDataIncludeCacheBids"`
	TargetDataIncludeCacheVast  bool                            `json:"targetDataIncludeCacheVast"`
	EventsDataEnabledForAccount bool                            `json:"eventsDataEnabledForAccount"`
	EventsDataEnabledForRequest bool                            `json:"eventsDataEnabledForRequest"`
	DebugLog                    DebugLog                        `json:"debugLog,omitempty"`
}

type pbsBid struct {
	Bid     *openrtb2.Bid          `json:"bid"`
	BidType openrtb_ext.BidType    `json:"bidType"`
	Bidder  openrtb_ext.BidderName `json:"bidder"`
}

type mockCache struct {
	scheme string
	host   string
	path   string
	items  []prebid_cache_client.Cacheable
}

func (c *mockCache) GetExtCacheData() (scheme string, host string, path string) {
	return c.scheme, c.host, c.path
}

func (c *mockCache) GetPutUrl() string {
	return ""
}

func (c *mockCache) PutJson(ctx context.Context, values []prebid_cache_client.Cacheable) ([]string, []error) {
	c.items = values
	return []string{"", "", "", "", ""}, nil
}
