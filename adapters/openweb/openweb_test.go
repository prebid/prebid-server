package openweb

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderOpenWeb, config.Adapter{
		Endpoint: "https://pbs.openwebmp.com/pbs"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "openwebtest", bidder)
}

func TestBuildEndpoint(t *testing.T) {
	tests := []struct {
		name      string
		adapter   *adapter
		testParam int8
		org       string
		expected  string
	}{
		{
			name: "production endpoint with org",
			adapter: &adapter{
				endpoint: "https://pbs.openwebmp.com/pbs",
			},
			testParam: 0,
			org:       "test_org",
			expected:  "https://pbs.openwebmp.com/pbs?publisher_id=test_org",
		},
		{
			name: "test endpoint when testParam is 1",
			adapter: &adapter{
				endpoint: "https://pbs.openwebmp.com/pbs",
			},
			testParam: 1,
			org:       "test_org",
			expected:  "https://pbs.openwebmp.com/pbs-test?publisher_id=test_org",
		},
		{
			name: "empty org parameter",
			adapter: &adapter{
				endpoint: "https://pbs.openwebmp.com/pbs",
			},
			testParam: 0,
			org:       "",
			expected:  "https://pbs.openwebmp.com/pbs?publisher_id=",
		},
		{
			name: "testParam other than 1 uses production endpoint",
			adapter: &adapter{
				endpoint: "https://pbs.openwebmp.com/pbs",
			},
			testParam: 2,
			org:       "test_org",
			expected:  "https://pbs.openwebmp.com/pbs?publisher_id=test_org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.adapter.buildEndpoint(tt.testParam, tt.org)
			if result != tt.expected {
				t.Errorf("buildEndpoint() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
