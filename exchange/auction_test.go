package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

			runCacheSpec(t, fileDisplayName, specData, true, false)
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

			runCacheSpec(t, fileDisplayName, specData, false, true)
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

			runCacheSpec(t, fileDisplayName, multiImpSpecData, false, true)
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

func runCacheSpec(t *testing.T, fileDisplayName string, specData *cacheSpec, bids bool, vast bool) {
	// bid := make([]pbsOrtbBid, 5)
	var bid *pbsOrtbBid
	winningBids := make(map[string]*pbsOrtbBid)
	winningBidsByBidder := make(map[string]map[openrtb_ext.BidderName]*pbsOrtbBid)
	roundedPrices := make(map[*pbsOrtbBid]string)
	bidCategory := make(map[string]string)
	for i, pbsBid := range specData.PbsBids {
		if _, ok := winningBidsByBidder[pbsBid.Bid.ImpID]; !ok {
			winningBidsByBidder[pbsBid.Bid.ImpID] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
		}
		bid = &pbsOrtbBid{
			bid:     pbsBid.Bid,
			bidType: pbsBid.BidType,
		}
		if _, ok := winningBids[pbsBid.Bid.ImpID]; !ok {
			winningBids[pbsBid.Bid.ImpID] = bid
		}
		winningBidsByBidder[pbsBid.Bid.ImpID][pbsBid.Bidder] = bid
		if len(pbsBid.Bid.Cat) == 1 {
			bidCategory[pbsBid.Bid.ImpID] = pbsBid.Bid.Cat[0]
		}
		roundedPrices[bid] = strconv.FormatFloat(bid.bid.Price, 'f', 2, 64)
		// Marshal the bid for the expected cacheables
		cjson, _ := json.Marshal(bid.bid)
		specData.ExpectedCacheables[i].Data = cjson
		specData.ExpectedCacheables[i].Key = pbsBid.Bid.Cat[0]
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
		includeCacheBids:  specData.TargetDataIncludeCacheBids || bids,
		includeCacheVast:  specData.TargetDataIncludeCacheVast || vast,
	}

	testAuction := &auction{
		winningBids:         winningBids,
		winningBidsByBidder: winningBidsByBidder,
		roundedPrices:       roundedPrices,
	}
	_ = testAuction.doCache(ctx, cache, targData, &specData.BidRequest, 60, &specData.DefaultTTLs, bidCategory)

	//asserting we have items with the same Data and TTLSeconds fields
	if !vast {
		if len(specData.ExpectedCacheables) == len(cache.items) {
			var ht map[string]int = make(map[string]int)
			var compareString string

			for _, cExpected := range specData.ExpectedCacheables {
				compareString = string(cExpected.Data) + string(cExpected.TTLSeconds)
				ht[compareString] += 1
			}
			for _, cFound := range cache.items {
				compareString = string(cFound.Data) + string(cFound.TTLSeconds)
				ht[compareString] -= 1
			}
			for _, freq := range ht {
				if freq > 0 {
					t.Errorf("%s:  Less elements were cached than expected \n", fileDisplayName)
				} else if freq < 0 {
					t.Errorf("%s:  More elements were cached than expected \n", fileDisplayName)
				}
			}
		}
	}
	//asserting we generated the Keys we expected
	for _, cExpected := range specData.ExpectedCacheables {
		found := false
		keyNotFound := ""
		for _, cFound := range cache.items {
			// make sure Key value is as expected
			if cExpected.Key == "" || strings.HasPrefix(cExpected.Key, cFound.Key) {
				found = true
				keyNotFound = cExpected.Key
			}
		}
		if !found {
			t.Errorf("Key \"%s\" was expected to get cached with a uuid but it was not\n", keyNotFound)
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
}

type pbsBid struct {
	Bid     *openrtb.Bid           `json:"bid"`
	BidType openrtb_ext.BidType    `json:"bidType"`
	Bidder  openrtb_ext.BidderName `json:"bidder"`
}

type mockCache struct {
	items []prebid_cache_client.Cacheable
}

func (c *mockCache) PutJson(ctx context.Context, values []prebid_cache_client.Cacheable) ([]string, []error) {
	c.items = values
	return []string{"", "", "", "", ""}, nil
}
