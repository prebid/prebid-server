package exchange

import (
	"encoding/json"
	"testing"

	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/gdpr"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestGetGDPR(t *testing.T) {
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
		{
			description: "Regs Ext GDPR = null, GPPSID has tcf2",
			giveRegs:    &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr": null}`), GPPSID: []int8{2}},
			wantGDPR:    gdpr.SignalYes,
		},
		{
			description: "Regs Ext GDPR = 1, GPPSID has uspv1",
			giveRegs:    &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr": 1}`), GPPSID: []int8{6}},
			wantGDPR:    gdpr.SignalNo,
		},
		{
			description: "Regs Ext GDPR = 0, GPPSID has tcf2",
			giveRegs:    &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr": 0}`), GPPSID: []int8{2}},
			wantGDPR:    gdpr.SignalYes,
		},
		{
			description: "Regs Ext is nil, GPPSID has tcf2",
			giveRegs:    &openrtb2.Regs{GPPSID: []int8{2}},
			wantGDPR:    gdpr.SignalYes,
		},
		{
			description: "Regs Ext is nil, GPPSID has uspv1",
			giveRegs:    &openrtb2.Regs{GPPSID: []int8{6}},
			wantGDPR:    gdpr.SignalNo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			req := openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: tt.giveRegs,
				},
			}
			result, err := getGDPR(&req)
			assert.Equal(t, tt.wantGDPR, result)

			if tt.wantError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetConsent(t *testing.T) {
	tests := []struct {
		description string
		giveUser    *openrtb2.User
		giveGPP     gpplib.GppContainer
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
		{
			description: "User Ext is nil, GPP has no GDPR",
			giveUser:    &openrtb2.User{Ext: nil},
			giveGPP:     gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{6}, Sections: []gpplib.Section{&upsv1Section}},
			wantConsent: "",
		},
		{
			description: "User Ext is nil, GPP has GDPR",
			giveUser:    &openrtb2.User{Ext: nil},
			giveGPP:     gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{2}, Sections: []gpplib.Section{&tcf1Section}},
			wantConsent: "BOS2bx5OS2bx5ABABBAAABoAAAAAFA",
		},
		{
			description: "User Ext has GDPR, GPP has GDPR",
			giveUser:    &openrtb2.User{Ext: json.RawMessage(`{"consent": "BSOMECONSENT"}`)},
			giveGPP:     gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{2}, Sections: []gpplib.Section{&tcf1Section}},
			wantConsent: "BOS2bx5OS2bx5ABABBAAABoAAAAAFA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			req := openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: tt.giveUser,
				},
			}

			result, err := getConsent(&req, tt.giveGPP)
			assert.Equal(t, tt.wantConsent, result, tt.description)

			if tt.wantError {
				assert.NotNil(t, err, tt.description)
			} else {
				assert.Nil(t, err, tt.description)
			}
		})
	}
}

var upsv1Section mockGPPSection = mockGPPSection{sectionID: 6, value: "1YNY"}
var tcf1Section mockGPPSection = mockGPPSection{sectionID: 2, value: "BOS2bx5OS2bx5ABABBAAABoAAAAAFA"}

type mockGPPSection struct {
	sectionID gppConstants.SectionID
	value     string
}

func (ms mockGPPSection) GetID() gppConstants.SectionID {
	return ms.sectionID
}

func (ms mockGPPSection) GetValue() string {
	return ms.value
}

func (ms mockGPPSection) Encode(bool) []byte {
	return nil
}
