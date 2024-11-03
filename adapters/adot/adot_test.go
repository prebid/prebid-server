package adot

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const testsBidderEndpoint = "https://dsp.adotmob.com/headerbidding{PUBLISHER_PATH}/bidrequest"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdot, config.Adapter{
		Endpoint: testsBidderEndpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adottest", bidder)
}

func TestMediaTypeError(t *testing.T) {
	_, err := getMediaTypeForBid(nil)

	assert.Error(t, err)

	byteInvalid, _ := json.Marshal(&adotBidExt{Adot: bidExt{"invalid"}})
	_, err = getMediaTypeForBid(&openrtb2.Bid{Ext: json.RawMessage(byteInvalid)})

	assert.Error(t, err)
}

func TestBidResponseNoContent(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdot, config.Adapter{
		Endpoint: "https://dsp.adotmob.com/headerbidding{PUBLISHER_PATH}/bidrequest"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidResponse, err := bidder.MakeBids(nil, nil, &adapters.ResponseData{StatusCode: 204})
	if bidResponse != nil {
		t.Fatalf("the bid response should be nil since the bidder status is No Content")
	} else if err != nil {
		t.Fatalf("the error should be nil since the bidder status is 204 : No Content")
	}
}

func TestMediaTypeForBid(t *testing.T) {
	byteBanner, _ := json.Marshal(&adotBidExt{Adot: bidExt{"banner"}})
	byteVideo, _ := json.Marshal(&adotBidExt{Adot: bidExt{"video"}})
	byteNative, _ := json.Marshal(&adotBidExt{Adot: bidExt{"native"}})

	bidTypeBanner, _ := getMediaTypeForBid(&openrtb2.Bid{Ext: json.RawMessage(byteBanner)})
	if bidTypeBanner != openrtb_ext.BidTypeBanner {
		t.Errorf("the type is not the valid one. actual: %v, expected: %v", bidTypeBanner, openrtb_ext.BidTypeBanner)
	}

	bidTypeVideo, _ := getMediaTypeForBid(&openrtb2.Bid{Ext: json.RawMessage(byteVideo)})
	if bidTypeVideo != openrtb_ext.BidTypeVideo {
		t.Errorf("the type is not the valid one. actual: %v, expected: %v", bidTypeVideo, openrtb_ext.BidTypeVideo)
	}

	bidTypeNative, _ := getMediaTypeForBid(&openrtb2.Bid{Ext: json.RawMessage(byteNative)})
	if bidTypeNative != openrtb_ext.BidTypeNative {
		t.Errorf("the type is not the valid one. actual: %v, expected: %v", bidTypeNative, openrtb_ext.BidTypeVideo)
	}
}

func TestResolveMacros(t *testing.T) {
	bid := &openrtb2.Bid{AdM: "adm:imp_${AUCTION_PRICE} amd:creativeview_${AUCTION_PRICE}", NURL: "nurl_${AUCTION_PRICE}", Price: 123.45}
	resolveMacros(bid)
	assert.Equal(t, "adm:imp_123.45 amd:creativeview_123.45", bid.AdM)
	assert.Equal(t, "nurl_123.45", bid.NURL)
}

func TestGetImpAdotExt(t *testing.T) {
	ext := &openrtb2.Imp{Ext: json.RawMessage(`{"bidder":{"publisherPath": "/hubvisor"}}`)}
	adotExt := getImpAdotExt(ext)
	assert.Equal(t, adotExt.PublisherPath, "/hubvisor")

	emptyBidderExt := &openrtb2.Imp{Ext: json.RawMessage(`{"bidder":{}}`)}
	emptyAdotBidderExt := getImpAdotExt(emptyBidderExt)
	assert.NotNil(t, emptyAdotBidderExt)
	assert.Equal(t, emptyAdotBidderExt.PublisherPath, "")

	emptyExt := &openrtb2.Imp{Ext: json.RawMessage(`{}`)}
	emptyAdotExt := getImpAdotExt(emptyExt)
	assert.Nil(t, emptyAdotExt)
}
