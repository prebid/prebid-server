package events

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleAccountServiceErrors(t *testing.T) {
	tests := map[string]struct {
		fetcher      *mockAccountsFetcher
		cfg          *config.Configuration
		accountID    string
		wantCode     int
		wantResponse string
	}{
		"bad-request": {
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
			accountID:    "testacc",
			wantCode:     400,
			wantResponse: "Invalid request: some error\nInvalid request: Prebid-server could not verify the Account ID. Please reach out to the prebid server host.\n",
		},
		"malformed-account-config": {
			fetcher: &mockAccountsFetcher{
				Fail:  true,
				Error: &errortypes.MalformedAcct{},
			},
			cfg: &config.Configuration{
				MaxRequestSize: maxSize,
				VTrack: config.VTrack{
					TimeoutMS: int64(2000), AllowUnknownBidder: false,
				},
			},
			accountID:    "malformed_acct",
			wantCode:     500,
			wantResponse: "Invalid request: The prebid-server account config for account id \"malformed_acct\" is malformed. Please reach out to the prebid server host.\n",
		},
		"service-unavailable": {
			fetcher: &mockAccountsFetcher{
				Fail: false,
			},
			cfg: &config.Configuration{
				AccountDefaults: config.Account{},
				AccountRequired: true,
				MaxRequestSize:  maxSize,
				VTrack: config.VTrack{
					TimeoutMS: int64(2000), AllowUnknownBidder: false,
				},
			},
			accountID:    "disabled_acct",
			wantCode:     503,
			wantResponse: "Invalid request: Prebid-server has disabled Account ID: disabled_acct, please reach out to the prebid server host.\n",
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
			accountID:    "testacc",
			wantCode:     504,
			wantResponse: "Invalid request: context deadline exceeded\nInvalid request: Prebid-server could not verify the Account ID. Please reach out to the prebid server host.\n",
		},
	}

	for name, test := range tests {
		handlers := []struct {
			name string
			h    httprouter.Handle
			r    *http.Request
		}{
			vast(t, test.cfg, test.fetcher, test.accountID),
			event(test.cfg, test.fetcher, test.accountID),
		}

		for _, handler := range handlers {
			t.Run(handler.name+"-"+name, func(t *testing.T) {
				test.cfg.MarshalAccountDefaults()

				recorder := httptest.NewRecorder()

				// execute
				handler.h(recorder, handler.r, nil)
				d, err := io.ReadAll(recorder.Result().Body)
				require.NoError(t, err)

				// validate
				assert.Equal(t, test.wantCode, recorder.Result().StatusCode)
				assert.Equal(t, test.wantResponse, string(d))
			})
		}
	}
}

func event(cfg *config.Configuration, fetcher stored_requests.AccountFetcher, accountID string) struct {
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
		h:    NewEventEndpoint(cfg, fetcher, nil, &metrics.MetricsEngineMock{}),
		r:    httptest.NewRequest("GET", "/event?t=win&b=test&ts=1234&f=b&x=1&a="+accountID, strings.NewReader("")),
	}
}

func vast(t *testing.T, cfg *config.Configuration, fetcher stored_requests.AccountFetcher, accountID string) struct {
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
		h:    NewVTrackEndpoint(cfg, fetcher, &vtrackMockCacheClient{}, config.BidderInfos{}, &metrics.MetricsEngineMock{}),
		r:    httptest.NewRequest("POST", "/vtrack?a="+accountID, strings.NewReader(vtrackBody)),
	}
}
