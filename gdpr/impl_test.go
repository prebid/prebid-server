package gdpr

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/go-gdpr/vendorlist2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestDisallowOnEmptyConsent(t *testing.T) {
	emptyConsent := ""
	perms := permissionsImpl{
		cfg:              &tcf2Config{},
		fetchVendorList:  failedListFetcher,
		gdprDefaultValue: "0",
		hostVendorID:     3,
		vendorIDs:        nil,
		gdprSignal:       SignalYes,
		consent:          emptyConsent,
	}

	allowSync, err := perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderAppnexus)
	assertBoolsEqual(t, false, allowSync)
	assertNilErr(t, err)
	allowSync, err = perms.HostCookiesAllowed(context.Background())
	assertBoolsEqual(t, false, allowSync)
	assertNilErr(t, err)
}

func TestAllowOnSignalNo(t *testing.T) {
	emptyConsent := ""
	perms := permissionsImpl{
		gdprSignal: SignalNo,
		consent:    emptyConsent,
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background())
	assert.Equal(t, true, allowSync)
	assert.Nil(t, err)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderAppnexus)
	assert.Equal(t, true, allowSync)
	assert.Nil(t, err)
}

func TestAllowedSyncs(t *testing.T) {
	vendor2AndPurpose1Consent := "CPGWbY_PGWbY_GYAAAENABCAAIAAAAAAAAAAACEAAAAA"
	vendorListData := MarshalVendorList(vendorList{
		VendorListVersion: 2,
		Vendors: map[string]*vendor{
			"2": {
				ID:       2,
				Purposes: []int{1},
			},
		},
	})

	tcf2AggConfig := tcf2Config{
		HostConfig: config.TCF2{
			Purpose1: config.TCF2Purpose{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
		},
	}
	tcf2AggConfig.HostConfig.PurposeConfigs = map[consentconstants.Purpose]*config.TCF2Purpose{
		consentconstants.Purpose(1): &tcf2AggConfig.HostConfig.Purpose1,
	}

	perms := permissionsImpl{
		cfg:          &tcf2AggConfig,
		hostVendorID: 2,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
		},
		fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
			2: {
				1: parseVendorListDataV2(t, vendorListData),
			},
		}),
		purposeEnforcerBuilder: NewPurposeEnforcerBuilder(&tcf2AggConfig),
		gdprSignal:             SignalYes,
		consent:                vendor2AndPurpose1Consent,
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background())
	assertNilErr(t, err)
	assertBoolsEqual(t, true, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderAppnexus)
	assertNilErr(t, err)
	assertBoolsEqual(t, true, allowSync)
}

func TestProhibitedPurposes(t *testing.T) {
	vendor2NoPurpose1Consent := "CPGWkCaPGWkCaApAAAENABCAAAAAAAAAAAAAABEAAAAA"
	vendorListData := MarshalVendorList(vendorList{
		VendorListVersion: 2,
		Vendors: map[string]*vendor{
			"2": {
				ID:       2,
				Purposes: []int{1},
			},
		},
	})

	tcf2AggConfig := tcf2Config{
		HostConfig: config.TCF2{
			Purpose1: config.TCF2Purpose{
				EnforcePurpose: true,
			},
		},
	}
	tcf2AggConfig.HostConfig.PurposeConfigs = map[consentconstants.Purpose]*config.TCF2Purpose{
		consentconstants.Purpose(1): &tcf2AggConfig.HostConfig.Purpose1,
	}

	perms := permissionsImpl{
		cfg:          &tcf2AggConfig,
		hostVendorID: 2,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
		},
		fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
			2: {
				1: parseVendorListDataV2(t, vendorListData),
			},
		}),
		purposeEnforcerBuilder: NewPurposeEnforcerBuilder(&tcf2AggConfig),
		gdprSignal:             SignalYes,
		consent:                vendor2NoPurpose1Consent,
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background())
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderAppnexus)
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)
}

func TestProhibitedVendors(t *testing.T) {
	purpose1NoVendorConsent := "CPGWkCaPGWkCaApAAAENABCAAIAAAAAAAAAAABAAAAAA"
	vendorListData := MarshalVendorList(vendorList{
		VendorListVersion: 2,
		Vendors: map[string]*vendor{
			"2": {
				ID:       2,
				Purposes: []int{1},
			},
		},
	})

	tcf2AggConfig := tcf2Config{
		HostConfig: config.TCF2{
			Purpose1: config.TCF2Purpose{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
		},
	}
	tcf2AggConfig.HostConfig.PurposeConfigs = map[consentconstants.Purpose]*config.TCF2Purpose{
		consentconstants.Purpose(1): &tcf2AggConfig.HostConfig.Purpose1,
	}

	perms := permissionsImpl{
		cfg:          &tcf2AggConfig,
		hostVendorID: 2,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
		},
		fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
			2: {
				1: parseVendorListDataV2(t, vendorListData),
			},
		}),
		purposeEnforcerBuilder: NewPurposeEnforcerBuilder(&tcf2AggConfig),
		gdprSignal:             SignalYes,
		consent:                purpose1NoVendorConsent,
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background())
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderPubmatic)
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)
}

func TestMalformedConsent(t *testing.T) {
	perms := permissionsImpl{
		hostVendorID:    2,
		fetchVendorList: listFetcher(nil),
		gdprSignal:      SignalYes,
		consent:         "BON",
	}

	sync, err := perms.HostCookiesAllowed(context.Background())
	assertErr(t, err, true)
	assertBoolsEqual(t, false, sync)
}

func TestAllowActivities(t *testing.T) {
	bidderAllowedByConsent := openrtb_ext.BidderAppnexus
	aliasedBidderAllowedByConsent := openrtb_ext.BidderName("appnexus1")
	bidderBlockedByConsent := openrtb_ext.BidderRubicon
	vendor2AndPurpose2Consent := "CPGWbY_PGWbY_GYAAAENABCAAEAAAAAAAAAAACEAAAAA"

	tests := []struct {
		description           string
		bidderName            openrtb_ext.BidderName
		bidderCoreName        openrtb_ext.BidderName
		publisherID           string
		gdpr                  Signal
		consent               string
		passID                bool
		weakVendorEnforcement bool
		aliasGVLIDs           map[string]uint16
	}{
		{
			description:    "Allow PI - Non standard publisher",
			bidderName:     bidderBlockedByConsent,
			bidderCoreName: bidderBlockedByConsent,
			publisherID:    "appNexusAppID",
			gdpr:           SignalYes,
			consent:        vendor2AndPurpose2Consent,
			passID:         true,
		},
		{
			description:    "Allow PI - known vendor with No GDPR",
			bidderName:     bidderBlockedByConsent,
			bidderCoreName: bidderBlockedByConsent,
			gdpr:           SignalNo,
			consent:        vendor2AndPurpose2Consent,
			passID:         true,
		},
		{
			description:    "Allow PI - known vendor with Yes GDPR",
			bidderName:     bidderAllowedByConsent,
			bidderCoreName: bidderAllowedByConsent,
			gdpr:           SignalYes,
			consent:        vendor2AndPurpose2Consent,
			passID:         true,
		},
		{
			description:    "Allow PI - known Alias vendor GVLID with Yes GDPR",
			bidderName:     aliasedBidderAllowedByConsent,
			bidderCoreName: bidderAllowedByConsent,
			gdpr:           SignalYes,
			consent:        vendor2AndPurpose2Consent,
			passID:         true,
			aliasGVLIDs:    map[string]uint16{"appnexus1": 2},
		},
		{
			description:    "Don't allow PI - known alias vendor with Yes GDPR, alias vendor does not consent to purpose 2",
			bidderName:     aliasedBidderAllowedByConsent,
			bidderCoreName: bidderAllowedByConsent,
			gdpr:           SignalYes,
			consent:        vendor2AndPurpose2Consent,
			passID:         false,
			aliasGVLIDs:    map[string]uint16{"appnexus1": 1},
		},
		{
			description:    "Allow PI - known vendor with Ambiguous GDPR and empty consent",
			bidderName:     bidderAllowedByConsent,
			bidderCoreName: bidderAllowedByConsent,
			gdpr:           SignalAmbiguous,
			consent:        "",
			passID:         true,
		},
		{
			description:    "Allow PI - known vendor with Ambiguous GDPR and non-empty consent",
			bidderName:     bidderAllowedByConsent,
			bidderCoreName: bidderAllowedByConsent,
			gdpr:           SignalAmbiguous,
			consent:        vendor2AndPurpose2Consent,
			passID:         true,
		},
		{
			description:    "Don't allow PI - known vendor with Yes GDPR and empty consent",
			bidderName:     bidderAllowedByConsent,
			bidderCoreName: bidderAllowedByConsent,
			gdpr:           SignalYes,
			consent:        "",
			passID:         false,
		},
		{
			description:    "Don't allow PI - default vendor with Yes GDPR and non-empty consent",
			bidderName:     bidderBlockedByConsent,
			bidderCoreName: bidderBlockedByConsent,
			gdpr:           SignalYes,
			consent:        vendor2AndPurpose2Consent,
			passID:         false,
		},
	}
	vendorListData := MarshalVendorList(vendorList{
		VendorListVersion: 1,
		Vendors: map[string]*vendor{
			"2": {
				ID:       2,
				Purposes: []int{2},
			},
		},
	})
	tcf2AggConfig := allPurposesEnabledTCF2Config()

	perms := permissionsImpl{
		cfg:                   &tcf2AggConfig,
		hostVendorID:          2,
		nonStandardPublishers: map[string]struct{}{"appNexusAppID": {}},
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
		},
		fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
			2: {
				1: parseVendorListDataV2(t, vendorListData),
			},
		}),
		purposeEnforcerBuilder: NewPurposeEnforcerBuilder(&tcf2AggConfig),
	}

	for _, tt := range tests {
		perms.aliasGVLIDs = tt.aliasGVLIDs
		perms.consent = tt.consent
		perms.gdprSignal = tt.gdpr
		perms.publisherID = tt.publisherID

		permissions := perms.AuctionActivitiesAllowed(context.Background(), tt.bidderCoreName, tt.bidderName)

		assert.Equal(t, tt.passID, permissions.PassID, tt.description)
	}
}

func TestAllowActivitiesBidderWithoutGVLID(t *testing.T) {
	bidderWithoutGVLID := openrtb_ext.BidderPangle
	purpose2Consent := "CPuDXznPuDXznMOAAAENCZCAAEAAAAAAAAAAAAAAAAAA"
	noPurposeConsent := "CPuDXznPuDXznMOAAAENCZCAAAAAAAAAAAAAAAAAAAAA"

	tests := []struct {
		name                    string
		enforceAlgoID           config.TCF2EnforcementAlgo
		vendorExceptions        map[string]struct{}
		basicEnforcementVendors map[string]struct{}
		consent                 string
		allowBidRequest         bool
		passID                  bool
	}{
		{
			name:          "full_enforcement_no_exceptions_user_consents_to_purpose_2",
			enforceAlgoID: config.TCF2FullEnforcement,
			consent:       purpose2Consent,
		},
		{
			name:             "full_enforcement_vendor_exception_user_consents_to_purpose_2",
			enforceAlgoID:    config.TCF2FullEnforcement,
			vendorExceptions: map[string]struct{}{string(bidderWithoutGVLID): {}},
			consent:          purpose2Consent,
			allowBidRequest:  true,
			passID:           true,
		},
		{
			name:    "basic_enforcement_no_exceptions_user_consents_to_purpose_2",
			consent: purpose2Consent,
		},
		{
			name:             "basic_enforcement_vendor_exception_user_consents_to_purpose_2",
			vendorExceptions: map[string]struct{}{string(bidderWithoutGVLID): {}},
			consent:          purpose2Consent,
			allowBidRequest:  true,
			passID:           true,
		},
		{
			name:                    "full_enforcement_soft_vendor_exception_user_consents_to_purpose_2", // allow bid request and pass ID
			enforceAlgoID:           config.TCF2FullEnforcement,
			basicEnforcementVendors: map[string]struct{}{string(bidderWithoutGVLID): {}},
			consent:                 purpose2Consent,
			allowBidRequest:         true,
			passID:                  true,
		},
		{
			name:                    "basic_enforcement_soft_vendor_exception_user_consents_to_purpose_2", // allow bid request and pass ID
			enforceAlgoID:           config.TCF2BasicEnforcement,
			basicEnforcementVendors: map[string]struct{}{string(bidderWithoutGVLID): {}},
			consent:                 purpose2Consent,
			allowBidRequest:         true,
			passID:                  true,
		},
		{
			name:                    "full_enforcement_soft_vendor_exception_user_consents_to_purpose_4",
			enforceAlgoID:           config.TCF2FullEnforcement,
			basicEnforcementVendors: map[string]struct{}{string(bidderWithoutGVLID): {}},
			consent:                 noPurposeConsent,
			allowBidRequest:         false,
			passID:                  false,
		},
		{
			name:                    "basic_enforcement_soft_vendor_exception_user_consents_to_purpose_4",
			enforceAlgoID:           config.TCF2BasicEnforcement,
			basicEnforcementVendors: map[string]struct{}{string(bidderWithoutGVLID): {}},
			consent:                 noPurposeConsent,
			allowBidRequest:         false,
			passID:                  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcf2AggConfig := allPurposesEnabledTCF2Config()
			tcf2AggConfig.AccountConfig.BasicEnforcementVendorsMap = tt.basicEnforcementVendors
			tcf2AggConfig.HostConfig.Purpose2.VendorExceptionMap = tt.vendorExceptions
			tcf2AggConfig.HostConfig.Purpose2.EnforceAlgoID = tt.enforceAlgoID
			tcf2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(2)] = &tcf2AggConfig.HostConfig.Purpose2

			perms := permissionsImpl{
				cfg:                   &tcf2AggConfig,
				consent:               tt.consent,
				gdprSignal:            SignalYes,
				hostVendorID:          2,
				nonStandardPublishers: map[string]struct{}{},
				vendorIDs:             map[openrtb_ext.BidderName]uint16{},
				fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
					2: {
						153: parseVendorListDataV2(t, MarshalVendorList(vendorList{GVLSpecificationVersion: 2, VendorListVersion: 153, Vendors: map[string]*vendor{}})),
					},
				}),
				purposeEnforcerBuilder: NewPurposeEnforcerBuilder(&tcf2AggConfig),
			}

			permissions := perms.AuctionActivitiesAllowed(context.Background(), bidderWithoutGVLID, bidderWithoutGVLID)
			assert.Equal(t, tt.allowBidRequest, permissions.AllowBidRequest)
			assert.Equal(t, tt.passID, permissions.PassID)
		})
	}
}

func buildVendorList34() vendorList {
	return vendorList{
		VendorListVersion: 2,
		Vendors: map[string]*vendor{
			"2": {
				ID:       2,
				Purposes: []int{1},
			},
			"6": {
				ID:               6,
				Purposes:         []int{1, 2, 4},
				LegIntPurposes:   []int{7},
				SpecialFeatures:  []int{1},
				FlexiblePurposes: []int{1, 2, 4, 7},
			},
			"8": {
				ID:             8,
				Purposes:       []int{1, 7},
				LegIntPurposes: []int{2, 4},
			},
			"10": {
				ID:              10,
				Purposes:        []int{2, 4, 7},
				SpecialFeatures: []int{1},
			},
			"20": {
				ID:               20,
				Purposes:         []int{1},
				LegIntPurposes:   []int{2, 7},
				FlexiblePurposes: []int{2, 7},
			},
			"32": {
				ID:       32,
				Purposes: []int{1, 2, 4, 7},
			},
		},
	}
}

func allPurposesEnabledTCF2Config() (TCF2AggConfig tcf2Config) {
	TCF2AggConfig = tcf2Config{
		HostConfig: config.TCF2{
			Enabled:         true,
			Purpose1:        config.TCF2Purpose{EnforceAlgoID: config.TCF2FullEnforcement, EnforcePurpose: true, EnforceVendors: true},
			Purpose2:        config.TCF2Purpose{EnforceAlgoID: config.TCF2FullEnforcement, EnforcePurpose: true, EnforceVendors: true},
			Purpose3:        config.TCF2Purpose{EnforceAlgoID: config.TCF2FullEnforcement, EnforcePurpose: true, EnforceVendors: true},
			Purpose4:        config.TCF2Purpose{EnforceAlgoID: config.TCF2FullEnforcement, EnforcePurpose: true, EnforceVendors: true},
			Purpose5:        config.TCF2Purpose{EnforceAlgoID: config.TCF2FullEnforcement, EnforcePurpose: true, EnforceVendors: true},
			Purpose6:        config.TCF2Purpose{EnforceAlgoID: config.TCF2FullEnforcement, EnforcePurpose: true, EnforceVendors: true},
			Purpose7:        config.TCF2Purpose{EnforceAlgoID: config.TCF2FullEnforcement, EnforcePurpose: true, EnforceVendors: true},
			Purpose8:        config.TCF2Purpose{EnforceAlgoID: config.TCF2FullEnforcement, EnforcePurpose: true, EnforceVendors: true},
			Purpose9:        config.TCF2Purpose{EnforceAlgoID: config.TCF2FullEnforcement, EnforcePurpose: true, EnforceVendors: true},
			Purpose10:       config.TCF2Purpose{EnforceAlgoID: config.TCF2FullEnforcement, EnforcePurpose: true, EnforceVendors: true},
			SpecialFeature1: config.TCF2SpecialFeature{Enforce: true},
		},
		AccountConfig: config.AccountGDPR{
			PurposeConfigs: map[consentconstants.Purpose]*config.AccountGDPRPurpose{
				consentconstants.Purpose(1):  {},
				consentconstants.Purpose(2):  {},
				consentconstants.Purpose(3):  {},
				consentconstants.Purpose(4):  {},
				consentconstants.Purpose(5):  {},
				consentconstants.Purpose(6):  {},
				consentconstants.Purpose(7):  {},
				consentconstants.Purpose(8):  {},
				consentconstants.Purpose(9):  {},
				consentconstants.Purpose(10): {},
			},
		},
	}
	TCF2AggConfig.HostConfig.PurposeConfigs = map[consentconstants.Purpose]*config.TCF2Purpose{
		consentconstants.Purpose(1):  &TCF2AggConfig.HostConfig.Purpose1,
		consentconstants.Purpose(2):  &TCF2AggConfig.HostConfig.Purpose2,
		consentconstants.Purpose(3):  &TCF2AggConfig.HostConfig.Purpose3,
		consentconstants.Purpose(4):  &TCF2AggConfig.HostConfig.Purpose4,
		consentconstants.Purpose(5):  &TCF2AggConfig.HostConfig.Purpose5,
		consentconstants.Purpose(6):  &TCF2AggConfig.HostConfig.Purpose6,
		consentconstants.Purpose(7):  &TCF2AggConfig.HostConfig.Purpose7,
		consentconstants.Purpose(8):  &TCF2AggConfig.HostConfig.Purpose8,
		consentconstants.Purpose(9):  &TCF2AggConfig.HostConfig.Purpose9,
		consentconstants.Purpose(10): &TCF2AggConfig.HostConfig.Purpose10,
	}
	return
}

type testDef struct {
	description           string
	bidder                openrtb_ext.BidderName
	consent               string
	allowBidRequest       bool
	passGeo               bool
	passID                bool
	weakVendorEnforcement bool
	bidderCoreName        openrtb_ext.BidderName
	aliasGVLIDs           map[string]uint16
}

func TestAllowActivitiesGeoAndID(t *testing.T) {
	vendorListData := MarshalVendorList(buildVendorList34())

	perms := permissionsImpl{
		hostVendorID:          2,
		nonStandardPublishers: map[string]struct{}{"appNexusAppID": {}},
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus:        2,
			openrtb_ext.BidderPubmatic:        6,
			openrtb_ext.BidderRubicon:         8,
			openrtb_ext.BidderOpenx:           20,
			openrtb_ext.BidderAudienceNetwork: 55,
		},
		fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
			2: {
				34: parseVendorListDataV2(t, vendorListData),
				74: parseVendorListDataV2(t, vendorListData),
			},
		}),
		gdprSignal: SignalYes,
	}

	// COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA : full consents to purposes and vendors 2, 6, 8 and special feature 1 opt-in
	testDefs := []testDef{
		{
			description:     "Appnexus vendor test, insufficient purposes claimed",
			bidder:          openrtb_ext.BidderAppnexus,
			bidderCoreName:  openrtb_ext.BidderAppnexus,
			consent:         "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBidRequest: false,
			passGeo:         false,
			passID:          false,
		},
		{
			description:     "Pubmatic Alias vendor test, insufficient purposes claimed",
			bidder:          "pubmatic1",
			bidderCoreName:  openrtb_ext.BidderPubmatic,
			consent:         "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBidRequest: false,
			passGeo:         false,
			passID:          false,
			aliasGVLIDs:     map[string]uint16{"pubmatic1": 1},
		},
		{
			description:           "Appnexus vendor test, insufficient purposes claimed, basic enforcement",
			bidder:                openrtb_ext.BidderAppnexus,
			bidderCoreName:        openrtb_ext.BidderAppnexus,
			consent:               "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBidRequest:       true,
			passGeo:               true,
			passID:                true,
			weakVendorEnforcement: true,
		},
		{
			description:           "Unknown vendor test, insufficient purposes claimed, basic enforcement",
			bidder:                openrtb_ext.BidderAudienceNetwork,
			bidderCoreName:        openrtb_ext.BidderAudienceNetwork,
			consent:               "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBidRequest:       true,
			passGeo:               true,
			passID:                true,
			weakVendorEnforcement: true,
		},
		{
			description:     "Pubmatic vendor test, flex purposes claimed",
			bidder:          openrtb_ext.BidderPubmatic,
			bidderCoreName:  openrtb_ext.BidderPubmatic,
			consent:         "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBidRequest: true,
			passGeo:         true,
			passID:          true,
		},
		{
			description:     "Pubmatic Alias vendor test, flex purposes claimed",
			bidder:          "pubmatic1",
			bidderCoreName:  openrtb_ext.BidderPubmatic,
			consent:         "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBidRequest: true,
			passGeo:         true,
			passID:          true,
			aliasGVLIDs:     map[string]uint16{"pubmatic1": 6},
		},
		{
			description:     "Rubicon vendor test, Specific purposes/LIs claimed, no geo claimed",
			bidder:          openrtb_ext.BidderRubicon,
			bidderCoreName:  openrtb_ext.BidderRubicon,
			consent:         "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBidRequest: true,
			passGeo:         false,
			passID:          true,
		},
		{
			// This requires publisher restrictions on any claimed purposes, 2-10. Vendor must declare all claimed purposes
			// as flex with legit interest as primary.
			// Using vendor 20 for this.
			description:     "OpenX vendor test, Specific purposes/LIs claimed, no geo claimed, Publisher restrictions apply",
			bidder:          openrtb_ext.BidderOpenx,
			bidderCoreName:  openrtb_ext.BidderOpenx,
			consent:         "CPAavcCPAavcCAGABCFRBKCsAP_AAH_AAAqIHFNf_X_fb3_j-_59_9t0eY1f9_7_v-0zjgeds-8Nyd_X_L8X5mM7vB36pq4KuR4Eu3LBAQdlHOHcTUmw6IkVqTPsbk2Mr7NKJ7PEinMbe2dYGH9_n9XT_ZKY79_____7__-_____7_f__-__3_vp9V---wOJAIMBAUAgAEMAAQIFCIQAAQhiQAAAABBCIBQJIAEqgAWVwEdoIEACAxAQgQAgBBQgwCAAQAAJKAgBACwQCAAiAQAAgAEAIAAEIAILACQEAAAEAJCAAiACECAgiAAg5DAgIgCCAFABAAAuJDACAMooASBAPGQGAAKAAqACGAEwALgAjgBlgDUAHZAPsA_ACMAFLAK2AbwBMQCbAFogLYAYEAw8BkQDOQGeAM-EQHwAVABWAC4AIYAZAAywBqADZAHYAPwAgABGAClgFPANYAdUA-QCGwEOgIvASIAmwBOwCkQFyAMCAYSAw8Bk4DOQGfCQAYADgBzgN_CQTgAEAALgAoACoAGQAOAAeABAACIAFQAMIAaABqADyAIYAigBMgCqAKwAWAAuABvADmAHoAQ0AiACJgEsAS4AmgBSgC3AGGAMgAZcA1ADVAGyAO8AewA-IB9gH6AQAAjABQQClgFPAL8AYoA1gBtADcAG8AOIAegA-QCGwEOgIqAReAkQBMQCZQE2AJ2AUOApEBYoC2AFyALvAYEAwYBhIDDQGHgMiAZIAycBlwDOQGfANIAadA1gDWQoAEAYQaBIACoAKwAXABDADIAGWANQAbIA7AB-AEAAIKARgApYBT4C0ALSAawA3gB1QD5AIbAQ6Ai8BIgCbAE7AKRAXIAwIBhIDDwGMAMnAZyAzwBnwcAEAA4Bv4qA2ABQAFQAQwAmABcAEcAMsAagA7AB-AEYAKXAWgBaQDeAJBATEAmwBTYC2AFyAMCAYeAyIBnIDPAGfANyHQWQAFwAUABUADIAHAAQAAiABdADAAMYAaABqADwAH0AQwBFACZAFUAVgAsABcADEAGYAN4AcwA9ACGAERAJYAmABNACjAFKALEAW4AwwBkADKAGiANQAbIA3wB3gD2gH2AfoBGACVAFBAKeAWKAtAC0gFzALyAX4AxQBuADiQHTAdQA9ACGwEOgIiAReAkEBIgCbAE7AKHAU0AqwBYsC2ALZAXAAuQBdoC7wGEgMNAYeAxIBjADHgGSAMnAZUAywBlwDOQGfANEgaQBpIDSwGnANYAbGPABAIqAb-QgZgALAAoABkAEQALgAYgBDACYAFUALgAYgAzABvAD0AI4AWIAygBqADfAHfAPsA_ACMAFBAKGAU-AtAC0gF-AMUAdQA9ACQQEiAJsAU0AsUBaMC2ALaAXAAuQBdoDDwGJAMiAZOAzkBngDPgGiANJAaWA4AlAyAAQAAsACgAGQAOAAigBgAGIAPAAiABMACqAFwAMQAZgA2gCGgEQARIAowBSgC3AGEAMoAaoA2QB3gD8AIwAU-AtAC0gGKANwAcQA6gCHQEXgJEATYAsUBbAC7QGHgMiAZOAywBnIDPAGfANIAawA4AmACARUA38pBBAAXABQAFQAMgAcABAACKAGAAYwA0ADUAHkAQwBFACYAFIAKoAWAAuABiADMAHMAQwAiABRgClAFiALcAZQA0QBqgDZAHfAPsA_ACMAFBAKGAVsAuYBeQDaAG4APQAh0BF4CRAE2AJ2AUOApoBWwCxQFsALgAXIAu0BhoDDwGMAMiAZIAycBlwDOQGeAM-gaQBpMDWANZAbGVABAA-Ab-A.YAAAAAAAAAAA",
			allowBidRequest: true,
			passGeo:         false,
			passID:          true,
		},
	}

	for _, td := range testDefs {

		tcf2AggConfig := allPurposesEnabledTCF2Config()
		if td.weakVendorEnforcement {
			tcf2AggConfig.AccountConfig.BasicEnforcementVendorsMap = map[string]struct{}{string(td.bidder): {}}
		}
		perms.cfg = &tcf2AggConfig
		perms.aliasGVLIDs = td.aliasGVLIDs
		perms.consent = td.consent
		perms.purposeEnforcerBuilder = NewPurposeEnforcerBuilder(&tcf2AggConfig)

		permissions := perms.AuctionActivitiesAllowed(context.Background(), td.bidderCoreName, td.bidder)
		assert.EqualValuesf(t, td.allowBidRequest, permissions.AllowBidRequest, "AllowBid failure on %s", td.description)
		assert.EqualValuesf(t, td.passGeo, permissions.PassGeo, "PassGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.passID, permissions.PassID, "PassID failure on %s", td.description)
	}
}

func TestAllowActivitiesWhitelist(t *testing.T) {
	// user specifies consent and LI for all purposes, and purpose and LI vendor consent for vendors 2, 6 and 8
	const fullConsentToPurposesAndVendorsTwoSixEight = "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA"

	vendorListData := MarshalVendorList(buildVendorList34())
	tcf2AggConfig := allPurposesEnabledTCF2Config()

	perms := permissionsImpl{
		cfg:                   &tcf2AggConfig,
		hostVendorID:          2,
		nonStandardPublishers: map[string]struct{}{"appNexusAppID": {}},
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
			2: {
				34: parseVendorListDataV2(t, vendorListData),
			},
		}),
		purposeEnforcerBuilder: NewPurposeEnforcerBuilder(&tcf2AggConfig),
		aliasGVLIDs:            map[string]uint16{},
		consent:                fullConsentToPurposesAndVendorsTwoSixEight,
		gdprSignal:             SignalYes,
		publisherID:            "appNexusAppID",
	}

	// Assert that an item that otherwise would not be allowed PI access, gets approved because it is found in the GDPR.NonStandardPublishers array
	permissions := perms.AuctionActivitiesAllowed(context.Background(), openrtb_ext.BidderAppnexus, openrtb_ext.BidderAppnexus)
	assert.EqualValuesf(t, true, permissions.PassGeo, "PassGeo failure")
	assert.EqualValuesf(t, true, permissions.PassID, "PassID failure")
}

func TestAllowActivitiesPubRestrict(t *testing.T) {
	vendorListData := MarshalVendorList(buildVendorList34())
	tcf2AggConfig := allPurposesEnabledTCF2Config()

	perms := permissionsImpl{
		cfg:          &tcf2AggConfig,
		hostVendorID: 2,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 32,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
			2: {
				15: parseVendorListDataV2(t, vendorListData),
			},
		}),
		purposeEnforcerBuilder: NewPurposeEnforcerBuilder(&tcf2AggConfig),
		gdprSignal:             SignalYes,
	}

	// COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA - vendors 1-10 legit interest only,
	// Pub restriction on purpose 7, consent only ... no allowPI will pass, no special feature 1 consent
	testDefs := []testDef{
		{
			description:    "Appnexus vendor test, insufficient purposes claimed",
			bidder:         openrtb_ext.BidderAppnexus,
			bidderCoreName: openrtb_ext.BidderAppnexus,
			consent:        "COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA",
			passGeo:        false,
			passID:         false,
			aliasGVLIDs:    map[string]uint16{},
		},
		{
			description:    "Pubmatic vendor test, flex purposes claimed",
			bidder:         openrtb_ext.BidderPubmatic,
			bidderCoreName: openrtb_ext.BidderPubmatic,
			consent:        "COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA",
			passGeo:        false,
			passID:         false,
			aliasGVLIDs:    map[string]uint16{},
		},
		{
			description:    "Pubmatic Alias vendor test, flex purposes claimed",
			bidder:         "pubmatic1",
			bidderCoreName: openrtb_ext.BidderPubmatic,
			consent:        "COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA",
			passGeo:        false,
			passID:         false,
			aliasGVLIDs:    map[string]uint16{"pubmatic1": 32},
		},
		{
			description:    "Rubicon vendor test, Specific purposes/LIs claimed, no geo claimed",
			bidder:         openrtb_ext.BidderRubicon,
			bidderCoreName: openrtb_ext.BidderRubicon,
			consent:        "COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA",
			passGeo:        false,
			passID:         true,
			aliasGVLIDs:    map[string]uint16{},
		},
	}

	for _, td := range testDefs {
		perms.aliasGVLIDs = td.aliasGVLIDs
		perms.consent = td.consent

		permissions := perms.AuctionActivitiesAllowed(context.Background(), td.bidderCoreName, td.bidder)
		assert.EqualValuesf(t, td.passGeo, permissions.PassGeo, "PassGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.passID, permissions.PassID, "PassID failure on %s", td.description)
	}
}

func TestAllowSync(t *testing.T) {
	const fullConsentToPurposesAndVendorsTwoSixEight = "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA"

	vendorListData := MarshalVendorList(buildVendorList34())
	tcf2AggConfig := allPurposesEnabledTCF2Config()

	perms := permissionsImpl{
		cfg:          &tcf2AggConfig,
		hostVendorID: 2,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
			2: {
				34: parseVendorListDataV2(t, vendorListData),
			},
		}),
		purposeEnforcerBuilder: NewPurposeEnforcerBuilder(&tcf2AggConfig),
		gdprSignal:             SignalYes,
		consent:                fullConsentToPurposesAndVendorsTwoSixEight,
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background())
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, true, allowSync, "HostCookiesAllowed failure")

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderRubicon)
	assert.NoErrorf(t, err, "Error processing BidderSyncAllowed")
	assert.EqualValuesf(t, true, allowSync, "BidderSyncAllowed failure")
}

func TestProhibitedPurposeSync(t *testing.T) {
	const fullConsentToPurposesAndVendorsTwoSixEight = "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA"

	vendorList34 := buildVendorList34()
	vendorList34.Vendors["8"].Purposes = []int{7}
	vendorListData := MarshalVendorList(vendorList34)

	tcf2AggConfig := allPurposesEnabledTCF2Config()

	perms := permissionsImpl{
		cfg:          &tcf2AggConfig,
		hostVendorID: 8,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
			2: {
				34: parseVendorListDataV2(t, vendorListData),
			},
		}),
		purposeEnforcerBuilder: NewPurposeEnforcerBuilder(&tcf2AggConfig),
		gdprSignal:             SignalYes,
		consent:                fullConsentToPurposesAndVendorsTwoSixEight,
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background())
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, false, allowSync, "HostCookiesAllowed failure")

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderRubicon)
	assert.NoErrorf(t, err, "Error processing BidderSyncAllowed")
	assert.EqualValuesf(t, false, allowSync, "BidderSyncAllowed failure")
}

func TestProhibitedVendorSync(t *testing.T) {
	const fullConsentToPurposesAndVendorsTwoSixEight = "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA"

	vendorListData := MarshalVendorList(buildVendorList34())
	tcf2AggConfig := allPurposesEnabledTCF2Config()

	perms := permissionsImpl{
		cfg:          &tcf2AggConfig,
		hostVendorID: 10,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
			openrtb_ext.BidderOpenx:    10,
		},
		fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
			2: {
				34: parseVendorListDataV2(t, vendorListData),
			},
		}),
		purposeEnforcerBuilder: NewPurposeEnforcerBuilder(&tcf2AggConfig),
		gdprSignal:             SignalYes,
		consent:                fullConsentToPurposesAndVendorsTwoSixEight,
	}

	// COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA : full consents to purposes for vendors 2, 6, 8
	allowSync, err := perms.HostCookiesAllowed(context.Background())
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, false, allowSync, "HostCookiesAllowed failure")

	// Permission disallowed due to consent string not including vendor 10.
	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderOpenx)
	assert.NoErrorf(t, err, "Error processing BidderSyncAllowed")
	assert.EqualValuesf(t, false, allowSync, "BidderSyncAllowed failure")
}

func parseVendorListDataV2(t *testing.T, data string) vendorlist.VendorList {
	t.Helper()
	parsed, err := vendorlist2.ParseEagerly([]byte(data))
	if err != nil {
		t.Fatalf("Failed to parse vendor list data. %v", err)
	}
	return parsed
}

func listFetcher(specVersionLists map[uint16]map[uint16]vendorlist.VendorList) func(context.Context, uint16, uint16) (vendorlist.VendorList, error) {
	return func(ctx context.Context, specVersion, listVersion uint16) (vendorlist.VendorList, error) {
		if lists, ok := specVersionLists[specVersion]; ok {
			if data, ok := lists[listVersion]; ok {
				return data, nil
			}
		}
		return nil, fmt.Errorf("spec version %d vendor list %d not found", specVersion, listVersion)
	}
}

func failedListFetcher(ctx context.Context, specVersion, listVersion uint16) (vendorlist.VendorList, error) {
	return nil, errors.New("vendor list can't be fetched")
}

func assertNilErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func assertErr(t *testing.T, err error, badConsent bool) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected error did not occur.")
		return
	}
	_, isBadConsent := err.(*ErrorMalformedConsent)
	assertBoolsEqual(t, badConsent, isBadConsent)
}

func assertBoolsEqual(t *testing.T, expected bool, actual bool) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %t, got %t", expected, actual)
	}
}

func TestAllowActivitiesBidRequests(t *testing.T) {
	purpose2AndVendorConsent := "CPF_61ePF_61eFxAAAENAiCAAEAAAAAAAAAAADAQAAAAAA"
	purpose2ConsentWithoutVendorConsent := "CPF_61ePF_61eFxAAAENAiCAAEAAAAAAAAAAABIAAAAA"

	purpose2AndVendorLI := "CPF_61ePF_61eFxAAAENAiCAAAAAAEAAAAAAAAAAIAIAAA"
	purpose2LIWithoutVendorLI := "CPF_61ePF_61eFxAAAENAiCAAAAAAEAAAAAAABIAAAAA"

	testDefs := []struct {
		description            string
		purpose2EnforcePurpose bool
		purpose2EnforceVendors bool
		bidder                 openrtb_ext.BidderName
		bidderCoreName         openrtb_ext.BidderName
		consent                string
		allowBidRequest        bool
		passGeo                bool
		passID                 bool
		aliasGVLIDs            map[string]uint16
	}{
		{
			description:            "Bid blocked - p2 enabled, user consents to p2 but not vendor, vendor consents to p2",
			purpose2EnforcePurpose: true,
			purpose2EnforceVendors: true,
			bidder:                 openrtb_ext.BidderPubmatic,
			bidderCoreName:         openrtb_ext.BidderPubmatic,
			consent:                purpose2ConsentWithoutVendorConsent,
			allowBidRequest:        false,
			passGeo:                false,
			passID:                 false,
		},
		{
			description:            "Bid allowed - p2 enabled, user consents to p2 and vendor, alias vendor consents to p2",
			purpose2EnforcePurpose: true,
			purpose2EnforceVendors: true,
			bidder:                 "pubmatic1",
			bidderCoreName:         openrtb_ext.BidderPubmatic,
			consent:                purpose2AndVendorConsent,
			allowBidRequest:        true,
			passGeo:                false,
			passID:                 true,
			aliasGVLIDs:            map[string]uint16{"pubmatic1": 6},
		},
		{
			description:            "Bid blocked - p2 enabled, user consents to p2 and vendor, alias vendor does not consent to p2",
			purpose2EnforcePurpose: true,
			purpose2EnforceVendors: true,
			bidder:                 "pubmatic1",
			bidderCoreName:         openrtb_ext.BidderPubmatic,
			consent:                purpose2AndVendorConsent,
			allowBidRequest:        false,
			passGeo:                false,
			passID:                 false,
			aliasGVLIDs:            map[string]uint16{"pubmatic1": 1},
		},
		{
			description:            "Bid allowed - p2 enabled not enforcing vendors, user consents to p2 but not vendor, vendor consents to p2",
			purpose2EnforcePurpose: true,
			purpose2EnforceVendors: false,
			bidder:                 openrtb_ext.BidderPubmatic,
			bidderCoreName:         openrtb_ext.BidderPubmatic,
			consent:                purpose2ConsentWithoutVendorConsent,
			allowBidRequest:        true,
			passGeo:                false,
			passID:                 false,
		},
		{
			description:            "Bid allowed - p2 disabled and enforcing vendors, user consents to p2 but not vendor, vendor consents to p2",
			purpose2EnforcePurpose: false,
			purpose2EnforceVendors: true,
			bidder:                 openrtb_ext.BidderPubmatic,
			bidderCoreName:         openrtb_ext.BidderPubmatic,
			consent:                purpose2ConsentWithoutVendorConsent,
			allowBidRequest:        false,
			passGeo:                false,
			passID:                 false,
		},
		{
			description:            "Bid allowed - p2 disabled not enforcing vendors, user consents to p2 but not vendor, vendor consents to p2",
			purpose2EnforcePurpose: false,
			purpose2EnforceVendors: false,
			bidder:                 openrtb_ext.BidderPubmatic,
			bidderCoreName:         openrtb_ext.BidderPubmatic,
			consent:                purpose2ConsentWithoutVendorConsent,
			allowBidRequest:        true,
			passGeo:                false,
			passID:                 false,
		},
		{
			description:            "Bid allowed - p2 disabled and enforcing vendors, user consents to p2 and vendor, vendor consents to p2",
			purpose2EnforcePurpose: false,
			purpose2EnforceVendors: true,
			bidder:                 openrtb_ext.BidderPubmatic,
			bidderCoreName:         openrtb_ext.BidderPubmatic,
			consent:                purpose2AndVendorConsent,
			allowBidRequest:        true,
			passGeo:                false,
			passID:                 true,
		},
		{
			description:            "Bid allowed - p2 enabled, user consents to p2 and vendor, vendor consents to p2",
			purpose2EnforcePurpose: true,
			purpose2EnforceVendors: true,
			bidder:                 openrtb_ext.BidderPubmatic,
			bidderCoreName:         openrtb_ext.BidderPubmatic,
			consent:                purpose2AndVendorConsent,
			allowBidRequest:        true,
			passGeo:                false,
			passID:                 true,
		},
		{
			description:            "Bid blocked - p2 enabled, user consents to p2 LI but not vendor, vendor consents to p2",
			purpose2EnforcePurpose: true,
			purpose2EnforceVendors: true,
			bidder:                 openrtb_ext.BidderRubicon,
			bidderCoreName:         openrtb_ext.BidderRubicon,
			consent:                purpose2LIWithoutVendorLI,
			allowBidRequest:        false,
			passGeo:                false,
			passID:                 false,
		},
		{
			description:            "Bid allowed - p2 enabled, user consents to p2 LI and vendor, vendor consents to p2",
			purpose2EnforcePurpose: true,
			purpose2EnforceVendors: true,
			bidder:                 openrtb_ext.BidderRubicon,
			bidderCoreName:         openrtb_ext.BidderRubicon,
			consent:                purpose2AndVendorLI,
			allowBidRequest:        true,
			passGeo:                false,
			passID:                 true,
		},
		{
			description:            "Bid allowed - p2 enabled not enforcing vendors, user consents to p2 LI but not vendor, vendor consents to p2",
			purpose2EnforcePurpose: true,
			purpose2EnforceVendors: false,
			bidder:                 openrtb_ext.BidderPubmatic,
			bidderCoreName:         openrtb_ext.BidderPubmatic,
			consent:                purpose2AndVendorLI,
			allowBidRequest:        true,
			passGeo:                false,
			passID:                 false,
		},
	}

	for _, td := range testDefs {
		vendorListData := MarshalVendorList(buildVendorList34())

		perms := permissionsImpl{
			hostVendorID: 2,
			vendorIDs: map[openrtb_ext.BidderName]uint16{
				openrtb_ext.BidderPubmatic: 6,
				openrtb_ext.BidderRubicon:  8,
			},
			fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
				2: {
					34: parseVendorListDataV2(t, vendorListData),
				},
			}),
			aliasGVLIDs: td.aliasGVLIDs,
			consent:     td.consent,
			gdprSignal:  SignalYes,
		}

		tcf2AggConfig := allPurposesEnabledTCF2Config()
		tcf2AggConfig.HostConfig.Purpose2.EnforcePurpose = td.purpose2EnforcePurpose
		tcf2AggConfig.HostConfig.Purpose2.EnforceVendors = td.purpose2EnforceVendors
		p2Config := tcf2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(2)]
		p2Config.EnforcePurpose = td.purpose2EnforcePurpose
		p2Config.EnforceVendors = td.purpose2EnforceVendors
		tcf2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(2)] = p2Config
		tcf2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(2)] = &tcf2AggConfig.HostConfig.Purpose2
		perms.cfg = &tcf2AggConfig
		perms.purposeEnforcerBuilder = NewPurposeEnforcerBuilder(&tcf2AggConfig)

		permissions := perms.AuctionActivitiesAllowed(context.Background(), td.bidderCoreName, td.bidder)
		assert.EqualValuesf(t, td.allowBidRequest, permissions.AllowBidRequest, "AllowBid failure on %s", td.description)
		assert.EqualValuesf(t, td.passGeo, permissions.PassGeo, "PassGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.passID, permissions.PassID, "PassID failure on %s", td.description)
	}
}

func TestAllowActivitiesVendorException(t *testing.T) {
	appnexus := string(openrtb_ext.BidderAppnexus)
	noPurposeOrVendorConsentAndPubRestrictsP2 := "CPF_61ePF_61eFxAAAENAiCAAAAAAAAAAAAAACEAAAACEAAgAgAA"
	noPurposeOrVendorConsentAndPubRestrictsNone := "CPF_61ePF_61eFxAAAENAiCAAAAAAAAAAAAAACEAAAAA"

	testDefs := []struct {
		description           string
		p2VendorExceptionMap  map[string]struct{}
		sf1VendorExceptionMap map[openrtb_ext.BidderName]struct{}
		bidder                openrtb_ext.BidderName
		consent               string
		allowBidRequest       bool
		passGeo               bool
		passID                bool
		bidderCoreName        openrtb_ext.BidderName
	}{
		{
			description:          "Bid/ID blocked by publisher - p2 enabled with p2 vendor exception, pub restricts p2 for vendor",
			p2VendorExceptionMap: map[string]struct{}{appnexus: {}},
			bidder:               openrtb_ext.BidderAppnexus,
			bidderCoreName:       openrtb_ext.BidderAppnexus,
			consent:              noPurposeOrVendorConsentAndPubRestrictsP2,
			allowBidRequest:      false,
			passGeo:              false,
			passID:               false,
		},
		{
			description:           "Bid/ID allowed by vendor exception - p2 enabled with p2 vendor exception, pub restricts none",
			p2VendorExceptionMap:  map[string]struct{}{appnexus: {}},
			sf1VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
			bidder:                openrtb_ext.BidderAppnexus,
			bidderCoreName:        openrtb_ext.BidderAppnexus,
			consent:               noPurposeOrVendorConsentAndPubRestrictsNone,
			allowBidRequest:       true,
			passGeo:               false,
			passID:                true,
		},
		{
			description:           "Geo blocked - sf1 enabled but no consent",
			p2VendorExceptionMap:  map[string]struct{}{},
			sf1VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
			bidder:                openrtb_ext.BidderAppnexus,
			bidderCoreName:        openrtb_ext.BidderAppnexus,
			consent:               noPurposeOrVendorConsentAndPubRestrictsNone,
			allowBidRequest:       false,
			passGeo:               false,
			passID:                false,
		},
		{
			description:           "Geo allowed by vendor exception - sf1 enabled with sf1 vendor exception",
			p2VendorExceptionMap:  map[string]struct{}{},
			sf1VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderAppnexus: {}},
			bidder:                openrtb_ext.BidderAppnexus,
			bidderCoreName:        openrtb_ext.BidderAppnexus,
			consent:               noPurposeOrVendorConsentAndPubRestrictsNone,
			allowBidRequest:       false,
			passGeo:               true,
			passID:                false,
		},
	}

	for _, td := range testDefs {
		vendorListData := MarshalVendorList(buildVendorList34())
		perms := permissionsImpl{
			hostVendorID: 2,
			vendorIDs: map[openrtb_ext.BidderName]uint16{
				openrtb_ext.BidderAppnexus: 32,
			},
			fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
				2: {
					34: parseVendorListDataV2(t, vendorListData),
				},
			}),
			aliasGVLIDs: map[string]uint16{},
			consent:     td.consent,
			gdprSignal:  SignalYes,
		}

		tcf2AggConfig := allPurposesEnabledTCF2Config()
		tcf2AggConfig.HostConfig.Purpose2.VendorExceptionMap = td.p2VendorExceptionMap
		tcf2AggConfig.HostConfig.SpecialFeature1.Enforce = true
		tcf2AggConfig.HostConfig.SpecialFeature1.VendorExceptionMap = td.sf1VendorExceptionMap
		tcf2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(2)] = &tcf2AggConfig.HostConfig.Purpose2
		tcf2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(3)] = &tcf2AggConfig.HostConfig.Purpose3
		perms.cfg = &tcf2AggConfig
		perms.purposeEnforcerBuilder = NewPurposeEnforcerBuilder(&tcf2AggConfig)

		permissions := perms.AuctionActivitiesAllowed(context.Background(), td.bidderCoreName, td.bidder)
		assert.EqualValuesf(t, td.allowBidRequest, permissions.AllowBidRequest, "AllowBid failure on %s", td.description)
		assert.EqualValuesf(t, td.passGeo, permissions.PassGeo, "PassGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.passID, permissions.PassID, "PassID failure on %s", td.description)
	}
}

func TestBidderSyncAllowedVendorException(t *testing.T) {
	appnexus := string(openrtb_ext.BidderAppnexus)
	noPurposeOrVendorConsentAndPubRestrictsP1 := "CPF_61ePF_61eFxAAAENAiCAAAAAAAAAAAAAAQAAAAAAAAAAIIACACA"
	noPurposeOrVendorConsentAndPubRestrictsNone := "CPF_61ePF_61eFxAAAENAiCAAAAAAAAAAAAAACEAAAAA"

	testDefs := []struct {
		description          string
		p1VendorExceptionMap map[string]struct{}
		bidder               openrtb_ext.BidderName
		consent              string
		allowSync            bool
	}{
		{
			description:          "Sync blocked by no consent - p1 enabled, no p1 vendor exception, pub restricts none",
			p1VendorExceptionMap: map[string]struct{}{},
			bidder:               openrtb_ext.BidderAppnexus,
			consent:              noPurposeOrVendorConsentAndPubRestrictsNone,
			allowSync:            false,
		},
		{
			description:          "Sync blocked by publisher - p1 enabled with p1 vendor exception, pub restricts p1 for vendor",
			p1VendorExceptionMap: map[string]struct{}{appnexus: {}},
			bidder:               openrtb_ext.BidderAppnexus,
			consent:              noPurposeOrVendorConsentAndPubRestrictsP1,
			allowSync:            false,
		},
		{
			description:          "Sync allowed by vendor exception - p1 enabled with p1 vendor exception, pub restricts none",
			p1VendorExceptionMap: map[string]struct{}{appnexus: {}},
			bidder:               openrtb_ext.BidderAppnexus,
			consent:              noPurposeOrVendorConsentAndPubRestrictsNone,
			allowSync:            true,
		},
	}

	for _, td := range testDefs {
		vendorListData := MarshalVendorList(buildVendorList34())
		perms := permissionsImpl{
			hostVendorID: 2,
			vendorIDs: map[openrtb_ext.BidderName]uint16{
				openrtb_ext.BidderAppnexus: 32,
			},
			fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
				2: {
					34: parseVendorListDataV2(t, vendorListData),
				},
			}),
			consent:    td.consent,
			gdprSignal: SignalYes,
		}

		tcf2AggConfig := allPurposesEnabledTCF2Config()
		tcf2AggConfig.HostConfig.Purpose1.VendorExceptionMap = td.p1VendorExceptionMap
		tcf2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(1)] = &tcf2AggConfig.HostConfig.Purpose1
		perms.cfg = &tcf2AggConfig
		perms.purposeEnforcerBuilder = NewPurposeEnforcerBuilder(&tcf2AggConfig)

		allowSync, err := perms.BidderSyncAllowed(context.Background(), td.bidder)
		assert.NoErrorf(t, err, "Error processing BidderSyncAllowed for %s", td.description)
		assert.EqualValuesf(t, td.allowSync, allowSync, "AllowSync failure on %s", td.description)
	}
}

func TestDefaultPermissions(t *testing.T) {
	tests := []struct {
		description      string
		purpose2Enforced bool
		feature1Enforced bool
		wantPermissions  AuctionPermissions
	}{
		{
			description: "Neither enforced",
			wantPermissions: AuctionPermissions{
				AllowBidRequest: true,
				PassGeo:         true,
				PassID:          false,
			},
		},
		{
			description:      "Purpose 2 enforced only",
			purpose2Enforced: true,
			wantPermissions: AuctionPermissions{
				AllowBidRequest: false,
				PassGeo:         true,
				PassID:          false,
			},
		},
		{
			description:      "Feature 1 enforced only",
			feature1Enforced: true,
			wantPermissions: AuctionPermissions{
				AllowBidRequest: true,
				PassGeo:         false,
				PassID:          false,
			},
		},
		{
			description:      "Both enforced",
			purpose2Enforced: true,
			feature1Enforced: true,
			wantPermissions: AuctionPermissions{
				AllowBidRequest: false,
				PassGeo:         false,
				PassID:          false,
			},
		},
	}

	for _, tt := range tests {
		perms := permissionsImpl{}

		tcf2AggConfig := allPurposesEnabledTCF2Config()
		tcf2AggConfig.HostConfig.Purpose2.EnforcePurpose = tt.purpose2Enforced
		tcf2AggConfig.HostConfig.SpecialFeature1.Enforce = tt.feature1Enforced
		tcf2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(2)] = &tcf2AggConfig.HostConfig.Purpose2
		perms.cfg = &tcf2AggConfig

		result := perms.defaultPermissions()

		assert.Equal(t, result, tt.wantPermissions, tt.description)
	}
}

func TestVendorListSelection(t *testing.T) {
	policyVersion3WithVendor2AndPurpose1Consent := "CPGWbY_PGWbY_GYAAAENABDAAIAAAAAAAAAAACEAAAAA"
	policyVersion4WithVendor2AndPurpose1Consent := "CPGWbY_PGWbY_GYAAAENABEAAIAAAAAAAAAAACEAAAAA"

	specVersion2vendorListData := MarshalVendorList(vendorList{
		GVLSpecificationVersion: 2,
		VendorListVersion:       2,
		Vendors: map[string]*vendor{
			"2": {
				ID:       2,
				Purposes: []int{},
			},
		},
	})
	specVersion3vendorListData := MarshalVendorList(vendorList{
		GVLSpecificationVersion: 3,
		VendorListVersion:       2,
		Vendors: map[string]*vendor{
			"2": {
				ID:       2,
				Purposes: []int{1},
			},
		},
	})

	tcf2AggConfig := tcf2Config{
		HostConfig: config.TCF2{
			Purpose1: config.TCF2Purpose{
				EnforcePurpose: true,
				EnforceVendors: true,
			},
		},
	}
	tcf2AggConfig.HostConfig.PurposeConfigs = map[consentconstants.Purpose]*config.TCF2Purpose{
		consentconstants.Purpose(1): &tcf2AggConfig.HostConfig.Purpose1,
	}

	perms := permissionsImpl{
		cfg:          &tcf2AggConfig,
		hostVendorID: 2,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
		},
		fetchVendorList: listFetcher(map[uint16]map[uint16]vendorlist.VendorList{
			2: {
				1: parseVendorListDataV2(t, specVersion2vendorListData),
			},
			3: {
				1: parseVendorListDataV2(t, specVersion3vendorListData),
			},
		}),
		purposeEnforcerBuilder: NewPurposeEnforcerBuilder(&tcf2AggConfig),
		gdprSignal:             SignalYes,
	}

	tests := []struct {
		name              string
		consent           string
		expectedAllowSync bool
		expectedErr       bool
	}{
		{
			name:              "consent_tcf_policy_version_3_uses_gvl_spec_version_2",
			consent:           policyVersion3WithVendor2AndPurpose1Consent,
			expectedAllowSync: false,
		},
		{
			name:              "consent_tcf_policy_version_4_uses_gvl_spec_version_3",
			consent:           policyVersion4WithVendor2AndPurpose1Consent,
			expectedAllowSync: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms.consent = tt.consent
			allowSync, err := perms.HostCookiesAllowed(context.Background())
			assert.Equal(t, tt.expectedAllowSync, allowSync)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
