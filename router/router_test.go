package router

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

const adapterDirectory = "../adapters"

type testValidator struct{}

func (validator *testValidator) Validate(name openrtb_ext.BidderName, ext json.RawMessage) error {
	return nil
}

func (validator *testValidator) Schema(name openrtb_ext.BidderName) string {
	if name == openrtb_ext.BidderAppnexus {
		return "{\"appnexus\":true}"
	} else {
		return "{\"appnexus\":false}"
	}
}

func ensureHasKey(t *testing.T, data map[string]json.RawMessage, key string) {
	t.Helper()
	if _, ok := data[key]; !ok {
		t.Errorf("Expected map to produce a schema for adapter: %s", key)
	}
}

func TestNewJsonDirectoryServer(t *testing.T) {
	handler := NewJsonDirectoryServer(&testValidator{})
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/whatever", nil)
	handler(recorder, request, nil)

	var data map[string]json.RawMessage
	json.Unmarshal(recorder.Body.Bytes(), &data)

	// Make sure that every adapter has a json schema by the same name associated with it.
	adapterFiles, err := ioutil.ReadDir(adapterDirectory)
	if err != nil {
		t.Fatalf("Failed to open the adapters directory: %v", err)
	}

	for _, adapterFile := range adapterFiles {
		if adapterFile.IsDir() && adapterFile.Name() != "adapterstest" {
			ensureHasKey(t, data, adapterFile.Name())
		}
	}
}

func TestExchangeMap(t *testing.T) {
	exchanges := newExchangeMap(&config.Configuration{})
	for bidderName := range exchanges {
		// OpenRTB doesn't support hardcoded aliases... so this test skips districtm,
		// which was the only alias in the legacy adapter map.
		if _, ok := openrtb_ext.BidderMap[bidderName]; bidderName != "districtm" && !ok {
			t.Errorf("Bidder %s exists in exchange, but is not a part of the BidderMap.", bidderName)
		}
	}
}

// Prevents #648
func TestCORSSupport(t *testing.T) {
	const origin = "https://publisher-domain.com"
	handler := func(w http.ResponseWriter, r *http.Request) {}
	cors := SupportCORS(http.HandlerFunc(handler))
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("OPTIONS", "http://some-domain.com/openrtb2/auction", nil)
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "origin")
	req.Header.Set("Origin", origin)

	if !assert.NoError(t, err) {
		return
	}
	cors.ServeHTTP(rr, req)
	assert.Equal(t, origin, rr.Header().Get("Access-Control-Allow-Origin"))
}
