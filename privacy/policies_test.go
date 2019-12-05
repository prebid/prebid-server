package privacy

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestWrite(t *testing.T) {
	polciies := Policies{
		GDPR: gdpr.Policy{Consent: "anyConsent"},
		CCPA: ccpa.Policy{Value: "anyValue"},
	}
	expectedRequest := &openrtb.BidRequest{
		Regs: &openrtb.Regs{
			Ext: json.RawMessage(`{"us_privacy":"anyValue"}`)},
		User: &openrtb.User{
			Ext: json.RawMessage(`{"consent":"anyConsent"}`)},
	}

	request := &openrtb.BidRequest{}
	err := polciies.Write(request)

	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, request)
}

func TestWriteWithErrorFromGDPR(t *testing.T) {
	polciies := Policies{
		GDPR: gdpr.Policy{Consent: "anyConsent"},
		CCPA: ccpa.Policy{Value: "anyValue"},
	}
	request := &openrtb.BidRequest{
		User: &openrtb.User{
			Ext: json.RawMessage(`malformed`)},
	}

	err := polciies.Write(request)

	assert.Error(t, err)
}

func TestWriteWithErrorFromCCPA(t *testing.T) {
	polciies := Policies{
		GDPR: gdpr.Policy{Consent: "anyConsent"},
		CCPA: ccpa.Policy{Value: "anyValue"},
	}
	request := &openrtb.BidRequest{
		Regs: &openrtb.Regs{
			Ext: json.RawMessage(`malformed`)},
	}

	err := polciies.Write(request)

	assert.Error(t, err)
}
