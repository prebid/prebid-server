package pubstack

import (
	"encoding/json"
	"github.com/magiconair/properties/assert"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics"
	"io/ioutil"
	"os"
	"testing"
)

var IntakeURL = "http://openrtb.preview.pubstack.io/v1/openrtb2/auction"

func loadJsonFromFile() (*analytics.AuctionObject, error) {
	req, err := os.Open("mocks/mock_openrtb_request.json")
	if err != nil {
		return nil, err
	}
	defer req.Close()

	reqCtn := openrtb.BidRequest{}
	reqPayload, err := ioutil.ReadAll(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(reqPayload, &reqCtn)
	if err != nil {
		return nil, err
	}

	res, err := os.Open("mocks/mock_openrtb_response.json")
	if err != nil {
		return nil, err
	}
	defer res.Close()

	resCtn := openrtb.BidResponse{}
	resPayload, err := ioutil.ReadAll(res)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resPayload, &resCtn)
	if err != nil {
		return nil, err
	}

	return &analytics.AuctionObject{
		Request:  &reqCtn,
		Response: &resCtn,
	}, nil
}

func TestPubstackModule(t *testing.T) {
	ao, err := loadJsonFromFile()
	assert.Equal(t, err, nil)

	payload, err := jsonifyAuctionObject(ao, "test-scope")
	assert.Equal(t, err, nil)

	err = sendPayloadToTarget(payload, IntakeURL)
	assert.Equal(t, err, nil)
}
