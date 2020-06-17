package privacy

import (
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAny(t *testing.T) {
	testCases := []struct {
		enforcement Enforcement
		expected    bool
		description string
	}{
		{
			description: "All False",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPR:    false,
				GDPRGeo: false,
				LMT:     false,
			},
			expected: false,
		},
		{
			description: "All True",
			enforcement: Enforcement{
				CCPA:    true,
				COPPA:   true,
				GDPR:    true,
				GDPRGeo: true,
				LMT:     true,
			},
			expected: true,
		},
		{
			description: "Mixed",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   true,
				GDPR:    false,
				GDPRGeo: false,
				LMT:     true,
			},
			expected: true,
		},
	}

	for _, test := range testCases {
		result := test.enforcement.Any()
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestApply(t *testing.T) {
	testCases := []struct {
		description        string
		enforcement        Enforcement
		ampGDPRException   bool
		expectedDeviceIPv6 ScrubStrategyIPV6
		expectedDeviceGeo  ScrubStrategyGeo
		expectedUser       ScrubStrategyUser
		expectedUserGeo    ScrubStrategyGeo
	}{
		{
			description: "All Enforced",
			enforcement: Enforcement{
				CCPA:    true,
				COPPA:   true,
				GDPR:    true,
				GDPRGeo: true,
				LMT:     true,
			},
			ampGDPRException:   false,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest32,
			expectedDeviceGeo:  ScrubStrategyGeoFull,
			expectedUser:       ScrubStrategyUserIDAndDemographic,
			expectedUserGeo:    ScrubStrategyGeoFull,
		},
		{
			description: "CCPA Only",
			enforcement: Enforcement{
				CCPA:    true,
				COPPA:   false,
				GDPR:    false,
				GDPRGeo: false,
				LMT:     false,
			},
			ampGDPRException:   false,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserID,
			expectedUserGeo:    ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "COPPA Only",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   true,
				GDPR:    false,
				GDPRGeo: false,
				LMT:     false,
			},
			ampGDPRException:   false,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest32,
			expectedDeviceGeo:  ScrubStrategyGeoFull,
			expectedUser:       ScrubStrategyUserIDAndDemographic,
			expectedUserGeo:    ScrubStrategyGeoFull,
		},
		{
			description: "GDPR Only",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPR:    true,
				GDPRGeo: true,
				LMT:     false,
			},
			ampGDPRException:   false,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserID,
			expectedUserGeo:    ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "GDPR Only, ampGDPRException",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPR:    true,
				GDPRGeo: true,
				LMT:     false,
			},
			ampGDPRException:   true,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserNone,
			expectedUserGeo:    ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "CCPA Only, ampGDPRException",
			enforcement: Enforcement{
				CCPA:    true,
				COPPA:   false,
				GDPR:    false,
				GDPRGeo: false,
				LMT:     false,
			},
			ampGDPRException:   true,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserID,
			expectedUserGeo:    ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "COPPA and GDPR, ampGDPRException",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   true,
				GDPR:    true,
				GDPRGeo: true,
				LMT:     false,
			},
			ampGDPRException:   true,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest32,
			expectedDeviceGeo:  ScrubStrategyGeoFull,
			expectedUser:       ScrubStrategyUserIDAndDemographic,
			expectedUserGeo:    ScrubStrategyGeoFull,
		},
		{
			description: "GDPR Only, no Geo",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPR:    true,
				GDPRGeo: false,
				LMT:     false,
			},
			ampGDPRException:   false,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoNone,
			expectedUser:       ScrubStrategyUserID,
			expectedUserGeo:    ScrubStrategyGeoNone,
		},
		{
			description: "GDPR Only, Geo only",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPR:    false,
				GDPRGeo: true,
				LMT:     false,
			},
			ampGDPRException:   false,
			expectedDeviceIPv6: ScrubStrategyIPV6None,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserNone,
			expectedUserGeo:    ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "LMT Only",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPR:    false,
				GDPRGeo: false,
				LMT:     true,
			},
			ampGDPRException:   false,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserID,
			expectedUserGeo:    ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "LMT Only, ampGDPRException",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPR:    false,
				GDPRGeo: false,
				LMT:     true,
			},
			ampGDPRException:   true,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserID,
			expectedUserGeo:    ScrubStrategyGeoReducedPrecision,
		},
	}

	for _, test := range testCases {
		req := &openrtb.BidRequest{
			Device: &openrtb.Device{},
			User:   &openrtb.User{},
		}
		replacedDevice := &openrtb.Device{}
		replacedUser := &openrtb.User{}

		m := &mockScrubber{}
		m.On("ScrubDevice", req.Device, test.expectedDeviceIPv6, test.expectedDeviceGeo).Return(replacedDevice).Once()
		m.On("ScrubUser", req.User, test.expectedUser, test.expectedUserGeo).Return(replacedUser).Once()

		test.enforcement.apply(req, test.ampGDPRException, m)

		m.AssertExpectations(t)
		assert.Same(t, replacedDevice, req.Device, "Device")
		assert.Same(t, replacedUser, req.User, "User")
	}
}

func TestApplyNoneApplicable(t *testing.T) {
	req := &openrtb.BidRequest{}

	m := &mockScrubber{}

	enforcement := Enforcement{
		CCPA:  false,
		COPPA: false,
		GDPR:  false,
		LMT:   false,
	}
	enforcement.apply(req, false, m)

	m.AssertNotCalled(t, "ScrubDevice")
	m.AssertNotCalled(t, "ScrubUser")
}

func TestApplyNil(t *testing.T) {
	m := &mockScrubber{}

	enforcement := Enforcement{}
	enforcement.apply(nil, false, m)

	m.AssertNotCalled(t, "ScrubDevice")
	m.AssertNotCalled(t, "ScrubUser")
}

type mockScrubber struct {
	mock.Mock
}

func (m *mockScrubber) ScrubDevice(device *openrtb.Device, ipv6 ScrubStrategyIPV6, geo ScrubStrategyGeo) *openrtb.Device {
	args := m.Called(device, ipv6, geo)
	return args.Get(0).(*openrtb.Device)
}

func (m *mockScrubber) ScrubUser(user *openrtb.User, strategy ScrubStrategyUser, geo ScrubStrategyGeo) *openrtb.User {
	args := m.Called(user, strategy, geo)
	return args.Get(0).(*openrtb.User)
}
