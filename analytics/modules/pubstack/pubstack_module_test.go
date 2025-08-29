package pubstack

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestNewModuleErrors(t *testing.T) {
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
		_, err := NewModule(&http.Client{}, "scope", "http://example.com", tt.refreshDelay, 100, tt.maxByteSize, tt.maxTime, clock.NewMock())
		assert.Error(t, err, tt.description)
	}
}

func TestNewModuleSuccess(t *testing.T) {
	tests := []struct {
		description string
		feature     string
		logObject   func(analytics.Module)
	}{
		{
			description: "auction events are only published when logging an auction object with auction feature on",
			feature:     auction,
			logObject: func(module analytics.Module) {
				module.LogAuctionObject(&analytics.AuctionObject{Status: http.StatusOK})
			},
		},
		{
			description: "AMP events are only published when logging an AMP object with AMP feature on",
			feature:     amp,
			logObject: func(module analytics.Module) {
				module.LogAmpObject(&analytics.AmpObject{Status: http.StatusOK})
			},
		},
		{
			description: "video events are only published when logging a video object with video feature on",
			feature:     video,
			logObject: func(module analytics.Module) {
				module.LogVideoObject(&analytics.VideoObject{Status: http.StatusOK})
			},
		},
		{
			description: "cookie events are only published when logging a cookie object with cookie feature on",
			feature:     cookieSync,
			logObject: func(module analytics.Module) {
				module.LogCookieSyncObject(&analytics.CookieSyncObject{Status: http.StatusOK})
			},
		},
		{
			description: "setUID events are only published when logging a setUID object with setUID feature on",
			feature:     setUID,
			logObject: func(module analytics.Module) {
				module.LogSetUIDObject(&analytics.SetUIDObject{Status: http.StatusOK})
			},
		},
		{
			description: "Ignore excluded fields from marshal",
			feature:     auction,
			logObject: func(module analytics.Module) {
				module.LogAuctionObject(&analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "123",
									StatusCode: 34,
									Ext:        &openrtb_ext.NonBidExt{Prebid: openrtb_ext.ExtResponseNonBidPrebid{Bid: openrtb_ext.NonBidObject{}}},
								},
							},
						},
					},
				})
			},
		},
	}

	for _, tt := range tests {
		// original config with the feature disabled so no events should be sent
		origConfig := &Configuration{
			Features: map[string]bool{
				tt.feature: false,
			},
		}

		// updated config with the feature enabled so events should be sent
		updatedConfig := &Configuration{
			Features: map[string]bool{
				tt.feature: true,
			},
		}

		// create server with an intake endpoint that PBS hits when events are sent
		mux := http.NewServeMux()
		intakeChannel := make(chan int)
		mux.HandleFunc("/intake/"+tt.feature+"/", func(res http.ResponseWriter, req *http.Request) {
			intakeChannel <- 1
		})
		server := httptest.NewServer(mux)
		client := server.Client()

		// set the event server url on each of the configs
		origConfig.Endpoint = server.URL
		updatedConfig.Endpoint = server.URL

		// instantiate module with a manual config update task
		clockMock := clock.NewMock()
		configTask := fakeConfigUpdateTask{}
		module, err := NewModuleWithConfigTask(client, "scope", server.URL, 100, "1B", "1s", &configTask, clockMock)
		assert.NoError(t, err, tt.description)

		pubstack, _ := module.(*PubstackModule)

		// original config
		configTask.Push(origConfig)
		time.Sleep(10 * time.Millisecond)                            // allow time for the module to load the original config
		tt.logObject(pubstack)                                       // attempt to log; no event channel created because feature is disabled in original config
		clockMock.Add(1 * time.Second)                               // trigger event channel sending
		assertChanNone(t, intakeChannel, tt.description+":original") // verify no event was received

		// updated config
		configTask.Push(updatedConfig)
		time.Sleep(10 * time.Millisecond)                          // allow time for the server to start serving the updated config
		tt.logObject(pubstack)                                     // attempt to log; event channel should be created because feature is enabled in updated config
		clockMock.Add(1 * time.Second)                             // trigger event channel sending
		assertChanOne(t, intakeChannel, tt.description+":updated") // verify an event was received

		// no config change
		configTask.Push(updatedConfig)
		time.Sleep(10 * time.Millisecond)                            // allow time for the server to determine no config change
		tt.logObject(pubstack)                                       // attempt to log; event channel should still be created from loading updated config
		clockMock.Add(1 * time.Second)                               // trigger event channel sending
		assertChanOne(t, intakeChannel, tt.description+":no_change") // verify an event was received

		// shutdown
		pubstack.sigTermCh <- os.Kill                                // simulate os shutdown signal
		time.Sleep(10 * time.Millisecond)                            // allow time for the server to switch to shutdown generated config
		tt.logObject(pubstack)                                       // attempt to log; event channel should be closed from the os kill signal
		clockMock.Add(1 * time.Second)                               // trigger event channel sending
		assertChanNone(t, intakeChannel, tt.description+":shutdown") // verify no event was received
	}
}

func assertChanNone(t *testing.T, c <-chan int, msgAndArgs ...interface{}) bool {
	t.Helper()
	select {
	case <-c:
		return assert.Fail(t, "Should NOT receive an event, but did", msgAndArgs...)
	case <-time.After(100 * time.Millisecond):
		return true
	}
}

func assertChanOne(t *testing.T, c <-chan int, msgAndArgs ...interface{}) bool {
	t.Helper()
	select {
	case <-c:
		return true
	case <-time.After(200 * time.Millisecond):
		return assert.Fail(t, "Should receive an event, but did NOT", msgAndArgs...)
	}
}

type fakeConfigUpdateTask struct {
	configChan chan *Configuration
}

func (f *fakeConfigUpdateTask) Start(stop <-chan struct{}) <-chan *Configuration {
	f.configChan = make(chan *Configuration)
	return f.configChan
}

func (f *fakeConfigUpdateTask) Push(c *Configuration) {
	f.configChan <- c
}
