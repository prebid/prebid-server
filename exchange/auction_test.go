package exchange

import (
	"context"
	"encoding/json"
	"testing"

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

func TestDoCache(t *testing.T) {
	bidRequest := &openrtb.BidRequest{
		Imp: []openrtb.Imp{
			openrtb.Imp{
				ID:  "oneImp",
				Exp: 300,
			},
			openrtb.Imp{
				ID: "twoImp",
			},
		},
	}
	winningBidsByBidder := make(map[string]map[openrtb_ext.BidderName]*pbsOrtbBid)
	roundedPrices := make(map[*pbsOrtbBid]string)
	winningBidsByBidder["oneImp"] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
	winningBidsByBidder["oneImp"][openrtb_ext.BidderAppnexus] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			Price: 7.64,
			Exp:   600,
		},
	}
	roundedPrices[winningBidsByBidder["oneImp"][openrtb_ext.BidderAppnexus]] = "7.64"
	winningBidsByBidder["oneImp"][openrtb_ext.BidderPubmatic] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			Price: 5.64,
			Exp:   200,
		},
	}
	roundedPrices[winningBidsByBidder["oneImp"][openrtb_ext.BidderPubmatic]] = "5.64"
	winningBidsByBidder["oneImp"][openrtb_ext.BidderOpenx] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			Price: 2.3,
		},
	}
	roundedPrices[winningBidsByBidder["oneImp"][openrtb_ext.BidderOpenx]] = "2.3"
	winningBidsByBidder["twoImp"] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
	winningBidsByBidder["twoImp"][openrtb_ext.BidderAppnexus] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			Price: 1.64,
		},
	}
	roundedPrices[winningBidsByBidder["twoImp"][openrtb_ext.BidderAppnexus]] = "1.64"
	winningBidsByBidder["twoImp"][openrtb_ext.BidderRubicon] = &pbsOrtbBid{
		bid: &openrtb.Bid{
			Price: 7.64,
			Exp:   900,
		},
	}
	roundedPrices[winningBidsByBidder["twoImp"][openrtb_ext.BidderRubicon]] = "7.64"
	testAuction := &auction{
		winningBidsByBidder: winningBidsByBidder,
	}
	ctx := context.Background()
	cache := &mockCache{}

	testAuction.doCache(ctx, cache, true, false, bidRequest, 60)
	json1, _ := json.Marshal(winningBidsByBidder["oneImp"][openrtb_ext.BidderAppnexus])
	cacheable1 := prebid_cache_client.Cacheable{
		Type:       prebid_cache_client.TypeJSON,
		TTLSeconds: 660,
		Data:       json1,
	}
	assert.Contains(t, "AppNexus1 cacheable not found", cache.items, cacheable1)
}

type mockCache struct {
	items []prebid_cache_client.Cacheable
}

func (c *mockCache) PutJson(ctx context.Context, values []prebid_cache_client.Cacheable) []string {
	c.items = values
	return []string{"", "", "", "", ""}
}
