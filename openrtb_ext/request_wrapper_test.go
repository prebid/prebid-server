package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/stretchr/testify/assert"
)

// Some minimal tests to get code coverage above 30%. The real tests are when other modules use these structures.

func TestUserExt(t *testing.T) {
	userExt := &UserExt{}

	userExt.unmarshal(nil)
	assert.Equal(t, false, userExt.Dirty(), "New UserExt should not be dirty.")
	assert.Nil(t, userExt.GetConsent(), "Empty UserExt should have nil consent")
	assert.Nil(t, userExt.GetEid(), "Empty UserExt should have nil eid")
	assert.Nil(t, userExt.GetPrebid(), "Empty UserExt should have nil prebid")

	newConsent := "NewConsent"
	userExt.SetConsent(&newConsent)
	assert.Equal(t, "NewConsent", *userExt.GetConsent(), "UserExt consent is incorrect")

	eid := openrtb2.EID{Source: "source", UIDs: []openrtb2.UID{{ID: "id"}}}
	newEid := []openrtb2.EID{eid}
	userExt.SetEid(&newEid)
	assert.Equal(t, []openrtb2.EID{eid}, *userExt.GetEid(), "UserExt eid is incorrect")

	buyerIDs := map[string]string{"buyer": "id"}
	newPrebid := ExtUserPrebid{BuyerUIDs: buyerIDs}
	userExt.SetPrebid(&newPrebid)
	assert.Equal(t, ExtUserPrebid{BuyerUIDs: buyerIDs}, *userExt.GetPrebid(), "UserExt prebid is incorrect")

	assert.Equal(t, true, userExt.Dirty(), "UserExt should be dirty after field updates")

	updatedUserExt, err := userExt.marshal()
	assert.Nil(t, err, "Marshalling UserExt after updating should not cause an error")

	expectedUserExt := json.RawMessage(`{"consent":"NewConsent","prebid":{"buyeruids":{"buyer":"id"}},"eids":[{"source":"source","uids":[{"id":"id"}]}]}`)
	assert.JSONEq(t, string(expectedUserExt), string(updatedUserExt), "Marshalled UserExt is incorrect")

	assert.Equal(t, false, userExt.Dirty(), "UserExt should not be dirty after marshalling")
}

func TestRebuildUserExt(t *testing.T) {
	strA := "a"
	strB := "b"

	testCases := []struct {
		description           string
		request               openrtb2.BidRequest
		requestUserExtWrapper UserExt
		expectedRequest       openrtb2.BidRequest
	}{
		{
			description:           "Nil - Not Dirty",
			request:               openrtb2.BidRequest{},
			requestUserExtWrapper: UserExt{},
			expectedRequest:       openrtb2.BidRequest{},
		},
		{
			description:           "Nil - Dirty",
			request:               openrtb2.BidRequest{},
			requestUserExtWrapper: UserExt{consent: &strA, consentDirty: true},
			expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
		},
		{
			description:           "Nil - Dirty - No Change",
			request:               openrtb2.BidRequest{},
			requestUserExtWrapper: UserExt{consent: nil, consentDirty: true},
			expectedRequest:       openrtb2.BidRequest{},
		},
		{
			description:           "Empty - Not Dirty",
			request:               openrtb2.BidRequest{User: &openrtb2.User{}},
			requestUserExtWrapper: UserExt{},
			expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
		},
		{
			description:           "Empty - Dirty",
			request:               openrtb2.BidRequest{User: &openrtb2.User{}},
			requestUserExtWrapper: UserExt{consent: &strB, consentDirty: true},
			expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"b"}`)}},
		},
		{
			description:           "Empty - Dirty - No Change",
			request:               openrtb2.BidRequest{User: &openrtb2.User{}},
			requestUserExtWrapper: UserExt{consent: nil, consentDirty: true},
			expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
		},
		{
			description:           "Populated - Not Dirty",
			request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
			requestUserExtWrapper: UserExt{},
			expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
		},
		{
			description:           "Populated - Dirty",
			request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
			requestUserExtWrapper: UserExt{consent: &strB, consentDirty: true},
			expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"b"}`)}},
		},
		{
			description:           "Populated - Dirty - No Change",
			request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
			requestUserExtWrapper: UserExt{consent: &strA, consentDirty: true},
			expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
		},
		{
			description:           "Populated - Dirty - Cleared",
			request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
			requestUserExtWrapper: UserExt{consent: nil, consentDirty: true},
			expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
		},
	}

	for _, test := range testCases {
		// create required filed in the test loop to keep test declaration easier to read
		test.requestUserExtWrapper.ext = make(map[string]json.RawMessage)

		w := RequestWrapper{BidRequest: &test.request, userExt: &test.requestUserExtWrapper}
		w.RebuildRequest()
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}

func TestRebuildDeviceExt(t *testing.T) {
	prebidContent1 := ExtDevicePrebid{Interstitial: &ExtDeviceInt{MinWidthPerc: 1}}
	prebidContent2 := ExtDevicePrebid{Interstitial: &ExtDeviceInt{MinWidthPerc: 2}}

	testCases := []struct {
		description             string
		request                 openrtb2.BidRequest
		requestDeviceExtWrapper DeviceExt
		expectedRequest         openrtb2.BidRequest
	}{
		{
			description:             "Nil - Not Dirty",
			request:                 openrtb2.BidRequest{},
			requestDeviceExtWrapper: DeviceExt{},
			expectedRequest:         openrtb2.BidRequest{},
		},
		{
			description:             "Nil - Dirty",
			request:                 openrtb2.BidRequest{},
			requestDeviceExtWrapper: DeviceExt{prebid: &prebidContent1, prebidDirty: true},
			expectedRequest:         openrtb2.BidRequest{Device: &openrtb2.Device{Ext: json.RawMessage(`{"prebid":{"interstitial":{"minwidthperc":1,"minheightperc":0}}}`)}},
		},
		{
			description:             "Nil - Dirty - No Change",
			request:                 openrtb2.BidRequest{},
			requestDeviceExtWrapper: DeviceExt{prebid: nil, prebidDirty: true},
			expectedRequest:         openrtb2.BidRequest{},
		},
		{
			description:             "Empty - Not Dirty",
			request:                 openrtb2.BidRequest{Device: &openrtb2.Device{}},
			requestDeviceExtWrapper: DeviceExt{},
			expectedRequest:         openrtb2.BidRequest{Device: &openrtb2.Device{}},
		},
		{
			description:             "Empty - Dirty",
			request:                 openrtb2.BidRequest{Device: &openrtb2.Device{}},
			requestDeviceExtWrapper: DeviceExt{prebid: &prebidContent1, prebidDirty: true},
			expectedRequest:         openrtb2.BidRequest{Device: &openrtb2.Device{Ext: json.RawMessage(`{"prebid":{"interstitial":{"minwidthperc":1,"minheightperc":0}}}`)}},
		},
		{
			description:             "Empty - Dirty - No Change",
			request:                 openrtb2.BidRequest{Device: &openrtb2.Device{}},
			requestDeviceExtWrapper: DeviceExt{prebid: nil, prebidDirty: true},
			expectedRequest:         openrtb2.BidRequest{Device: &openrtb2.Device{}},
		},
		{
			description:             "Populated - Not Dirty",
			request:                 openrtb2.BidRequest{Device: &openrtb2.Device{Ext: json.RawMessage(`{"prebid":{"interstitial":{"minwidthperc":1,"minheightperc":0}}}`)}},
			requestDeviceExtWrapper: DeviceExt{},
			expectedRequest:         openrtb2.BidRequest{Device: &openrtb2.Device{Ext: json.RawMessage(`{"prebid":{"interstitial":{"minwidthperc":1,"minheightperc":0}}}`)}},
		},
		{
			description:             "Populated - Dirty",
			request:                 openrtb2.BidRequest{Device: &openrtb2.Device{Ext: json.RawMessage(`{"prebid":{"interstitial":{"minwidthperc":1,"minheightperc":0}}}`)}},
			requestDeviceExtWrapper: DeviceExt{prebid: &prebidContent2, prebidDirty: true},
			expectedRequest:         openrtb2.BidRequest{Device: &openrtb2.Device{Ext: json.RawMessage(`{"prebid":{"interstitial":{"minwidthperc":2,"minheightperc":0}}}`)}},
		},
		{
			description:             "Populated - Dirty - No Change",
			request:                 openrtb2.BidRequest{Device: &openrtb2.Device{Ext: json.RawMessage(`{"prebid":{"interstitial":{"minwidthperc":1,"minheightperc":0}}}`)}},
			requestDeviceExtWrapper: DeviceExt{prebid: &prebidContent1, prebidDirty: true},
			expectedRequest:         openrtb2.BidRequest{Device: &openrtb2.Device{Ext: json.RawMessage(`{"prebid":{"interstitial":{"minwidthperc":1,"minheightperc":0}}}`)}},
		},
		{
			description:             "Populated - Dirty - Cleared",
			request:                 openrtb2.BidRequest{Device: &openrtb2.Device{Ext: json.RawMessage(`{"prebid":{"interstitial":{"minwidthperc":1,"minheightperc":0}}}`)}},
			requestDeviceExtWrapper: DeviceExt{prebid: nil, prebidDirty: true},
			expectedRequest:         openrtb2.BidRequest{Device: &openrtb2.Device{}},
		},
	}

	for _, test := range testCases {
		// create required filed in the test loop to keep test declaration easier to read
		test.requestDeviceExtWrapper.ext = make(map[string]json.RawMessage)

		w := RequestWrapper{BidRequest: &test.request, deviceExt: &test.requestDeviceExtWrapper}
		w.RebuildRequest()
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}

func TestRebuildRequestExt(t *testing.T) {
	prebidContent1 := ExtRequestPrebid{Integration: "1"}
	prebidContent2 := ExtRequestPrebid{Integration: "2"}

	testCases := []struct {
		description              string
		request                  openrtb2.BidRequest
		requestRequestExtWrapper RequestExt
		expectedRequest          openrtb2.BidRequest
	}{
		{
			description:              "Empty - Not Dirty",
			request:                  openrtb2.BidRequest{},
			requestRequestExtWrapper: RequestExt{},
			expectedRequest:          openrtb2.BidRequest{},
		},
		{
			description:              "Empty - Dirty",
			request:                  openrtb2.BidRequest{},
			requestRequestExtWrapper: RequestExt{prebid: &prebidContent1, prebidDirty: true},
			expectedRequest:          openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"integration":"1"}}`)},
		},
		{
			description:              "Empty - Dirty - No Change",
			request:                  openrtb2.BidRequest{},
			requestRequestExtWrapper: RequestExt{prebid: nil, prebidDirty: true},
			expectedRequest:          openrtb2.BidRequest{},
		},
		{
			description:              "Populated - Not Dirty",
			request:                  openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"integration":"1"}}`)},
			requestRequestExtWrapper: RequestExt{},
			expectedRequest:          openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"integration":"1"}}`)},
		},
		{
			description:              "Populated - Dirty",
			request:                  openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"integration":"1"}}`)},
			requestRequestExtWrapper: RequestExt{prebid: &prebidContent2, prebidDirty: true},
			expectedRequest:          openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"integration":"2"}}`)},
		},
		{
			description:              "Populated - Dirty - No Change",
			request:                  openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"integration":"1"}}`)},
			requestRequestExtWrapper: RequestExt{prebid: &prebidContent1, prebidDirty: true},
			expectedRequest:          openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"integration":"1"}}`)},
		},
		{
			description:              "Populated - Dirty - Cleared",
			request:                  openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"integration":"1"}}`)},
			requestRequestExtWrapper: RequestExt{prebid: nil, prebidDirty: true},
			expectedRequest:          openrtb2.BidRequest{},
		},
	}

	for _, test := range testCases {
		// create required filed in the test loop to keep test declaration easier to read
		test.requestRequestExtWrapper.ext = make(map[string]json.RawMessage)

		w := RequestWrapper{BidRequest: &test.request, requestExt: &test.requestRequestExtWrapper}
		w.RebuildRequest()
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}

func TestRebuildAppExt(t *testing.T) {
	prebidContent1 := ExtAppPrebid{Source: "1"}
	prebidContent2 := ExtAppPrebid{Source: "2"}

	testCases := []struct {
		description          string
		request              openrtb2.BidRequest
		requestAppExtWrapper AppExt
		expectedRequest      openrtb2.BidRequest
	}{
		{
			description:          "Nil - Not Dirty",
			request:              openrtb2.BidRequest{},
			requestAppExtWrapper: AppExt{},
			expectedRequest:      openrtb2.BidRequest{},
		},
		{
			description:          "Nil - Dirty",
			request:              openrtb2.BidRequest{},
			requestAppExtWrapper: AppExt{prebid: &prebidContent1, prebidDirty: true},
			expectedRequest:      openrtb2.BidRequest{App: &openrtb2.App{Ext: json.RawMessage(`{"prebid":{"source":"1"}}`)}},
		},
		{
			description:          "Nil - Dirty - No Change",
			request:              openrtb2.BidRequest{},
			requestAppExtWrapper: AppExt{prebid: nil, prebidDirty: true},
			expectedRequest:      openrtb2.BidRequest{},
		},
		{
			description:          "Empty - Not Dirty",
			request:              openrtb2.BidRequest{App: &openrtb2.App{}},
			requestAppExtWrapper: AppExt{},
			expectedRequest:      openrtb2.BidRequest{App: &openrtb2.App{}},
		},
		{
			description:          "Empty - Dirty",
			request:              openrtb2.BidRequest{App: &openrtb2.App{}},
			requestAppExtWrapper: AppExt{prebid: &prebidContent1, prebidDirty: true},
			expectedRequest:      openrtb2.BidRequest{App: &openrtb2.App{Ext: json.RawMessage(`{"prebid":{"source":"1"}}`)}},
		},
		{
			description:          "Empty - Dirty - No Change",
			request:              openrtb2.BidRequest{App: &openrtb2.App{}},
			requestAppExtWrapper: AppExt{prebid: nil, prebidDirty: true},
			expectedRequest:      openrtb2.BidRequest{App: &openrtb2.App{}},
		},
		{
			description:          "Populated - Not Dirty",
			request:              openrtb2.BidRequest{App: &openrtb2.App{Ext: json.RawMessage(`{"prebid":{"source":"1"}}`)}},
			requestAppExtWrapper: AppExt{},
			expectedRequest:      openrtb2.BidRequest{App: &openrtb2.App{Ext: json.RawMessage(`{"prebid":{"source":"1"}}`)}},
		},
		{
			description:          "Populated - Dirty",
			request:              openrtb2.BidRequest{App: &openrtb2.App{Ext: json.RawMessage(`{"prebid":{"source":"1"}}`)}},
			requestAppExtWrapper: AppExt{prebid: &prebidContent2, prebidDirty: true},
			expectedRequest:      openrtb2.BidRequest{App: &openrtb2.App{Ext: json.RawMessage(`{"prebid":{"source":"2"}}`)}},
		},
		{
			description:          "Populated - Dirty - No Change",
			request:              openrtb2.BidRequest{App: &openrtb2.App{Ext: json.RawMessage(`{"prebid":{"source":"1"}}`)}},
			requestAppExtWrapper: AppExt{prebid: &prebidContent1, prebidDirty: true},
			expectedRequest:      openrtb2.BidRequest{App: &openrtb2.App{Ext: json.RawMessage(`{"prebid":{"source":"1"}}`)}},
		},
		{
			description:          "Populated - Dirty - Cleared",
			request:              openrtb2.BidRequest{App: &openrtb2.App{Ext: json.RawMessage(`{"prebid":{"source":"1"}}`)}},
			requestAppExtWrapper: AppExt{prebid: nil, prebidDirty: true},
			expectedRequest:      openrtb2.BidRequest{App: &openrtb2.App{}},
		},
	}

	for _, test := range testCases {
		// create required filed in the test loop to keep test declaration easier to read
		test.requestAppExtWrapper.ext = make(map[string]json.RawMessage)

		w := RequestWrapper{BidRequest: &test.request, appExt: &test.requestAppExtWrapper}
		w.RebuildRequest()
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}

func TestRebuildSiteExt(t *testing.T) {
	int1 := int8(1)
	int2 := int8(2)

	testCases := []struct {
		description           string
		request               openrtb2.BidRequest
		requestSiteExtWrapper SiteExt
		expectedRequest       openrtb2.BidRequest
	}{
		{
			description:           "Nil - Not Dirty",
			request:               openrtb2.BidRequest{},
			requestSiteExtWrapper: SiteExt{},
			expectedRequest:       openrtb2.BidRequest{},
		},
		{
			description:           "Nil - Dirty",
			request:               openrtb2.BidRequest{},
			requestSiteExtWrapper: SiteExt{amp: &int1, ampDirty: true},
			expectedRequest:       openrtb2.BidRequest{Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)}},
		},
		{
			description:           "Nil - Dirty - No Change",
			request:               openrtb2.BidRequest{},
			requestSiteExtWrapper: SiteExt{amp: nil, ampDirty: true},
			expectedRequest:       openrtb2.BidRequest{},
		},
		{
			description:           "Empty - Not Dirty",
			request:               openrtb2.BidRequest{Site: &openrtb2.Site{}},
			requestSiteExtWrapper: SiteExt{},
			expectedRequest:       openrtb2.BidRequest{Site: &openrtb2.Site{}},
		},
		{
			description:           "Empty - Dirty",
			request:               openrtb2.BidRequest{Site: &openrtb2.Site{}},
			requestSiteExtWrapper: SiteExt{amp: &int1, ampDirty: true},
			expectedRequest:       openrtb2.BidRequest{Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)}},
		},
		{
			description:           "Empty - Dirty - No Change",
			request:               openrtb2.BidRequest{Site: &openrtb2.Site{}},
			requestSiteExtWrapper: SiteExt{amp: nil, ampDirty: true},
			expectedRequest:       openrtb2.BidRequest{Site: &openrtb2.Site{}},
		},
		{
			description:           "Populated - Not Dirty",
			request:               openrtb2.BidRequest{Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)}},
			requestSiteExtWrapper: SiteExt{},
			expectedRequest:       openrtb2.BidRequest{Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)}},
		},
		{
			description:           "Populated - Dirty",
			request:               openrtb2.BidRequest{Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)}},
			requestSiteExtWrapper: SiteExt{amp: &int2, ampDirty: true},
			expectedRequest:       openrtb2.BidRequest{Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":2}`)}},
		},
		{
			description:           "Populated - Dirty - No Change",
			request:               openrtb2.BidRequest{Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)}},
			requestSiteExtWrapper: SiteExt{amp: &int1, ampDirty: true},
			expectedRequest:       openrtb2.BidRequest{Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)}},
		},
		{
			description:           "Populated - Dirty - Cleared",
			request:               openrtb2.BidRequest{Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)}},
			requestSiteExtWrapper: SiteExt{amp: nil, ampDirty: true},
			expectedRequest:       openrtb2.BidRequest{Site: &openrtb2.Site{}},
		},
	}

	for _, test := range testCases {
		// create required filed in the test loop to keep test declaration easier to read
		test.requestSiteExtWrapper.ext = make(map[string]json.RawMessage)

		w := RequestWrapper{BidRequest: &test.request, siteExt: &test.requestSiteExtWrapper}
		w.RebuildRequest()
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}

func TestRebuildSourceExt(t *testing.T) {
	schainContent1 := openrtb2.SupplyChain{Ver: "1"}
	schainContent2 := openrtb2.SupplyChain{Ver: "2"}

	testCases := []struct {
		description             string
		request                 openrtb2.BidRequest
		requestSourceExtWrapper SourceExt
		expectedRequest         openrtb2.BidRequest
	}{
		{
			description:             "Nil - Not Dirty",
			request:                 openrtb2.BidRequest{},
			requestSourceExtWrapper: SourceExt{},
			expectedRequest:         openrtb2.BidRequest{},
		},
		{
			description:             "Nil - Dirty",
			request:                 openrtb2.BidRequest{},
			requestSourceExtWrapper: SourceExt{schain: &schainContent1, schainDirty: true},
			expectedRequest:         openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":0,"nodes":null,"ver":"1"}}`)}},
		},
		{
			description:             "Nil - Dirty - No Change",
			request:                 openrtb2.BidRequest{},
			requestSourceExtWrapper: SourceExt{schain: nil, schainDirty: true},
			expectedRequest:         openrtb2.BidRequest{},
		},
		{
			description:             "Empty - Not Dirty",
			request:                 openrtb2.BidRequest{Source: &openrtb2.Source{}},
			requestSourceExtWrapper: SourceExt{},
			expectedRequest:         openrtb2.BidRequest{Source: &openrtb2.Source{}},
		},
		{
			description:             "Empty - Dirty",
			request:                 openrtb2.BidRequest{Source: &openrtb2.Source{}},
			requestSourceExtWrapper: SourceExt{schain: &schainContent1, schainDirty: true},
			expectedRequest:         openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":0,"nodes":null,"ver":"1"}}`)}},
		},
		{
			description:             "Empty - Dirty - No Change",
			request:                 openrtb2.BidRequest{Source: &openrtb2.Source{}},
			requestSourceExtWrapper: SourceExt{schain: nil, schainDirty: true},
			expectedRequest:         openrtb2.BidRequest{Source: &openrtb2.Source{}},
		},
		{
			description:             "Populated - Not Dirty",
			request:                 openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":0,"nodes":null,"ver":"1"}}`)}},
			requestSourceExtWrapper: SourceExt{},
			expectedRequest:         openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":0,"nodes":null,"ver":"1"}}`)}},
		},
		{
			description:             "Populated - Dirty",
			request:                 openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":0,"nodes":null,"ver":"1"}}`)}},
			requestSourceExtWrapper: SourceExt{schain: &schainContent2, schainDirty: true},
			expectedRequest:         openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":0,"nodes":null,"ver":"2"}}`)}},
		},
		{
			description:             "Populated - Dirty - No Change",
			request:                 openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":0,"nodes":null,"ver":"1"}}`)}},
			requestSourceExtWrapper: SourceExt{schain: &schainContent1, schainDirty: true},
			expectedRequest:         openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":0,"nodes":null,"ver":"1"}}`)}},
		},
		{
			description:             "Populated - Dirty - Cleared",
			request:                 openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":0,"nodes":null,"ver":"1"}}`)}},
			requestSourceExtWrapper: SourceExt{schain: nil, schainDirty: true},
			expectedRequest:         openrtb2.BidRequest{Source: &openrtb2.Source{}},
		},
	}

	for _, test := range testCases {
		// create required filed in the test loop to keep test declaration easier to read
		test.requestSourceExtWrapper.ext = make(map[string]json.RawMessage)

		w := RequestWrapper{BidRequest: &test.request, sourceExt: &test.requestSourceExtWrapper}
		w.RebuildRequest()
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}
