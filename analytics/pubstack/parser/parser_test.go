package parser

import (
	"encoding/json"
	"github.com/magiconair/properties/assert"
	"github.com/mxmCherry/openrtb"
	"io/ioutil"
	"os"
	"testing"
)

func mapFileToObject(path string, tg interface{}) error {
	fl, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fl.Close()

	data, err := ioutil.ReadAll(fl)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(data), tg)
	if err == nil {
		return err
	}

	return nil
}

func TestParser(t *testing.T) {
	testRq := openrtb.BidRequest{}
	testRp := openrtb.BidResponse{}
	p := NewParser("test-scope")

	err := mapFileToObject("mocks/mock_openrtb_request.json", &testRq)
	assert.Equal(t, err, nil)
	err = mapFileToObject("mocks/mock_openrtb_response.json", &testRp)
	assert.Equal(t, err, nil)

	ret := p.Feed(&testRq, &testRp)
	// expect 2 auctions from 1 openrtb request
	assert.Equal(t, len(ret), 2)

	// expect requested sizes auction
	assert.Equal(t, ret[0].Sizes, []string{"970x250"})
	assert.Equal(t, ret[1].Sizes, []string{"300x250", "300x600"})

	// expect 2 bids on first auction
	assert.Equal(t, len(ret[0].BidRequests), 3)

	// expect 1 bid on first auction
	assert.Equal(t, len(ret[1].BidRequests), 1)

	// expect correct bidder codes
	// all N/A need to be changed when appnexus parsing will be working
	assert.Equal(t, ret[0].BidRequests[0].BidderCode, "N/A")
	assert.Equal(t, ret[0].BidRequests[1].BidderCode, "N/A")
	assert.Equal(t, ret[0].BidRequests[2].BidderCode, "improvedigital")
	assert.Equal(t, ret[1].BidRequests[0].BidderCode, "N/A")

	// Expect the correct ammounts in bid requests
	assert.Equal(t, ret[0].BidRequests[0].Cpm == float32(0.500000), true)
	assert.Equal(t, ret[0].BidRequests[1].Cpm == float32(0.0), true)
	assert.Equal(t, ret[0].BidRequests[2].Cpm == float32(0.51), true)
	assert.Equal(t, ret[1].BidRequests[0].Cpm == float32(0.5234), true)

	// Expect the correct state for bids
	assert.Equal(t, ret[0].BidRequests[0].State, "bid")
	assert.Equal(t, ret[0].BidRequests[1].State, "noBid")
	assert.Equal(t, ret[1].BidRequests[0].State, "bid")

	// Expect the correct scopeId
	assert.Equal(t, ret[0].ScopeId, "test-scope")
	assert.Equal(t, ret[1].ScopeId, "test-scope")
}
