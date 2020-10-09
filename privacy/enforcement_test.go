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
		ampGDPRException   bool
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
			ampGDPRException:   false,
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4Lowest8,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserID,
			expectedUserGeo:    ScrubStrategyGeoReducedPrecision,
		},
		{
			description: "GDPR Only - Full - AMP Exception",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPRGeo: true,
				GDPRID:  true,
				LMT:     false,
			},
			ampGDPRException:   true,
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4Lowest8,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserNone,
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
			ampGDPRException:   false,
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4None,
			expectedDeviceIPv6: ScrubStrategyIPV6None,
			expectedDeviceGeo:  ScrubStrategyGeoNone,
			expectedUser:       ScrubStrategyUserID,
			expectedUserGeo:    ScrubStrategyGeoNone,
		},
		{
			description: "GDPR Only - ID Only - AMP Exception",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   false,
				GDPRGeo: false,
				GDPRID:  true,
				LMT:     false,
			},
			ampGDPRException:   true,
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4None,
			expectedDeviceIPv6: ScrubStrategyIPV6None,
			expectedDeviceGeo:  ScrubStrategyGeoNone,
			expectedUser:       ScrubStrategyUserNone,
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
			ampGDPRException:   false,
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
			description: "Interactions: COPPA Only + AMP Exception",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   true,
				GDPRGeo: false,
				GDPRID:  false,
				LMT:     false,
			},
			ampGDPRException:   true,
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4Lowest8,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest32,
			expectedDeviceGeo:  ScrubStrategyGeoFull,
			expectedUser:       ScrubStrategyUserIDAndDemographic,
			expectedUserGeo:    ScrubStrategyGeoFull,
		},
		{
			description: "Interactions: COPPA + GDPR Full + AMP Exception",
			enforcement: Enforcement{
				CCPA:    false,
				COPPA:   true,
				GDPRGeo: true,
				GDPRID:  true,
				LMT:     false,
			},
			ampGDPRException:   true,
			expectedDeviceID:   ScrubStrategyDeviceIDAll,
			expectedDeviceIPv4: ScrubStrategyIPV4Lowest8,
			expectedDeviceIPv6: ScrubStrategyIPV6Lowest32,
			expectedDeviceGeo:  ScrubStrategyGeoFull,
			expectedUser:       ScrubStrategyUserIDAndDemographic,
			expectedUserGeo:    ScrubStrategyGeoFull,
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
		m.On("ScrubDevice", req.Device, test.expectedDeviceID, test.expectedDeviceIPv4, test.expectedDeviceIPv6, test.expectedDeviceGeo).Return(replacedDevice).Once()
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
		CCPA:    false,
		COPPA:   false,
		GDPRGeo: false,
		GDPRID:  false,
		LMT:     false,
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

func (m *mockScrubber) ScrubDevice(device *openrtb.Device, id ScrubStrategyDeviceID, ipv4 ScrubStrategyIPV4, ipv6 ScrubStrategyIPV6, geo ScrubStrategyGeo) *openrtb.Device {
	args := m.Called(device, id, ipv4, ipv6, geo)
	return args.Get(0).(*openrtb.Device)
}

func (m *mockScrubber) ScrubUser(user *openrtb.User, strategy ScrubStrategyUser, geo ScrubStrategyGeo) *openrtb.User {
	args := m.Called(user, strategy, geo)
	return args.Get(0).(*openrtb.User)
}
