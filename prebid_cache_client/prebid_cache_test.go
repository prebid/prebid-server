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
var (
	MaxValueLength = 1024 * 10
	MaxNumValues   = 10
)

type putAnyObject struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

type putAnyRequest struct {
	Puts []putAnyObject `json:"puts"`
}

func DummyPrebidCacheServer(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read the request body.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	var put putAnyRequest

	err = json.Unmarshal(body, &put)
	if err != nil {
		http.Error(w, "Request body "+string(body)+" is not valid JSON.", http.StatusBadRequest)
		return
	}

	if len(put.Puts) > MaxNumValues {
		http.Error(w, fmt.Sprintf("More keys than allowed: %d", MaxNumValues), http.StatusBadRequest)
		return
	}

	resp := response{
		Responses: make([]responseObject, len(put.Puts)),
	}
	for i, p := range put.Puts {
		resp.Responses[i].UUID = fmt.Sprintf("UUID-%d", i+1) // deterministic for testing
		if len(p.Value) > MaxValueLength {
			http.Error(w, fmt.Sprintf("Value is larger than allowed size: %d", MaxValueLength), http.StatusBadRequest)
			return
		}
		if len(p.Value) == 0 {
			http.Error(w, "Missing value.", http.StatusBadRequest)
			return
		}
		if p.Type != "xml" && p.Type != "json" {
			http.Error(w, fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type), http.StatusBadRequest)
			return
		}
	}

	bytes, err := json.Marshal(&resp)
	if err != nil {
		http.Error(w, "Failed to serialize UUIDs into JSON.", http.StatusInternalServerError)
		return
	}
	if delay > 0 {
		<-time.After(delay)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func TestPrebidClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyPrebidCacheServer))
	defer server.Close()

	cobj := make([]*CacheObject, 3)

	// example bids
	cobj[0] = &CacheObject{
		IsVideo: false,
		Value: &BidCache{
			Adm:    "{\"type\":\"ID\",\"bid_id\":\"8255649814109237089\",\"placement_id\":\"1995257847363113_1997038003851764\",\"resolved_placement_id\":\"1995257847363113_1997038003851764\",\"sdk_version\":\"4.25.0-appnexus.bidding\",\"device_id\":\"87ECBA49-908A-428F-9DE7-4B9CED4F486C\",\"template\":7,\"payload\":\"null\"}",
			NURL:   "https://www.facebook.com/audiencenetwork/nurl/?partner=442648859414574&app=1995257847363113&placement=1997038003851764&auction=d3013e9e-ca55-4a86-9baa-d44e31355e1d&impression=bannerad1&request=7187783259538616534&bid=3832427901228167009&ortb_loss_code=0",
			Width:  300,
			Height: 250,
		},
	}
	cobj[1] = &CacheObject{
		IsVideo: false,
		Value: &BidCache{
			Adm:    "<script type=\"application/javascript\" src=\"http://nym1-ib.adnxs.com/ab?e=wqT_3QLVBqBVAwAAAwDWAAUBCN_rpM4FEIziq9qV-8avPRiq8Nq0r-ek-wcqLQkAAAECCOA_EQEHNAAA4D8ZAAAAQOF6hD8hERIAKREJoDCV4t0EOL4HQL4HSAJQy5aGI1it2kRgAGiRQHj14AOAAQCKAQNVU0SSBQbwUpgBrAKgAfoBqAEBsAEAuAECwAEDyAEC0AEC2AEA4AEA8AEAigI7dWYoJ2EnLCAxMzk5NzAwLCAxNTA2MzU4NzUxKTt1ZigncicsIDczNTAxNTE1Nh4A8I2SAu0BIXREUGp6d2llLTVJSEVNdVdoaU1ZQUNDdDJrUXdBRGdBUUFSSXZnZFFsZUxkQkZnQVlOSUdhQUJ3WEhqU0E0QUJYSWdCMGdPUUFRR1lBUUdnQVFHb0FRT3dBUUM1QVNtTGlJTUFBT0Ffd1FFcGk0aURBQURnUDhrQmtzSzlsZXQwMGpfWkFRQUFBAQMkUEFfNEFFQTlRRQEOLEFtQUlBb0FJQXRRSQUQAHYNCIh3QUlBeUFJQTRBSUE2QUlBLUFJQWdBTUVrQU1BbUFNQnFBTwXQaHVnTUpUbGxOTWpveU9USXmaAi0hOWdoQW5naQUgAEUN8ChyZHBFSUFRb0FEbzIwAFjYAugH4ALH0wHyAhEKBkFEVl9JRBIHMSlqBRQIQ1BHBRQYMzU0NjYyNwEUCAVDUAET9BUBCDE0OTkwNzUwgAMBiAMBkAMAmAMUoAMBqgMAwAOsAsgDANIDKAgAEiQ0NzJhYjY4MS03MDUxLTQzMjktOTc5MS1hZTI4YTg4ZWJmNmLYAwDgAwDoAwL4AwCABACSBAkvb3BlbnJ0YjKYBACoBLj7A7IEDAgAEAAYACAAMAA4ALgEAMAEAMgEANIECU5ZTTI6MjkyMtoEAggB4AQA8ATLloYjggUZQXBwTmV4dXMuUHJlYmlkTW9iaWxlRGVtb4gFAZgFAKAF____________AaoFJERDMzVGRjlGLTA0RjUtNDBFQi1CRDJFLTA1MzY5QjVCOUMxNsAFAMkFAAAAAAAA8D_SBQkJAAAAAAAAAADYBQHgBQA.&s=49790274de0e076a2b8b9577c2cce27ff3919239&pp=${AUCTION_PRICE}&\"></script>",
			Width:  300,
			Height: 250,
		},
	}
	cobj[2] = &CacheObject{
		IsVideo: true,
		Value:   "<script type=\"application/javascript\" src=\"http://nym1-ib.adnxs.com/ab?e=wqT_3QLVBqBVAwAAAwDWAAUBCN_rpM4FEIziq9qV-8avPRiq8Nq0r-ek-wcqLQkAAAECCOA_EQEHNAAA4D8ZAAAAQOF6hD8hERIAKREJoDCV4t0EOL4HQL4HSAJQy5aGI1it2kRgAGiRQHj14AOAAQCKAQNVU0SSBQbwUpgBrAKgAfoBqAEBsAEAuAECwAEDyAEC0AEC2AEA4AEA8AEAigI7dWYoJ2EnLCAxMzk5NzAwLCAxNTA2MzU4NzUxKTt1ZigncicsIDczNTAxNTE1Nh4A8I2SAu0BIXREUGp6d2llLTVJSEVNdVdoaU1ZQUNDdDJrUXdBRGdBUUFSSXZnZFFsZUxkQkZnQVlOSUdhQUJ3WEhqU0E0QUJYSWdCMGdPUUFRR1lBUUdnQVFHb0FRT3dBUUM1QVNtTGlJTUFBT0Ffd1FFcGk0aURBQURnUDhrQmtzSzlsZXQwMGpfWkFRQUFBAQMkUEFfNEFFQTlRRQEOLEFtQUlBb0FJQXRRSQUQAHYNCIh3QUlBeUFJQTRBSUE2QUlBLUFJQWdBTUVrQU1BbUFNQnFBTwXQaHVnTUpUbGxOTWpveU9USXmaAi0hOWdoQW5naQUgAEUN8ChyZHBFSUFRb0FEbzIwAFjYAugH4ALH0wHyAhEKBkFEVl9JRBIHMSlqBRQIQ1BHBRQYMzU0NjYyNwEUCAVDUAET9BUBCDE0OTkwNzUwgAMBiAMBkAMAmAMUoAMBqgMAwAOsAsgDANIDKAgAEiQ0NzJhYjY4MS03MDUxLTQzMjktOTc5MS1hZTI4YTg4ZWJmNmLYAwDgAwDoAwL4AwCABACSBAkvb3BlbnJ0YjKYBACoBLj7A7IEDAgAEAAYACAAMAA4ALgEAMAEAMgEANIECU5ZTTI6MjkyMtoEAggB4AQA8ATLloYjggUZQXBwTmV4dXMuUHJlYmlkTW9iaWxlRGVtb4gFAZgFAKAF____________AaoFJERDMzVGRjlGLTA0RjUtNDBFQi1CRDJFLTA1MzY5QjVCOUMxNsAFAMkFAAAAAAAA8D_SBQkJAAAAAAAAAADYBQHgBQA.&s=49790274de0e076a2b8b9577c2cce27ff3919239&pp=${AUCTION_PRICE}&\"></script>",
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
		t.Errorf("Second object UUID was '%s', should have been 'UUID-2'", cobj[1].UUID)
	}
	if cobj[2].UUID != "UUID-3" {
		t.Errorf("Third object UUID was '%s', should have been 'UUID-3'", cobj[2].UUID)
	}

	delay = 5 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err = Put(ctx, cobj)
	if err == nil {
		t.Fatalf("pbc put succeeded but should have timed out")
	}
}

// Prevents #197
func TestEmptyBids(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("The server should not be called.")
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	InitPrebidCache(server.URL)

	if err := Put(context.Background(), []*CacheObject{}); err != nil {
		t.Errorf("Error on Put: %v", err)
	}
}
