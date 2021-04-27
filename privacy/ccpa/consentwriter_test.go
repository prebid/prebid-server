package ccpa

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

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
		writer := ConsentWriter{consent}

		reqWrapper := &openrtb_ext.RequestWrapper{Request: test.request}
		var err error
		err1 := reqWrapper.ExtractRegExt()
		if err1 == nil {
			writer.Write(reqWrapper)
			if reqWrapper.Request != nil {
				err = reqWrapper.Sync()
			}
		}
		assertError(t, test.expectedError, err, test.description)
		assert.Equal(t, test.expected, reqWrapper.Request, test.description)
	}
}
