package ccpa

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

func TestConsentWriterWrite(t *testing.T) {
	consent := "anyConsent"
	testCases := []struct {
		description   string
		request       *openrtb.BidRequest
		expected      *openrtb.BidRequest
		expectedError bool
	}{
		{
			description: "Success",
			request:     &openrtb.BidRequest{},
			expected: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
			},
		},
		{
			description: "Nil Request",
			request:     nil,
			expected:    nil,
		},
		{
			description: "Error With Regs.Ext - Does Not Mutate",
			request: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`malformed}`)},
			},
			expectedError: true,
			expected: &openrtb.BidRequest{
				Regs: &openrtb.Regs{Ext: json.RawMessage(`malformed}`)},
			},
		},
	}

	for _, test := range testCases {
		writer := &consentWriter{consent}

		err := writer.Write(test.request)

		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, test.request, test.description)
	}
}

func TestNewConsentWriter(t *testing.T) {
	testCases := []string{
		"",
		"anyConsent",
	}

	for _, test := range testCases {
		writer := NewConsentWriter(test).(consentWriter)
		assert.Equal(t, test, writer.consent)
	}
}
