package pubstack

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/analytics"
	"github.com/stretchr/testify/assert"
)

func loadJSONFromFile() (*analytics.AuctionObject, error) {
	req, err := os.Open("mocks/mock_openrtb_request.json")
	if err != nil {
		return nil, err
	}
	defer req.Close()

	reqCtn := openrtb2.BidRequest{}
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

	resCtn := openrtb2.BidResponse{}
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

func TestPubstackModuleErrors(t *testing.T) {
	tests := []struct {
		description  string
		refreshDelay string
		maxByteSize  string
		maxTime      string
	}{
		{
			description:  "refresh delay is in an invalid format",
			refreshDelay: "1invalid",
			maxByteSize:  "90MB",
			maxTime:      "15m",
		},
		{
			description:  "max byte size is in an invalid format",
			refreshDelay: "1h",
			maxByteSize:  "90invalid",
			maxTime:      "15m",
		},
		{
			description:  "max time is in an invalid format",
			refreshDelay: "1h",
			maxByteSize:  "90MB",
			maxTime:      "15invalid",
		},
	}

	for _, tt := range tests {
		_, err := NewPubstackModule(&http.Client{}, "scope", "http://example.com", tt.refreshDelay, 100, tt.maxByteSize, tt.maxTime)
		assert.NotNil(t, err, tt.description)
	}
}

func TestPubstackModuleSuccess(t *testing.T) {
	tests := []struct {
		description string
		feature     string
		logObject   func(analytics.PBSAnalyticsModule)
	}{
		{
			description: "auction events are only published when logging an auction object with auction feature on",
			feature:     auction,
			logObject: func(module analytics.PBSAnalyticsModule) {
				module.LogAuctionObject(&analytics.AuctionObject{Status: http.StatusOK})
			},
		},
		{
			description: "AMP events are only published when logging an AMP object with AMP feature on",
			feature:     amp,
			logObject: func(module analytics.PBSAnalyticsModule) {
				module.LogAmpObject(&analytics.AmpObject{Status: http.StatusOK})
			},
		},
		{
			description: "video events are only published when logging a video object with video feature on",
			feature:     video,
			logObject: func(module analytics.PBSAnalyticsModule) {
				module.LogVideoObject(&analytics.VideoObject{Status: http.StatusOK})
			},
		},
		{
			description: "cookie events are only published when logging a cookie object with cookie feature on",
			feature:     cookieSync,
			logObject: func(module analytics.PBSAnalyticsModule) {
				module.LogCookieSyncObject(&analytics.CookieSyncObject{Status: http.StatusOK})
			},
		},
		{
			description: "setUID events are only published when logging a setUID object with setUID feature on",
			feature:     setUID,
			logObject: func(module analytics.PBSAnalyticsModule) {
				module.LogSetUIDObject(&analytics.SetUIDObject{Status: http.StatusOK})
			},
		},
	}

	for _, tt := range tests {
		// original config is loaded when the module is created
		// the feature is disabled so no events should be sent
		origConfig := &Configuration{
			Features: map[string]bool{
				tt.feature: false,
			},
		}
		// updated config is hot-reloaded after some time passes
		// the feature is enabled so events should be sent
		updatedConfig := &Configuration{
			Features: map[string]bool{
				tt.feature: true,
			},
		}

		// create server with root endpoint that returns the current config
		// add an intake endpoint that PBS hits when events are sent
		rootCount := 0
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
			rootCount++
			defer req.Body.Close()

			if rootCount > 1 {
				if data, err := json.Marshal(updatedConfig); err != nil {
					res.WriteHeader(http.StatusBadRequest)
				} else {
					res.Write(data)
				}
			} else {
				if data, err := json.Marshal(origConfig); err != nil {
					res.WriteHeader(http.StatusBadRequest)
				} else {
					res.Write(data)
				}
			}
		})

		intakeChannel := make(chan int) // using a channel rather than examining the count directly to avoid race
		mux.HandleFunc("/intake/"+tt.feature+"/", func(res http.ResponseWriter, req *http.Request) {
			intakeChannel <- 1
		})
		server := httptest.NewServer(mux)
		client := server.Client()

		// set the server url on each of the configs
		origConfig.Endpoint = server.URL
		updatedConfig.Endpoint = server.URL

		// instantiate module with 25ms config refresh rate
		module, err := NewPubstackModule(client, "scope", server.URL, "15ms", 100, "1B", "10ms")
		assert.Nil(t, err, tt.description)

		// allow time for the module to load the original config
		time.Sleep(10 * time.Millisecond)

		pubstack, _ := module.(*PubstackModule)
		// attempt to log but no event channel was created because the feature is disabled in the original config
		tt.logObject(pubstack)

		// verify no event was received over a 10ms period
		assertChanNone(t, intakeChannel, tt.description)

		// allow time for the server to start serving the updated config
		time.Sleep(10 * time.Millisecond)

		// attempt to log; the event channel should have been created because the feature is enabled in updated config
		tt.logObject(pubstack)

		// verify an event was received within 10ms
		assertChanOne(t, intakeChannel, tt.description)
	}
}

func assertChanNone(t *testing.T, c <-chan int, msgAndArgs ...interface{}) bool {
	select {
	case <-c:
		return assert.Fail(t, "Should NOT receive an event, but did", msgAndArgs...)
	case <-time.After(10 * time.Millisecond):
		return true
	}
}

func assertChanOne(t *testing.T, c <-chan int, msgAndArgs ...interface{}) bool {
	select {
	case <-c:
		return true
	case <-time.After(10 * time.Millisecond):
		return assert.Fail(t, "Should receive an event, but did NOT", msgAndArgs...)
	}
}
