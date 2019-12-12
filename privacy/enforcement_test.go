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
			enforcement: Enforcement{
				CCPA:  false,
				COPPA: false,
				GDPR:  false,
			},
			expected:    false,
			description: "All False",
		},
		{
			enforcement: Enforcement{
				CCPA:  true,
				COPPA: true,
				GDPR:  true,
			},
			expected:    true,
			description: "All True",
		},
		{
			enforcement: Enforcement{
				CCPA:  false,
				COPPA: true,
				GDPR:  false,
			},
			expected:    true,
			description: "Mixed",
		},
	}

	for _, test := range testCases {
		result := test.enforcement.Any()
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestApply(t *testing.T) {
	testCases := []struct {
		enforcement             Enforcement
		isAMP                   bool
		expectedDeviceMacAndIFA bool
		expectedDeviceIPv6      ScrubStrategyIPV6
		expectedDeviceGeo       ScrubStrategyGeo
		expectedUser            ScrubStrategyUser
		expectedUserGeo         ScrubStrategyGeo
		description             string
	}{
		{
			enforcement: Enforcement{
				CCPA:  true,
				COPPA: true,
				GDPR:  true,
			},
			isAMP:                   true,
			expectedDeviceMacAndIFA: true,
			expectedDeviceIPv6:      ScrubStrategyIPV6Lowest32,
			expectedDeviceGeo:       ScrubStrategyGeoFull,
			expectedUser:            ScrubStrategyUserFull,
			expectedUserGeo:         ScrubStrategyGeoFull,
			description:             "All Enforced - Most Strict",
		},
		{
			enforcement: Enforcement{
				CCPA:  false,
				COPPA: true,
				GDPR:  false,
			},
			isAMP:                   false,
			expectedDeviceMacAndIFA: true,
			expectedDeviceIPv6:      ScrubStrategyIPV6Lowest32,
			expectedDeviceGeo:       ScrubStrategyGeoFull,
			expectedUser:            ScrubStrategyUserFull,
			expectedUserGeo:         ScrubStrategyGeoFull,
			description:             "COPPA",
		},
		{
			enforcement: Enforcement{
				CCPA:  false,
				COPPA: false,
				GDPR:  true,
			},
			isAMP:                   false,
			expectedDeviceMacAndIFA: false,
			expectedDeviceIPv6:      ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:       ScrubStrategyGeoReducedPrecision,
			expectedUser:            ScrubStrategyUserBuyerIDOnly,
			expectedUserGeo:         ScrubStrategyGeoReducedPrecision,
			description:             "GDPR",
		},
		{
			enforcement: Enforcement{
				CCPA:  false,
				COPPA: false,
				GDPR:  true,
			},
			isAMP:                   true,
			expectedDeviceMacAndIFA: false,
			expectedDeviceIPv6:      ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:       ScrubStrategyGeoReducedPrecision,
			expectedUser:            ScrubStrategyUserNone,
			expectedUserGeo:         ScrubStrategyGeoReducedPrecision,
			description:             "GDPR For AMP",
		},
		{
			enforcement: Enforcement{
				CCPA:  true,
				COPPA: false,
				GDPR:  false,
			},
			isAMP:                   false,
			expectedDeviceMacAndIFA: false,
			expectedDeviceIPv6:      ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:       ScrubStrategyGeoReducedPrecision,
			expectedUser:            ScrubStrategyUserBuyerIDOnly,
			expectedUserGeo:         ScrubStrategyGeoReducedPrecision,
			description:             "CCPA",
		},
		{
			enforcement: Enforcement{
				CCPA:  true,
				COPPA: false,
				GDPR:  false,
			},
			isAMP:                   true,
			expectedDeviceMacAndIFA: false,
			expectedDeviceIPv6:      ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:       ScrubStrategyGeoReducedPrecision,
			expectedUser:            ScrubStrategyUserBuyerIDOnly,
			expectedUserGeo:         ScrubStrategyGeoReducedPrecision,
			description:             "CCPA For AMP",
		},
		{
			enforcement: Enforcement{
				CCPA:  true,
				COPPA: false,
				GDPR:  true,
			},
			isAMP:                   true,
			expectedDeviceMacAndIFA: false,
			expectedDeviceIPv6:      ScrubStrategyIPV6Lowest16,
			expectedDeviceGeo:       ScrubStrategyGeoReducedPrecision,
			expectedUser:            ScrubStrategyUserNone,
			expectedUserGeo:         ScrubStrategyGeoReducedPrecision,
			description:             "GDPR And CCPA For AMP",
		},
	}

	for _, test := range testCases {
		req := &openrtb.BidRequest{
			Device: &openrtb.Device{DIDSHA1: "before"},
			User:   &openrtb.User{ID: "before"},
		}
		device := &openrtb.Device{DIDSHA1: "after"}
		user := &openrtb.User{ID: "after"}

		m := &mockScrubber{}
		m.On("ScrubDevice", req.Device, test.expectedDeviceMacAndIFA, test.expectedDeviceIPv6, test.expectedDeviceGeo).Return(device).Once()
		m.On("ScrubUser", req.User, test.expectedUser, test.expectedUserGeo).Return(user).Once()

		test.enforcement.apply(req, test.isAMP, m)

		m.AssertExpectations(t)
		assert.Equal(t, device, req.Device, "Device Set Correctly")
		assert.Equal(t, user, req.User, "User Set Correctly")
	}
}

func TestApplyNoneApplicable(t *testing.T) {
	enforcement := Enforcement{}
	device := &openrtb.Device{DIDSHA1: "original"}
	user := &openrtb.User{ID: "original"}
	req := &openrtb.BidRequest{
		Device: device,
		User:   user,
	}

	m := &mockScrubber{}

	enforcement.apply(req, true, m)

	m.AssertNotCalled(t, "ScrubDevice")
	m.AssertNotCalled(t, "ScrubUser")
	assert.Equal(t, device, req.Device, "Device Set Correctly")
	assert.Equal(t, user, req.User, "User Set Correctly")
}

type mockScrubber struct {
	mock.Mock
}

func (m *mockScrubber) ScrubDevice(device *openrtb.Device, macAndIFA bool, ipv6 ScrubStrategyIPV6, geo ScrubStrategyGeo) *openrtb.Device {
	args := m.Called(device, macAndIFA, ipv6, geo)
	return args.Get(0).(*openrtb.Device)
}

func (m *mockScrubber) ScrubUser(user *openrtb.User, strategy ScrubStrategyUser, geo ScrubStrategyGeo) *openrtb.User {
	args := m.Called(user, strategy, geo)
	return args.Get(0).(*openrtb.User)
}
