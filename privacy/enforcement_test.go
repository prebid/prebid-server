package privacy

import (
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
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
				GDPRGeo: false,
				GDPRID:  false,
				LMT:     false,
			},
			expected: false,
		},
		{
			description: "All True",
			enforcement: Enforcement{
				CCPA:    true,
				COPPA:   true,
				GDPRGeo: true,
				GDPRID:  true,
				LMT:     true,
			},
			expected: true,
		},
		{
			description: "Mixed",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   true,
				GDPRGeo: false,
				GDPRID:  false,
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
		expectedDeviceID   ScrubStrategyDeviceID
		expectedDeviceIPv4 ScrubStrategyIPV4
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
				GDPRGeo: true,
				GDPRID:  true,
				LMT:     true,
			},
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4Lowest8,
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
				GDPRGeo: false,
				GDPRID:  false,
				LMT:     false,
			},
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4Lowest8,
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
				GDPRGeo: false,
				GDPRID:  false,
				LMT:     false,
			},
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4Lowest8,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest32,
			expectedDeviceGeo:  ScrubStrategyGeoFull,
			expectedUser:       ScrubStrategyUserIDAndDemographic,
			expectedUserGeo:    ScrubStrategyGeoFull,
		},
		{
			description: "GDPR Only - Full",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPRGeo: true,
				GDPRID:  true,
				LMT:     false,
			},
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4Lowest8,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserID,
			expectedUserGeo:    ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "GDPR Only - ID Only",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPRGeo: false,
				GDPRID:  true,
				LMT:     false,
			},
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4None,
			expectedDeviceIPv6: ScrubStrategyIPV6None,
			expectedDeviceGeo:  ScrubStrategyGeoNone,
			expectedUser:       ScrubStrategyUserID,
			expectedUserGeo:    ScrubStrategyGeoNone,
		},
		{
			description: "GDPR Only - Geo Only",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPRGeo: true,
				GDPRID:  false,
				LMT:     false,
			},
			expectedDeviceID:   ScrubStrategyDeviceIDNone,
			expectedDeviceIPv4: ScrubStrategyIPV4Lowest8,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserNone,
			expectedUserGeo:    ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "LMT Only",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPRGeo: false,
				GDPRID:  false,
				LMT:     true,
			},
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4Lowest8,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserID,
			expectedUserGeo:    ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "Interactions: COPPA + GDPR Full",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   true,
				GDPRGeo: true,
				GDPRID:  true,
				LMT:     false,
			},
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4Lowest8,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest32,
			expectedDeviceGeo:  ScrubStrategyGeoFull,
			expectedUser:       ScrubStrategyUserIDAndDemographic,
			expectedUserGeo:    ScrubStrategyGeoFull,
		},
	}

	for _, test := range testCases {
		req := &openrtb2.BidRequest{
			Device: &openrtb2.Device{},
			User:   &openrtb2.User{},
		}
		replacedDevice := &openrtb2.Device{}
		replacedUser := &openrtb2.User{}

		m := &mockScrubber{}
		m.On("ScrubDevice", req.Device, test.expectedDeviceID, test.expectedDeviceIPv4, test.expectedDeviceIPv6, test.expectedDeviceGeo).Return(replacedDevice).Once()
		m.On("ScrubUser", req.User, test.expectedUser, test.expectedUserGeo).Return(replacedUser).Once()

		test.enforcement.apply(req, m)

		m.AssertExpectations(t)
		assert.Same(t, replacedDevice, req.Device, "Device")
		assert.Same(t, replacedUser, req.User, "User")
	}
}

func TestApplyNoneApplicable(t *testing.T) {
	req := &openrtb2.BidRequest{}

	m := &mockScrubber{}

	enforcement := Enforcement{
		CCPA:    false,
		COPPA:   false,
		GDPRGeo: false,
		GDPRID:  false,
		LMT:     false,
	}
	enforcement.apply(req, m)

	m.AssertNotCalled(t, "ScrubDevice")
	m.AssertNotCalled(t, "ScrubUser")
}

func TestApplyNil(t *testing.T) {
	m := &mockScrubber{}

	enforcement := Enforcement{}
	enforcement.apply(nil, m)

	m.AssertNotCalled(t, "ScrubDevice")
	m.AssertNotCalled(t, "ScrubUser")
}

type mockScrubber struct {
	mock.Mock
}

func (m *mockScrubber) ScrubDevice(device *openrtb2.Device, id ScrubStrategyDeviceID, ipv4 ScrubStrategyIPV4, ipv6 ScrubStrategyIPV6, geo ScrubStrategyGeo) *openrtb2.Device {
	args := m.Called(device, id, ipv4, ipv6, geo)
	return args.Get(0).(*openrtb2.Device)
}

func (m *mockScrubber) ScrubUser(user *openrtb2.User, strategy ScrubStrategyUser, geo ScrubStrategyGeo) *openrtb2.User {
	args := m.Called(user, strategy, geo)
	return args.Get(0).(*openrtb2.User)
}
