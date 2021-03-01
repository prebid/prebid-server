package ccpa

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestConsentWriter(t *testing.T) {
	consent := "anyConsent"
	testCases := []struct {
		description   string
		request       *openrtb.BidRequest
		expected      *openrtb.BidRequest
		expectedError bool
	}{
		{
			description: "Nil Request",
			request:     nil,
			expected:    nil,
		},
		{
			description: "Success",
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
			},
		},
		{
			description: "Error With Regs.Ext - Does Not Mutate",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`malformed}`)},
			},
			expectedError: false,
			expected: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`malformed}`)},
			},
		},
	}

	for _, test := range testCases {
		writer := ConsentWriter{consent}

		reqWrapper := &openrtb_ext.RequestWrapper{Request: test.request}
		var err error
		writer.Write(reqWrapper)

		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, test.request, test.description)
	}
}
