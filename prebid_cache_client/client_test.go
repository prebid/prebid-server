package prebid_cache_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	metricsConf "github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEmptyPut(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("The server should not be called.")
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	metricsMock := &metrics.MetricsEngineMock{}

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

	metricsMock := &metrics.MetricsEngineMock{}
	metricsMock.On("RecordPrebidCacheRequestTime", false, mock.Anything).Once()

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
	testCases := []struct {
		description         string
		cacheable           []Cacheable
		expectedItems       int
		expectedPayloadSize int
	}{
		{
			description: "1 Item",
			cacheable: []Cacheable{
				{
					Type: TypeJSON,
					Data: json.RawMessage("true"),
				},
			},
			expectedItems:       1,
			expectedPayloadSize: 39,
		},
		{
			description: "2 Items",
			cacheable: []Cacheable{
				{
					Type: TypeJSON,
					Data: json.RawMessage("true"),
				},
				{
					Type: TypeJSON,
					Data: json.RawMessage("false"),
				},
			},
			expectedItems:       2,
			expectedPayloadSize: 69,
		},
	}

	// Initialize Stub Server
	stubHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	stubServer := httptest.NewServer(stubHandler)
	defer stubServer.Close()

	// Run Tests
	for _, testCase := range testCases {
		metricsMock := &metrics.MetricsEngineMock{}
		metricsMock.On("RecordPrebidCacheRequestTime", false, mock.Anything).Once()

		client := &clientImpl{
			httpClient: stubServer.Client(),
			putUrl:     stubServer.URL,
			metrics:    metricsMock,
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		ids, errs := client.PutJson(ctx, testCase.cacheable)

		expectedErrorMessage := fmt.Sprintf("Items=%v, Payload Size=%v", testCase.expectedItems, testCase.expectedPayloadSize)

		assert.Equal(t, testCase.expectedItems, len(ids), testCase.description+":ids")
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "Error sending the request to Prebid Cache: context canceled", testCase.description+":error")
		assert.Contains(t, errs[0].Error(), expectedErrorMessage, testCase.description+":error_dimensions")
		metricsMock.AssertExpectations(t)
	}
}

func TestSuccessfulPut(t *testing.T) {
	server := httptest.NewServer(newHandler(2))
	defer server.Close()

	metricsMock := &metrics.MetricsEngineMock{}
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
		BidID:      "bid",
		Bidder:     "bdr",
		Timestamp:  123456789,
	}
	expected := string(`{"type":"json","ttlseconds":300,"value":{},"bidid":"bid","bidder":"bdr","timestamp":123456789}`)
	_ = encodeValueToBuffer(testCache, false, buf)
	actual := buf.String()
	assertStringEqual(t, expected, actual)
}

// The following test asserts that the cache client's GetExtCacheData() implementation is able to pull return the exact Path and Host that were
// specified in Prebid-Server's configuration, no substitutions nor default values.
func TestStripCacheHostAndPath(t *testing.T) {
	inCacheURL := config.Cache{ExpectedTimeMillis: 10}
	type aTest struct {
		inExtCacheURL  config.ExternalCache
		expectedScheme string
		expectedHost   string
		expectedPath   string
	}
	testInput := []aTest{
		{
			inExtCacheURL: config.ExternalCache{
				Scheme: "",
				Host:   "prebid-server.prebid.org",
				Path:   "/pbcache/endpoint",
			},
			expectedScheme: "",
			expectedHost:   "prebid-server.prebid.org",
			expectedPath:   "/pbcache/endpoint",
		},
		{
			inExtCacheURL: config.ExternalCache{
				Scheme: "https",
				Host:   "prebid-server.prebid.org",
				Path:   "/pbcache/endpoint",
			},
			expectedScheme: "https",
			expectedHost:   "prebid-server.prebid.org",
			expectedPath:   "/pbcache/endpoint",
		},
		{
			inExtCacheURL: config.ExternalCache{
				Scheme: "",
				Host:   "prebidcache.net",
				Path:   "",
			},
			expectedScheme: "",
			expectedHost:   "prebidcache.net",
			expectedPath:   "",
		},
		{
			inExtCacheURL: config.ExternalCache{
				Scheme: "",
				Host:   "",
				Path:   "",
			},
			expectedScheme: "",
			expectedHost:   "",
			expectedPath:   "",
		},
		{
			inExtCacheURL: config.ExternalCache{
				Scheme: "",
				Host:   "prebid-server.prebid.org",
				Path:   "pbcache/endpoint",
			},
			expectedScheme: "",
			expectedHost:   "prebid-server.prebid.org",
			expectedPath:   "/pbcache/endpoint",
		},
		{
			inExtCacheURL: config.ExternalCache{
				Scheme: "",
				Host:   "prebidcache.net",
				Path:   "/",
			},
			expectedScheme: "",
			expectedHost:   "prebidcache.net",
			expectedPath:   "",
		},
	}
	for _, test := range testInput {
		cacheClient := NewClient(&http.Client{}, &inCacheURL, &test.inExtCacheURL, &metricsConf.NilMetricsEngine{})
		scheme, host, path := cacheClient.GetExtCacheData()

		assert.Equal(t, test.expectedScheme, scheme)
		assert.Equal(t, test.expectedHost, host)
		assert.Equal(t, test.expectedPath, path)
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

type handlerResponseObject struct {
	UUID string `json:"uuid"`
}

type handlerResponse struct {
	Responses []handlerResponseObject `json:"responses"`
}

func newHandler(numResponses int) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := handlerResponse{
			Responses: make([]handlerResponseObject, numResponses),
		}
		for i := 0; i < numResponses; i++ {
			resp.Responses[i].UUID = strconv.Itoa(i)
		}

		respBytes, _ := jsonutil.Marshal(resp)
		w.Write(respBytes)
	})
}
