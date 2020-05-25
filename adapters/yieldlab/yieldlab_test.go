package yieldlab

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testURL = "https://ad.yieldlab.net/testing/"

var testCacheBuster cacheBuster = func() string {
	return "testing"
}

var testWeekGenerator weekGenerator = func() string {
	return "33"
}

func newTestYieldlabBidder(endpoint string) *YieldlabAdapter {
	return &YieldlabAdapter{
		endpoint:    endpoint,
		cacheBuster: testCacheBuster,
		getWeek:     testWeekGenerator,
	}
}

func TestNewYieldlabBidder(t *testing.T) {
	bid := NewYieldlabBidder(testURL)
	assert.NotNil(t, bid)
	assert.Equal(t, bid.endpoint, testURL)
	assert.NotNil(t, bid.cacheBuster)
	assert.NotNil(t, bid.getWeek)
}

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "yieldlabtest", newTestYieldlabBidder(testURL))
}

func Test_splitSize(t *testing.T) {
	type args struct {
		size string
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		want1   uint64
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				size: "300x800",
			},
			want:    300,
			want1:   800,
			wantErr: false,
		},
		{
			name: "empty",
			args: args{
				size: "",
			},
			want:    0,
			want1:   0,
			wantErr: false,
		},
		{
			name: "invalid",
			args: args{
				size: "test",
			},
			want:    0,
			want1:   0,
			wantErr: false,
		},
		{
			name: "invalid_height",
			args: args{
				size: "200xtest",
			},
			want:    0,
			want1:   0,
			wantErr: true,
		},
		{
			name: "invalid_width",
			args: args{
				size: "testx200",
			},
			want:    0,
			want1:   0,
			wantErr: true,
		},
		{
			name: "invalid_separator",
			args: args{
				size: "200y200",
			},
			want:    0,
			want1:   0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := splitSize(tt.args.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("splitSize() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("splitSize() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestYieldlabAdapter_makeEndpointURL_invalidEndpoint(t *testing.T) {
	bid := NewYieldlabBidder("test$:/something§")
	_, err := bid.makeEndpointURL(nil, nil)
	assert.Error(t, err)
}
