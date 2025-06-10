package yieldlab

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
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
	bidder, buildErr := Builder(openrtb_ext.BidderYieldlab, config.Adapter{
		Endpoint: testURL}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.NoError(t, buildErr)
	assert.NotNil(t, bidder)

	bidderYieldlab := bidder.(*YieldlabAdapter)
	assert.Equal(t, testURL, bidderYieldlab.endpoint)
	assert.NotNil(t, bidderYieldlab.cacheBuster)
	assert.NotNil(t, bidderYieldlab.getWeek)
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

func Test_makeNodeValue(t *testing.T) {
	int8TestCase := int8(8)
	tests := []struct {
		name      string
		nodeParam interface{}
		expected  string
	}{
		{
			name:      "string with special characters",
			nodeParam: "AZ09-._~:/?#[]@!$%&'()*+,;=",
			expected:  "AZ09-._~%3A%2F%3F%23%5B%5D%40%21%24%25%26%27%28%29%2A%2B%2C%3B%3D",
		},
		{
			name:      "int8 pointer",
			nodeParam: &int8TestCase,
			expected:  "8",
		},
		{
			name:      "int",
			nodeParam: 8,
			expected:  "8",
		},
		{
			name:      "free form data",
			nodeParam: json.RawMessage(`{"foo":"bar"}`),
			expected:  "%7B%22foo%22%3A%22bar%22%7D",
		},
		{
			name:      "unknown type (bool)",
			nodeParam: true,
			expected:  "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := makeNodeValue(test.nodeParam)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func Test_makeSupplyChain(t *testing.T) {
	hp := int8(1)
	tests := []struct {
		name     string
		param    openrtb2.SupplyChain
		expected string
	}{
		{
			name: "No nodes",
			param: openrtb2.SupplyChain{
				Ver:      "1.0",
				Complete: 1,
			},
			expected: "",
		},
		{
			name: "Not all fields",
			param: openrtb2.SupplyChain{
				Ver:      "1.0",
				Complete: 1,
				Nodes: []openrtb2.SupplyChainNode{
					{
						ASI: "exchange1.com",
						SID: "12345",
						HP:  &hp,
					},
				},
			},

			expected: "1.0,1!exchange1.com,12345,1,,,,",
		},
		{
			name: "All fields handled in correct order",
			param: openrtb2.SupplyChain{
				Ver:      "1.0",
				Complete: 1,
				Nodes: []openrtb2.SupplyChainNode{
					{
						ASI:    "exchange1.com",
						SID:    "12345",
						HP:     &hp,
						RID:    "bid-request-1",
						Name:   "publisher",
						Domain: "publisher.com",
						Ext:    []byte("{\"ext\":\"test\"}"),
					},
				},
			},
			expected: "1.0,1!exchange1.com,12345,1,bid-request-1,publisher,publisher.com,%7B%22ext%22%3A%22test%22%7D",
		},
		{
			name: "handle simple node.ext type (string)",
			param: openrtb2.SupplyChain{
				Ver:      "1.0",
				Complete: 1,
				Nodes: []openrtb2.SupplyChainNode{
					{
						Ext: []byte("\"ext\""),
					},
				},
			},
			expected: "1.0,1!,,,,,,%22ext%22",
		},
		{
			name: "handle simple node.ext type (int)",
			param: openrtb2.SupplyChain{
				Ver:      "1.0",
				Complete: 1,
				Nodes: []openrtb2.SupplyChainNode{
					{
						Ext: []byte(strconv.Itoa(1)),
					},
				},
			},
			expected: "1.0,1!,,,,,,1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := makeSupplyChain(test.param)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func Test_makeDSATransparencyUrlParam(t *testing.T) {
	tests := []struct {
		name           string
		transparencies []dsaTransparency
		expected       string
	}{
		{
			name:           "No transparency objects",
			transparencies: []dsaTransparency{},
			expected:       "",
		},
		{
			name:           "Nil transparency",
			transparencies: nil,
			expected:       "",
		},
		{
			name: "Params without a domain",
			transparencies: []dsaTransparency{
				{
					Params: []int{1, 2},
				},
			},
			expected: "",
		},
		{
			name: "Params without a params",
			transparencies: []dsaTransparency{
				{
					Domain: "domain.com",
				},
			},
			expected: "domain.com",
		},
		{
			name: "One object; No Params",
			transparencies: []dsaTransparency{
				{
					Domain: "domain.com",
					Params: []int{},
				},
			},
			expected: "domain.com",
		},
		{
			name: "One object; One Param",
			transparencies: []dsaTransparency{
				{
					Domain: "domain.com",
					Params: []int{1},
				},
			},
			expected: "domain.com~1",
		},
		{
			name: "Three domain objects",
			transparencies: []dsaTransparency{
				{
					Domain: "domain1.com",
					Params: []int{1, 2},
				},
				{
					Domain: "domain2.com",
					Params: []int{3, 4},
				},
				{
					Domain: "domain3.com",
					Params: []int{5, 6},
				},
			},
			expected: "domain1.com~1_2~~domain2.com~3_4~~domain3.com~5_6",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := makeDSATransparencyURLParam(test.transparencies)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func Test_getDSA_invalidRequestExt(t *testing.T) {
	req := &openrtb2.BidRequest{
		Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"DSA":"wrongValueType"}`)},
	}

	dsa, err := getDSA(req)

	assert.NotNil(t, err)
	assert.Nil(t, dsa)
}

func TestYieldlabAdapter_makeEndpointURL_invalidEndpoint(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderYieldlab, config.Adapter{
		Endpoint: "test$:/something§"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderYieldlab := bidder.(*YieldlabAdapter)
	_, err := bidderYieldlab.makeEndpointURL(nil, nil)
	assert.Error(t, err)
}
