package pubstack

import (
	"bytes"
	"encoding/json"
	"github.com/prebid/prebid-server/analytics/pubstack/eventchannel"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics"
	"github.com/stretchr/testify/assert"
)

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

	// Loading Issues
	_, err := NewPubstackModule("scope", "http://localhost:11287", "1z", 100, "90MB", "15m")
	assert.NotEqual(t, err, nil) // should raise an error since  we can't parse args // configRefreshDelay

	_, err = NewPubstackModule("scope", "http://localhost:11287", "1h", 100, "90z", "15m")
	assert.NotEqual(t, err, nil) // should raise an error since  we can't parse args // maxByte

	_, err = NewPubstackModule("scope", "http://localhost:11287", "1h", 100, "90MB", "15z")
	assert.NotEqual(t, err, nil) // should raise an error since  we can't parse args // maxTime

	// Loading OK
	module, err := NewPubstackModule("scope", "http://localhost:11287", "1h", 100, "90MB", "15m")
	assert.Equal(t, err, nil)

	// Default Configuration
	pubstack, ok := module.(*PubstackModule)
	assert.Equal(t, ok, true) //PBSAnalyticsModule is also a PubstackModule
	assert.Equal(t, len(pubstack.cfg.Features), 5)
	assert.Equal(t, pubstack.cfg.Features[auction], false)
	assert.Equal(t, pubstack.cfg.Features[video], false)
	assert.Equal(t, pubstack.cfg.Features[amp], false)
	assert.Equal(t, pubstack.cfg.Features[setUID], false)
	assert.Equal(t, pubstack.cfg.Features[cookieSync], false)

	assert.Equal(t, len(pubstack.eventChannels), 0)

	// Process Auction Event
	data := bytes.Buffer{}
	send := func(payload []byte) {
		data.Write(payload)
	}
	mockedEvent, err := loadJsonFromFile()

	pubstack.eventChannels[auction] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)
	pubstack.LogAuctionObject(mockedEvent)
	time.Sleep(2 * time.Millisecond) // process channel
	assert.Equal(t, data.Len(), 0)

	// Hot-Reload config
	newFeatures := make(map[string]bool)
	newFeatures[auction] = true
	newFeatures[video] = true
	newFeatures[amp] = false
	newFeatures[cookieSync] = false
	newFeatures[setUID] = false

	newConfig := &Configuration{
		ScopeId:  "new-scope",
		Endpoint: "new-endpoint",
		Features: newFeatures,
	}

	pubstack.configCh <- newConfig
	time.Sleep(2 * time.Millisecond) // process channel
	assert.Equal(t, len(pubstack.cfg.Features), 5)
	assert.Equal(t, pubstack.cfg.Features[auction], true)
	assert.Equal(t, pubstack.cfg.Features[video], true)
	assert.Equal(t, pubstack.cfg.Features[amp], false)
	assert.Equal(t, pubstack.cfg.Features[setUID], false)
	assert.Equal(t, pubstack.cfg.Features[cookieSync], false)
	assert.Equal(t, pubstack.cfg.ScopeId, "new-scope")
	assert.Equal(t, pubstack.cfg.Endpoint, "new-endpoint")
	assert.Equal(t, len(pubstack.eventChannels), 2)

	data.Reset()
	pubstack.eventChannels[auction] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)
	pubstack.LogAuctionObject(mockedEvent)
	time.Sleep(2 * time.Millisecond) // process channel (auction is disabled)
	assert.NotEqual(t, data.Len(), 0)

}
