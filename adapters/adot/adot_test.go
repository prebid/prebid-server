package adot

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const testsBidderEndpoint = "https://dsp.adotmob.com/headerbidding/bidrequest"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdot, config.Adapter{
		Endpoint: testsBidderEndpoint})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adottest", bidder)
}

//Test the media type error
func TestMediaTypeError(t *testing.T) {
	_, err := getMediaTypeForBid(nil)

	assert.Error(t, err)

	byteInvalid, _ := json.Marshal(&adotBidExt{Adot: bidExt{"invalid"}})
	_, err = getMediaTypeForBid(&openrtb2.Bid{Ext: json.RawMessage(byteInvalid)})

	assert.Error(t, err)
}

//Test the bid response when the bidder return a status code 204
func TestBidResponseNoContent(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdot, config.Adapter{
		Endpoint: "https://dsp.adotmob.com/headerbidding/bidrequest"})

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

//Test the media type for a bid response
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
