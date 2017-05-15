package prebid_cache_client

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fmt"
)

var delay time.Duration

func DummyPrebidCacheServer(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var pr putRequest
	err = json.Unmarshal(body, &pr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := response{
		Responses: make([]responseObject, len(pr.Puts)),
	}
	for i, _ := range pr.Puts {
		resp.Responses[i].UUID = fmt.Sprintf("UUID-%d", i+1) // deterministic for testing
	}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if delay > 0 {
		<-time.After(delay)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func TestPrebidClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyPrebidCacheServer))
	defer server.Close()

	cobj := make([]*CacheObject, 2)

	cobj[0] = &CacheObject{
		Value: "random",
	}
	cobj[1] = &CacheObject{
		Value: "totally awesome value!",
	}

	InitPrebidCache(server.URL)

	ctx := context.TODO()
	err := Put(ctx, cobj)
	if err != nil {
		t.Fatalf("pbc put failed: %v", err)
	}

	if cobj[0].UUID != "UUID-1" {
		t.Errorf("First object UUID was '%s', should have been 'UUID-1'", cobj[0].UUID)
	}
	if cobj[1].UUID != "UUID-2" {
		t.Errorf("Second object UUID was '%s', should have been 'UUID-2'", cobj[0].UUID)
	}

	delay = 5 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err = Put(ctx, cobj)
	if err == nil {
		t.Fatalf("pbc put succeeded but should have timed out")
	}
}
