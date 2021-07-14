package exchange

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestExtractGDPR(t *testing.T) {
	tests := []struct {
		description string
		giveRegs    *openrtb2.Regs
		wantGDPR    gdpr.Signal
		wantError   bool
	}{
		{
			description: "Regs Ext GDPR = 0",
			giveRegs:    &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr": 0}`)},
			wantGDPR:    gdpr.SignalNo,
		},
		{
			description: "Regs Ext GDPR = 1",
			giveRegs:    &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr": 1}`)},
			wantGDPR:    gdpr.SignalYes,
		},
		{
			description: "Regs Ext GDPR = null",
			giveRegs:    &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr": null}`)},
			wantGDPR:    gdpr.SignalAmbiguous,
		},
		{
			description: "Regs is nil",
			giveRegs:    nil,
			wantGDPR:    gdpr.SignalAmbiguous,
		},
		{
			description: "Regs Ext is nil",
			giveRegs:    &openrtb2.Regs{Ext: nil},
			wantGDPR:    gdpr.SignalAmbiguous,
		},
		{
			description: "JSON unmarshal error",
			giveRegs:    &openrtb2.Regs{Ext: json.RawMessage(`{"`)},
			wantGDPR:    gdpr.SignalAmbiguous,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		bidReq := openrtb2.BidRequest{
			Regs: tt.giveRegs,
		}

		result, err := extractGDPR(&bidReq)
		assert.Equal(t, tt.wantGDPR, result, tt.description)

		if tt.wantError {
			assert.NotNil(t, err, tt.description)
		} else {
			assert.Nil(t, err, tt.description)
		}
	}
}

func TestExtractConsent(t *testing.T) {
	tests := []struct {
		description string
		giveUser    *openrtb2.User
		wantConsent string
		wantError   bool
	}{
		{
			description: "User Ext Consent is not empty",
			giveUser:    &openrtb2.User{Ext: json.RawMessage(`{"consent": "BOS2bx5OS2bx5ABABBAAABoAAAAAFA"}`)},
			wantConsent: "BOS2bx5OS2bx5ABABBAAABoAAAAAFA",
		},
		{
			description: "User Ext Consent is empty",
			giveUser:    &openrtb2.User{Ext: json.RawMessage(`{"consent": ""}`)},
			wantConsent: "",
		},
		{
			description: "User Ext is nil",
			giveUser:    &openrtb2.User{Ext: nil},
			wantConsent: "",
		},
		{
			description: "User is nil",
			giveUser:    nil,
			wantConsent: "",
		},
		{
			description: "JSON unmarshal error",
			giveUser:    &openrtb2.User{Ext: json.RawMessage(`{`)},
			wantConsent: "",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		bidReq := openrtb2.BidRequest{
			User: tt.giveUser,
		}

		result, err := extractConsent(&bidReq)
		assert.Equal(t, tt.wantConsent, result, tt.description)

		if tt.wantError {
			assert.NotNil(t, err, tt.description)
		} else {
			assert.Nil(t, err, tt.description)
		}
	}
}
