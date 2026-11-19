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

	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/metrics"
	metricsConf "github.com/prebid/prebid-server/v4/metrics/config"
	"github.com/prebid/prebid-server/v4/util/jsonutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestEncodeValuesStringFields(t *testing.T) {
	for _, field := range []string{"type", "key", "bidid", "bidder"} {
		t.Run(field, func(t *testing.T) {
			original := "quote\" backslash\\ newline\n tab\t null\x00 unicode\u2028 café"
			value := Cacheable{Type: TypeJSON, Data: json.RawMessage(`{"raw": [1, true, null]}`), TTLSeconds: 300, Timestamp: 123}
			switch field {
			case "type":
				value.Type = PayloadType(original)
			case "key":
				value.Key = original
			case "bidid":
				value.BidID = original
			case "bidder":
				value.Bidder = original
			}
			encoded, err := encodeValues([]Cacheable{value})
			require.NoError(t, err)
			var got struct {
				Puts []Cacheable `json:"puts"`
			}
			require.NoError(t, json.Unmarshal(encoded, &got))
			require.Len(t, got.Puts, 1)
			assert.Equal(t, value, got.Puts[0])
		})
	}
}

func TestEncodeValuesKeyCannotAddPut(t *testing.T) {
	values := []Cacheable{
		{Type: TypeJSON, Data: json.RawMessage(`true`), Key: `safe"},{"type":"json","value":true,"key":"extra`},
		{Type: TypeJSON, Data: json.RawMessage(`{"other":true}`), Key: "neighbor", TTLSeconds: 30},
	}
	encoded, err := encodeValues(values)
	require.NoError(t, err)
	var got struct {
		Puts []Cacheable `json:"puts"`
	}
	require.NoError(t, json.Unmarshal(encoded, &got))
	assert.Equal(t, values, got.Puts)
}

func TestEncodeValuesOptionalFields(t *testing.T) {
	for _, number := range []int64{0, -1} {
		encoded, err := encodeValues([]Cacheable{{Type: TypeJSON, Data: json.RawMessage(`{"n":1}`), TTLSeconds: number, Timestamp: number}})
		require.NoError(t, err)
		assert.JSONEq(t, `{"puts":[{"type":"json","value":{"n":1}}]}`, string(encoded))
	}
}

func TestPutJsonResponseEntries(t *testing.T) {
	tests := []struct {
		name, response string
		ids            []string
		errorText      string
	}{
		{name: "exact", response: `{"responses":[{"uuid":"first"},{"uuid":"second"}]}`, ids: []string{"first", "second"}},
		{name: "fewer", response: `{"responses":[{"uuid":"first"}]}`, ids: []string{"first", ""}},
		{name: "empty", response: `{"responses":[]}`, ids: []string{"", ""}},
		{name: "excess", response: `{"responses":[{"uuid":"first"},{"uuid":"second"},{"uuid":"extra"}]}`, ids: []string{"", ""}, errorText: "more responses than requested"},
		{name: "excess_invalid_entry", response: `{"responses":[{"uuid":"first"},{"uuid":"second"},{}]}`, ids: []string{"", ""}, errorText: "more responses than requested"},
		{name: "excess_multiple", response: `{"responses":[{"uuid":"first"},{"uuid":"second"},{"uuid":"extra"},{"uuid":"another"}]}`, ids: []string{"", ""}, errorText: "more responses than requested"},
		{name: "invalid_uuid", response: `{"responses":[{"uuid":123},{"uuid":"second"}]}`, ids: []string{"", "second"}, errorText: "at index 0"},
		{name: "missing_uuid", response: `{"responses":[{}, {"uuid":"second"}]}`, ids: []string{"", "second"}, errorText: "bad value at index 0"},
		{name: "missing_responses", response: `{}`, ids: []string{"", ""}, errorText: "Error interpreting"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(test.response))
			}))
			defer server.Close()
			metricsMock := &metrics.MetricsEngineMock{}
			metricsMock.On("RecordPrebidCacheRequestTime", true, mock.Anything).Once()
			client := &clientImpl{httpClient: server.Client(), putUrl: server.URL, metrics: metricsMock}
			var ids []string
			var errs []error
			require.NotPanics(t, func() {
				ids, errs = client.PutJson(context.Background(), []Cacheable{
					{Type: TypeJSON, Data: json.RawMessage(`true`)},
					{Type: TypeJSON, Data: json.RawMessage(`false`)},
				})
			})
			assert.Equal(t, test.ids, ids)
			if test.errorText == "" {
				assert.Empty(t, errs)
			} else {
				require.Len(t, errs, 1)
				assert.Contains(t, errs[0].Error(), test.errorText)
			}
			metricsMock.AssertExpectations(t)
		})
	}
}

func BenchmarkEncodeValues(b *testing.B) {
	for _, test := range []struct {
		name  string
		value Cacheable
	}{
		{name: "json", value: Cacheable{Type: TypeJSON, Data: json.RawMessage(`{"id":"bid-1","price":1.2}`), TTLSeconds: 300}},
		{name: "vtrack", value: Cacheable{Type: TypeXML, Data: json.RawMessage(`"<VAST version=\"3.0\"></VAST>"`), TTLSeconds: 300, Key: "cache-key", BidID: "bid-id", Bidder: "bidder", Timestamp: 123456789}},
	} {
		b.Run(test.name, func(b *testing.B) {
			values := []Cacheable{test.value}
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := encodeValues(values); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
