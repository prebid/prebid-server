package pubstack

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigUpdateHttpTask(t *testing.T) {
	// configure test config endpoint
	var isFirstQuery bool = true
	var isFirstQueryMutex sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		isFirstQueryMutex.Lock()
		defer isFirstQueryMutex.Unlock()

		defer req.Body.Close()

		if isFirstQuery {
			res.Write([]byte(`{ "scopeId":  "scope1", "endpoint": "https://pubstack.io", "features": { "auction": true, "cookiesync": true }}`))
		} else {
			res.Write([]byte(`{ "scopeId":  "scope2", "endpoint": "https://pubstack.io", "features": { "auction": false, "cookiesync": false }}`))
		}

		isFirstQuery = false
	})
	server := httptest.NewServer(mux)
	client := server.Client()

	// create task
	task, err := NewConfigUpdateHttpTask(client, "scope", server.URL, "5ms")
	require.NoError(t, err, "create config task")

	// start task
	stopChan := make(chan struct{})
	configChan := task.Start(stopChan)

	// read initial config
	expectedInitialConfig := &Configuration{ScopeID: "scope1", Endpoint: "https://pubstack.io", Features: map[string]bool{"auction": true, "cookiesync": true}}
	assertConfigChanOne(t, configChan, expectedInitialConfig, "initial config")

	// read updated config
	expectedUpdatedConfig := &Configuration{ScopeID: "scope2", Endpoint: "https://pubstack.io", Features: map[string]bool{"auction": false, "cookiesync": false}}
	assertConfigChanOne(t, configChan, expectedUpdatedConfig, "updated config")

	// stop task
	close(stopChan)

	// no further updates
	assertConfigChanNone(t, configChan)
}

func TestNewConfigUpdateHttpTaskErrors(t *testing.T) {
	tests := []struct {
		description          string
		givenEndpoint        string
		givenRefreshInterval string
		expectedError        string
	}{
		{
			description:          "refresh interval invalid",
			givenEndpoint:        "http://valid.com",
			givenRefreshInterval: "invalid",
			expectedError:        `fail to parse the module args, arg=analytics.pubstack.configuration_refresh_delay: time: invalid duration "invalid"`,
		},
		{
			description:          "endpoint invalid",
			givenEndpoint:        "://invalid.com",
			givenRefreshInterval: "10ms",
			expectedError:        `parse "://invalid.com/bootstrap?scopeId=anyScope": missing protocol scheme`,
		},
	}

	for _, test := range tests {
		task, err := NewConfigUpdateHttpTask(nil, "anyScope", test.givenEndpoint, test.givenRefreshInterval)
		assert.Nil(t, task, test.description)
		assert.EqualError(t, err, test.expectedError, test.description)
	}
}

func assertConfigChanNone(t *testing.T, c <-chan *Configuration) bool {
	t.Helper()
	select {
	case <-c:
		return assert.Fail(t, "received a unexpected configuration channel event")
	case <-time.After(200 * time.Millisecond):
		return true
	}
}

func assertConfigChanOne(t *testing.T, c <-chan *Configuration, expectedConfig *Configuration, msgAndArgs ...interface{}) bool {
	t.Helper()
	select {
	case v := <-c:
		return assert.Equal(t, expectedConfig, v, msgAndArgs...)
	case <-time.After(200 * time.Millisecond):
		return assert.Fail(t, "Should receive an event, but did NOT", msgAndArgs...)
	}
}
