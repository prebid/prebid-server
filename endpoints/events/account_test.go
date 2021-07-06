package events

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/stretchr/testify/assert"
)

func TestHandleAccountServiceErrors(t *testing.T) {
	tests := map[string]struct {
		fetcher *mockAccountsFetcher
		cfg     *config.Configuration
		want    struct {
			code     int
			response string
		}
	}{
		"badRequest": {
			fetcher: &mockAccountsFetcher{
				Fail:  true,
				Error: errors.New("some error"),
			},
			cfg: &config.Configuration{
				AccountDefaults: config.Account{Disabled: true},
				AccountRequired: true,
				MaxRequestSize:  maxSize,
				VTrack: config.VTrack{
					TimeoutMS: int64(2000), AllowUnknownBidder: false,
				},
			},
			want: struct {
				code     int
				response string
			}{
				code:     400,
				response: "Invalid request: some error\nInvalid request: Prebid-server could not verify the Account ID. Please reach out to the prebid server host.\n",
			},
		},
		"serviceUnavailable": {
			fetcher: &mockAccountsFetcher{
				Fail: false,
			},
			cfg: &config.Configuration{
				BlacklistedAcctMap: map[string]bool{"testacc": true},
				MaxRequestSize:     maxSize,
				VTrack: config.VTrack{
					TimeoutMS: int64(2000), AllowUnknownBidder: false,
				},
			},
			want: struct {
				code     int
				response string
			}{
				code:     503,
				response: "Invalid request: Prebid-server has disabled Account ID: testacc, please reach out to the prebid server host.\n",
			},
		},
		"timeout": {
			fetcher: &mockAccountsFetcher{
				Fail:       false,
				DurationMS: 50,
			},
			cfg: &config.Configuration{
				AccountDefaults: config.Account{Disabled: true},
				AccountRequired: true,
				Event: config.Event{
					TimeoutMS: 1,
				},
				MaxRequestSize: maxSize,
				VTrack: config.VTrack{
					TimeoutMS:          int64(1),
					AllowUnknownBidder: false,
				},
			},
			want: struct {
				code     int
				response string
			}{
				code:     504,
				response: "Invalid request: context deadline exceeded\nInvalid request: Prebid-server could not verify the Account ID. Please reach out to the prebid server host.\n",
			},
		},
	}

	for name, test := range tests {

		handlers := []struct {
			name string
			h    httprouter.Handle
			r    *http.Request
		}{
			vast(t, test.cfg, test.fetcher),
			event(test.cfg, test.fetcher),
		}

		for _, handler := range handlers {
			t.Run(handler.name+"-"+name, func(t *testing.T) {
				test.cfg.MarshalAccountDefaults()

				recorder := httptest.NewRecorder()

				// execute
				handler.h(recorder, handler.r, nil)
				d, err := ioutil.ReadAll(recorder.Result().Body)
				if err != nil {
					t.Fatal(err)
				}

				// validate
				assert.Equal(t, test.want.code, recorder.Result().StatusCode, fmt.Sprintf("Expected %d", test.want.code))
				assert.Equal(t, test.want.response, string(d))
			})
		}
	}
}

func event(cfg *config.Configuration, fetcher stored_requests.AccountFetcher) struct {
	name string
	h    httprouter.Handle
	r    *http.Request
} {
	return struct {
		name string
		h    httprouter.Handle
		r    *http.Request
	}{
		name: "event",
		h:    NewEventEndpoint(cfg, fetcher, nil),
		r:    httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=1&a=testacc", strings.NewReader("")),
	}
}

func vast(t *testing.T, cfg *config.Configuration, fetcher stored_requests.AccountFetcher) struct {
	name string
	h    httprouter.Handle
	r    *http.Request
} {
	vtrackBody, err := getValidVTrackRequestBody(true, true)
	if err != nil {
		t.Fatal(err)
	}

	return struct {
		name string
		h    httprouter.Handle
		r    *http.Request
	}{
		name: "vast",
		h:    NewVTrackEndpoint(cfg, fetcher, &vtrackMockCacheClient{}, config.BidderInfos{}),
		r:    httptest.NewRequest("POST", "/vtrack?a=testacc", strings.NewReader(vtrackBody)),
	}
}
