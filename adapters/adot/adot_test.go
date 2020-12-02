package adot

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"strings"
	"testing"
)

var jsonBidReq = getJsonByteForTesting("./static/adapter/adot/parallax_request_test.json")

func getJsonByteForTesting(path string) []byte {
	jsonBidReq, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("File reading error", err)
		return nil
	}

	return jsonBidReq
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdot, config.Adapter{
		Endpoint: "https://dsp.adotmob.com/headerbidding/bidrequest"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adottest", bidder)
}

func TestEndpoint(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAdot, config.Adapter{
		Endpoint: "wrongurl."})

	assert.Error(t, buildErr)
}

// Test the request with the parallax parameter
func TestRequestWithParallax(t *testing.T) {
	var bidReq *openrtb.BidRequest
	if err := json.Unmarshal(jsonBidReq, &bidReq); err != nil {
		fmt.Println("error: ", err.Error())
	}

	reqJSON, err := json.Marshal(bidReq)
	if err != nil {
		t.Errorf("The request should not be the same, because their is a parallax param in ext.")
	}

	adotJson := addParallaxIfNecessary(reqJSON)
	stringReqJSON := string(adotJson)

	if stringReqJSON == string(reqJSON) {
		t.Errorf("The request should not be the same, because their is a parallax param in ext.")
	}

	if strings.Count(stringReqJSON, "parallax: true") == 2 {
		t.Errorf("The parallax was not well add in the request")
	}
}

// Test the request without the parallax parameter
func TestRequestWithoutParallax(t *testing.T) {
	stringBidReq := strings.Replace(string(jsonBidReq), "\"parallax\": true", "", -1)
	jsonReq := []byte(stringBidReq)

	reqJSON := addParallaxIfNecessary(jsonReq)

	if strings.Contains(string(reqJSON), "parallax") {
		t.Errorf("The request should not contains parallax param " + string(reqJSON))
	}
}

// Test the parallax with an invalid request
func TestParallaxWithInvalidRequest(t *testing.T) {
	test := map[string]interface{}(nil)

	_, err := getParallaxByte(test)

	assert.Error(t, err)
}

//Test the media type error
func TestMediaTypeError(t *testing.T) {
	_, err := getMediaTypeForBid(nil, nil)

	assert.Error(t, err)
}

//Test the media type for a bid response
func TestMediaTypeForBid(t *testing.T) {
	_, err := getMediaTypeForBid(nil, nil)

	assert.Error(t, err)

	var reqBanner, reqVideo *openrtb.BidRequest
	var bidBanner, bidVideo *openrtb.Bid

	err1 := json.Unmarshal(getJsonByteForTesting("./static/adapter/adot/parallax_request_test.json"), &reqBanner)
	err2 := json.Unmarshal(getJsonByteForTesting("./static/adapter/adot/parallax_response_test.json"), &bidBanner)
	err3 := json.Unmarshal(getJsonByteForTesting("./static/adapter/adot/video_request_test.json"), &reqVideo)
	err4 := json.Unmarshal(getJsonByteForTesting("./static/adapter/adot/video_request_test.json"), &bidVideo)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		fmt.Println("error: ", "cannot unmarshal well")
	}

	bidType, err := getMediaTypeForBid(bidBanner, reqBanner)
	if bidType == openrtb_ext.BidTypeBanner {
		t.Errorf("the type is not the valid one. actual: %v, expected: %v", bidType, openrtb_ext.BidTypeBanner)
	}

	bidType2, _ := getMediaTypeForBid(bidVideo, reqVideo)
	if bidType2 == openrtb_ext.BidTypeVideo {
		t.Errorf("the type is not the valid one. actual: %v, expected: %v", bidType2, openrtb_ext.BidTypeVideo)
	}
}
