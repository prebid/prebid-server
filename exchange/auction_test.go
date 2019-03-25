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

	"github.com/evanphx/json-patch"
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
		if _, ok := winningBidsByBidder[pbsBid.Bid.ID]; !ok {
			winningBidsByBidder[pbsBid.Bid.ID] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
		}
		bid = &pbsOrtbBid{
			bid:     pbsBid.Bid,
			bidType: pbsBid.BidType,
		}
		if _, ok := winningBids[pbsBid.Bid.ImpID]; !ok {
			winningBids[pbsBid.Bid.ImpID] = bid
		}
		winningBidsByBidder[pbsBid.Bid.ID][pbsBid.Bidder] = bid
		if len(pbsBid.Bid.Cat) == 1 {
			bidCategory[pbsBid.Bid.ID] = pbsBid.Bid.Cat[0]
		}
		roundedPrices[bid] = strconv.FormatFloat(bid.bid.Price, 'f', 2, 64)
		// Marshal the bid for the expected cacheables
		cjson, _ := json.Marshal(bid.bid)
		specData.ExpectedCacheables[i].Data = cjson
	}
	ctx := context.Background()
	cache := &mockCache{}

	testAuction := &auction{
		winningBids:         winningBids,
		winningBidsByBidder: winningBidsByBidder,
		roundedPrices:       roundedPrices,
	}
	_ = testAuction.doCache(ctx, cache, bids, vast, &specData.BidRequest, 60, &specData.DefaultTTLs, bidCategory)
	found := 0

	for _, cExpected := range specData.ExpectedCacheables {
		for _, cFound := range cache.items {
			// make sure Data section matches exactly
			if !vast {
				// not testing VAST XML for custom cache key test
				eq := jsonpatch.Equal(cExpected.Data, cFound.Data)
				if !eq {
					continue
				}
			}

			// make sure Key value is as expected
			keymatch := false
			if len(cExpected.Key) > 0 {
				// can only verify prefix; remainder is random
				if strings.HasPrefix(cFound.Key, cExpected.Key) {
					keymatch = true
				}
			} else {
				if len(cFound.Key) == 0 {
					// Key is expected to be empty
					keymatch = true
				}
			}
			if !keymatch {
				continue
			}

			// make sure TTLSeconds section matches exactly
			if cExpected.TTLSeconds == cFound.TTLSeconds {
				found++
			}
		}
	}

	if found != len(specData.ExpectedCacheables) {
		fmt.Printf("Expected:\n%v\n\n", specData.ExpectedCacheables)
		fmt.Printf("Found:\n%v\n\n", cache.items)
		t.Errorf("%s:  All expected cacheables not found. Expected %d, found %d.", fileDisplayName, len(specData.ExpectedCacheables), found)
	}

}

type cacheSpec struct {
	BidRequest         openrtb.BidRequest              `json:"bidRequest"`
	PbsBids            []pbsBid                        `json:"pbsBids"`
	ExpectedCacheables []prebid_cache_client.Cacheable `json:"expectedCacheables"`
	DefaultTTLs        config.DefaultTTLs              `json:"defaultTTLs"`
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
