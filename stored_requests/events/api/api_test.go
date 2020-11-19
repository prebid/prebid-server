package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/caches/memory"
	"github.com/prebid/prebid-server/stored_requests/events"
)

func TestGoodRequests(t *testing.T) {
	cache := stored_requests.Cache{
		Requests: memory.NewCache(256*1024, -1, "Request"),
		Imps:     memory.NewCache(256*1024, -1, "Imp"),
		Accounts: memory.NewCache(256*1024, -1, "Account"),
	}
	id := "1"
	config := fmt.Sprintf(`{"id": "%s"}`, id)
	initialValue := map[string]json.RawMessage{id: json.RawMessage(config)}
	cache.Requests.Save(context.Background(), initialValue)
	cache.Imps.Save(context.Background(), initialValue)

	apiEvents, endpoint := NewEventsAPI()

	// create channels to syncronize
	updateOccurred := make(chan struct{})
	invalidateOccurred := make(chan struct{})
	listener := events.NewEventListener(
		func() { updateOccurred <- struct{}{} },
		func() { invalidateOccurred <- struct{}{} },
	)

	go listener.Listen(cache, apiEvents)
	defer listener.Stop()

	config = fmt.Sprintf(`{"id": "%s", "updated": true}`, id)
	update := fmt.Sprintf(`{"requests": {"%s": %s}, "imps": {"%s": %s}}`, id, config, id, config)
	request := newRequest("POST", update)

	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Unexpected error from request: %s", recorder.Body.String())
	}

	<-updateOccurred
	reqData := cache.Requests.Get(context.Background(), []string{id})
	impData := cache.Imps.Get(context.Background(), []string{id})
	assertHasValue(t, reqData, id, config)
	assertHasValue(t, impData, id, config)

	invalidation := fmt.Sprintf(`{"requests": ["%s"], "imps": ["%s"]}`, id, id)
	request = newRequest("DELETE", invalidation)

	recorder = httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Unexpected error from request: %s", recorder.Body.String())
	}

	<-invalidateOccurred
	reqData = cache.Requests.Get(context.Background(), []string{id})
	impData = cache.Imps.Get(context.Background(), []string{id})
	assertMapLength(t, 0, reqData)
	assertMapLength(t, 0, impData)
}

func TestBadRequests(t *testing.T) {
	cache := stored_requests.Cache{
		Requests: memory.NewCache(256*1024, -1, "Requests"),
		Imps:     memory.NewCache(256*1024, -1, "Imps"),
	}
	apiEvents, endpoint := NewEventsAPI()
	listener := events.SimpleEventListener()
	go listener.Listen(cache, apiEvents)
	defer listener.Stop()

	update := "NOT JSON"
	request := newRequest("POST", update)

	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected error from request, got OK")
	}

	invalidation := "NOT JSON"
	request = newRequest("DELETE", invalidation)

	recorder = httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected error from request, got OK")
	}

	request = newRequest("GET", "")
	recorder = httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected error from request, got OK")
	}
}

func newRequest(method string, body string) *http.Request {
	return httptest.NewRequest(method, "/stored_requests", strings.NewReader(body))
}

func assertMapLength(t *testing.T, expectedLen int, theMap map[string]json.RawMessage) {
	t.Helper()
	if len(theMap) != expectedLen {
		t.Errorf("Wrong map length. Expected %d, Got %d.", expectedLen, len(theMap))
	}
}

func assertHasValue(t *testing.T, m map[string]json.RawMessage, key string, val string) {
	t.Helper()
	realVal, ok := m[key]
	if !ok {
		t.Errorf("Map missing required key: %s", key)
	}
	if val != string(realVal) {
		t.Errorf("Unexpected value at key %s. Expected %s, Got %s", key, val, string(realVal))
	}
}
