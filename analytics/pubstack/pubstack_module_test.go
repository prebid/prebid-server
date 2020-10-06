package pubstack

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

func TestPubstackModuleErrors(t *testing.T) {
	tests := []struct {
		giveRefreshDelay string
		giveMaxByteSize  string
		giveMaxTime      string
	}{
		{
			giveRefreshDelay: "1z",
			giveMaxByteSize:  "90MB",
			giveMaxTime:      "15m",
		},
		{
			giveRefreshDelay: "1h",
			giveMaxByteSize:  "90z",
			giveMaxTime:      "15m",
		},
		{
			giveRefreshDelay: "1z",
			giveMaxByteSize:  "90MB",
			giveMaxTime:      "15z",
		},
	}

	for _, tt := range tests {
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()
			data, _ := json.Marshal(&Configuration{})
			res.Write(data)
		}))
		client := server.Client()

		defer server.Close()

		_, err := NewPubstackModule(client, "scope", "http://example.com", tt.giveRefreshDelay, 100, tt.giveMaxByteSize, tt.giveMaxTime)
		assert.NotNil(t, err) // should raise an error since  we can't parse args // configRefreshDelay
	}
}

func TestPubstackModuleSuccess(t *testing.T) {
	tests := []struct {
		feature   string
		logObject func(analytics.PBSAnalyticsModule)
	}{
		{
			feature: auction,
			logObject: func(module analytics.PBSAnalyticsModule) {
				module.LogAuctionObject(&analytics.AuctionObject{Status: http.StatusOK})
			},
		},
		{
			feature: amp,
			logObject: func(module analytics.PBSAnalyticsModule) {
				module.LogAmpObject(&analytics.AmpObject{Status: http.StatusOK})
			},
		},
		{
			feature: video,
			logObject: func(module analytics.PBSAnalyticsModule) {
				module.LogVideoObject(&analytics.VideoObject{Status: http.StatusOK})
			},
		},
		{
			feature: cookieSync,
			logObject: func(module analytics.PBSAnalyticsModule) {
				module.LogCookieSyncObject(&analytics.CookieSyncObject{Status: http.StatusOK})
			},
		},
		{
			feature: setUID,
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
				data, _ := json.Marshal(updatedConfig)
				res.Write(data)
			} else {
				data, _ := json.Marshal(origConfig)
				res.Write(data)
			}
		})

		intakeCount := 0
		intakeChannel := make(chan int) // using a channel rather than examining the count directly to avoid race
		mux.HandleFunc("/intake/"+tt.feature+"/", func(res http.ResponseWriter, req *http.Request) {
			intakeCount++
			intakeChannel <- intakeCount
		})
		server := httptest.NewServer(mux)
		client := server.Client()

		// set the server url on each of the configs
		origConfig.Endpoint = server.URL
		updatedConfig.Endpoint = server.URL

		// instantiate module with 25ms config refresh rate
		module, err := NewPubstackModule(client, "scope", server.URL, "25ms", 100, "1B", "10ms")
		assert.Nil(t, err)

		// allow time for the module to load the original config
		time.Sleep(10 * time.Millisecond)

		pubstack, _ := module.(*PubstackModule)
		// attempt to log but no event channel was created because the feature is disabled in the original config
		tt.logObject(pubstack)

		eventCount := 0

		select {
		case eventCount = <-intakeChannel:
		case <-time.After(10 * time.Millisecond):
		}

		assert.Equal(t, 0, eventCount)

		// allow time for the server to start serving the updated config
		time.Sleep(10 * time.Millisecond)

		// attempt to log; the event channel should have been created because the feature is enabled in updated config
		tt.logObject(pubstack)

		select {
		case eventCount = <-intakeChannel:
		case <-time.After(10 * time.Millisecond):
		}

		assert.Equal(t, 1, eventCount)
	}
}
