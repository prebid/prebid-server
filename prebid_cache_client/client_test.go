package prebid_cache_client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbsmetrics"
	metricsConf "github.com/prebid/prebid-server/pbsmetrics/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Prevents #197
func TestEmptyPut(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("The server should not be called.")
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	metricsMock := &pbsmetrics.MetricsEngineMock{}

	client := &clientImpl{
		httpClient: server.Client(),
		putUrl:     server.URL,
		metrics:    metricsMock,
	}
	ids, _ := client.PutJson(context.Background(), nil)
	assertIntEqual(t, len(ids), 0)
	ids, _ = client.PutJson(context.Background(), []Cacheable{})
	assertIntEqual(t, len(ids), 0)

	metricsMock.AssertNotCalled(t, "RecordPrebidCacheRequestTime")
}

func TestBadResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	metricsMock := &pbsmetrics.MetricsEngineMock{}
	metricsMock.On("RecordPrebidCacheRequestTime", true, mock.Anything).Once()

	client := &clientImpl{
		httpClient: server.Client(),
		putUrl:     server.URL,
		metrics:    metricsMock,
	}
	ids, _ := client.PutJson(context.Background(), []Cacheable{
		{
			Type: TypeJSON,
			Data: json.RawMessage("true"),
		}, {
			Type: TypeJSON,
			Data: json.RawMessage("false"),
		},
	})
	assertIntEqual(t, len(ids), 2)
	assertStringEqual(t, ids[0], "")
	assertStringEqual(t, ids[1], "")

	metricsMock.AssertExpectations(t)
}

func TestCancelledContext(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	metricsMock := &pbsmetrics.MetricsEngineMock{}
	metricsMock.On("RecordPrebidCacheRequestTime", false, mock.Anything).Once()

	client := &clientImpl{
		httpClient: server.Client(),
		putUrl:     server.URL,
		metrics:    metricsMock,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ids, _ := client.PutJson(ctx, []Cacheable{{
		Type: TypeJSON,
		Data: json.RawMessage("true"),
	},
	})
	assertIntEqual(t, len(ids), 1)
	assertStringEqual(t, ids[0], "")

	metricsMock.AssertExpectations(t)
}

func TestSuccessfulPut(t *testing.T) {
	server := httptest.NewServer(newHandler(2))
	defer server.Close()

	metricsMock := &pbsmetrics.MetricsEngineMock{}
	metricsMock.On("RecordPrebidCacheRequestTime", true, mock.Anything).Once()

	client := &clientImpl{
		httpClient: server.Client(),
		putUrl:     server.URL,
		metrics:    metricsMock,
	}

	ids, _ := client.PutJson(context.Background(), []Cacheable{
		{
			Type:       TypeJSON,
			Data:       json.RawMessage("true"),
			TTLSeconds: 300,
		}, {
			Type: TypeJSON,
			Data: json.RawMessage("false"),
		},
	})
	assertIntEqual(t, len(ids), 2)
	assertStringEqual(t, ids[0], "0")
	assertStringEqual(t, ids[1], "1")

	metricsMock.AssertExpectations(t)
}

func TestEncodeValueToBuffer(t *testing.T) {
	buf := new(bytes.Buffer)
	testCache := Cacheable{
		Type:       TypeJSON,
		Data:       json.RawMessage(`{}`),
		TTLSeconds: 300,
	}
	expected := string(`{"type":"json","ttlseconds":300,"value":{}}`)
	_ = encodeValueToBuffer(testCache, false, buf)
	actual := buf.String()
	assertStringEqual(t, expected, actual)
}

// The following test asserts that the cache client's GetExtCacheData() implementation is able to pull return the exact Path and Host that were
// specified in Prebid-Server's configuration, no substitutions nor default values.
func TestStripCacheHostAndPath(t *testing.T) {
	inCacheURL := config.Cache{ExpectedTimeMillis: 10}
	type aTest struct {
		inExtCacheURL config.ExternalCache
		expectedHost  string
		expectedPath  string
	}
	testInput := []aTest{
		{
			inExtCacheURL: config.ExternalCache{
				Host: "prebid-server.prebid.org",
				Path: "/pbcache/endpoint",
			},
			expectedHost: "prebid-server.prebid.org",
			expectedPath: "/pbcache/endpoint",
		},
		{
			inExtCacheURL: config.ExternalCache{
				Host: "prebidcache.net",
				Path: "",
			},
			expectedHost: "prebidcache.net",
			expectedPath: "",
		},
		{
			inExtCacheURL: config.ExternalCache{
				Host: "",
				Path: "",
			},
			expectedHost: "",
			expectedPath: "",
		},
		{
			inExtCacheURL: config.ExternalCache{
				Host: "prebid-server.prebid.org",
				Path: "pbcache/endpoint",
			},
			expectedHost: "prebid-server.prebid.org",
			expectedPath: "/pbcache/endpoint",
		},
		{
			inExtCacheURL: config.ExternalCache{
				Host: "prebidcache.net",
				Path: "/",
			},
			expectedHost: "prebidcache.net",
			expectedPath: "",
		},
	}
	for _, test := range testInput {
		//start client
		cacheClient := NewClient(&inCacheURL, &test.inExtCacheURL, &metricsConf.DummyMetricsEngine{})
		cHost, cPath := cacheClient.GetExtCacheData()

		//assert
		assert.Equal(t, test.expectedHost, cHost)
		assert.Equal(t, test.expectedPath, cPath)
	}
}

func assertIntEqual(t *testing.T, expected, actual int) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %d, got %d", expected, actual)
	}
}

func assertStringEqual(t *testing.T, expected, actual string) {
	t.Helper()
	if expected != actual {
		t.Errorf(`Expected "%s", got "%s"`, expected, actual)
	}
}

func newHandler(numResponses int) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := response{
			Responses: make([]responseObject, numResponses),
		}
		for i := 0; i < numResponses; i++ {
			resp.Responses[i].UUID = strconv.Itoa(i)
		}

		respBytes, _ := json.Marshal(resp)
		w.Write(respBytes)
	})
}
