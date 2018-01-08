package prebid_cache_client

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"context"
	"github.com/mxmCherry/openrtb"
	"strconv"
	"encoding/json"
)

// Prevents #197
func TestEmptyPut(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("The server should not be called.")
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &clientImpl{
		httpClient: server.Client(),
		putUrl: "prebid.adnxs.com/cache",
	}
	ids := client.PutBids(context.Background(), nil)
	assertIntEqual(t, len(ids), 0)
	ids = client.PutBids(context.Background(), []*openrtb.Bid{})
	assertIntEqual(t, len(ids), 0)
}

func TestBadResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &clientImpl{
		httpClient: server.Client(),
		putUrl: "prebid.adnxs.com/cache",
	}
	ids := client.PutBids(context.Background(), []*openrtb.Bid{{}})
	assertIntEqual(t, len(ids), 1)
	assertStringEqual(t, ids[0], "")
}

func TestCancelledContext(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &clientImpl{
		httpClient: server.Client(),
		putUrl: "prebid.adnxs.com/cache",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ids := client.PutBids(ctx, []*openrtb.Bid{{}})
	assertIntEqual(t, len(ids), 1)
	assertStringEqual(t, ids[0], "")
}

func TestSuccessfulPut(t *testing.T) {
	server := httptest.NewServer(newHandler(2))
	defer server.Close()

	client := &clientImpl{
		httpClient: server.Client(),
		putUrl: "prebid.adnxs.com/cache",
	}

	ids := client.PutBids(context.Background(), []*openrtb.Bid{{}, {}})
	assertIntEqual(t, len(ids), 2)
	assertStringEqual(t, ids[0], "0")
	assertStringEqual(t, ids[1], "1")
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
