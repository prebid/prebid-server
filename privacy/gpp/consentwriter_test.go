package gpp

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestConsentWriter_Write(t *testing.T) {
	tests := []struct {
		name              string
		writer            ConsentWriter
		expectedGpp       string
		expectedGppSid    []int8
		expectedGppSidNil bool
	}{
		{
			name: "write both gpp and gpp_sid",
			writer: ConsentWriter{
				Consent: "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
				GppSid:  "2,4,6",
			},
			expectedGpp:       "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
			expectedGppSid:    []int8{2, 4, 6},
			expectedGppSidNil: false,
		},
		{
			name: "write only gpp string",
			writer: ConsentWriter{
				Consent: "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
				GppSid:  "",
			},
			expectedGpp:       "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
			expectedGppSid:    nil,
			expectedGppSidNil: true,
		},
		{
			name: "invalid gpp_sid results in nil GPPSID",
			writer: ConsentWriter{
				Consent: "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
				GppSid:  "malformed",
			},
			expectedGpp:       "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
			expectedGppSid:    nil,
			expectedGppSidNil: true,
		},
		{
			name: "nil request does not panic",
			writer: ConsentWriter{
				Consent: "test",
				GppSid:  "2",
			},
			expectedGpp:       "",
			expectedGppSid:    nil,
			expectedGppSidNil: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var req *openrtb2.BidRequest
			if test.name != "nil request does not panic" {
				req = &openrtb2.BidRequest{}
			}

			err := test.writer.Write(req)
			assert.NoError(t, err)

			if req != nil {
				assert.NotNil(t, req.Regs)
				assert.Equal(t, test.expectedGpp, req.Regs.GPP)

				if test.expectedGppSidNil {
					assert.Nil(t, req.Regs.GPPSID)
				} else {
					assert.Equal(t, test.expectedGppSid, req.Regs.GPPSID)
				}
			}
		})
	}
}
