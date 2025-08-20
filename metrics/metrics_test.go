package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEndpointFromRequestType(t *testing.T) {
	testCases := []struct {
		name          string
		inRequestType RequestType
		expected      EndpointType
	}{
		{
			name:          "request-type-openrtb2-web",
			inRequestType: ReqTypeORTB2Web,
			expected:      EndpointAuction,
		},
		{
			name:          "request-type-openrtb2-app",
			inRequestType: ReqTypeORTB2App,
			expected:      EndpointAuction,
		},
		{
			name:          "request-type-openrtb2-dooh",
			inRequestType: ReqTypeORTB2DOOH,
			expected:      EndpointAuction,
		},
		{
			name:          "request-type-amp",
			inRequestType: ReqTypeAMP,
			expected:      EndpointAmp,
		},
		{
			name:          "request-type-video",
			inRequestType: ReqTypeVideo,
			expected:      EndpointVideo,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := GetEndpointFromRequestType(tc.inRequestType)

			assert.Equal(t, tc.expected, out)
		})
	}
}
