package exchange

import (
	"testing"

	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
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
			giveRegs:    &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](0)},
			wantGDPR:    gdpr.SignalNo,
		},
		{
			description: "Regs Ext GDPR = 1",
			giveRegs:    &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](1)},
			wantGDPR:    gdpr.SignalYes,
		},
		{
			description: "Regs Ext GDPR = null",
			giveRegs:    &openrtb2.Regs{GDPR: nil},
			wantGDPR:    gdpr.SignalAmbiguous,
		},
		{
			description: "Regs is nil",
			giveRegs:    nil,
			wantGDPR:    gdpr.SignalAmbiguous,
		},
		{
			description: "Regs Ext GDPR = null, GPPSID has tcf2",
			giveRegs:    &openrtb2.Regs{GDPR: nil, GPPSID: []int8{2}},
			wantGDPR:    gdpr.SignalYes,
		},
		{
			description: "Regs Ext GDPR = 1, GPPSID has uspv1",
			giveRegs:    &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](1), GPPSID: []int8{6}},
			wantGDPR:    gdpr.SignalNo,
		},
		{
			description: "Regs Ext GDPR = 0, GPPSID has tcf2",
			giveRegs:    &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](0), GPPSID: []int8{2}},
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
			description: "User Consent is not empty",
			giveUser:    &openrtb2.User{Consent: "BOS2bx5OS2bx5ABABBAAABoAAAAAFA"},
			wantConsent: "BOS2bx5OS2bx5ABABBAAABoAAAAAFA",
		},
		{
			description: "User Consent is empty",
			giveUser:    &openrtb2.User{Consent: ""},
			wantConsent: "",
		},
		{
			description: "User is nil",
			giveUser:    nil,
			wantConsent: "",
		},
		{
			description: "User is nil, GPP has no GDPR",
			giveUser:    nil,
			giveGPP:     gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{6}, Sections: []gpplib.Section{&upsv1Section}},
			wantConsent: "",
		},
		{
			description: "User is nil, GPP has GDPR",
			giveUser:    nil,
			giveGPP:     gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{2}, Sections: []gpplib.Section{&tcf1Section}},
			wantConsent: "BOS2bx5OS2bx5ABABBAAABoAAAAAFA",
		},
		{
			description: "User has GDPR, GPP has GDPR",
			giveUser:    &openrtb2.User{Consent: "BSOMECONSENT"},
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

func TestSelectEEACountries(t *testing.T) {
	tests := []struct {
		description         string
		hostEEACountries    []string
		accountEEACountries []string
		expected            []string
	}{
		{
			description:         "Account_EEA_countries_provided",
			hostEEACountries:    []string{"UK", "DE"},
			accountEEACountries: []string{"FR", "IT"},
			expected:            []string{"FR", "IT"},
		},
		{
			description:         "Account_is_nil",
			hostEEACountries:    []string{"UK"},
			accountEEACountries: nil,
			expected:            []string{"UK"},
		},
		{
			description:         "Both_nil",
			hostEEACountries:    nil,
			accountEEACountries: nil,
			expected:            nil,
		},
		{
			description:         "Account_is_empty_slice",
			hostEEACountries:    []string{"UK"},
			accountEEACountries: []string{},
			expected:            []string{},
		},
		{
			description:         "Host_is_nil",
			hostEEACountries:    nil,
			accountEEACountries: []string{"DE"},
			expected:            []string{"DE"},
		},
		{
			description:         "Host_and_account_both_non-nil",
			hostEEACountries:    []string{"UK"},
			accountEEACountries: []string{"FR"},
			expected:            []string{"FR"},
		},
		{
			description:         "Host_is_empty_slice,_account_is_nil",
			hostEEACountries:    []string{},
			accountEEACountries: nil,
			expected:            []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := selectEEACountries(tt.hostEEACountries, tt.accountEEACountries)
			assert.Equal(t, tt.expected, result)
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
