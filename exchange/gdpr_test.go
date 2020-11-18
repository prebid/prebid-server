package exchange

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestExtractGDPR(t *testing.T) {
	tests := []struct {
		description string
		giveRegs    *openrtb.Regs
		wantGDPR    gdpr.GDPRState
	}{
		{
			description: "Regs Ext GDPR = 0",
			giveRegs:    &openrtb.Regs{Ext: json.RawMessage(`{"gdpr": 0}`)},
			wantGDPR:    gdpr.NoGDPR,
		},
		{
			description: "Regs Ext GDPR = 1",
			giveRegs:    &openrtb.Regs{Ext: json.RawMessage(`{"gdpr": 1}`)},
			wantGDPR:    gdpr.YesGDPR,
		},
		{
			description: "Regs Ext GDPR = null",
			giveRegs:    &openrtb.Regs{Ext: json.RawMessage(`{"gdpr": null}`)},
			wantGDPR:    gdpr.AmbiguousGDPR,
		},
		{
			description: "Regs is nil",
			giveRegs:    nil,
			wantGDPR:    gdpr.AmbiguousGDPR,
		},
		{
			description: "Regs Ext is nil",
			giveRegs:    &openrtb.Regs{Ext: nil},
			wantGDPR:    gdpr.AmbiguousGDPR,
		},
		{
			description: "JSON unmarshal error",
			giveRegs:    &openrtb.Regs{Ext: json.RawMessage(`{"`)},
			wantGDPR:    gdpr.AmbiguousGDPR,
		},
	}

	for _, tt := range tests {
		bidReq := openrtb.BidRequest{
			Regs: tt.giveRegs,
		}

		result := extractGDPR(&bidReq)
		assert.Equal(t, tt.wantGDPR, result, tt.description)
	}
}

func TestExtractConsent(t *testing.T) {
	tests := []struct {
		description string
		giveUser    *openrtb.User
		wantConsent string
	}{
		{
			description: "User Ext Consent is not empty",
			giveUser:    &openrtb.User{Ext: json.RawMessage(`{"consent": "BOS2bx5OS2bx5ABABBAAABoAAAAAFA"}`)},
			wantConsent: "BOS2bx5OS2bx5ABABBAAABoAAAAAFA",
		},
		{
			description: "User Ext Consent is empty",
			giveUser:    &openrtb.User{Ext: json.RawMessage(`{"consent": ""}`)},
			wantConsent: "",
		},
		{
			description: "User Ext is nil",
			giveUser:    &openrtb.User{Ext: nil},
			wantConsent: "",
		},
		{
			description: "User is nil",
			giveUser:    nil,
			wantConsent: "",
		},
		{
			description: "JSON unmarshal error",
			giveUser:    &openrtb.User{Ext: json.RawMessage(`{`)},
			wantConsent: "",
		},
	}

	for _, tt := range tests {
		bidReq := openrtb.BidRequest{
			User: tt.giveUser,
		}

		result := extractConsent(&bidReq)
		assert.Equal(t, tt.wantConsent, result, tt.description)
	}
}
