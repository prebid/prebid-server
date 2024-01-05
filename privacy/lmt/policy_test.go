package lmt

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestReadFromRequest(t *testing.T) {
	var one int8 = 1

	testCases := []struct {
		description    string
		request        *openrtb2.BidRequest
		expectedPolicy Policy
	}{
		{
			description: "Nil Request",
			request:     nil,
			expectedPolicy: Policy{
				Signal:         0,
				SignalProvided: false,
			},
		},
		{
			description: "Nil Device",
			request: &openrtb2.BidRequest{
				Device: nil,
			},
			expectedPolicy: Policy{
				Signal:         0,
				SignalProvided: false,
			},
		},
		{
			description: "Nil Device.Lmt",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					Lmt: nil,
				},
			},
			expectedPolicy: Policy{
				Signal:         0,
				SignalProvided: false,
			},
		},
		{
			description: "Enabled",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					Lmt: &one,
				},
			},
			expectedPolicy: Policy{
				Signal:         1,
				SignalProvided: true,
			},
		},
	}

	for _, test := range testCases {
		p := ReadFromRequest(test.request)
		assert.Equal(t, test.expectedPolicy, p, test.description)
	}
}

func TestCanEnforce(t *testing.T) {
	testCases := []struct {
		description string
		policy      Policy
		expected    bool
	}{
		{
			description: "Signal Not Provided - Zero",
			policy: Policy{
				Signal:         0,
				SignalProvided: false,
			},
			expected: false,
		},
		{
			description: "Signal Not Provided - One",
			policy: Policy{
				Signal:         1,
				SignalProvided: false,
			},
			expected: false,
		},
		{
			description: "Signal Not Provided - Other",
			policy: Policy{
				Signal:         42,
				SignalProvided: false,
			},
			expected: false,
		},
		{
			description: "Signal Provided - Zero",
			policy: Policy{
				Signal:         0,
				SignalProvided: true,
			},
			expected: true,
		},
		{
			description: "Signal Provided - One",
			policy: Policy{
				Signal:         1,
				SignalProvided: true,
			},
			expected: true,
		},
		{
			description: "Signal Provided - Other",
			policy: Policy{
				Signal:         42,
				SignalProvided: true,
			},
			expected: true,
		},
	}

	for _, test := range testCases {
		result := test.policy.CanEnforce()
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestShouldEnforce(t *testing.T) {
	testCases := []struct {
		description string
		policy      Policy
		expected    bool
	}{
		{
			description: "Signal Not Provided - Zero",
			policy: Policy{
				Signal:         0,
				SignalProvided: false,
			},
			expected: false,
		},
		{
			description: "Signal Not Provided - One",
			policy: Policy{
				Signal:         1,
				SignalProvided: false,
			},
			expected: false,
		},
		{
			description: "Signal Not Provided - Other",
			policy: Policy{
				Signal:         42,
				SignalProvided: false,
			},
			expected: false,
		},
		{
			description: "Signal Provided - Zero",
			policy: Policy{
				Signal:         0,
				SignalProvided: true,
			},
			expected: false,
		},
		{
			description: "Signal Provided - One",
			policy: Policy{
				Signal:         1,
				SignalProvided: true,
			},
			expected: true,
		},
		{
			description: "Signal Provided - Other",
			policy: Policy{
				Signal:         42,
				SignalProvided: true,
			},
			expected: false,
		},
	}

	for _, test := range testCases {
		result := test.policy.ShouldEnforce("")
		assert.Equal(t, test.expected, result, test.description)
	}
}
