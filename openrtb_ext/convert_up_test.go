package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/stretchr/testify/assert"
)

// func migrateGDPRFrom25To26(r *RequestWrapper) {
// 	// read and clear 2.5 location
// 	regsExt, _ := r.GetRegExt()
// 	gdpr25 := regsExt.GetGDPR()

// 	// move to 2.6 location
// 	if gdpr25 != nil {
// 		if r.Regs == nil {
// 			r.Regs = &openrtb2.Regs{}
// 		}
// 		r.Regs.GDPR = gdpr25
// 	}
// }

func TestMigrateGDPRFrom25To26(t *testing.T) {
	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
	}{
		{
			description:     "Not Present",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "2.5 Migrated To 2.6",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":0}`)}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0)}},
		},
		{
			description:     "2.5 Dropped",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0), Ext: json.RawMessage(`{"gdpr":1}`)}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0)}},
		},
		{
			description:     "2.6 Left Alone",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0)}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0)}},
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		migrateGDPRFrom25To26(w)
		assert.NoError(t, w.RebuildRequest())
		assert.Equal(t, test.expectedRequest, *w.BidRequest)
	}
}
