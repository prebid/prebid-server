package exchange

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/prebid_cache_client"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

func TestMakeVASTGiven(t *testing.T) {
	const expect = `<VAST version="3.0"></VAST>`
	bid := &openrtb.Bid{
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
	bid := &openrtb.Bid{
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
	if specFiles, err := ioutil.ReadDir("./cachetest"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./cachetest/" + specFile.Name()
			fileDisplayName := "exchange/cachetest/" + specFile.Name()
			specData, err := loadCacheSpec(fileName)
			if err != nil {
				t.Fatalf("Failed to load contents of file %s: %v", fileDisplayName, err)
			}

			runCacheSpec(t, fileDisplayName, specData)
		}
	} else {
		t.Fatalf("Failed to read contents of directory exchange/cachetest/: %v", err)
	}
}

// TestCacheJSON executes tests for all the *.json files in customcachekeytest.
// customcachekey.json test here verifies custom cache key is used for vast video
func TestCustomCacheKeyJSON(t *testing.T) {
	if specFiles, err := ioutil.ReadDir("./customcachekeytest"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./customcachekeytest/" + specFile.Name()
			fileDisplayName := "exchange/customcachekeytest/" + specFile.Name()
			specData, err := loadCacheSpec(fileName)
			if err != nil {
				t.Fatalf("Failed to load contents of file %s: %v", fileDisplayName, err)
			}

			runCacheSpec(t, fileDisplayName, specData)
		}
	} else {
		t.Fatalf("Failed to read contents of directory exchange/customcachekeytest/: %v", err)
	}
}

// TestMultiImpCache executes multi-Imp test cases found in *.json files in
// impcustomcachekeytest.
func TestCustomCacheKeyMultiImp(t *testing.T) {
	if specFiles, err := ioutil.ReadDir("./impcustomcachekeytest"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./impcustomcachekeytest/" + specFile.Name()
			fileDisplayName := "exchange/impcustomcachekeytest/" + specFile.Name()
			multiImpSpecData, err := loadCacheSpec(fileName)
			if err != nil {
				t.Fatalf("Failed to load contents of file %s: %v", fileDisplayName, err)
			}

			runCacheSpec(t, fileDisplayName, multiImpSpecData)
		}
	} else {
		t.Fatalf("Failed to read contents of directory exchange/customcachekeytest/: %v", err)
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

// runCacheSpec has been modified to handle multi-Imp and multi-bid Json test files,
// it cycles through the bids found in the test cases hardcoded in json files and
// finds the highest bid of every Imp.
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
	_ = testAuction.doCache(ctx, cache, targData, &specData.BidRequest, 60, &specData.DefaultTTLs, bidCategory, &specData.DebugLog)

	if len(specData.ExpectedCacheables) > len(cache.items) {
		t.Errorf("%s:  [CACHE_ERROR] Less elements were cached than expected \n", fileDisplayName)
	} else if len(specData.ExpectedCacheables) < len(cache.items) {
		t.Errorf("%s:  [CACHE_ERROR] More elements were cached than expected \n", fileDisplayName)
	} else { // len(specData.ExpectedCacheables) == len(cache.items)
		// We cached the exact number of elements we expected, now we compare them side by side in n^2
		var matched int = 0
		var formattedExpectedData string
		for i := 0; i < len(specData.ExpectedCacheables); i++ {
			if specData.ExpectedCacheables[i].Type == prebid_cache_client.TypeJSON {
				ExpectedData := strings.Replace(string(specData.ExpectedCacheables[i].Data), "\\", "", -1)
				ExpectedData = strings.Replace(ExpectedData, " ", "", -1)
				formattedExpectedData = ExpectedData[1 : len(ExpectedData)-1]
			} else {
				formattedExpectedData = string(specData.ExpectedCacheables[i].Data)
			}
			for j := 0; j < len(cache.items); j++ {
				if formattedExpectedData == string(cache.items[j].Data) &&
					specData.ExpectedCacheables[i].TTLSeconds == cache.items[j].TTLSeconds &&
					specData.ExpectedCacheables[i].Type == cache.items[j].Type &&
					len(specData.ExpectedCacheables[i].Key) <= len(cache.items[j].Key) &&
					specData.ExpectedCacheables[i].Key == cache.items[j].Key[:len(specData.ExpectedCacheables[i].Key)] {
					matched++
				}
			}
		}
		if matched != len(specData.ExpectedCacheables) {
			t.Errorf("%s: [CACHE_ERROR] One or more keys were not cached as we expected \n", fileDisplayName)
			t.FailNow()
		}
	}
}

type cacheSpec struct {
	BidRequest                  openrtb.BidRequest              `json:"bidRequest"`
	PbsBids                     []pbsBid                        `json:"pbsBids"`
	ExpectedCacheables          []prebid_cache_client.Cacheable `json:"expectedCacheables"`
	DefaultTTLs                 config.DefaultTTLs              `json:"defaultTTLs"`
	TargetDataIncludeWinners    bool                            `json:"targetDataIncludeWinners"`
	TargetDataIncludeBidderKeys bool                            `json:"targetDataIncludeBidderKeys"`
	TargetDataIncludeCacheBids  bool                            `json:"targetDataIncludeCacheBids"`
	TargetDataIncludeCacheVast  bool                            `json:"targetDataIncludeCacheVast"`
	DebugLog                    DebugLog                        `json:"debugLog,omitempty"`
}

type pbsBid struct {
	Bid     *openrtb.Bid           `json:"bid"`
	BidType openrtb_ext.BidType    `json:"bidType"`
	Bidder  openrtb_ext.BidderName `json:"bidder"`
}

type mockCache struct {
	items []prebid_cache_client.Cacheable
}

type cacheComparator struct {
	freq         int
	expectedKeys []string
	actualKeys   []string
}

func (c *mockCache) GetExtCacheData() (string, string) {
	return "", ""
}
func (c *mockCache) GetPutUrl() string {
	return ""
}
func (c *mockCache) PutJson(ctx context.Context, values []prebid_cache_client.Cacheable) ([]string, []error) {
	c.items = values
	return []string{"", "", "", "", ""}, nil
}
