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

func runCacheSpec(t *testing.T, fileDisplayName string, specData *cacheSpec) {
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
		winningBids:         winningBids,
		winningBidsByBidder: winningBidsByBidder,
		roundedPrices:       roundedPrices,
	}
	_ = testAuction.doCache(ctx, cache, targData, &specData.BidRequest, 60, &specData.DefaultTTLs, bidCategory)

	if len(specData.ExpectedCacheables) > len(cache.items) {
		//t.Errorf("%s:  Less elements were cached than expected \n", fileDisplayName)
		t.Errorf("%s:  [CACHE_ERROR] Less elements were cached than expected \n", fileDisplayName)
	} else if len(specData.ExpectedCacheables) < len(cache.items) {
		t.Errorf("%s:  [CACHE_ERROR] More elements were cached than expected \n", fileDisplayName)
	} else { // len(specData.ExpectedCacheables) == len(cache.items)
		var ht map[string]*cacheComparator = make(map[string]*cacheComparator)
		var formattedData string
		var compareString string

		for _, cExpected := range specData.ExpectedCacheables {
			formattedData = strings.Replace(string(cExpected.Data), "\\", "", -1)
			formattedData = strings.Replace(formattedData, " ", "", -1)
			compareString = fmt.Sprintf("%s_%d", formattedData, cExpected.TTLSeconds)
			// init cacheComparator element before hashing it. If needed
			if _, ok := ht[compareString]; !ok {
				ht[compareString] = &cacheComparator{0, make([]string, 0), make([]string, 0)}
			}
			ht[compareString].freq += 1
			if targData.includeCacheVast {
				ht[compareString].expectedKeys = append(ht[compareString].expectedKeys, cExpected.Key)
			}
		}
		for _, cFound := range cache.items {
			formattedData = strings.Replace(string(cFound.Data), "\\", "", -1)
			formattedData = strings.Replace(formattedData, " ", "", -1)
			compareString = fmt.Sprintf("\"%s\"_%d", formattedData, cFound.TTLSeconds)
			// init cacheComparator element before hashing it. If needed
			if _, ok := ht[compareString]; !ok {
				ht[compareString] = &cacheComparator{0, make([]string, 0), make([]string, 0)}
			}
			ht[compareString].freq -= 1
			if targData.includeCacheVast {
				ht[compareString].actualKeys = append(ht[compareString].actualKeys, cFound.Key)
			}
		}
		for k, cachedElements := range ht {
			if cachedElements.freq > 0 {
				t.Errorf("%s:  [CACHE_ERROR] Cache inconsistensy. Element %s was not expected to get cached\n", fileDisplayName, k)
			} else if cachedElements.freq < 0 {
				t.Errorf("%s:  [CACHE_ERROR] Cache inconsistensy. We cached some more elements (%s) than expected.\n", fileDisplayName, k)
			} else if targData.includeCacheVast {
				for i := 0; i < cachedElements.freq; i++ {
					found := false
					for j := 0; j < cachedElements.freq; j++ {
						if strings.HasPrefix(cachedElements.expectedKeys[i], cachedElements.actualKeys[j]) {
							found = true
						}
					}
					if !found {
						t.Errorf("[CACHE_ERROR] Key \"%s\" was expected to be cached but was not.\n", cachedElements.expectedKeys[i])
					}
				}
			}
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

type cacheComparator struct {
	freq         int
	expectedKeys []string
	actualKeys   []string
}

func (c *mockCache) PutJson(ctx context.Context, values []prebid_cache_client.Cacheable) ([]string, []error) {
	c.items = values
	return []string{"", "", "", "", ""}, nil
}
