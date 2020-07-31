package pubstack

import (
	"encoding/json"
	"github.com/prebid/prebid-server/analytics/pubstack/eventchannel"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	remoteConfig := &Configuration{}
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		data, _ := json.Marshal(remoteConfig)
		res.Write(data)
	}))
	client := server.Client()

	defer server.Close()

	// Loading Issues
	_, err := NewPubstackModule(client, "scope", server.URL, "1z", 100, "90MB", "15m")
	assert.NotNil(t, err) // should raise an error since  we can't parse args // configRefreshDelay

	_, err = NewPubstackModule(client, "scope", server.URL, "1h", 100, "90z", "15m")
	assert.NotNil(t, err) // should raise an error since  we can't parse args // maxByte

	_, err = NewPubstackModule(client, "scope", server.URL, "1h", 100, "90MB", "15z")
	assert.NotNil(t, err) // should raise an error since  we can't parse args // maxTime

	// Loading OK
	module, err := NewPubstackModule(client, "scope", server.URL, "10ms", 100, "90MB", "15m")
	assert.Nil(t, err)

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
	counter := 0
	send := func(_ []byte) error {
		counter++
		return nil
	}
	mockedEvent, err := loadJsonFromFile()
	if err != nil {
		t.Fail()
	}

	pubstack.eventChannels[auction] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)
	pubstack.eventChannels[video] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)
	pubstack.eventChannels[amp] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)
	pubstack.eventChannels[setUID] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)
	pubstack.eventChannels[cookieSync] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)

	pubstack.LogAuctionObject(mockedEvent)
	pubstack.LogAmpObject(&analytics.AmpObject{
		Status: http.StatusOK,
	})
	pubstack.LogCookieSyncObject(&analytics.CookieSyncObject{
		Status: http.StatusOK,
	})
	pubstack.LogVideoObject(&analytics.VideoObject{
		Status: http.StatusOK,
	})
	pubstack.LogSetUIDObject(&analytics.SetUIDObject{
		Status: http.StatusOK,
	})

	pubstack.closeAllEventChannels()
	time.Sleep(10 * time.Millisecond) // process channel
	assert.Equal(t, counter, 0)

	// Hot-Reload config
	newFeatures := make(map[string]bool)
	newFeatures[auction] = true
	newFeatures[video] = true
	newFeatures[amp] = true
	newFeatures[cookieSync] = true
	newFeatures[setUID] = true

	remoteConfig = &Configuration{
		ScopeID:  "new-scope",
		Endpoint: "new-endpoint",
		Features: newFeatures,
	}

	endpoint, _ := url.Parse(server.URL)
	pubstack.reloadConfig(endpoint)

	time.Sleep(2 * time.Millisecond) // process channel
	assert.Equal(t, len(pubstack.cfg.Features), 5)
	assert.Equal(t, pubstack.cfg.Features[auction], true)
	assert.Equal(t, pubstack.cfg.Features[video], true)
	assert.Equal(t, pubstack.cfg.Features[amp], true)
	assert.Equal(t, pubstack.cfg.Features[setUID], true)
	assert.Equal(t, pubstack.cfg.Features[cookieSync], true)
	assert.Equal(t, pubstack.cfg.ScopeID, "new-scope")
	assert.Equal(t, pubstack.cfg.Endpoint, "new-endpoint")
	assert.Equal(t, len(pubstack.eventChannels), 5)

	counter = 0
	pubstack.eventChannels[auction] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)
	pubstack.eventChannels[video] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)
	pubstack.eventChannels[amp] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)
	pubstack.eventChannels[setUID] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)
	pubstack.eventChannels[cookieSync] = eventchannel.NewEventChannel(send, 2000, 1, 10*time.Second)

	pubstack.LogAuctionObject(mockedEvent)
	pubstack.LogAmpObject(&analytics.AmpObject{
		Status: http.StatusOK,
	})
	pubstack.LogCookieSyncObject(&analytics.CookieSyncObject{
		Status: http.StatusOK,
	})
	pubstack.LogVideoObject(&analytics.VideoObject{
		Status: http.StatusOK,
	})
	pubstack.LogSetUIDObject(&analytics.SetUIDObject{
		Status: http.StatusOK,
	})
	pubstack.closeAllEventChannels()
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, counter, 5)

}
