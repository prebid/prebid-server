package gdpr

import (
	"testing"

	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestGetGDPR(t *testing.T) {
	tests := []struct {
		description string
		giveRegs    *openrtb2.Regs
		wantGDPR    Signal
		wantError   bool
	}{
		{
			description: "Regs Ext GDPR = 0",
			giveRegs:    &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](0)},
			wantGDPR:    SignalNo,
		},
		{
			description: "Regs Ext GDPR = 1",
			giveRegs:    &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](1)},
			wantGDPR:    SignalYes,
		},
		{
			description: "Regs Ext GDPR = null",
			giveRegs:    &openrtb2.Regs{GDPR: nil},
			wantGDPR:    SignalAmbiguous,
		},
		{
			description: "Regs is nil",
			giveRegs:    nil,
			wantGDPR:    SignalAmbiguous,
		},
		{
			description: "Regs Ext GDPR = null, GPPSID has tcf2",
			giveRegs:    &openrtb2.Regs{GDPR: nil, GPPSID: []int8{2}},
			wantGDPR:    SignalYes,
		},
		{
			description: "Regs Ext GDPR = 1, GPPSID has uspv1",
			giveRegs:    &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](1), GPPSID: []int8{6}},
			wantGDPR:    SignalNo,
		},
		{
			description: "Regs Ext GDPR = 0, GPPSID has tcf2",
			giveRegs:    &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](0), GPPSID: []int8{2}},
			wantGDPR:    SignalYes,
		},
		{
			description: "Regs Ext is nil, GPPSID has tcf2",
			giveRegs:    &openrtb2.Regs{GPPSID: []int8{2}},
			wantGDPR:    SignalYes,
		},
		{
			description: "Regs Ext is nil, GPPSID has uspv1",
			giveRegs:    &openrtb2.Regs{GPPSID: []int8{6}},
			wantGDPR:    SignalNo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			req := openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: tt.giveRegs,
				},
			}
			result, err := GetGDPR(&req)
			assert.Equal(t, tt.wantGDPR, result)

			if tt.wantError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestEnforceGDPR(t *testing.T) {
	tests := []struct {
		name            string
		giveSignal      Signal
		giveDefault     Signal
		giveChannelFlag bool
		wantResult      bool
	}{
		{
			name:            "gdpr-applies-with-yes-signal-and-channel-enabled",
			giveSignal:      SignalYes,
			giveDefault:     SignalYes,
			giveChannelFlag: true,
			wantResult:      true,
		},
		{
			name:            "gdpr-does-not-apply-with-no-signal-and-channel-enabled",
			giveSignal:      SignalNo,
			giveDefault:     SignalYes,
			giveChannelFlag: true,
			wantResult:      false,
		},
		{
			name:            "gdpr-applies-with-ambiguous-signal-and-default-yes-with-channel-enabled",
			giveSignal:      SignalAmbiguous,
			giveDefault:     SignalYes,
			giveChannelFlag: true,
			wantResult:      true,
		},
		{
			name:            "gdpr-does-not-apply-with-ambiguous-signal-and-default-no-with-channel-enabled",
			giveSignal:      SignalAmbiguous,
			giveDefault:     SignalNo,
			giveChannelFlag: true,
			wantResult:      false,
		},
		{
			name:            "gdpr-does-not-apply-with-yes-signal-and-channel-disabled",
			giveSignal:      SignalYes,
			giveDefault:     SignalYes,
			giveChannelFlag: false,
			wantResult:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnforceGDPR(tt.giveSignal, tt.giveDefault, tt.giveChannelFlag)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func TestGetConsent(t *testing.T) {
	tests := []struct {
		description string
		giveUser    *openrtb2.User
		giveGPP     gpplib.GppContainer
		wantConsent string
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

			result := GetConsent(&req, tt.giveGPP)
			assert.Equal(t, tt.wantConsent, result, tt.description)
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
			result := SelectEEACountries(tt.hostEEACountries, tt.accountEEACountries)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsEEACountry(t *testing.T) {
	eeaCountries := []string{"FRA", "DEU", "ITA", "ESP", "NLD"}

	tests := []struct {
		name     string
		country  string
		eeaList  []string
		expected bool
	}{
		{
			name:     "Country_in_EEA",
			country:  "FRA",
			eeaList:  eeaCountries,
			expected: true,
		},
		{
			name:     "Country_in_EEA_lowercase",
			country:  "fra",
			eeaList:  eeaCountries,
			expected: true,
		},
		{
			name:     "Country_not_in_EEA",
			country:  "USA",
			eeaList:  eeaCountries,
			expected: false,
		},
		{
			name:     "Empty_country_string",
			country:  "",
			eeaList:  eeaCountries,
			expected: false,
		},
		{
			name:     "EEA_list_is_empty",
			country:  "FRA",
			eeaList:  []string{},
			expected: false,
		},
		{
			name:     "EEA_list_is_nil",
			country:  "FRA",
			eeaList:  nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEEACountry(tt.country, tt.eeaList)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseGDPRDefaultValue(t *testing.T) {
	tests := []struct {
		name        string
		giveRequest *openrtb_ext.RequestWrapper
		giveDefault string
		giveEEA     []string
		wantResult  Signal
	}{
		{
			name: "geo-nil-cfg-default-0",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Geo: nil,
					},
				},
			},
			giveDefault: "0",
			giveEEA:     []string{"DEU", "FRA"},
			wantResult:  SignalNo,
		},
		{
			name: "user-geo-present-eea-empty",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Geo: &openrtb2.Geo{Country: "DEU"},
					},
				},
			},
			giveDefault: "0",
			giveEEA:     []string{},
			wantResult:  SignalNo,
		},
		{
			name: "user-geo-present-geo-country-in-eea",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Geo: &openrtb2.Geo{Country: "DEU"},
					},
				},
			},
			giveDefault: "0",
			giveEEA:     []string{"DEU", "FRA"},
			wantResult:  SignalYes,
		},
		{
			name: "user-geo-present-geo-country-not-in-eea-but-properly-formatted",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Geo: &openrtb2.Geo{Country: "USA"},
					},
				},
			},
			giveDefault: "1",
			giveEEA:     []string{"DEU", "FRA"},
			wantResult:  SignalNo,
		},
		{
			name: "user-geo-present-geo-country-not-in-eea-but-improperly-formatted",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Geo: &openrtb2.Geo{Country: "US"},
					},
				},
			},
			giveDefault: "1",
			giveEEA:     []string{"DEU", "FRA"},
			wantResult:  SignalYes,
		},
		{
			name: "device-geo-present-country-not-in-eea",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{Country: "USA"},
					},
				},
			},
			giveDefault: "1",
			giveEEA:     []string{"DEU", "FRA"},
			wantResult:  SignalNo,
		},
		{
			name: "user-and-device-geo-present-user-geo-country-selected",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Geo: &openrtb2.Geo{Country: "DEU"},
					},
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{Country: "USA"},
					},
				},
			},
			giveDefault: "0",
			giveEEA:     []string{"DEU", "FRA"},
			wantResult:  SignalYes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseGDPRDefaultValue(tt.giveRequest, tt.giveDefault, tt.giveEEA)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}
