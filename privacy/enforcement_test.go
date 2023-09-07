package privacy

import (
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAnyLegacy(t *testing.T) {
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
		result := test.enforcement.AnyLegacy()
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestApplyGDPR(t *testing.T) {
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
			expectedDeviceIPv4: ScrubStrategyIPV4Subnet,
			expectedDeviceIPv6: ScrubStrategyIPV6Subnet,
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
			expectedDeviceIPv4: ScrubStrategyIPV4Subnet,
			expectedDeviceIPv6: ScrubStrategyIPV6Subnet,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserIDAndDemographic,
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
			expectedDeviceIPv4: ScrubStrategyIPV4Subnet,
			expectedDeviceIPv6: ScrubStrategyIPV6Subnet,
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
			expectedDeviceIPv4: ScrubStrategyIPV4Subnet,
			expectedDeviceIPv6: ScrubStrategyIPV6Subnet,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserIDAndDemographic,
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
			expectedUser:       ScrubStrategyUserIDAndDemographic,
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
			expectedDeviceIPv4: ScrubStrategyIPV4Subnet,
			expectedDeviceIPv6: ScrubStrategyIPV6Subnet,
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
			expectedDeviceIPv4: ScrubStrategyIPV4Subnet,
			expectedDeviceIPv6: ScrubStrategyIPV6Subnet,
			expectedDeviceGeo:  ScrubStrategyGeoReducedPrecision,
			expectedUser:       ScrubStrategyUserIDAndDemographic,
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
			expectedDeviceIPv4: ScrubStrategyIPV4Subnet,
			expectedDeviceIPv6: ScrubStrategyIPV6Subnet,
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

func TestApplyToggle(t *testing.T) {
	testCases := []struct {
		description                  string
		enforcement                  Enforcement
		expectedScrubRequestExecuted bool
		expectedScrubUserExecuted    bool
		expectedScrubDeviceExecuted  bool
	}{
		{
			description: "All enforced - only ScrubRequest execution expected",
			enforcement: Enforcement{
				CCPA:       true,
				COPPA:      true,
				GDPRGeo:    true,
				GDPRID:     true,
				LMT:        true,
				UFPD:       true,
				Eids:       true,
				PreciseGeo: true,
				TID:        true,
			},
			expectedScrubRequestExecuted: true,
			expectedScrubUserExecuted:    false,
			expectedScrubDeviceExecuted:  false,
		},
		{
			description: "All Legacy and no activities - ScrubUser and ScrubDevice execution expected",
			enforcement: Enforcement{
				CCPA:       true,
				COPPA:      true,
				GDPRGeo:    true,
				GDPRID:     true,
				LMT:        true,
				UFPD:       false,
				Eids:       false,
				PreciseGeo: false,
				TID:        false,
			},
			expectedScrubRequestExecuted: false,
			expectedScrubUserExecuted:    true,
			expectedScrubDeviceExecuted:  true,
		},
		{
			description: "Some Legacy and some activities - ScrubRequest, ScrubUser and ScrubDevice execution expected",
			enforcement: Enforcement{
				CCPA:       true,
				COPPA:      true,
				GDPRGeo:    true,
				GDPRID:     true,
				LMT:        true,
				UFPD:       true,
				Eids:       false,
				PreciseGeo: false,
				TID:        false,
			},
			expectedScrubRequestExecuted: true,
			expectedScrubUserExecuted:    true,
			expectedScrubDeviceExecuted:  true,
		},
		{
			description: "Some Legacy and some activities - ScrubRequest execution expected",
			enforcement: Enforcement{
				CCPA:       true,
				COPPA:      true,
				GDPRGeo:    true,
				GDPRID:     true,
				LMT:        true,
				UFPD:       true,
				Eids:       true,
				PreciseGeo: true,
				TID:        false,
			},
			expectedScrubRequestExecuted: true,
			expectedScrubUserExecuted:    false,
			expectedScrubDeviceExecuted:  false,
		},
		{
			description: "Some Legacy and some activities overlap - ScrubRequest and ScrubUser execution expected",
			enforcement: Enforcement{
				CCPA:       true,
				COPPA:      true,
				GDPRGeo:    true,
				GDPRID:     true,
				LMT:        true,
				UFPD:       true,
				Eids:       false,
				PreciseGeo: true,
				TID:        false,
			},
			expectedScrubRequestExecuted: true,
			expectedScrubUserExecuted:    true,
			expectedScrubDeviceExecuted:  false,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			req := &openrtb2.BidRequest{
				Device: &openrtb2.Device{},
				User:   &openrtb2.User{},
			}
			replacedDevice := &openrtb2.Device{}
			replacedUser := &openrtb2.User{}

			m := &mockScrubber{}

			if test.expectedScrubRequestExecuted {
				m.On("ScrubRequest", req, test.enforcement).Return(req).Once()
			}
			if test.expectedScrubUserExecuted {
				m.On("ScrubUser", req.User, ScrubStrategyUserIDAndDemographic, ScrubStrategyGeoFull).Return(replacedUser).Once()
			}
			if test.expectedScrubDeviceExecuted {
				m.On("ScrubDevice", req.Device, ScrubStrategyDeviceIDAll, ScrubStrategyIPV4Subnet, ScrubStrategyIPV6Subnet, ScrubStrategyGeoFull).Return(replacedDevice).Once()
			}

			test.enforcement.apply(req, m)

			m.AssertExpectations(t)

		})
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

		UFPD:       false,
		PreciseGeo: false,
		TID:        false,
		Eids:       false,
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

func (m *mockScrubber) ScrubRequest(bidRequest *openrtb2.BidRequest, enforcement Enforcement) *openrtb2.BidRequest {
	args := m.Called(bidRequest, enforcement)
	return args.Get(0).(*openrtb2.BidRequest)
}

func (m *mockScrubber) ScrubDevice(device *openrtb2.Device, id ScrubStrategyDeviceID, ipv4 ScrubStrategyIPV4, ipv6 ScrubStrategyIPV6, geo ScrubStrategyGeo) *openrtb2.Device {
	args := m.Called(device, id, ipv4, ipv6, geo)
	return args.Get(0).(*openrtb2.Device)
}

func (m *mockScrubber) ScrubUser(user *openrtb2.User, strategy ScrubStrategyUser, geo ScrubStrategyGeo) *openrtb2.User {
	args := m.Called(user, strategy, geo)
	return args.Get(0).(*openrtb2.User)
}
