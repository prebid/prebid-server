package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestCloneRequestWrapper(t *testing.T) {
	testCases := []struct {
		name        string
		reqWrap     *RequestWrapper
		reqWrapCopy *RequestWrapper                             // manual copy of above ext object to verify against
		mutator     func(t *testing.T, reqWrap *RequestWrapper) // function to modify the Ext object
	}{
		{
			name:        "Nil", // Verify the nil case
			reqWrap:     nil,
			reqWrapCopy: nil,
			mutator:     func(t *testing.T, reqWrap *RequestWrapper) {},
		},
		{
			name: "NoMutate",
			reqWrap: &RequestWrapper{
				impWrappers: []*ImpWrapper{
					{
						impExt: &ImpExt{prebid: &ExtImpPrebid{Options: &Options{EchoVideoAttrs: true}}, prebidDirty: true, tid: "fun"},
					},
					{
						impExt: &ImpExt{tid: "star"},
					},
				},
				userExt:   &UserExt{consentDirty: true},
				deviceExt: &DeviceExt{extDirty: true},
				requestExt: &RequestExt{
					prebid: &ExtRequestPrebid{Integration: "derivative"},
				},
				appExt:    &AppExt{prebidDirty: true},
				regExt:    &RegExt{usPrivacy: "foo"},
				siteExt:   &SiteExt{amp: ptrutil.ToPtr[int8](1)},
				sourceExt: &SourceExt{schainDirty: true},
			},
			reqWrapCopy: &RequestWrapper{
				impWrappers: []*ImpWrapper{
					{
						impExt: &ImpExt{prebid: &ExtImpPrebid{Options: &Options{EchoVideoAttrs: true}}, prebidDirty: true, tid: "fun"},
					},
					{
						impExt: &ImpExt{tid: "star"},
					},
				},
				userExt:   &UserExt{consentDirty: true},
				deviceExt: &DeviceExt{extDirty: true},
				requestExt: &RequestExt{
					prebid: &ExtRequestPrebid{Integration: "derivative"},
				},
				appExt:    &AppExt{prebidDirty: true},
				regExt:    &RegExt{usPrivacy: "foo"},
				siteExt:   &SiteExt{amp: ptrutil.ToPtr[int8](1)},
				sourceExt: &SourceExt{schainDirty: true},
			},
			mutator: func(t *testing.T, reqWrap *RequestWrapper) {},
		},
		{
			name: "General",
			reqWrap: &RequestWrapper{
				impWrappers: []*ImpWrapper{
					{
						impExt: &ImpExt{prebid: &ExtImpPrebid{Options: &Options{EchoVideoAttrs: true}}, prebidDirty: true, tid: "fun"},
					},
					{
						impExt: &ImpExt{tid: "star"},
					},
				},
				userExt:   &UserExt{consentDirty: true},
				deviceExt: &DeviceExt{extDirty: true},
				requestExt: &RequestExt{
					prebid: &ExtRequestPrebid{Integration: "derivative"},
				},
				appExt:    &AppExt{prebidDirty: true},
				regExt:    &RegExt{usPrivacy: "foo"},
				siteExt:   &SiteExt{amp: ptrutil.ToPtr[int8](1)},
				sourceExt: &SourceExt{schainDirty: true},
			},
			reqWrapCopy: &RequestWrapper{
				impWrappers: []*ImpWrapper{
					{
						impExt: &ImpExt{prebid: &ExtImpPrebid{Options: &Options{EchoVideoAttrs: true}}, prebidDirty: true, tid: "fun"},
					},
					{
						impExt: &ImpExt{tid: "star"},
					},
				},
				userExt:   &UserExt{consentDirty: true},
				deviceExt: &DeviceExt{extDirty: true},
				requestExt: &RequestExt{
					prebid: &ExtRequestPrebid{Integration: "derivative"},
				},
				appExt:    &AppExt{prebidDirty: true},
				regExt:    &RegExt{usPrivacy: "foo"},
				siteExt:   &SiteExt{amp: ptrutil.ToPtr[int8](1)},
				sourceExt: &SourceExt{schainDirty: true},
			},
			mutator: func(t *testing.T, reqWrap *RequestWrapper) {
				reqWrap.impWrappers[1].impExt.prebidDirty = true
				reqWrap.impWrappers[0] = nil
				reqWrap.impWrappers = append(reqWrap.impWrappers, &ImpWrapper{impExt: &ImpExt{tid: "star"}})
				reqWrap.impWrappers = nil
				reqWrap.userExt = nil
				reqWrap.deviceExt = nil
				reqWrap.requestExt = nil
				reqWrap.appExt = nil
				reqWrap.regExt = nil
				reqWrap.siteExt = nil
				reqWrap.sourceExt = nil
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := test.reqWrap.Clone()
			test.mutator(t, test.reqWrap)
			assert.Equal(t, test.reqWrapCopy, clone)
		})
	}
}

func TestUserExt(t *testing.T) {
	userExt := &UserExt{}

	userExt.unmarshal(nil)
	assert.Equal(t, false, userExt.Dirty(), "New UserExt should not be dirty.")
	assert.Nil(t, userExt.GetConsent(), "Empty UserExt should have nil consent")
	assert.Nil(t, userExt.GetEid(), "Empty UserExt should have nil eid")
	assert.Nil(t, userExt.GetPrebid(), "Empty UserExt should have nil prebid")
	assert.Nil(t, userExt.GetConsentedProvidersSettingsIn(), "Empty UserExt should have nil consentedProvidersSettings")

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

	consentedProvidersSettings := &ConsentedProvidersSettingsIn{ConsentedProvidersString: "1~X.X.X"}
	userExt.SetConsentedProvidersSettingsIn(consentedProvidersSettings)
	assert.Equal(t, &ConsentedProvidersSettingsIn{ConsentedProvidersString: "1~X.X.X"}, userExt.GetConsentedProvidersSettingsIn(), "UserExt consentedProvidersSettings is incorrect")
	assert.Equal(t, true, userExt.Dirty(), "UserExt should be dirty after field updates")
	cpsIn := userExt.GetConsentedProvidersSettingsIn()
	assert.Equal(t, "1~X.X.X", cpsIn.ConsentedProvidersString, "UserExt consentedProviders is incorrect")

	consentedProvidersString := "1~1.35.41.101"
	cpsIn.ConsentedProvidersString = consentedProvidersString
	userExt.SetConsentedProvidersSettingsIn(cpsIn)
	cpsIn = userExt.GetConsentedProvidersSettingsIn()
	assert.Equal(t, "1~1.35.41.101", cpsIn.ConsentedProvidersString, "UserExt consentedProviders is incorrect")

	cpsOut := &ConsentedProvidersSettingsOut{}
	//cpsOut.ConsentedProvidersList = make([]int, 0, 1)
	cpsOut.ConsentedProvidersList = append(cpsOut.ConsentedProvidersList, ParseConsentedProvidersString(consentedProvidersString)...)
	assert.Len(t, cpsOut.ConsentedProvidersList, 4, "UserExt consentedProvidersList is incorrect")
	userExt.SetConsentedProvidersSettingsOut(cpsOut)

	updatedUserExt, err := userExt.marshal()
	assert.Nil(t, err, "Marshalling UserExt after updating should not cause an error")

	expectedUserExt := json.RawMessage(`{"consent":"NewConsent","prebid":{"buyeruids":{"buyer":"id"}},"consented_providers_settings":{"consented_providers":[1,35,41,101]},"ConsentedProvidersSettings":{"consented_providers":"1~1.35.41.101"},"eids":[{"source":"source","uids":[{"id":"id"}]}]}`)
	assert.JSONEq(t, string(expectedUserExt), string(updatedUserExt), "Marshalled UserExt is incorrect")

	assert.Equal(t, false, userExt.Dirty(), "UserExt should not be dirty after marshalling")
}

func TestRebuildImp(t *testing.T) {
	var (
		prebid     = &ExtImpPrebid{IsRewardedInventory: openrtb2.Int8Ptr(1)}
		prebidJson = json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)
	)

	testCases := []struct {
		description       string
		request           openrtb2.BidRequest
		requestImpWrapper []*ImpWrapper
		expectedRequest   openrtb2.BidRequest
		expectedError     string
	}{
		{
			description:       "Empty - Never Accessed",
			request:           openrtb2.BidRequest{},
			requestImpWrapper: nil,
			expectedRequest:   openrtb2.BidRequest{},
		},
		{
			description:       "One - Never Accessed",
			request:           openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "1"}}},
			requestImpWrapper: nil,
			expectedRequest:   openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "1"}}},
		},
		{
			description:       "One - Accessed - Dirty",
			request:           openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "1"}}},
			requestImpWrapper: []*ImpWrapper{{Imp: &openrtb2.Imp{ID: "2"}, impExt: &ImpExt{prebid: prebid, prebidDirty: true}}},
			expectedRequest:   openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "2", Ext: prebidJson}}},
		},
		{
			description:       "One - Accessed - Error",
			request:           openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "1"}}},
			requestImpWrapper: []*ImpWrapper{{Imp: nil, impExt: &ImpExt{}}},
			expectedError:     "ImpWrapper RebuildImp called on a nil Imp",
		},
		{
			description:       "Many - Accessed - Dirty / Not Dirty",
			request:           openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "1"}, {ID: "2"}}},
			requestImpWrapper: []*ImpWrapper{{Imp: &openrtb2.Imp{ID: "1"}, impExt: &ImpExt{}}, {Imp: &openrtb2.Imp{ID: "2"}, impExt: &ImpExt{prebid: prebid, prebidDirty: true}}},
			expectedRequest:   openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "1"}, {ID: "2", Ext: prebidJson}}},
		},
	}

	for _, test := range testCases {
		// create required filed in the test loop to keep test declaration easier to read
		for _, w := range test.requestImpWrapper {
			w.impExt.ext = make(map[string]json.RawMessage)
		}

		w := RequestWrapper{BidRequest: &test.request, impWrappers: test.requestImpWrapper, impWrappersAccessed: test.requestImpWrapper != nil}
		err := w.RebuildRequest()

		if test.expectedError != "" {
			assert.EqualError(t, err, test.expectedError, test.description)
		} else {
			assert.NoError(t, err, test.description)
			assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
		}
	}
}

func TestRebuildUserExt(t *testing.T) {
	strA := "a"
	strB := "b"

	type testCase struct {
		description           string
		request               openrtb2.BidRequest
		requestUserExtWrapper UserExt
		expectedRequest       openrtb2.BidRequest
	}
	testGroups := []struct {
		groupDesc string
		tests     []testCase
	}{
		{
			groupDesc: "Consent string tests",
			tests: []testCase{
				// Nil req.User
				{
					description:           "Nil req.User - UserExt Not Dirty",
					request:               openrtb2.BidRequest{},
					requestUserExtWrapper: UserExt{},
					expectedRequest:       openrtb2.BidRequest{},
				},
				{
					description:           "Nil req.User - Dirty UserExt",
					request:               openrtb2.BidRequest{},
					requestUserExtWrapper: UserExt{consent: &strA, consentDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
				},
				{
					description:           "Nil req.User - Dirty UserExt but consent is nil - No Change",
					request:               openrtb2.BidRequest{},
					requestUserExtWrapper: UserExt{consent: nil, consentDirty: true},
					expectedRequest:       openrtb2.BidRequest{},
				},
				// Nil req.User.Ext
				{
					description:           "Nil req.User.Ext - Not Dirty - No Change",
					request:               openrtb2.BidRequest{User: &openrtb2.User{}},
					requestUserExtWrapper: UserExt{},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
				},
				{
					description:           "Nil req.User.Ext - Dirty with valid consent string - Expect consent string to be added",
					request:               openrtb2.BidRequest{User: &openrtb2.User{}},
					requestUserExtWrapper: UserExt{consent: &strB, consentDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"b"}`)}},
				},
				{
					description:           "Nil req.User.Ext - Dirty UserExt with nil consent string- No Change",
					request:               openrtb2.BidRequest{User: &openrtb2.User{}},
					requestUserExtWrapper: UserExt{consent: nil, consentDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
				},
				// Not-nil req.User.Ext
				{
					description:           "Populated - Not Dirty - No Change",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
					requestUserExtWrapper: UserExt{},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
				},
				{
					description:           "Populated - Dirty - Consent string overriden",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
					requestUserExtWrapper: UserExt{consent: &strB, consentDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"b"}`)}},
				},
				{
					description:           "Populated - Dirty but consent string is equal to the one already set - No Change",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
					requestUserExtWrapper: UserExt{consent: &strA, consentDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
				},
				{
					description:           "Populated - Dirty UserExt with nil consent string - Cleared",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"a"}`)}},
					requestUserExtWrapper: UserExt{consent: nil, consentDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
				},
			},
		},
		{
			groupDesc: "Consented provider settings string form",
			tests: []testCase{
				// Nil req.User
				{
					description:           "Nil req.User - Dirty UserExt with nil consentedProviderSettings - No Change",
					request:               openrtb2.BidRequest{},
					requestUserExtWrapper: UserExt{},
					expectedRequest:       openrtb2.BidRequest{},
				},
				{
					description:           "Nil req.User - Dirty UserExt with empty consentedProviderSettings - No Change",
					request:               openrtb2.BidRequest{},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsIn: &ConsentedProvidersSettingsIn{}, consentedProvidersSettingsInDirty: true},
					expectedRequest:       openrtb2.BidRequest{},
				},
				{
					description:           "Nil req.User - Dirty UserExt with populated consentedProviderSettings - ConsentedProvidersSettings are added",
					request:               openrtb2.BidRequest{},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsIn: &ConsentedProvidersSettingsIn{ConsentedProvidersString: "ConsentedProvidersString"}, consentedProvidersSettingsInDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"ConsentedProvidersSettings":{"consented_providers":"ConsentedProvidersString"}}`)}},
				},
				// Nil req.User.Ext
				{
					description:           "Nil req.User.Ext - Dirty UserExt with nil consentedProviderSettings - No Change",
					request:               openrtb2.BidRequest{User: &openrtb2.User{}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsIn: nil, consentedProvidersSettingsInDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
				},
				{
					description:           "Nil req.User.Ext - Dirty UserExt with empty consentedProviderSettings - No Change",
					request:               openrtb2.BidRequest{User: &openrtb2.User{}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsIn: &ConsentedProvidersSettingsIn{}, consentedProvidersSettingsInDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
				},
				{
					description:           "Nil req.User.Ext - Dirty UserExt with populated consentedProviderSettings - ConsentedProvidersSettings are added",
					request:               openrtb2.BidRequest{User: &openrtb2.User{}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsIn: &ConsentedProvidersSettingsIn{ConsentedProvidersString: "ConsentedProvidersString"}, consentedProvidersSettingsInDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"ConsentedProvidersSettings":{"consented_providers":"ConsentedProvidersString"}}`)}},
				},
				// Not-nil req.User.Ext
				{
					description:           "Populated req.User.Ext - Not Dirty UserExt - No Change",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"ConsentedProvidersSettings":{"consented_providers":"1~X.X.X.X"}}`)}},
					requestUserExtWrapper: UserExt{},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"ConsentedProvidersSettings":{"consented_providers":"1~X.X.X.X"}}`)}},
				},
				{
					description:           "Populated req.User.Ext - Dirty UserExt with nil consentedProviderSettings - Populated req.User.Ext gets overriden with nil User.Ext",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"ConsentedProvidersSettings":{"consented_providers":"1~X.X.X.X"}}`)}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsIn: nil, consentedProvidersSettingsInDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
				},
				{
					description:           "Populated req.User.Ext - Dirty UserExt with empty consentedProviderSettings - Populated req.User.Ext gets overriden with nil User.Ext",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"1~X.X.X.X":{"consented_providers":"1~X.X.X.X"}}`)}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsIn: &ConsentedProvidersSettingsIn{}, consentedProvidersSettingsInDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
				},
				{
					description:           "Populated req.User.Ext - Dirty UserExt with populated consentedProviderSettings - ConsentedProvidersSettings are overriden",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"ConsentedProvidersSettings":{"consented_providers":"1~X.X.X.X"}}`)}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsIn: &ConsentedProvidersSettingsIn{ConsentedProvidersString: "1~1.35.41.101"}, consentedProvidersSettingsInDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"ConsentedProvidersSettings":{"consented_providers":"1~1.35.41.101"}}`)}},
				},
			},
		},
		{
			groupDesc: "Consented provider settings array",
			tests: []testCase{
				// Nil req.User
				{
					description:           "Nil req.User - Dirty UserExt with nil consentedProviderSettings - No Change",
					request:               openrtb2.BidRequest{},
					requestUserExtWrapper: UserExt{},
					expectedRequest:       openrtb2.BidRequest{},
				},
				{
					description:           "Nil req.User - Dirty UserExt with empty consentedProviderSettings - No Change",
					request:               openrtb2.BidRequest{},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsOut: &ConsentedProvidersSettingsOut{}, consentedProvidersSettingsOutDirty: true},
					expectedRequest:       openrtb2.BidRequest{},
				},
				{
					description:           "Nil req.User - Dirty UserExt with populated consentedProviderSettings - ConsentedProvidersSettings are added",
					request:               openrtb2.BidRequest{},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsOut: &ConsentedProvidersSettingsOut{ConsentedProvidersList: []int{1, 35, 41, 101}}, consentedProvidersSettingsOutDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consented_providers_settings":{"consented_providers":[1,35,41,101]}}`)}},
				},
				// Nil req.User.Ext
				{
					description:           "Nil req.User.Ext - Dirty UserExt with nil consentedProviderSettings - No Change",
					request:               openrtb2.BidRequest{User: &openrtb2.User{}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsOut: nil, consentedProvidersSettingsOutDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
				},
				{
					description:           "Nil req.User.Ext - Dirty UserExt with empty consentedProviderSettings - No Change",
					request:               openrtb2.BidRequest{User: &openrtb2.User{}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsOut: &ConsentedProvidersSettingsOut{}, consentedProvidersSettingsOutDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
				},
				{
					description:           "Nil req.User.Ext - Dirty UserExt with populated consentedProviderSettings - ConsentedProvidersSettings are added",
					request:               openrtb2.BidRequest{User: &openrtb2.User{}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsOut: &ConsentedProvidersSettingsOut{ConsentedProvidersList: []int{1, 35, 41, 101}}, consentedProvidersSettingsOutDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consented_providers_settings":{"consented_providers":[1,35,41,101]}}`)}},
				},
				// Not-nil req.User.Ext
				{
					description:           "Populated req.User.Ext - Not Dirty UserExt - No Change",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consented_providers_settings":{"consented_providers":[1,35,41,101]}}`)}},
					requestUserExtWrapper: UserExt{},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consented_providers_settings":{"consented_providers":[1,35,41,101]}}`)}},
				},
				{
					description:           "Populated req.User.Ext - Dirty UserExt with nil consentedProviderSettingsOut - Populated req.User.Ext gets overriden with nil User.Ext",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consented_providers_settings":{"consented_providers":[1,35,41,101]}}`)}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsOut: nil, consentedProvidersSettingsOutDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
				},
				{
					description:           "Populated req.User.Ext - Dirty UserExt with empty consentedProviderSettingsOut - Populated req.User.Ext gets overriden with nil User.Ext",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consented_providers_settings":{"consented_providers":[1,35,41,101]}}`)}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsOut: &ConsentedProvidersSettingsOut{}, consentedProvidersSettingsOutDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{}},
				},
				{
					description:           "Populated req.User.Ext - Dirty UserExt with populated consentedProviderSettingsOut - consented_providers list elements are overriden",
					request:               openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consented_providers_settings":{"consented_providers":[1,35,41,101]}}`)}},
					requestUserExtWrapper: UserExt{consentedProvidersSettingsOut: &ConsentedProvidersSettingsOut{ConsentedProvidersList: []int{35, 36, 240}}, consentedProvidersSettingsOutDirty: true},
					expectedRequest:       openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consented_providers_settings":{"consented_providers":[35,36,240]}}`)}},
				},
			},
		},
	}

	for _, group := range testGroups {
		for _, test := range group.tests {
			// create required filed in the test loop to keep test declaration easier to read
			test.requestUserExtWrapper.ext = make(map[string]json.RawMessage)

			w := RequestWrapper{BidRequest: &test.request, userExt: &test.requestUserExtWrapper}
			w.RebuildRequest()
			assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
		}
	}
}

func TestUserExtUnmarshal(t *testing.T) {
	type testInput struct {
		userExt *UserExt
		extJson json.RawMessage
	}
	testCases := []struct {
		desc        string
		in          testInput
		expectError bool
	}{
		{
			desc: "UserExt.ext is not nil, don't expect error",
			in: testInput{
				userExt: &UserExt{
					ext: map[string]json.RawMessage{
						"eids": json.RawMessage(`[{"source":"value"}]`),
					},
				},
				extJson: json.RawMessage(`{"prebid":{"buyeruids":{"elem1":"value1"}}}`),
			},
			expectError: false,
		},
		{
			desc: "UserExt.ext is dirty, don't expect error",
			in: testInput{
				userExt: &UserExt{extDirty: true},
				extJson: json.RawMessage(`{"prebid":{"buyeruids":{"elem1":"value1"}}}`),
			},
			expectError: false,
		},
		// Eids
		{
			desc: "Has eids and it is valid JSON",
			in: testInput{
				userExt: &UserExt{},
				extJson: json.RawMessage(`{"eids":[{"source":"value"}]}`),
			},
			expectError: false,
		},
		{
			desc: "Has malformed eids expect error",
			in: testInput{
				userExt: &UserExt{},
				extJson: json.RawMessage(`{"eids":123}`),
			},
			expectError: true,
		},
		// prebid
		{
			desc: "Has prebid and it is valid JSON",
			in: testInput{
				userExt: &UserExt{},
				extJson: json.RawMessage(`{"prebid":{"buyeruids":{"elem1":"value1"}}}`),
			},
			expectError: false,
		},
		{
			desc: "Has malformed prebid expect error",
			in: testInput{
				userExt: &UserExt{},
				extJson: json.RawMessage(`{"prebid":{"buyeruids":123}}`),
			},
			expectError: true,
		},
		// ConsentedProvidersSettings
		{
			desc: "Has ConsentedProvidersSettings and it is valid JSON",
			in: testInput{
				userExt: &UserExt{},
				extJson: json.RawMessage(`{"ConsentedProvidersSettings":{"consented_providers":"ConsentedProvidersString"}}`),
			},
			expectError: false,
		},
		{
			desc: "Has malformed ConsentedProvidersSettings expect error",
			in: testInput{
				userExt: &UserExt{},
				extJson: json.RawMessage(`{"ConsentedProvidersSettings":{"consented_providers":123}}`),
			},
			expectError: true,
		},
		// consented_providers_settings
		{
			desc: "Has consented_providers_settings and it is valid JSON",
			in: testInput{
				userExt: &UserExt{},
				extJson: json.RawMessage(`{"consented_providers_settings":{"consented_providers":[2,25]}}`),
			},
			expectError: false,
		},
		{
			desc: "Has malformed consented_providers_settings expect error",
			in: testInput{
				userExt: &UserExt{},
				extJson: json.RawMessage(`{"consented_providers_settings":{"consented_providers":123}}`),
			},
			expectError: true,
		},
	}
	for _, tc := range testCases {
		err := tc.in.userExt.unmarshal(tc.in.extJson)

		if tc.expectError {
			assert.Error(t, err, tc.desc)
		} else {
			assert.NoError(t, err, tc.desc)
		}
	}
}

func TestCloneUserExt(t *testing.T) {
	testCases := []struct {
		name        string
		userExt     *UserExt
		userExtCopy *UserExt                             // manual copy of above ext object to verify against
		mutator     func(t *testing.T, userExt *UserExt) // function to modify the Ext object
	}{
		{
			name:        "Nil", // Verify the nil case
			userExt:     nil,
			userExtCopy: nil,
			mutator:     func(t *testing.T, user *UserExt) {},
		},
		{
			name: "NoMutate",
			userExt: &UserExt{
				ext:          map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				consent:      ptrutil.ToPtr("Myconsent"),
				consentDirty: true,
				prebid: &ExtUserPrebid{
					BuyerUIDs: map[string]string{"A": "X", "B": "Y"},
				},
				prebidDirty: true,
				eids:        &[]openrtb2.EID{},
			},
			userExtCopy: &UserExt{
				ext:          map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				consent:      ptrutil.ToPtr("Myconsent"),
				consentDirty: true,
				prebid: &ExtUserPrebid{
					BuyerUIDs: map[string]string{"A": "X", "B": "Y"},
				},
				prebidDirty: true,
				eids:        &[]openrtb2.EID{},
			},
			mutator: func(t *testing.T, user *UserExt) {},
		},
		{
			name: "General",
			userExt: &UserExt{
				ext:          map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				consent:      ptrutil.ToPtr("Myconsent"),
				consentDirty: true,
				prebid: &ExtUserPrebid{
					BuyerUIDs: map[string]string{"A": "X", "B": "Y"},
				},
				prebidDirty: true,
				eids:        &[]openrtb2.EID{},
			},
			userExtCopy: &UserExt{
				ext:          map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				consent:      ptrutil.ToPtr("Myconsent"),
				consentDirty: true,
				prebid: &ExtUserPrebid{
					BuyerUIDs: map[string]string{"A": "X", "B": "Y"},
				},
				prebidDirty: true,
				eids:        &[]openrtb2.EID{},
			},
			mutator: func(t *testing.T, user *UserExt) {
				user.ext["A"] = json.RawMessage(`G`)
				user.ext["C"] = json.RawMessage(`L`)
				user.extDirty = true
				user.consent = nil
				user.consentDirty = false
				user.prebid.BuyerUIDs["A"] = "C"
				user.prebid.BuyerUIDs["C"] = "A"
				user.prebid = nil
			},
		},
		{
			name: "EIDs",
			userExt: &UserExt{
				eids: &[]openrtb2.EID{
					{
						Source: "Sauce",
						UIDs: []openrtb2.UID{
							{ID: "A", AType: 5, Ext: json.RawMessage(`{}`)},
							{ID: "B", AType: 1, Ext: json.RawMessage(`{"extra": "stuff"}`)},
						},
					},
					{
						Source: "Moon",
						UIDs: []openrtb2.UID{
							{ID: "G", AType: 3, Ext: json.RawMessage(`{}`)},
							{ID: "D", AType: 1},
						},
					},
				},
			},
			userExtCopy: &UserExt{
				eids: &[]openrtb2.EID{
					{
						Source: "Sauce",
						UIDs: []openrtb2.UID{
							{ID: "A", AType: 5, Ext: json.RawMessage(`{}`)},
							{ID: "B", AType: 1, Ext: json.RawMessage(`{"extra": "stuff"}`)},
						},
					},
					{
						Source: "Moon",
						UIDs: []openrtb2.UID{
							{ID: "G", AType: 3, Ext: json.RawMessage(`{}`)},
							{ID: "D", AType: 1},
						},
					},
				},
			},
			mutator: func(t *testing.T, userExt *UserExt) {
				eids := *userExt.eids
				eids[0].UIDs[1].ID = "G2"
				eids[1].UIDs[0].AType = 0
				eids[0].UIDs = append(eids[0].UIDs, openrtb2.UID{ID: "Z", AType: 2})
				eids = append(eids, openrtb2.EID{Source: "Blank"})
				userExt.eids = nil
			},
		},
		{
			name: "ConsentedProviders",
			userExt: &UserExt{
				consentedProvidersSettingsIn: &ConsentedProvidersSettingsIn{
					ConsentedProvidersString: "A,B,C",
				},
				consentedProvidersSettingsOut: &ConsentedProvidersSettingsOut{
					ConsentedProvidersList: []int{1, 2, 3, 4},
				},
			},
			userExtCopy: &UserExt{
				consentedProvidersSettingsIn: &ConsentedProvidersSettingsIn{
					ConsentedProvidersString: "A,B,C",
				},
				consentedProvidersSettingsOut: &ConsentedProvidersSettingsOut{
					ConsentedProvidersList: []int{1, 2, 3, 4},
				},
			},
			mutator: func(t *testing.T, userExt *UserExt) {
				userExt.consentedProvidersSettingsIn.ConsentedProvidersString = "B,C,D"
				userExt.consentedProvidersSettingsIn = &ConsentedProvidersSettingsIn{
					ConsentedProvidersString: "G,H,I",
				}
				userExt.consentedProvidersSettingsOut.ConsentedProvidersList[1] = 5
				userExt.consentedProvidersSettingsOut.ConsentedProvidersList = append(userExt.consentedProvidersSettingsOut.ConsentedProvidersList, 7)
				userExt.consentedProvidersSettingsOut = nil
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := test.userExt.Clone()
			test.mutator(t, test.userExt)
			assert.Equal(t, test.userExtCopy, clone)
		})
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

func TestCloneRequestExt(t *testing.T) {
	testCases := []struct {
		name       string
		reqExt     *RequestExt
		reqExtCopy *RequestExt                            // manual copy of above ext object to verify against
		mutator    func(t *testing.T, reqExt *RequestExt) // function to modify the Ext object
	}{
		{
			name:       "Nil", // Verify the nil case
			reqExt:     nil,
			reqExtCopy: nil,
			mutator:    func(t *testing.T, reqExt *RequestExt) {},
		},
		{
			name: "NoMutate", // Verify the nil case
			reqExt: &RequestExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`{}`), "B": json.RawMessage(`{"foo":"bar"}`)},
				extDirty: true,
				prebid: &ExtRequestPrebid{
					BidderParams: json.RawMessage(`{}`),
				},
			},
			reqExtCopy: &RequestExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`{}`), "B": json.RawMessage(`{"foo":"bar"}`)},
				extDirty: true,
				prebid: &ExtRequestPrebid{
					BidderParams: json.RawMessage(`{}`),
				},
			},
			mutator: func(t *testing.T, reqExt *RequestExt) {},
		},
		{
			name: "General", // Verify the nil case
			reqExt: &RequestExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`{}`), "B": json.RawMessage(`{"foo":"bar"}`)},
				extDirty: true,
				prebid: &ExtRequestPrebid{
					BidderParams: json.RawMessage(`{}`),
				},
			},
			reqExtCopy: &RequestExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`{}`), "B": json.RawMessage(`{"foo":"bar"}`)},
				extDirty: true,
				prebid: &ExtRequestPrebid{
					BidderParams: json.RawMessage(`{}`),
				},
			},
			mutator: func(t *testing.T, reqExt *RequestExt) {
				reqExt.ext["A"] = json.RawMessage(`"string"`)
				reqExt.ext["C"] = json.RawMessage(`{}`)
				reqExt.extDirty = false
				reqExt.prebid.Channel = &ExtRequestPrebidChannel{Name: "Bob"}
				reqExt.prebid.BidderParams = nil
				reqExt.prebid = nil
			},
		},
		{
			name: "SChain", // Verify the nil case
			reqExt: &RequestExt{
				schain: &openrtb2.SupplyChain{
					Complete: 1,
					Ver:      "1.1",
					Nodes: []openrtb2.SupplyChainNode{
						{ASI: "Is a", RID: "off", HP: ptrutil.ToPtr[int8](1)},
						{ASI: "other", RID: "drift", HP: ptrutil.ToPtr[int8](0)},
					},
				},
			},
			reqExtCopy: &RequestExt{
				schain: &openrtb2.SupplyChain{
					Complete: 1,
					Ver:      "1.1",
					Nodes: []openrtb2.SupplyChainNode{
						{ASI: "Is a", RID: "off", HP: ptrutil.ToPtr[int8](1)},
						{ASI: "other", RID: "drift", HP: ptrutil.ToPtr[int8](0)},
					},
				},
			},
			mutator: func(t *testing.T, reqExt *RequestExt) {
				reqExt.schain.Complete = 0
				reqExt.schain.Ver = "1.2"
				reqExt.schain.Nodes[0].ASI = "some"
				reqExt.schain.Nodes[1].HP = nil
				reqExt.schain.Nodes = append(reqExt.schain.Nodes, openrtb2.SupplyChainNode{ASI: "added"})
				reqExt.schain = nil
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := test.reqExt.Clone()
			test.mutator(t, test.reqExt)
			assert.Equal(t, test.reqExtCopy, clone)
		})
	}

}

func TestCloneDeviceExt(t *testing.T) {
	testCases := []struct {
		name       string
		devExt     *DeviceExt
		devExtCopy *DeviceExt                            // manual copy of above ext object to verify against
		mutator    func(t *testing.T, devExt *DeviceExt) // function to modify the Ext object
	}{
		{
			name:       "Nil", // Verify the nil case
			devExt:     nil,
			devExtCopy: nil,
			mutator:    func(t *testing.T, devExt *DeviceExt) {},
		},
		{
			name: "NoMutate",
			devExt: &DeviceExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`{}`), "B": json.RawMessage(`{"foo":"bar"}`)},
				extDirty: true,
				prebid: &ExtDevicePrebid{
					Interstitial: &ExtDeviceInt{MinWidthPerc: 65.0, MinHeightPerc: 75.0},
				},
			},
			devExtCopy: &DeviceExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`{}`), "B": json.RawMessage(`{"foo":"bar"}`)},
				extDirty: true,
				prebid: &ExtDevicePrebid{
					Interstitial: &ExtDeviceInt{MinWidthPerc: 65.0, MinHeightPerc: 75.0},
				},
			},
			mutator: func(t *testing.T, devExt *DeviceExt) {},
		},
		{
			name: "General",
			devExt: &DeviceExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`{}`), "B": json.RawMessage(`{"foo":"bar"}`)},
				extDirty: true,
				prebid: &ExtDevicePrebid{
					Interstitial: &ExtDeviceInt{MinWidthPerc: 65.0, MinHeightPerc: 75.0},
				},
			},
			devExtCopy: &DeviceExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`{}`), "B": json.RawMessage(`{"foo":"bar"}`)},
				extDirty: true,
				prebid: &ExtDevicePrebid{
					Interstitial: &ExtDeviceInt{MinWidthPerc: 65, MinHeightPerc: 75},
				},
			},
			mutator: func(t *testing.T, devExt *DeviceExt) {
				devExt.ext["A"] = json.RawMessage(`"string"`)
				devExt.ext["C"] = json.RawMessage(`{}`)
				devExt.extDirty = false
				devExt.prebid.Interstitial.MinHeightPerc = 55
				devExt.prebid.Interstitial = &ExtDeviceInt{MinWidthPerc: 80}
				devExt.prebid = nil
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := test.devExt.Clone()
			test.mutator(t, test.devExt)
			assert.Equal(t, test.devExtCopy, clone)
		})
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

func TestCloneAppExt(t *testing.T) {
	testCases := []struct {
		name       string
		appExt     *AppExt
		appExtCopy *AppExt                            // manual copy of above ext object to verify against
		mutator    func(t *testing.T, appExt *AppExt) // function to modify the Ext object
	}{
		{
			name:       "Nil", // Verify the nil case
			appExt:     nil,
			appExtCopy: nil,
			mutator:    func(t *testing.T, appExt *AppExt) {},
		},
		{
			name: "NoMutate",
			appExt: &AppExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				prebid: &ExtAppPrebid{
					Source:  "Sauce",
					Version: "2.2",
				},
			},
			appExtCopy: &AppExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				prebid: &ExtAppPrebid{
					Source:  "Sauce",
					Version: "2.2",
				},
			},
			mutator: func(t *testing.T, appExt *AppExt) {},
		},
		{
			name: "General",
			appExt: &AppExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				prebid: &ExtAppPrebid{
					Source:  "Sauce",
					Version: "2.2",
				},
			},
			appExtCopy: &AppExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				prebid: &ExtAppPrebid{
					Source:  "Sauce",
					Version: "2.2",
				},
			},
			mutator: func(t *testing.T, appExt *AppExt) {
				appExt.ext["A"] = json.RawMessage(`"string"`)
				appExt.ext["C"] = json.RawMessage(`{}`)
				appExt.extDirty = false
				appExt.prebid.Source = "foobar"
				appExt.prebid = nil
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := test.appExt.Clone()
			test.mutator(t, test.appExt)
			assert.Equal(t, test.appExtCopy, clone)
		})
	}
}

func TestCloneRegExt(t *testing.T) {
	testCases := []struct {
		name       string
		regExt     *RegExt
		regExtCopy *RegExt                            // manual copy of above ext object to verify against
		mutator    func(t *testing.T, regExt *RegExt) // function to modify the Ext object
	}{
		{
			name:       "Nil", // Verify the nil case
			regExt:     nil,
			regExtCopy: nil,
			mutator:    func(t *testing.T, appExt *RegExt) {},
		},
		{
			name: "NoMutate",
			regExt: &RegExt{
				ext:            map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty:       true,
				gdpr:           ptrutil.ToPtr[int8](1),
				usPrivacy:      "priv",
				usPrivacyDirty: true,
			},
			regExtCopy: &RegExt{
				ext:            map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty:       true,
				gdpr:           ptrutil.ToPtr[int8](1),
				usPrivacy:      "priv",
				usPrivacyDirty: true,
			},
			mutator: func(t *testing.T, appExt *RegExt) {},
		},
		{
			name: "General",
			regExt: &RegExt{
				ext:            map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty:       true,
				gdpr:           ptrutil.ToPtr[int8](1),
				usPrivacy:      "priv",
				usPrivacyDirty: true,
			},
			regExtCopy: &RegExt{
				ext:            map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty:       true,
				gdpr:           ptrutil.ToPtr[int8](1),
				usPrivacy:      "priv",
				usPrivacyDirty: true,
			},
			mutator: func(t *testing.T, appExt *RegExt) {
				appExt.ext["A"] = json.RawMessage(`"string"`)
				appExt.ext["C"] = json.RawMessage(`{}`)
				appExt.extDirty = false
				appExt.gdpr = nil
				appExt.gdprDirty = true
				appExt.usPrivacy = "Other"
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := test.regExt.Clone()
			test.mutator(t, test.regExt)
			assert.Equal(t, test.regExtCopy, clone)
		})
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

func TestCloneSiteExt(t *testing.T) {
	testCases := []struct {
		name        string
		siteExt     *SiteExt
		siteExtCopy *SiteExt                             // manual copy of above ext object to verify against
		mutator     func(t *testing.T, siteExt *SiteExt) // function to modify the Ext object
	}{
		{
			name:        "Nil", // Verify the nil case
			siteExt:     nil,
			siteExtCopy: nil,
			mutator:     func(t *testing.T, siteExt *SiteExt) {},
		},
		{
			name: "NoMutate",
			siteExt: &SiteExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				amp:      ptrutil.ToPtr[int8](1),
			},
			siteExtCopy: &SiteExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				amp:      ptrutil.ToPtr[int8](1),
			},
			mutator: func(t *testing.T, siteExt *SiteExt) {},
		},
		{
			name: "General",
			siteExt: &SiteExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				amp:      ptrutil.ToPtr[int8](1),
			},
			siteExtCopy: &SiteExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				amp:      ptrutil.ToPtr[int8](1),
			},
			mutator: func(t *testing.T, siteExt *SiteExt) {
				siteExt.ext["A"] = json.RawMessage(`"string"`)
				siteExt.ext["C"] = json.RawMessage(`{}`)
				siteExt.extDirty = false
				siteExt.amp = nil
				siteExt.ampDirty = true
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := test.siteExt.Clone()
			test.mutator(t, test.siteExt)
			assert.Equal(t, test.siteExtCopy, clone)
		})
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

func TestCloneSourceExt(t *testing.T) {
	testCases := []struct {
		name          string
		sourceExt     *SourceExt
		sourceExtCopy *SourceExt                               // manual copy of above ext object to verify against
		mutator       func(t *testing.T, sourceExt *SourceExt) // function to modify the Ext object
	}{
		{
			name:          "Nil", // Verify the nil case
			sourceExt:     nil,
			sourceExtCopy: nil,
			mutator:       func(t *testing.T, sourceExt *SourceExt) {},
		},
		{
			name: "NoMutate",
			sourceExt: &SourceExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				schain: &openrtb2.SupplyChain{
					Complete: 1,
					Ver:      "1.1",
					Nodes: []openrtb2.SupplyChainNode{
						{ASI: "Is a", RID: "off", HP: ptrutil.ToPtr[int8](1)},
						{ASI: "other", RID: "drift", HP: ptrutil.ToPtr[int8](0)},
					},
				},
			},
			sourceExtCopy: &SourceExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				schain: &openrtb2.SupplyChain{
					Complete: 1,
					Ver:      "1.1",
					Nodes: []openrtb2.SupplyChainNode{
						{ASI: "Is a", RID: "off", HP: ptrutil.ToPtr[int8](1)},
						{ASI: "other", RID: "drift", HP: ptrutil.ToPtr[int8](0)},
					},
				},
			},
			mutator: func(t *testing.T, sourceExt *SourceExt) {
				sourceExt.ext["A"] = json.RawMessage(`"string"`)
				sourceExt.ext["C"] = json.RawMessage(`{}`)
				sourceExt.extDirty = false
				sourceExt.schain.Complete = 0
				sourceExt.schain.Ver = "1.2"
				sourceExt.schain.Nodes[0].ASI = "some"
				sourceExt.schain.Nodes[1].HP = nil
				sourceExt.schain.Nodes = append(sourceExt.schain.Nodes, openrtb2.SupplyChainNode{ASI: "added"})
				sourceExt.schain = nil

			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := test.sourceExt.Clone()
			test.mutator(t, test.sourceExt)
			assert.Equal(t, test.sourceExtCopy, clone)
		})
	}
}

func TestImpWrapperRebuildImp(t *testing.T) {
	var (
		isRewardedInventoryOne int8 = 1
		isRewardedInventoryTwo int8 = 2
	)

	testCases := []struct {
		description   string
		imp           openrtb2.Imp
		impExtWrapper ImpExt
		expectedImp   openrtb2.Imp
	}{
		{
			description:   "Empty - Not Dirty",
			imp:           openrtb2.Imp{},
			impExtWrapper: ImpExt{},
			expectedImp:   openrtb2.Imp{},
		},
		{
			description:   "Empty - Dirty",
			imp:           openrtb2.Imp{},
			impExtWrapper: ImpExt{prebid: &ExtImpPrebid{IsRewardedInventory: &isRewardedInventoryOne}, prebidDirty: true},
			expectedImp:   openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)},
		},
		{
			description:   "Empty - Dirty - No Change",
			imp:           openrtb2.Imp{},
			impExtWrapper: ImpExt{prebid: nil, prebidDirty: true},
			expectedImp:   openrtb2.Imp{},
		},
		{
			description:   "Populated - Not Dirty",
			imp:           openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)},
			impExtWrapper: ImpExt{},
			expectedImp:   openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)},
		},
		{
			description:   "Populated - Dirty",
			imp:           openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)},
			impExtWrapper: ImpExt{prebid: &ExtImpPrebid{IsRewardedInventory: &isRewardedInventoryTwo}, prebidDirty: true},
			expectedImp:   openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":2}}`)},
		},
		{
			description:   "Populated - Dirty - No Change",
			imp:           openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)},
			impExtWrapper: ImpExt{prebid: &ExtImpPrebid{IsRewardedInventory: &isRewardedInventoryOne}, prebidDirty: true},
			expectedImp:   openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)},
		},
		{
			description:   "Populated - Dirty - Cleared",
			imp:           openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)},
			impExtWrapper: ImpExt{prebid: nil, prebidDirty: true},
			expectedImp:   openrtb2.Imp{},
		},
		{
			description:   "Populated - Dirty - Empty Prebid Object",
			imp:           openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)},
			impExtWrapper: ImpExt{prebid: &ExtImpPrebid{IsRewardedInventory: nil}, prebidDirty: true},
			expectedImp:   openrtb2.Imp{},
		},
		{
			description:   "Populated Tid - Dirty",
			imp:           openrtb2.Imp{Ext: json.RawMessage(`{"tid": "some-tid"}`)},
			impExtWrapper: ImpExt{tidDirty: true, tid: "12345"},
			expectedImp:   openrtb2.Imp{Ext: json.RawMessage(`{"tid":"12345"}`)},
		},
		{
			description:   "Populated Tid - Dirty - No Change",
			imp:           openrtb2.Imp{Ext: json.RawMessage(`{"tid": "some-tid"}`)},
			impExtWrapper: ImpExt{tid: "some-tid", tidDirty: true},
			expectedImp:   openrtb2.Imp{Ext: json.RawMessage(`{"tid":"some-tid"}`)},
		},
		{
			description:   "Populated Tid - Dirty - Cleared",
			imp:           openrtb2.Imp{Ext: json.RawMessage(`{"tid":"some-tid"}`)},
			impExtWrapper: ImpExt{tid: "", tidDirty: true},
			expectedImp:   openrtb2.Imp{},
		},
	}

	for _, test := range testCases {
		// create required filed in the test loop to keep test declaration easier to read
		test.impExtWrapper.ext = make(map[string]json.RawMessage)

		w := &ImpWrapper{Imp: &test.imp, impExt: &test.impExtWrapper}
		w.RebuildImp()
		assert.Equal(t, test.expectedImp, *w.Imp, test.description)
	}
}

func TestImpWrapperGetImpExt(t *testing.T) {
	var isRewardedInventoryOne int8 = 1

	testCases := []struct {
		description    string
		givenWrapper   ImpWrapper
		expectedImpExt ImpExt
		expectedError  string
	}{
		{
			description:    "Empty",
			givenWrapper:   ImpWrapper{},
			expectedImpExt: ImpExt{ext: make(map[string]json.RawMessage)},
		},
		{
			description:  "Populated - Ext",
			givenWrapper: ImpWrapper{Imp: &openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1},"other":42,"tid":"test-tid","gpid":"test-gpid","data":{"adserver":{"name":"ads","adslot":"adslot123"},"pbadslot":"pbadslot123"}}`)}},
			expectedImpExt: ImpExt{
				ext: map[string]json.RawMessage{
					"prebid": json.RawMessage(`{"is_rewarded_inventory":1}`),
					"other":  json.RawMessage(`42`),
					"tid":    json.RawMessage(`"test-tid"`),
					"gpid":   json.RawMessage(`"test-gpid"`),
					"data":   json.RawMessage(`{"adserver":{"name":"ads","adslot":"adslot123"},"pbadslot":"pbadslot123"}`),
				},
				tid:  "test-tid",
				gpId: "test-gpid",
				data: &ExtImpData{
					AdServer: &ExtImpDataAdServer{
						Name:   "ads",
						AdSlot: "adslot123",
					},
					PbAdslot: "pbadslot123",
				},
				prebid: &ExtImpPrebid{IsRewardedInventory: &isRewardedInventoryOne},
			},
		},
		{
			description: "Already Retrieved - Dirty - Not Unmarshalled Again",
			givenWrapper: ImpWrapper{
				Imp:    &openrtb2.Imp{Ext: json.RawMessage(`{"will":"be ignored"}`)},
				impExt: &ImpExt{ext: map[string]json.RawMessage{"foo": json.RawMessage("bar")}}},
			expectedImpExt: ImpExt{ext: map[string]json.RawMessage{"foo": json.RawMessage("bar")}},
		},
		{
			description:   "Error - Ext",
			givenWrapper:  ImpWrapper{Imp: &openrtb2.Imp{Ext: json.RawMessage(`malformed`)}},
			expectedError: "invalid character 'm' looking for beginning of value",
		},
		{
			description:   "Error - Ext - Prebid",
			givenWrapper:  ImpWrapper{Imp: &openrtb2.Imp{Ext: json.RawMessage(`{"prebid":malformed}`)}},
			expectedError: "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		impExt, err := test.givenWrapper.GetImpExt()
		if test.expectedError != "" {
			assert.EqualError(t, err, test.expectedError, test.description)
		} else {
			assert.NoError(t, err, test.description)
			assert.Equal(t, test.expectedImpExt, *impExt, test.description)
		}
	}
}

func TestImpExtTid(t *testing.T) {
	impExt := &ImpExt{}

	impExt.unmarshal(nil)
	assert.Equal(t, false, impExt.Dirty(), "New impext should not be dirty.")
	assert.Empty(t, impExt.GetTid(), "Empty ImpExt should have  empty tid")

	newTid := "tid"
	impExt.SetTid(newTid)
	assert.Equal(t, "tid", impExt.GetTid(), "ImpExt tid is incorrect")
	assert.Equal(t, true, impExt.Dirty(), "New impext should be dirty.")
}

func TestCloneImpWrapper(t *testing.T) {
	testCases := []struct {
		name           string
		impWrapper     *ImpWrapper
		impWrapperCopy *ImpWrapper                                // manual copy of above ext object to verify against
		mutator        func(t *testing.T, impWrapper *ImpWrapper) // function to modify the Ext object
	}{
		{
			name:           "Nil", // Verify the nil case
			impWrapper:     nil,
			impWrapperCopy: nil,
			mutator:        func(t *testing.T, impWrapper *ImpWrapper) {},
		},
		{
			name: "NoMutate",
			impWrapper: &ImpWrapper{
				impExt: &ImpExt{
					tid: "occupied",
				},
			},
			impWrapperCopy: &ImpWrapper{
				impExt: &ImpExt{
					tid: "occupied",
				},
			},
			mutator: func(t *testing.T, impWrapper *ImpWrapper) {},
		},
		{
			name: "General",
			impWrapper: &ImpWrapper{
				impExt: &ImpExt{
					tid: "occupied",
				},
			},
			impWrapperCopy: &ImpWrapper{
				impExt: &ImpExt{
					tid: "occupied",
				},
			},
			mutator: func(t *testing.T, impWrapper *ImpWrapper) {
				impWrapper.impExt.extDirty = true
				impWrapper.impExt.tid = "Something"
				impWrapper.impExt = nil
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := test.impWrapper.Clone()
			test.mutator(t, test.impWrapper)
			assert.Equal(t, test.impWrapperCopy, clone)
		})
	}
}

func TestCloneImpExt(t *testing.T) {
	testCases := []struct {
		name       string
		impExt     *ImpExt
		impExtCopy *ImpExt                            // manual copy of above ext object to verify against
		mutator    func(t *testing.T, impExt *ImpExt) // function to modify the Ext object
	}{
		{
			name:       "Nil", // Verify the nil case
			impExt:     nil,
			impExtCopy: nil,
			mutator:    func(t *testing.T, impExt *ImpExt) {},
		},
		{
			name: "NoMutate",
			impExt: &ImpExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				tid:      "TID",
			},
			impExtCopy: &ImpExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				tid:      "TID",
			},
			mutator: func(t *testing.T, impExt *ImpExt) {},
		},
		{
			name: "General",
			impExt: &ImpExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				tid:      "TID",
			},
			impExtCopy: &ImpExt{
				ext:      map[string]json.RawMessage{"A": json.RawMessage(`X`), "B": json.RawMessage(`Y`)},
				extDirty: true,
				tid:      "TID",
			},
			mutator: func(t *testing.T, impExt *ImpExt) {
				impExt.ext["A"] = json.RawMessage(`"string"`)
				impExt.ext["C"] = json.RawMessage(`{}`)
				impExt.extDirty = false
				impExt.tid = "other"
				impExt.tidDirty = true
			},
		},
		{
			name: "Prebid",
			impExt: &ImpExt{
				prebid: &ExtImpPrebid{
					StoredRequest:         &ExtStoredRequest{ID: "abc123"},
					StoredAuctionResponse: &ExtStoredAuctionResponse{ID: "123abc"},
					StoredBidResponse: []ExtStoredBidResponse{
						{ID: "foo", Bidder: "bar", ReplaceImpId: ptrutil.ToPtr(true)},
						{ID: "def", Bidder: "xyz", ReplaceImpId: ptrutil.ToPtr(false)},
					},
					IsRewardedInventory: ptrutil.ToPtr[int8](1),
					Bidder: map[string]json.RawMessage{
						"abc": json.RawMessage(`{}`),
						"def": json.RawMessage(`{"alpha":"beta"}`),
					},
					Options:     &Options{EchoVideoAttrs: true},
					Passthrough: json.RawMessage(`{"foo":"bar"}`),
					Floors: &ExtImpPrebidFloors{
						FloorRule:      "Rule 16",
						FloorRuleValue: 16.17,
						FloorValue:     6.7,
					},
				},
			},
			impExtCopy: &ImpExt{
				prebid: &ExtImpPrebid{
					StoredRequest:         &ExtStoredRequest{ID: "abc123"},
					StoredAuctionResponse: &ExtStoredAuctionResponse{ID: "123abc"},
					StoredBidResponse: []ExtStoredBidResponse{
						{ID: "foo", Bidder: "bar", ReplaceImpId: ptrutil.ToPtr(true)},
						{ID: "def", Bidder: "xyz", ReplaceImpId: ptrutil.ToPtr(false)},
					},
					IsRewardedInventory: ptrutil.ToPtr[int8](1),
					Bidder: map[string]json.RawMessage{
						"abc": json.RawMessage(`{}`),
						"def": json.RawMessage(`{"alpha":"beta"}`),
					},
					Options:     &Options{EchoVideoAttrs: true},
					Passthrough: json.RawMessage(`{"foo":"bar"}`),
					Floors: &ExtImpPrebidFloors{
						FloorRule:      "Rule 16",
						FloorRuleValue: 16.17,
						FloorValue:     6.7,
					},
				},
			},
			mutator: func(t *testing.T, impExt *ImpExt) {
				impExt.prebid.StoredRequest.ID = "seventy"
				impExt.prebid.StoredRequest = nil
				impExt.prebid.StoredAuctionResponse.ID = "xyz"
				impExt.prebid.StoredAuctionResponse = nil
				impExt.prebid.StoredBidResponse[0].ID = "alpha"
				impExt.prebid.StoredBidResponse[1].ReplaceImpId = nil
				impExt.prebid.StoredBidResponse[0] = ExtStoredBidResponse{ID: "o", Bidder: "k", ReplaceImpId: ptrutil.ToPtr(false)}
				impExt.prebid.StoredBidResponse = append(impExt.prebid.StoredBidResponse, ExtStoredBidResponse{ID: "jay", Bidder: "walk"})
				impExt.prebid.IsRewardedInventory = nil
				impExt.prebid.Bidder["def"] = json.RawMessage(``)
				delete(impExt.prebid.Bidder, "abc")
				impExt.prebid.Bidder["xyz"] = json.RawMessage(`{"jar":5}`)
				impExt.prebid.Options.EchoVideoAttrs = false
				impExt.prebid.Options = nil
				impExt.prebid.Passthrough = json.RawMessage(`{}`)
				impExt.prebid.Floors.FloorRule = "Friday"
				impExt.prebid.Floors.FloorMinCur = "EUR"
				impExt.prebid.Floors = nil
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := test.impExt.Clone()
			test.mutator(t, test.impExt)
			assert.Equal(t, test.impExtCopy, clone)
		})
	}
}
