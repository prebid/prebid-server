package pubstack

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics"
	"github.com/stretchr/testify/assert"
)

var received = 0
var updateConfig = 0

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

func mockBoot(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query().Get("scopeId")
	if qp == "no-auction-scope" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"scopeId\":\"test-scope\",\"endpoint\":\"http://localhost:11287\",\"features\":{\"amp\":false,\"auction\":false,\"cookiesync\":true,\"setuid\":false,\"video\":false}}"))
	} else if qp == "test-scope" {
		if updateConfig == 0 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{\"scopeId\":\"test-scope\",\"endpoint\":\"http://localhost:11287\",\"features\":{\"amp\":false,\"auction\":true,\"cookiesync\":true,\"setuid\":false,\"video\":false}}"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{\"scopeId\":\"test-scope\",\"endpoint\":\"http://localhost:11287\",\"features\":{\"amp\":true,\"auction\":true,\"cookiesync\":true,\"setuid\":true,\"video\":true}}"))
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func mockEvent(w http.ResponseWriter, r *http.Request) {
	gzr, err := gzip.NewReader(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	data, err := ioutil.ReadAll(gzr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	payloads := bytes.SplitN(data, []byte("\n"), -1)
	received += len(payloads) - 1 // to account for trailing \n
	w.WriteHeader(http.StatusOK)
}

func mockedServer() {
	http.HandleFunc("/bootstrap", mockBoot)
	http.HandleFunc("/intake/auction", mockEvent)
	http.ListenAndServe(":11287", nil)
}

func TestPubstackModule(t *testing.T) {
	go mockedServer()

	_, err := NewPubstackModule("bad scope", "http://localhost:11287", "3h", 2, "900MB", "3h")
	assert.NotEqual(t, err, nil) // should raise an error since we can't configure the scope

	pm, err := NewPubstackModule("test-scope", "http://localhost:11287", "3h", 2, "900MB", "3h")
	assert.NotNil(t, pm)

	pbmodule, ok := pm.(*PubstackModule)
	assert.Equal(t, ok, true) // assert pubstack module is effectively a PBSAnalyticsModule
	assert.Equal(t, len(pbmodule.eventChannels), 2)

	eventOne, err := loadJsonFromFile()
	assert.Equal(t, err, nil)
	// due to the buffer configuration, we should not have data received by the handler
	pbmodule.LogAuctionObject(eventOne)
	time.Sleep(1 * time.Second)
	assert.Equal(t, received, 0)

	eventTwo, err := loadJsonFromFile()
	assert.Equal(t, err, nil)
	// due to the buffer configuration, we should have 2 events received
	pbmodule.LogAuctionObject(eventTwo)
	time.Sleep(1 * time.Second)
	assert.Equal(t, received, 2)

	// reset received counter
	received = 0

	// due to features settings we should send the event right away but the auction collection is disabled
	pm, err = NewPubstackModule("no-auction-scope", "http://localhost:11287", "3h", 1, "1MB", "3h")
	assert.Equal(t, err, nil)
	pbmodule, ok = pm.(*PubstackModule)
	assert.Equal(t, ok, true)
	pbmodule.LogAuctionObject(eventOne)
	pbmodule.LogAuctionObject(eventTwo)
	time.Sleep(1 * time.Second)
	assert.Equal(t, received, 0)

	// test configuration update
	pm, err = NewPubstackModule("test-scope", "http://localhost:11287", "10s", 2, "900MB", "3h") // create new module which update its conf every 10 seconds
	assert.Equal(t, err, nil)
	pbmodule, ok = pm.(*PubstackModule)
	assert.Equal(t, ok, true)
	assert.Equal(t, len(pbmodule.eventChannels), 2)
	updateConfig = 1             // force config update
	time.Sleep(15 * time.Second) // wait for the config to update
	// update should enable all features so chanmap size should be 5 instead of 2
	assert.Equal(t, len(pbmodule.eventChannels), 5)
}
