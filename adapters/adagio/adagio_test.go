package adagio

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func buildFakeBidRequest() openrtb2.BidRequest {
	imp1 := openrtb2.Imp{
		ID:     "1",
		Banner: &openrtb2.Banner{},
		Ext: json.RawMessage(`{"bidder": {"organizationId": "1000", "site": "test-site"}}`),
	}
	imp2 := openrtb2.Imp{
		ID:    "2",
		Banner: &openrtb2.Banner{},
		Video: &openrtb2.Video{},
		Ext: json.RawMessage(`{"bidder": {"organizationId": "1000", "site": "test-site"}}`),
	}
	imp3 := openrtb2.Imp{
		ID:     "3",
		Native: &openrtb2.Native{},
		Ext: json.RawMessage(`{"bidder": {"organizationId": "1000", "site": "test-site"}}`),
	}

	fakeBidRequest := openrtb2.BidRequest{
		ID: "abc-123",
		Imp: []openrtb2.Imp{imp1, imp2, imp3},
	}

	return fakeBidRequest
}

func buildFakeRequestData(t *testing.T, fakeBidRequest openrtb2.BidRequest) adapters.RequestData {
	imps, err := json.Marshal(fakeBidRequest.Imp)
	if err != nil {
		t.Fatalf("Json marshal failed: %v", err)
	}

	fakeRequestData := adapters.RequestData{
		Method: "POST",
		Body: []byte(fmt.Sprintf(`{"imp": %v}`, imps)),
	}

	return fakeRequestData
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdagio, config.Adapter{
		Endpoint: "http://localhost/prebid_server"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adagiotest", bidder)
}

func TestMakeRequests(t *testing.T) {
	fakeBidRequest := buildFakeBidRequest()
	fakeBidRequest.Test = 1

	bidder, buildErr := Builder(openrtb_ext.BidderAdagio, config.Adapter{
		Endpoint: "http://localhost/prebid_server"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	requestData, err := bidder.MakeRequests(&fakeBidRequest, nil)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(requestData))

	body := &openrtb2.BidRequest{}
	_ = json.Unmarshal(requestData[0].Body, body)

	assert.Equal(t, 3, len(body.Imp))
}

func TestAdapter_MakeRequestsGzip(t *testing.T) {
	fakeBidRequest := buildFakeBidRequest()

	bidder, buildErr := Builder(openrtb_ext.BidderAdagio, config.Adapter{
		Endpoint: "http://localhost/prebid_server"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	requestData, _ := bidder.MakeRequests(&fakeBidRequest, nil)
	assert.Equal(t, []string([]string{"gzip"}), requestData[0].Headers["Content-Encoding"])
}

func TestMakeBids(t *testing.T) {
	fakeBidRequest := buildFakeBidRequest()
	fakeRequestData := buildFakeRequestData(t, fakeBidRequest)

	bidder, buildErr := Builder(openrtb_ext.BidderAdagio, config.Adapter{
		Endpoint: "http://localhost/prebid_server"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	fakeBidResponse := &adapters.ResponseData{}

	fakeBidResponse.StatusCode = http.StatusForbidden
	_, err := bidder.MakeBids(&fakeBidRequest, &fakeRequestData, fakeBidResponse)
	assert.NotNil(t, err)

	fakeBidResponse.StatusCode = http.StatusInternalServerError
	_, err = bidder.MakeBids(&fakeBidRequest, &fakeRequestData, fakeBidResponse)
	assert.NotNil(t, err)

	fakeBidResponse.StatusCode = http.StatusOK
	fakeBidResponse.Body = []byte(`"imp": 1`)
	_, err = bidder.MakeBids(&fakeBidRequest, &fakeRequestData, fakeBidResponse)
	assert.NotNil(t, err)

	fakeBidResponse.Body = []byte(`{"id": "abc", "seatbid": [{"bid": [{"id": "internal-id", "impid": "1", "banner": {}}]}]}`)
	re, err := bidder.MakeBids(&fakeBidRequest, &fakeRequestData, fakeBidResponse)
	assert.Equal(t, 1, len(re.Bids))

}
