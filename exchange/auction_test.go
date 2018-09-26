package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

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

func TestDoCache(t *testing.T) {
	bidRequest := &openrtb.BidRequest{
		Imp: []openrtb.Imp{
			{
				ID:  "oneImp",
				Exp: 300,
			},
			{
				ID: "twoImp",
			},
		},
	}
	bid := make([]pbsOrtbBid, 5)
	winningBidsByBidder := make(map[string]map[openrtb_ext.BidderName]*pbsOrtbBid)
	roundedPrices := make(map[*pbsOrtbBid]string)
	winningBidsByBidder["oneImp"] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
	bid[0] = pbsOrtbBid{
		bid: &openrtb.Bid{
			Price: 7.64,
			Exp:   600,
		},
	}
	winningBidsByBidder["oneImp"][openrtb_ext.BidderAppnexus] = &bid[0]
	roundedPrices[winningBidsByBidder["oneImp"][openrtb_ext.BidderAppnexus]] = "7.64"
	bid[1] = pbsOrtbBid{
		bid: &openrtb.Bid{
			Price: 5.64,
			Exp:   200,
		},
	}
	winningBidsByBidder["oneImp"][openrtb_ext.BidderPubmatic] = &bid[1]
	roundedPrices[winningBidsByBidder["oneImp"][openrtb_ext.BidderPubmatic]] = "5.64"
	bid[2] = pbsOrtbBid{
		bid: &openrtb.Bid{
			Price: 2.3,
		},
	}
	winningBidsByBidder["oneImp"][openrtb_ext.BidderOpenx] = &bid[2]
	roundedPrices[winningBidsByBidder["oneImp"][openrtb_ext.BidderOpenx]] = "2.3"
	winningBidsByBidder["twoImp"] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
	bid[3] = pbsOrtbBid{
		bid: &openrtb.Bid{
			Price: 1.64,
		},
	}
	winningBidsByBidder["twoImp"][openrtb_ext.BidderAppnexus] = &bid[3]
	roundedPrices[winningBidsByBidder["twoImp"][openrtb_ext.BidderAppnexus]] = "1.64"
	bid[4] = pbsOrtbBid{
		bid: &openrtb.Bid{
			Price: 7.64,
			Exp:   900,
		},
	}
	winningBidsByBidder["twoImp"][openrtb_ext.BidderRubicon] = &bid[4]
	roundedPrices[winningBidsByBidder["twoImp"][openrtb_ext.BidderRubicon]] = "7.64"
	testAuction := &auction{
		winningBidsByBidder: winningBidsByBidder,
	}
	ctx := context.Background()
	cache := &mockCache{}

	_ = testAuction.doCache(ctx, cache, true, false, bidRequest, 60)
	json0, _ := json.Marshal(bid[0].bid)
	json1, _ := json.Marshal(bid[1].bid)
	json2, _ := json.Marshal(bid[2].bid)
	json3, _ := json.Marshal(bid[3].bid)
	json4, _ := json.Marshal(bid[4].bid)
	cacheables := make([]prebid_cache_client.Cacheable, 5)
	cacheables[0] = prebid_cache_client.Cacheable{
		Type:       prebid_cache_client.TypeJSON,
		TTLSeconds: 360,
		Data:       json0,
	}
	cacheables[1] = prebid_cache_client.Cacheable{
		Type:       prebid_cache_client.TypeJSON,
		TTLSeconds: 260,
		Data:       json1,
	}
	cacheables[2] = prebid_cache_client.Cacheable{
		Type:       prebid_cache_client.TypeJSON,
		TTLSeconds: 360,
		Data:       json2,
	}
	cacheables[3] = prebid_cache_client.Cacheable{
		Type:       prebid_cache_client.TypeJSON,
		TTLSeconds: 0,
		Data:       json3,
	}
	cacheables[4] = prebid_cache_client.Cacheable{
		Type:       prebid_cache_client.TypeJSON,
		TTLSeconds: 960,
		Data:       json4,
	}
	found := 0

	for _, cExpected := range cacheables {
		for _, cFound := range cache.items {
			eq := jsonpatch.Equal(cExpected.Data, cFound.Data)
			if cExpected.TTLSeconds == cFound.TTLSeconds && eq {
				found++
			}
		}
	}

	if found != 5 {
		fmt.Printf("Expected:\n%v\n\n", cacheables)
		fmt.Printf("Found:\n%v\n\n", cache.items)
		t.Errorf("All expected cacheables not found. Expected 5, found %d.", found)
	}

}

type mockCache struct {
	items []prebid_cache_client.Cacheable
}

func (c *mockCache) PutJson(ctx context.Context, values []prebid_cache_client.Cacheable) ([]string, []error) {
	c.items = values
	return []string{"", "", "", "", ""}, nil
}
