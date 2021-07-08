package exchange

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/prebid_cache_client"

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
		if specFiles, err := ioutil.ReadDir(dir); err == nil {
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
	specData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %v", filename, err)
	}

	var spec cacheSpec
	if err := json.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal JSON from file: %v", err)
	}

	return &spec, nil
}

// runCacheSpec cycles through the bids found in the json test cases and
// finds the highest bid of every Imp, then tests doCache() with resulting auction object
func runCacheSpec(t *testing.T, fileDisplayName string, specData *cacheSpec) {
	var bid *pbsOrtbBid
	winningBidsByImp := make(map[string]*pbsOrtbBid)
	winningBidsByBidder := make(map[string]map[openrtb_ext.BidderName]*pbsOrtbBid)
	roundedPrices := make(map[*pbsOrtbBid]string)
	bidCategory := make(map[string]string)

	// Traverse through the bid list found in the parsed in Json file
	for _, pbsBid := range specData.PbsBids {
		bid = &pbsOrtbBid{
			bid:     pbsBid.Bid,
			bidType: pbsBid.BidType,
		}
		cpm := bid.bid.Price

		// Map this bid if it's the highest we've seen from this Imp so far
		wbid, ok := winningBidsByImp[bid.bid.ImpID]
		if !ok || cpm > wbid.bid.Price {
			winningBidsByImp[bid.bid.ImpID] = bid
		}

		// Map this bid if it's the highest we've seen from this bidder so far
		if _, ok := winningBidsByBidder[bid.bid.ImpID]; ok {
			bestSoFar, ok := winningBidsByBidder[bid.bid.ImpID][pbsBid.Bidder]
			if !ok || cpm > bestSoFar.bid.Price {
				winningBidsByBidder[bid.bid.ImpID][pbsBid.Bidder] = bid
			}
		} else {
			winningBidsByBidder[bid.bid.ImpID] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
			winningBidsByBidder[bid.bid.ImpID][pbsBid.Bidder] = bid
		}

		if len(pbsBid.Bid.Cat) == 1 {
			bidCategory[pbsBid.Bid.ID] = pbsBid.Bid.Cat[0]
		}
		roundedPrices[bid] = strconv.FormatFloat(bid.bid.Price, 'f', 2, 64)
	}

	ctx := context.Background()
	cache := &mockCache{}

	targData := &targetData{
		priceGranularity: openrtb_ext.PriceGranularity{
			Precision: 2,
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
		winningBids:         winningBidsByImp,
		winningBidsByBidder: winningBidsByBidder,
		roundedPrices:       roundedPrices,
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
			if err := json.Unmarshal(expectedCacheable.Data, &expectedData); err != nil {
				t.Fatalf("Failed to decode expectedCacheables[%d].value: %v", i, err)
			}
			if s, ok := expectedData.(string); ok && expectedCacheable.Type == prebid_cache_client.TypeJSON {
				// decode again if we have pre-encoded json string values
				if err := json.Unmarshal([]byte(s), &expectedData); err != nil {
					t.Fatalf("Failed to re-decode expectedCacheables[%d].value :%v", i, err)
				}
			}
			for j, cachedItem := range cache.items {
				var actualData interface{}
				if err := json.Unmarshal(cachedItem.Data, &actualData); err != nil {
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
	bid1p077 := pbsOrtbBid{
		bid: &openrtb2.Bid{
			ImpID: "imp1",
			Price: 0.77,
		},
	}
	bid1p123 := pbsOrtbBid{
		bid: &openrtb2.Bid{
			ImpID: "imp1",
			Price: 1.23,
		},
	}
	bid1p230 := pbsOrtbBid{
		bid: &openrtb2.Bid{
			ImpID: "imp1",
			Price: 2.30,
		},
	}
	bid1p088d := pbsOrtbBid{
		bid: &openrtb2.Bid{
			ImpID:  "imp1",
			Price:  0.88,
			DealID: "SpecialDeal",
		},
	}
	bid1p166d := pbsOrtbBid{
		bid: &openrtb2.Bid{
			ImpID:  "imp1",
			Price:  1.66,
			DealID: "BigDeal",
		},
	}
	bid2p123 := pbsOrtbBid{
		bid: &openrtb2.Bid{
			ImpID: "imp2",
			Price: 1.23,
		},
	}
	bid2p144 := pbsOrtbBid{
		bid: &openrtb2.Bid{
			ImpID: "imp2",
			Price: 1.44,
		},
	}
	tests := []struct {
		description     string
		seatBids        map[openrtb_ext.BidderName]*pbsOrtbSeatBid
		numImps         int
		preferDeals     bool
		expectedAuction auction
	}{
		{
			description: "Basic auction test",
			seatBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"appnexus": {
					bids: []*pbsOrtbBid{&bid1p123},
				},
				"rubicon": {
					bids: []*pbsOrtbBid{&bid1p230},
				},
			},
			numImps:     1,
			preferDeals: false,
			expectedAuction: auction{
				winningBids: map[string]*pbsOrtbBid{
					"imp1": &bid1p230,
				},
				winningBidsByBidder: map[string]map[openrtb_ext.BidderName]*pbsOrtbBid{
					"imp1": {
						"appnexus": &bid1p123,
						"rubicon":  &bid1p230,
					},
				},
			},
		},
		{
			description: "Multi-imp auction",
			seatBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"appnexus": {
					bids: []*pbsOrtbBid{&bid1p230, &bid2p123},
				},
				"rubicon": {
					bids: []*pbsOrtbBid{&bid1p077, &bid2p144},
				},
				"openx": {
					bids: []*pbsOrtbBid{&bid1p123},
				},
			},
			numImps:     2,
			preferDeals: false,
			expectedAuction: auction{
				winningBids: map[string]*pbsOrtbBid{
					"imp1": &bid1p230,
					"imp2": &bid2p144,
				},
				winningBidsByBidder: map[string]map[openrtb_ext.BidderName]*pbsOrtbBid{
					"imp1": {
						"appnexus": &bid1p230,
						"rubicon":  &bid1p077,
						"openx":    &bid1p123,
					},
					"imp2": {
						"appnexus": &bid2p123,
						"rubicon":  &bid2p144,
					},
				},
			},
		},
		{
			description: "Basic auction with deals, no preference",
			seatBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"appnexus": {
					bids: []*pbsOrtbBid{&bid1p123},
				},
				"rubicon": {
					bids: []*pbsOrtbBid{&bid1p088d},
				},
			},
			numImps:     1,
			preferDeals: false,
			expectedAuction: auction{
				winningBids: map[string]*pbsOrtbBid{
					"imp1": &bid1p123,
				},
				winningBidsByBidder: map[string]map[openrtb_ext.BidderName]*pbsOrtbBid{
					"imp1": {
						"appnexus": &bid1p123,
						"rubicon":  &bid1p088d,
					},
				},
			},
		},
		{
			description: "Basic auction with deals, prefer deals",
			seatBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"appnexus": {
					bids: []*pbsOrtbBid{&bid1p123},
				},
				"rubicon": {
					bids: []*pbsOrtbBid{&bid1p088d},
				},
			},
			numImps:     1,
			preferDeals: true,
			expectedAuction: auction{
				winningBids: map[string]*pbsOrtbBid{
					"imp1": &bid1p088d,
				},
				winningBidsByBidder: map[string]map[openrtb_ext.BidderName]*pbsOrtbBid{
					"imp1": {
						"appnexus": &bid1p123,
						"rubicon":  &bid1p088d,
					},
				},
			},
		},
		{
			description: "Auction with 2 deals",
			seatBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"appnexus": {
					bids: []*pbsOrtbBid{&bid1p166d},
				},
				"rubicon": {
					bids: []*pbsOrtbBid{&bid1p088d},
				},
			},
			numImps:     1,
			preferDeals: true,
			expectedAuction: auction{
				winningBids: map[string]*pbsOrtbBid{
					"imp1": &bid1p166d,
				},
				winningBidsByBidder: map[string]map[openrtb_ext.BidderName]*pbsOrtbBid{
					"imp1": {
						"appnexus": &bid1p166d,
						"rubicon":  &bid1p088d,
					},
				},
			},
		},
		{
			description: "Auction with 3 bids and 2 deals",
			seatBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				"appnexus": {
					bids: []*pbsOrtbBid{&bid1p166d},
				},
				"rubicon": {
					bids: []*pbsOrtbBid{&bid1p088d},
				},
				"openx": {
					bids: []*pbsOrtbBid{&bid1p230},
				},
			},
			numImps:     1,
			preferDeals: true,
			expectedAuction: auction{
				winningBids: map[string]*pbsOrtbBid{
					"imp1": &bid1p166d,
				},
				winningBidsByBidder: map[string]map[openrtb_ext.BidderName]*pbsOrtbBid{
					"imp1": {
						"appnexus": &bid1p166d,
						"rubicon":  &bid1p088d,
						"openx":    &bid1p230,
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

type cacheComparator struct {
	freq         int
	expectedKeys []string
	actualKeys   []string
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
