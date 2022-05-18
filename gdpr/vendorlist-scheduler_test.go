package gdpr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/go-gdpr/api"
	"github.com/stretchr/testify/assert"
)

func TestGetVendorListScheduler(t *testing.T) {
	type args struct {
		interval   string
		timeout    string
		httpClient *http.Client
	}
	tests := []struct {
		name    string
		args    args
		want    *vendorListScheduler
		wantErr bool
	}{
		{
			name: "Test singleton",
			args: args{
				interval:   "1m",
				timeout:    "1s",
				httpClient: http.DefaultClient,
			},
			want:    GetExpectedVendorListScheduler("1m", "1s", http.DefaultClient),
			wantErr: false,
		},
		{
			name: "Test singleton again",
			args: args{
				interval:   "2m",
				timeout:    "2s",
				httpClient: http.DefaultClient,
			},
			want:    GetExpectedVendorListScheduler("2m", "2s", http.DefaultClient),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//Mark instance as nil for recreating new instance
			if tt.want == nil {
				//_instance = nil
			}

			got, err := GetVendorListScheduler(tt.args.interval, tt.args.timeout, tt.args.httpClient)
			if got != tt.want {
				t.Errorf("GetVendorListScheduler() got = %v, want %v", got, tt.want)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVendorListScheduler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func GetExpectedVendorListScheduler(interval string, timeout string, httpClient *http.Client) *vendorListScheduler {
	s, _ := GetVendorListScheduler(interval, timeout, httpClient)
	return s
}

func Test_vendorListScheduler_Start(t *testing.T) {
	type fields struct {
		scheduler *vendorListScheduler
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Start test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler, err := GetVendorListScheduler("1m", "30s", http.DefaultClient)
			assert.Nil(t, err, "error should be nil")
			assert.NotNil(t, scheduler, "scheduler instance should not be nil")

			scheduler.Start()

			assert.NotNil(t, scheduler.ticker, "ticker should not be nil")
			assert.True(t, scheduler.isStarted, "isStarted should be true")

			scheduler.Stop()
		})
	}
}

func Test_vendorListScheduler_Stop(t *testing.T) {
	type fields struct {
		scheduler *vendorListScheduler
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Stop test",
		},
		{
			name: "Calling stop again",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler, err := GetVendorListScheduler("1m", "30s", http.DefaultClient)
			assert.Nil(t, err, "error should be nil")
			assert.NotNil(t, scheduler, "scheduler instance should not be nil")

			scheduler.Start()
			scheduler.Stop()

			assert.Nil(t, scheduler.ticker, "ticker should not be nil")
			assert.False(t, scheduler.isStarted, "isStarted should be true")
		})
	}
}

func Test_vendorListScheduler_runLoadCache(t *testing.T) {
	type fields struct {
		scheduler *vendorListScheduler
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "runLoadCache caches all files",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			tt.fields.scheduler, err = GetVendorListScheduler("5m", "5m", http.DefaultClient)
			assert.Nil(t, err, "error should be nil")
			assert.False(t, tt.fields.scheduler.isStarted, "VendorListScheduler should not be already running")

			tt.fields.scheduler.timeout = 2 * time.Minute

			mockCacheSave := func(uint16, api.VendorList) {}
			latestVersion := saveOne(context.Background(), http.DefaultClient, VendorListURLMaker(0), mockCacheSave)

			cacheSave, cacheLoad = newVendorListCache()
			tt.fields.scheduler.runLoadCache()

			firstVersionToLoad := uint16(2)
			for i := latestVersion; i >= firstVersionToLoad; i-- {
				list := cacheLoad(i)
				assert.NotNil(t, list, "vendor-list file should be present in cache")
			}
		})
	}
}

func Benchmark_vendorListScheduler_runLoadCache(b *testing.B) {
	scheduler, err := GetVendorListScheduler("1m", "30m", http.DefaultClient)
	assert.Nil(b, err, "")
	assert.NotNil(b, scheduler, "")

	scheduler.timeout = 2 * time.Minute

	for n := 0; n < b.N; n++ {
		cacheSave, cacheLoad = newVendorListCache()
		scheduler.runLoadCache()
	}

}

func Test_vendorListScheduler_cacheFuncs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]string{
			1: vendorList1,
			2: vendorList2,
		},
	})))
	defer server.Close()
	config := testConfig()

	_ = NewVendorListFetcher(context.Background(), config, server.Client(), testURLMaker(server))

	assert.NotNil(t, cacheSave, "Error gdpr.cacheSave nil")
	assert.NotNil(t, cacheLoad, "Error gdpr.cacheLoad nil")
}
