package criteo

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestGetDeviceType(t *testing.T) {

	// Setup:
	deviceTypeCases := []struct {
		deviceType string
		expected   string
	}{
		{"ios", "idfa"},
		{"Ios", "idfa"},
		{"IOS", "idfa"},
		{"android", "gaid"},
		{"unknown", "unknown"},
		{"", "unknown"},
		{"qwerty", "unknown"},
		{"qWerty", "unknown"},
		{"abc", "unknown"},
	}

	for _, uc := range deviceTypeCases {
		// Execute:
		result := getDeviceType(uc.deviceType)

		// Verify:
		if uc.expected != result {
			t.Errorf("Bad getDeviceType for '%s'. Expected: %s, got %s", uc.deviceType, uc.expected, result)
		}
	}
}

func TestNewCriteoRequest(t *testing.T) {
	// Setup:
	var (
		dummyRequestID         = "random request ID"
		dummyPublisherBundleID = "bundleid"
		dummyPublisherURL      = "test.com"
		dummyPublisherSiteID   = "siteid"
		dummyUserDeviceID      = "random-device-id"
		dummyUserDeviceOS      = "android"
		dummyUserDeviceIDType  = "gaid"
		dummyUserCookieID      = "random-cookie-id"
		dummyUserIP            = "1.1.1.1"
		dummyUserUA            = "random UA"
		dummyGdprApplies       = true
		dummyGdprAppliesUint   = int8(1)
		dummyGdprConsentData   = "randomconsentdata"
		dummySlotID            = "11111111-1111-1111-11111111"
		dummySlotImpID         = "fake-imp-id-1"
		dummySlotZoneID        = int64(1)
	)

	fakeSlotIDGenerator := newFakeGuidGenerator(dummySlotID)

	// The request doesn't make any sense but aims to fill every single criteo request fields
	expectedCriteoRequest := criteoRequest{
		ID: dummyRequestID,
		Publisher: criteoPublisher{
			SiteID:   dummyPublisherSiteID,
			BundleID: dummyPublisherBundleID,
			URL:      dummyPublisherURL,
		},
		User: criteoUser{
			DeviceID:     dummyUserDeviceID,
			DeviceOS:     dummyUserDeviceOS,
			DeviceIDType: dummyUserDeviceIDType,
			CookieID:     dummyUserCookieID,
			IP:           dummyUserIP,
			UA:           dummyUserUA,
		},
		GdprConsent: criteoGdprConsent{
			GdprApplies: &dummyGdprApplies,
			ConsentData: dummyGdprConsentData,
		},
		Slots: []criteoRequestSlot{
			{
				SlotID: dummySlotID,
				ImpID:  dummySlotImpID,
				ZoneID: &dummySlotZoneID,
			},
		},
	}

	userExtJSON, _ := json.Marshal(&openrtb_ext.ExtUser{
		Consent: dummyGdprConsentData,
	})
	regsExtJSON, _ := json.Marshal(&openrtb_ext.ExtRegs{
		GDPR: &dummyGdprAppliesUint,
	})
	bidderExtJSON, _ := json.Marshal(&openrtb_ext.ExtImpCriteo{
		ZoneID: dummySlotZoneID,
	})
	impExtJSON, _ := json.Marshal(&adapters.ExtImpBidder{
		Bidder: bidderExtJSON,
	})
	incomingRequest := &openrtb2.BidRequest{
		ID: dummyRequestID,
		App: &openrtb2.App{
			Bundle: dummyPublisherBundleID,
		},
		Site: &openrtb2.Site{
			ID:   dummyPublisherSiteID,
			Page: dummyPublisherURL,
		},
		User: &openrtb2.User{
			BuyerUID: dummyUserCookieID,
			Ext:      userExtJSON,
		},
		Regs: &openrtb2.Regs{
			Ext: regsExtJSON,
		},
		Device: &openrtb2.Device{
			IFA: dummyUserDeviceID,
			OS:  dummyUserDeviceOS,
			IP:  dummyUserIP,
			UA:  dummyUserUA,
		},
		Imp: []openrtb2.Imp{
			{
				ID:  dummySlotImpID,
				Ext: impExtJSON,
			},
		},
	}

	// Execute:
	result, err := newCriteoRequest(fakeSlotIDGenerator, incomingRequest)

	// Verify:
	if err != nil {
		t.Errorf("newCriteoRequest has errors: %s", err)
	}

	if expectedCriteoRequest.ID != result.ID ||
		!reflect.DeepEqual(expectedCriteoRequest, result) ||
		!reflect.DeepEqual(expectedCriteoRequest.Publisher, result.Publisher) ||
		!reflect.DeepEqual(expectedCriteoRequest.User, result.User) ||
		!reflect.DeepEqual(expectedCriteoRequest.GdprConsent, result.GdprConsent) ||
		len(expectedCriteoRequest.Slots) != len(result.Slots) ||
		!reflect.DeepEqual(expectedCriteoRequest.Slots[0], result.Slots[0]) {
		actualResultJSON, _ := json.Marshal(result)
		expectedResultJSON, _ := json.Marshal(expectedCriteoRequest)
		t.Errorf("newCriteoRequest was incorrect, got '%s', want '%s'.", actualResultJSON, expectedResultJSON)
	}
}

func TestGetGdprConsent(t *testing.T) {
	// Setup:
	var (
		dummyGdprApplies     = true
		dummyGdprConsentData = "randomconsentdata"
		dummyGdprAppliesUint = int8(1)
	)

	expectedCriteoRequest := criteoRequest{
		GdprConsent: criteoGdprConsent{
			GdprApplies: &dummyGdprApplies,
			ConsentData: dummyGdprConsentData,
		},
	}

	userExtJSON, _ := json.Marshal(&openrtb_ext.ExtUser{
		Consent: dummyGdprConsentData,
	})
	regsExtJSON, _ := json.Marshal(&openrtb_ext.ExtRegs{
		GDPR: &dummyGdprAppliesUint,
	})
	incomingRequest := &openrtb2.BidRequest{
		User: &openrtb2.User{
			Ext: userExtJSON,
		},
		Regs: &openrtb2.Regs{
			Ext: regsExtJSON,
		},
	}

	var regsExt *openrtb_ext.ExtRegs
	if incomingRequest.Regs != nil {
		json.Unmarshal(incomingRequest.Regs.Ext, &regsExt)
	}

	// Execute:
	gdprConsent, _ := newCriteoGdprConsent(incomingRequest.User, regsExt)
	result := criteoRequest{
		GdprConsent: gdprConsent,
	}

	// Verify:
	if !reflect.DeepEqual(expectedCriteoRequest, result) {
		actualResultJSON, _ := json.Marshal(result)
		expectedResultJSON, _ := json.Marshal(expectedCriteoRequest)
		t.Errorf("getGdprConsent was incorrect, got '%s', want '%s'.", actualResultJSON, expectedResultJSON)
	}
}

func TestGetUser(t *testing.T) {
	// Setup:
	var (
		dummyUserDeviceID     = "random-device-id"
		dummyUserDeviceOS     = "android"
		dummyUserDeviceIDType = "gaid"
		dummyUserCookieID     = "random-cookie-id"
		dummyUserIP           = "1.1.1.1"
		dummyUserUA           = "random UA"
		dummyCcpaString       = "1YYY"
	)
	expectedCriteoRequest := &criteoRequest{
		User: criteoUser{
			DeviceID:     dummyUserDeviceID,
			DeviceOS:     dummyUserDeviceOS,
			DeviceIDType: dummyUserDeviceIDType,
			CookieID:     dummyUserCookieID,
			IP:           dummyUserIP,
			UA:           dummyUserUA,
			UspIab:       dummyCcpaString,
		},
	}

	regsExt := &openrtb_ext.ExtRegs{
		USPrivacy: dummyCcpaString,
	}
	regsExtData, err := json.Marshal(regsExt)
	if err != nil {
		t.Errorf("cannot marshal regsExt data")
	}

	incomingRequest := &openrtb2.BidRequest{
		User: &openrtb2.User{
			BuyerUID: dummyUserCookieID,
		},
		Device: &openrtb2.Device{
			IFA: dummyUserDeviceID,
			OS:  dummyUserDeviceOS,
			IP:  dummyUserIP,
			UA:  dummyUserUA,
		},
		Regs: &openrtb2.Regs{
			Ext: regsExtData,
		},
	}

	// Execute:
	result := &criteoRequest{
		User: newCriteoUser(incomingRequest.User, incomingRequest.Device, regsExt),
	}

	// Verify:
	if !reflect.DeepEqual(expectedCriteoRequest.User, result.User) {
		actualResultJSON, _ := json.Marshal(result)
		expectedResultJSON, _ := json.Marshal(expectedCriteoRequest)
		t.Errorf("getUser was incorrect, got '%s', want '%s'.", actualResultJSON, expectedResultJSON)
	}
}

func TestGetUser_NilUserAndDevice(t *testing.T) {
	// Setup:
	var dummyCcpaString = "1YYY"

	expectedCriteoRequest := &criteoRequest{}

	regsExt := &openrtb_ext.ExtRegs{
		USPrivacy: dummyCcpaString,
	}
	regsExtData, err := json.Marshal(regsExt)
	if err != nil {
		t.Errorf("cannot marshal regsExt data")
	}

	incomingRequest := &openrtb2.BidRequest{
		Regs: &openrtb2.Regs{
			Ext: regsExtData,
		},
	}

	// Execute:
	result := &criteoRequest{
		User: newCriteoUser(incomingRequest.User, incomingRequest.Device, regsExt),
	}

	// Verify:
	if !reflect.DeepEqual(expectedCriteoRequest.User, result.User) {
		actualResultJSON, _ := json.Marshal(result)
		expectedResultJSON, _ := json.Marshal(expectedCriteoRequest)
		t.Errorf("getUser was incorrect, got '%s', want '%s'.", actualResultJSON, expectedResultJSON)
	}
}

func TestPublisher(t *testing.T) {
	// Setup:
	var (
		dummyPublisherSiteID         = "siteid"
		dummyPublisherBundleID       = "bundleid"
		dummyPublisherURL            = "test.com"
		dummyNetworkID         int64 = 1234567
	)
	expectedCriteoRequest := &criteoRequest{
		Publisher: criteoPublisher{
			SiteID:    dummyPublisherSiteID,
			BundleID:  dummyPublisherBundleID,
			URL:       dummyPublisherURL,
			NetworkID: &dummyNetworkID,
		},
	}

	incomingRequest := &openrtb2.BidRequest{
		App: &openrtb2.App{
			Bundle: dummyPublisherBundleID,
		},
		Site: &openrtb2.Site{
			ID:   dummyPublisherSiteID,
			Page: dummyPublisherURL,
		},
	}

	// Execute:
	result := &criteoRequest{
		Publisher: newCriteoPublisher(&dummyNetworkID, incomingRequest.App, incomingRequest.Site),
	}

	// Verify:
	if !reflect.DeepEqual(expectedCriteoRequest.Publisher, result.Publisher) {
		actualResultJSON, _ := json.Marshal(result)
		expectedResultJSON, _ := json.Marshal(expectedCriteoRequest)
		t.Errorf("getPublisher was incorrect, got '%s', want '%s'.", actualResultJSON, expectedResultJSON)
	}
}

func TestGetRequestSlots(t *testing.T) {
	// Setup:
	var (
		dummySlotImpID  = "fake-imp-id-1"
		dummySlotZoneID = int64(1)
		dummySlotID     = "22222222-2222-2222-22222222"
	)

	fakeSlotIDGenerator := newFakeGuidGenerator(dummySlotID)

	expectedCriteoRequest := &criteoRequest{
		Slots: []criteoRequestSlot{
			{
				SlotID: dummySlotID,
				ImpID:  dummySlotImpID,
				ZoneID: &dummySlotZoneID,
			},
		},
	}

	bidderExtJSON, _ := json.Marshal(&openrtb_ext.ExtImpCriteo{
		ZoneID: dummySlotZoneID,
	})
	impExtJSON, _ := json.Marshal(&adapters.ExtImpBidder{
		Bidder: bidderExtJSON,
	})
	incomingRequest := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				ID:  dummySlotImpID,
				Ext: impExtJSON,
			},
		},
	}

	// Execute:
	slots, err := newCriteoRequestSlots(fakeSlotIDGenerator, incomingRequest.Imp)
	result := &criteoRequest{
		Slots: slots,
	}

	// Verify:
	if err != nil {
		t.Errorf("newCriteoRequestSlots has errors: %s", err)
	}

	if len(expectedCriteoRequest.Slots) != len(result.Slots) ||
		!reflect.DeepEqual(expectedCriteoRequest.Slots[0], result.Slots[0]) {
		actualResultJSON, _ := json.Marshal(result)
		expectedResultJSON, _ := json.Marshal(expectedCriteoRequest)
		t.Errorf("newCriteoRequest was incorrect, got '%s', want '%s'.", actualResultJSON, expectedResultJSON)
	}
}

func TestGetRequestMultipleSlots(t *testing.T) {
	// Setup:
	dummySlots := []struct {
		ID     string
		ZoneID int64
	}{
		{"fake-imp-id-1", 1},
		{"fake-imp-id-2", 2},
		{"fake-imp-id-3", 3},
		{"fake-imp-id-4", 4},
		{"fake-imp-id-5", 5},
	}

	incomingRequest := &openrtb2.BidRequest{
		Imp: make([]openrtb2.Imp, len(dummySlots)),
	}
	slots := make([]criteoRequestSlot, len(dummySlots))

	for i := range dummySlots {
		// Build expected slots
		slots[i] = criteoRequestSlot{
			ImpID:  dummySlots[i].ID,
			ZoneID: &dummySlots[i].ZoneID,
		}

		// Build incoming request imps
		bidderExtJSON, _ := json.Marshal(&openrtb_ext.ExtImpCriteo{
			ZoneID: dummySlots[i].ZoneID,
		})
		impExtJSON, _ := json.Marshal(&adapters.ExtImpBidder{
			Bidder: bidderExtJSON,
		})
		incomingRequest.Imp[i] = openrtb2.Imp{
			ID:  dummySlots[i].ID,
			Ext: impExtJSON,
		}
	}

	expectedCriteoRequestSlots, err := newCriteoRequestSlots(newFakeGuidGenerator(""), incomingRequest.Imp)
	expectedCriteoRequest := &criteoRequest{
		Slots: expectedCriteoRequestSlots,
	}

	// Execute:
	slotsResult, err := newCriteoRequestSlots(newFakeGuidGenerator(""), incomingRequest.Imp)
	result := &criteoRequest{
		Slots: slotsResult,
	}

	// Verify:
	if err != nil {
		t.Errorf("newCriteoRequestSlots has errors: %s", err)
	}

	if len(expectedCriteoRequest.Slots) != len(result.Slots) ||
		!reflect.DeepEqual(expectedCriteoRequest.Slots, result.Slots) {
		actualResultJSON, _ := json.Marshal(result)
		expectedResultJSON, _ := json.Marshal(expectedCriteoRequest)
		t.Errorf("newCriteoRequest was incorrect, got '%s', want '%s'.", actualResultJSON, expectedResultJSON)
	}
}
