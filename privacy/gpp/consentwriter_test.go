package gpp

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestConsentWriter_Write(t *testing.T) {
	tests := []struct {
		name           string
		nilReq         bool
		writer         ConsentWriter
		expectedGpp    string
		expectedGppSid []int8
	}{
		{
			name: "write both gpp and gpp_sid",
			writer: ConsentWriter{
				Consent: "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
				GppSid:  []int8{2, 4, 6},
			},
			expectedGpp:    "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
			expectedGppSid: []int8{2, 4, 6},
		},
		{
			name: "write only gpp string",
			writer: ConsentWriter{
				Consent: "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
				GppSid:  nil,
			},
			expectedGpp:    "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
			expectedGppSid: nil,
		},
		{
			name: "write only gpp_sid, empty consent",
			writer: ConsentWriter{
				Consent: "",
				GppSid:  []int8{2},
			},
			expectedGpp:    "",
			expectedGppSid: []int8{2},
		},
		{
			name:   "nil request does not panic",
			nilReq: true,
			writer: ConsentWriter{
				Consent: "test",
				GppSid:  []int8{2},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var req *openrtb2.BidRequest
			if !test.nilReq {
				req = &openrtb2.BidRequest{}
			}

			err := test.writer.Write(req)
			assert.NoError(t, err)

			if req != nil {
				assert.NotNil(t, req.Regs)
				assert.Equal(t, test.expectedGpp, req.Regs.GPP)
				assert.Equal(t, test.expectedGppSid, req.Regs.GPPSID)
			}
		})
	}
}
