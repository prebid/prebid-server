package ccpa

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

// RegExt.SetUSPrivacy() is the new ConsentWriter
func TestConsentWriter(t *testing.T) {
	consent := "anyConsent"
	testCases := []struct {
		description   string
		request       *openrtb2.BidRequest
		expected      *openrtb2.BidRequest
		expectedError bool
	}{
		{
			description: "Nil Request",
			request:     nil,
			expected:    nil,
		},
		{
			description: "Success",
			request:     &openrtb2.BidRequest{},
			expected: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
			},
		},
		{
			description: "Error With Regs.Ext - Does Not Mutate",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`malformed}`)},
			},
			expectedError: false,
			expected: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`malformed}`)},
			},
		},
	}

	for _, test := range testCases {

		reqWrapper := &openrtb_ext.RequestWrapper{BidRequest: test.request}
		var err error
		regsExt, err1 := reqWrapper.GetRegExt()
		if err1 == nil {
			regsExt.SetUSPrivacy(consent)
			if reqWrapper.BidRequest != nil {
				err = reqWrapper.RebuildRequest()
			}
		}
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, reqWrapper.BidRequest, test.description)
	}
}

func TestConsentWriterLegacy(t *testing.T) {
	consent := "anyConsent"
	testCases := []struct {
		description   string
		request       *openrtb2.BidRequest
		expected      *openrtb2.BidRequest
		expectedError bool
	}{
		{
			description: "Nil Request",
			request:     nil,
			expected:    nil,
		},
		{
			description: "Success",
			request:     &openrtb2.BidRequest{},
			expected: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"anyConsent"}`)},
			},
		},
		{
			description: "Error With Regs.Ext - Does Not Mutate",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`malformed}`)},
			},
			expectedError: true,
			expected: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`malformed}`)},
			},
		},
	}

	for _, test := range testCases {
		writer := ConsentWriter{consent}

		err := writer.Write(test.request)

		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, test.request, test.description)
	}
}
