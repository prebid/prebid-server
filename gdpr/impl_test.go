package gdpr

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/go-gdpr/vendorlist2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestDisallowOnEmptyConsent(t *testing.T) {
	perms := permissionsImpl{
		gdprDefaultValue: "0",
		hostVendorID:     3,
		vendorIDs:        nil,
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: failedListFetcher,
		},
	}

	cfg := tcf2Config{}

	allowSync, err := perms.BidderSyncAllowed(context.Background(), &cfg, openrtb_ext.BidderAppnexus, SignalYes, "")
	assertBoolsEqual(t, false, allowSync)
	assertNilErr(t, err)
	allowSync, err = perms.HostCookiesAllowed(context.Background(), &cfg, SignalYes, "")
	assertBoolsEqual(t, false, allowSync)
	assertNilErr(t, err)
}

func TestAllowOnSignalNo(t *testing.T) {
	perms := permissionsImpl{}
	emptyConsent := ""
	cfg := tcf2Config{}

	allowSync, err := perms.HostCookiesAllowed(context.Background(), &cfg, SignalNo, emptyConsent)
	assert.Equal(t, true, allowSync)
	assert.Nil(t, err)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), &cfg, openrtb_ext.BidderAppnexus, SignalNo, emptyConsent)
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

	perms := permissionsImpl{
		hostVendorID: 2,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	TCF2AggConfig := TCF2Config{
		HostConfig: config.TCF2{
			Purpose1: config.TCF2Purpose{
				EnforcePurpose: config.TCF2FullEnforcement,
				EnforceVendors: true,
			},
		},
	}
	TCF2AggConfig.HostConfig.PurposeConfigs = map[consentconstants.Purpose]*config.TCF2Purpose{
		consentconstants.Purpose(1): &TCF2AggConfig.HostConfig.Purpose1,
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background(), &TCF2AggConfig, SignalYes, vendor2AndPurpose1Consent)
	assertNilErr(t, err)
	assertBoolsEqual(t, true, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), &TCF2AggConfig, openrtb_ext.BidderAppnexus, SignalYes, vendor2AndPurpose1Consent)
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

	perms := permissionsImpl{
		hostVendorID: 2,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	TCF2AggConfig := TCF2Config{
		HostConfig: config.TCF2{
			Purpose1: config.TCF2Purpose{
				EnforcePurpose: config.TCF2FullEnforcement,
			},
		},
	}
	TCF2AggConfig.HostConfig.PurposeConfigs = map[consentconstants.Purpose]*config.TCF2Purpose{
		consentconstants.Purpose(1): &TCF2AggConfig.HostConfig.Purpose1,
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background(), &TCF2AggConfig, SignalYes, vendor2NoPurpose1Consent)
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), &TCF2AggConfig, openrtb_ext.BidderAppnexus, SignalYes, vendor2NoPurpose1Consent)
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
	perms := permissionsImpl{
		hostVendorID: 2,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	TCF2AggConfig := TCF2Config{
		HostConfig: config.TCF2{
			Purpose1: config.TCF2Purpose{
				EnforcePurpose: config.TCF2FullEnforcement,
				EnforceVendors: true,
			},
		},
	}
	TCF2AggConfig.HostConfig.PurposeConfigs = map[consentconstants.Purpose]*config.TCF2Purpose{
		consentconstants.Purpose(1): &TCF2AggConfig.HostConfig.Purpose1,
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background(), &TCF2AggConfig, SignalYes, purpose1NoVendorConsent)
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), &TCF2AggConfig, openrtb_ext.BidderPubmatic, SignalYes, purpose1NoVendorConsent)
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)
}

func TestMalformedConsent(t *testing.T) {
	perms := permissionsImpl{
		hostVendorID: 2,
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: listFetcher(nil),
		},
	}

	sync, err := perms.HostCookiesAllowed(context.Background(), &TCF2Config{}, SignalYes, "BON")
	assertErr(t, err, true)
	assertBoolsEqual(t, false, sync)
}

func TestAllowActivities(t *testing.T) {
	bidderAllowedByConsent := openrtb_ext.BidderAppnexus
	bidderBlockedByConsent := openrtb_ext.BidderRubicon
	vendor2AndPurpose2Consent := "CPGWbY_PGWbY_GYAAAENABCAAEAAAAAAAAAAACEAAAAA"

	tests := []struct {
		description           string
		bidderName            openrtb_ext.BidderName
		publisherID           string
		gdprDefaultValue      string
		gdpr                  Signal
		consent               string
		passID                bool
		weakVendorEnforcement bool
	}{
		{
			description:      "Allow PI - Non standard publisher",
			bidderName:       bidderBlockedByConsent,
			publisherID:      "appNexusAppID",
			gdprDefaultValue: "1",
			gdpr:             SignalYes,
			consent:          vendor2AndPurpose2Consent,
			passID:           true,
		},
		{
			description:      "Allow PI - known vendor with No GDPR",
			bidderName:       bidderBlockedByConsent,
			gdprDefaultValue: "1",
			gdpr:             SignalNo,
			consent:          vendor2AndPurpose2Consent,
			passID:           true,
		},
		{
			description:      "Allow PI - known vendor with Yes GDPR",
			bidderName:       bidderAllowedByConsent,
			gdprDefaultValue: "1",
			gdpr:             SignalYes,
			consent:          vendor2AndPurpose2Consent,
			passID:           true,
		},
		{
			description:      "PI allowed according to host setting gdprDefaultValue 0 - known vendor with ambiguous GDPR and empty consent",
			bidderName:       bidderAllowedByConsent,
			gdprDefaultValue: "0",
			gdpr:             SignalAmbiguous,
			consent:          "",
			passID:           true,
		},
		{
			description:      "PI allowed according to host setting gdprDefaultValue 0 - known vendor with ambiguous GDPR and non-empty consent",
			bidderName:       bidderAllowedByConsent,
			gdprDefaultValue: "0",
			gdpr:             SignalAmbiguous,
			consent:          vendor2AndPurpose2Consent,
			passID:           true,
		},
		{
			description:      "PI allowed according to host setting gdprDefaultValue 1 - known vendor with ambiguous GDPR and empty consent",
			bidderName:       bidderAllowedByConsent,
			gdprDefaultValue: "1",
			gdpr:             SignalAmbiguous,
			consent:          "",
			passID:           false,
		},
		{
			description:      "PI allowed according to host setting gdprDefaultValue 1 - known vendor with ambiguous GDPR and non-empty consent",
			bidderName:       bidderAllowedByConsent,
			gdprDefaultValue: "1",
			gdpr:             SignalAmbiguous,
			consent:          vendor2AndPurpose2Consent,
			passID:           true,
		},
		{
			description:      "Don't allow PI - known vendor with Yes GDPR and empty consent",
			bidderName:       bidderAllowedByConsent,
			gdprDefaultValue: "1",
			gdpr:             SignalYes,
			consent:          "",
			passID:           false,
		},
		{
			description:      "Don't allow PI - default vendor with Yes GDPR and non-empty consent",
			bidderName:       bidderBlockedByConsent,
			gdprDefaultValue: "1",
			gdpr:             SignalYes,
			consent:          vendor2AndPurpose2Consent,
			passID:           false,
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

	perms := permissionsImpl{
		hostVendorID:          2,
		nonStandardPublishers: map[string]struct{}{"appNexusAppID": {}},
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	TCF2AggConfig := allPurposesEnabledTCF2Config()

	for _, tt := range tests {
		perms.gdprDefaultValue = tt.gdprDefaultValue

		_, _, passID, err := perms.AuctionActivitiesAllowed(context.Background(), &TCF2AggConfig, tt.bidderName, tt.publisherID, tt.gdpr, tt.consent)

		assert.Nil(t, err, tt.description)
		assert.Equal(t, tt.passID, passID, tt.description)
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

func allPurposesEnabledTCF2Config() (TCF2AggConfig TCF2Config) {
	TCF2AggConfig = TCF2Config{
		HostConfig: config.TCF2{
			Enabled:         true,
			Purpose1:        config.TCF2Purpose{EnforcePurpose: config.TCF2FullEnforcement, EnforceVendors: true},
			Purpose2:        config.TCF2Purpose{EnforcePurpose: config.TCF2FullEnforcement, EnforceVendors: true},
			Purpose3:        config.TCF2Purpose{EnforcePurpose: config.TCF2FullEnforcement, EnforceVendors: true},
			Purpose4:        config.TCF2Purpose{EnforcePurpose: config.TCF2FullEnforcement, EnforceVendors: true},
			Purpose5:        config.TCF2Purpose{EnforcePurpose: config.TCF2FullEnforcement, EnforceVendors: true},
			Purpose6:        config.TCF2Purpose{EnforcePurpose: config.TCF2FullEnforcement, EnforceVendors: true},
			Purpose7:        config.TCF2Purpose{EnforcePurpose: config.TCF2FullEnforcement, EnforceVendors: true},
			Purpose8:        config.TCF2Purpose{EnforcePurpose: config.TCF2FullEnforcement, EnforceVendors: true},
			Purpose9:        config.TCF2Purpose{EnforcePurpose: config.TCF2FullEnforcement, EnforceVendors: true},
			Purpose10:       config.TCF2Purpose{EnforcePurpose: config.TCF2FullEnforcement, EnforceVendors: true},
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
	allowBid              bool
	passGeo               bool
	passID                bool
	weakVendorEnforcement bool
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
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
				74: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	TCF2AggConfig := allPurposesEnabledTCF2Config()

	// COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA : full consents to purposes and vendors 2, 6, 8 and special feature 1 opt-in
	testDefs := []testDef{
		{
			description: "Appnexus vendor test, insufficient purposes claimed",
			bidder:      openrtb_ext.BidderAppnexus,
			consent:     "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBid:    false,
			passGeo:     false,
			passID:      false,
		},
		{
			description:           "Appnexus vendor test, insufficient purposes claimed, basic enforcement",
			bidder:                openrtb_ext.BidderAppnexus,
			consent:               "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBid:              true,
			passGeo:               true,
			passID:                true,
			weakVendorEnforcement: true,
		},
		{
			description:           "Unknown vendor test, insufficient purposes claimed, basic enforcement",
			bidder:                openrtb_ext.BidderAudienceNetwork,
			consent:               "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBid:              true,
			passGeo:               true,
			passID:                true,
			weakVendorEnforcement: true,
		},
		{
			description: "Pubmatic vendor test, flex purposes claimed",
			bidder:      openrtb_ext.BidderPubmatic,
			consent:     "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBid:    true,
			passGeo:     true,
			passID:      true,
		},
		{
			description: "Rubicon vendor test, Specific purposes/LIs claimed, no geo claimed",
			bidder:      openrtb_ext.BidderRubicon,
			consent:     "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowBid:    true,
			passGeo:     false,
			passID:      true,
		},
		{
			// This requires publisher restrictions on any claimed purposes, 2-10. Vendor must declare all claimed purposes
			// as flex with legit interest as primary.
			// Using vendor 20 for this.
			description: "OpenX vendor test, Specific purposes/LIs claimed, no geo claimed, Publisher restrictions apply",
			bidder:      openrtb_ext.BidderOpenx,
			consent:     "CPAavcCPAavcCAGABCFRBKCsAP_AAH_AAAqIHFNf_X_fb3_j-_59_9t0eY1f9_7_v-0zjgeds-8Nyd_X_L8X5mM7vB36pq4KuR4Eu3LBAQdlHOHcTUmw6IkVqTPsbk2Mr7NKJ7PEinMbe2dYGH9_n9XT_ZKY79_____7__-_____7_f__-__3_vp9V---wOJAIMBAUAgAEMAAQIFCIQAAQhiQAAAABBCIBQJIAEqgAWVwEdoIEACAxAQgQAgBBQgwCAAQAAJKAgBACwQCAAiAQAAgAEAIAAEIAILACQEAAAEAJCAAiACECAgiAAg5DAgIgCCAFABAAAuJDACAMooASBAPGQGAAKAAqACGAEwALgAjgBlgDUAHZAPsA_ACMAFLAK2AbwBMQCbAFogLYAYEAw8BkQDOQGeAM-EQHwAVABWAC4AIYAZAAywBqADZAHYAPwAgABGAClgFPANYAdUA-QCGwEOgIvASIAmwBOwCkQFyAMCAYSAw8Bk4DOQGfCQAYADgBzgN_CQTgAEAALgAoACoAGQAOAAeABAACIAFQAMIAaABqADyAIYAigBMgCqAKwAWAAuABvADmAHoAQ0AiACJgEsAS4AmgBSgC3AGGAMgAZcA1ADVAGyAO8AewA-IB9gH6AQAAjABQQClgFPAL8AYoA1gBtADcAG8AOIAegA-QCGwEOgIqAReAkQBMQCZQE2AJ2AUOApEBYoC2AFyALvAYEAwYBhIDDQGHgMiAZIAycBlwDOQGfANIAadA1gDWQoAEAYQaBIACoAKwAXABDADIAGWANQAbIA7AB-AEAAIKARgApYBT4C0ALSAawA3gB1QD5AIbAQ6Ai8BIgCbAE7AKRAXIAwIBhIDDwGMAMnAZyAzwBnwcAEAA4Bv4qA2ABQAFQAQwAmABcAEcAMsAagA7AB-AEYAKXAWgBaQDeAJBATEAmwBTYC2AFyAMCAYeAyIBnIDPAGfANyHQWQAFwAUABUADIAHAAQAAiABdADAAMYAaABqADwAH0AQwBFACZAFUAVgAsABcADEAGYAN4AcwA9ACGAERAJYAmABNACjAFKALEAW4AwwBkADKAGiANQAbIA3wB3gD2gH2AfoBGACVAFBAKeAWKAtAC0gFzALyAX4AxQBuADiQHTAdQA9ACGwEOgIiAReAkEBIgCbAE7AKHAU0AqwBYsC2ALZAXAAuQBdoC7wGEgMNAYeAxIBjADHgGSAMnAZUAywBlwDOQGfANEgaQBpIDSwGnANYAbGPABAIqAb-QgZgALAAoABkAEQALgAYgBDACYAFUALgAYgAzABvAD0AI4AWIAygBqADfAHfAPsA_ACMAFBAKGAU-AtAC0gF-AMUAdQA9ACQQEiAJsAU0AsUBaMC2ALaAXAAuQBdoDDwGJAMiAZOAzkBngDPgGiANJAaWA4AlAyAAQAAsACgAGQAOAAigBgAGIAPAAiABMACqAFwAMQAZgA2gCGgEQARIAowBSgC3AGEAMoAaoA2QB3gD8AIwAU-AtAC0gGKANwAcQA6gCHQEXgJEATYAsUBbAC7QGHgMiAZOAywBnIDPAGfANIAawA4AmACARUA38pBBAAXABQAFQAMgAcABAACKAGAAYwA0ADUAHkAQwBFACYAFIAKoAWAAuABiADMAHMAQwAiABRgClAFiALcAZQA0QBqgDZAHfAPsA_ACMAFBAKGAVsAuYBeQDaAG4APQAh0BF4CRAE2AJ2AUOApoBWwCxQFsALgAXIAu0BhoDDwGMAMiAZIAycBlwDOQGeAM-gaQBpMDWANZAbGVABAA-Ab-A.YAAAAAAAAAAA",
			allowBid:    true,
			passGeo:     false,
			passID:      true,
		},
	}

	for _, td := range testDefs {
		if td.weakVendorEnforcement {
			TCF2AggConfig.AccountConfig.BasicEnforcementVendorsMap = map[string]struct{}{string(td.bidder): {}}
		}

		allowBid, passGeo, passID, err := perms.AuctionActivitiesAllowed(context.Background(), &TCF2AggConfig, td.bidder, "", SignalYes, td.consent)
		assert.NoErrorf(t, err, "Error processing AuctionActivitiesAllowed for %s", td.description)
		assert.EqualValuesf(t, td.allowBid, allowBid, "AllowBid failure on %s", td.description)
		assert.EqualValuesf(t, td.passGeo, passGeo, "PassGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.passID, passID, "PassID failure on %s", td.description)
	}
}

func TestAllowActivitiesWhitelist(t *testing.T) {
	vendorListData := MarshalVendorList(buildVendorList34())

	perms := permissionsImpl{
		hostVendorID:          2,
		nonStandardPublishers: map[string]struct{}{"appNexusAppID": {}},
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	TCF2AggConfig := allPurposesEnabledTCF2Config()

	// Assert that an item that otherwise would not be allowed PI access, gets approved because it is found in the GDPR.NonStandardPublishers array
	_, passGeo, passID, err := perms.AuctionActivitiesAllowed(context.Background(), &TCF2AggConfig, openrtb_ext.BidderAppnexus, "appNexusAppID", SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing AuctionActivitiesAllowed")
	assert.EqualValuesf(t, true, passGeo, "PassGeo failure")
	assert.EqualValuesf(t, true, passID, "PassID failure")
}

func TestAllowActivitiesPubRestrict(t *testing.T) {
	vendorListData := MarshalVendorList(buildVendorList34())

	perms := permissionsImpl{
		hostVendorID: 2,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 32,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				15: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	TCF2AggConfig := allPurposesEnabledTCF2Config()

	// COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA - vendors 1-10 legit interest only,
	// Pub restriction on purpose 7, consent only ... no allowPI will pass, no special feature 1 consent
	testDefs := []testDef{
		{
			description: "Appnexus vendor test, insufficient purposes claimed",
			bidder:      openrtb_ext.BidderAppnexus,
			consent:     "COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA",
			passGeo:     false,
			passID:      false,
		},
		{
			description: "Pubmatic vendor test, flex purposes claimed",
			bidder:      openrtb_ext.BidderPubmatic,
			consent:     "COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA",
			passGeo:     false,
			passID:      false,
		},
		{
			description: "Rubicon vendor test, Specific purposes/LIs claimed, no geo claimed",
			bidder:      openrtb_ext.BidderRubicon,
			consent:     "COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA",
			passGeo:     false,
			passID:      true,
		},
	}

	for _, td := range testDefs {
		_, passGeo, passID, err := perms.AuctionActivitiesAllowed(context.Background(), &TCF2AggConfig, td.bidder, "", SignalYes, td.consent)
		assert.NoErrorf(t, err, "Error processing AuctionActivitiesAllowed for %s", td.description)
		assert.EqualValuesf(t, td.passGeo, passGeo, "PassGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.passID, passID, "PassID failure on %s", td.description)
	}
}

func TestAllowSync(t *testing.T) {
	vendorListData := MarshalVendorList(buildVendorList34())

	perms := permissionsImpl{
		hostVendorID: 2,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	TCF2AggConfig := allPurposesEnabledTCF2Config()

	// COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA : full consensts to purposes and vendors 2, 6, 8
	allowSync, err := perms.HostCookiesAllowed(context.Background(), &TCF2AggConfig, SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, true, allowSync, "HostCookiesAllowed failure")

	allowSync, err = perms.BidderSyncAllowed(context.Background(), &TCF2AggConfig, openrtb_ext.BidderRubicon, SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing BidderSyncAllowed")
	assert.EqualValuesf(t, true, allowSync, "BidderSyncAllowed failure")
}

func TestProhibitedPurposeSync(t *testing.T) {
	vendorList34 := buildVendorList34()
	vendorList34.Vendors["8"].Purposes = []int{7}
	vendorListData := MarshalVendorList(vendorList34)

	perms := permissionsImpl{
		hostVendorID: 8,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	TCF2AggConfig := allPurposesEnabledTCF2Config()

	// COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA : full consents to purposes for vendors 2, 6, 8
	allowSync, err := perms.HostCookiesAllowed(context.Background(), &TCF2AggConfig, SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, false, allowSync, "HostCookiesAllowed failure")

	allowSync, err = perms.BidderSyncAllowed(context.Background(), &TCF2AggConfig, openrtb_ext.BidderRubicon, SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing BidderSyncAllowed")
	assert.EqualValuesf(t, false, allowSync, "BidderSyncAllowed failure")
}

func TestProhibitedVendorSync(t *testing.T) {
	vendorListData := MarshalVendorList(buildVendorList34())

	perms := permissionsImpl{
		hostVendorID: 10,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
			openrtb_ext.BidderOpenx:    10,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	TCF2AggConfig := allPurposesEnabledTCF2Config()

	// COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA : full consents to purposes for vendors 2, 6, 8
	allowSync, err := perms.HostCookiesAllowed(context.Background(), &TCF2AggConfig, SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, false, allowSync, "HostCookiesAllowed failure")

	// Permission disallowed due to consent string not including vendor 10.
	allowSync, err = perms.BidderSyncAllowed(context.Background(), &TCF2AggConfig, openrtb_ext.BidderOpenx, SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
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

func listFetcher(lists map[uint16]vendorlist.VendorList) func(context.Context, uint16) (vendorlist.VendorList, error) {
	return func(ctx context.Context, id uint16) (vendorlist.VendorList, error) {
		data, ok := lists[id]
		if ok {
			return data, nil
		} else {
			return nil, fmt.Errorf("vendorlist id=%d not found", id)
		}
	}
}

func failedListFetcher(ctx context.Context, id uint16) (vendorlist.VendorList, error) {
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
		purpose2EnforcePurpose string
		purpose2EnforceVendors bool
		bidder                 openrtb_ext.BidderName
		consent                string
		allowBid               bool
		passGeo                bool
		passID                 bool
	}{
		{
			description:            "Bid blocked - p2 enabled, user consents to p2 but not vendor, vendor consents to p2",
			purpose2EnforcePurpose: config.TCF2FullEnforcement,
			purpose2EnforceVendors: true,
			bidder:                 openrtb_ext.BidderPubmatic,
			consent:                purpose2ConsentWithoutVendorConsent,
			allowBid:               false,
			passGeo:                false,
			passID:                 false,
		},
		{
			description:            "Bid allowed - p2 enabled not enforcing vendors, user consents to p2 but not vendor, vendor consents to p2",
			purpose2EnforcePurpose: config.TCF2FullEnforcement,
			purpose2EnforceVendors: false,
			bidder:                 openrtb_ext.BidderPubmatic,
			consent:                purpose2ConsentWithoutVendorConsent,
			allowBid:               true,
			passGeo:                false,
			passID:                 true,
		},
		{
			description:            "Bid allowed - p2 disabled, user consents to p2 but not vendor, vendor consents to p2",
			purpose2EnforcePurpose: config.TCF2NoEnforcement,
			purpose2EnforceVendors: true,
			bidder:                 openrtb_ext.BidderPubmatic,
			consent:                purpose2ConsentWithoutVendorConsent,
			allowBid:               true,
			passGeo:                false,
			passID:                 false,
		},
		{
			description:            "Bid allowed - p2 enabled, user consents to p2 and vendor, vendor consents to p2",
			purpose2EnforcePurpose: config.TCF2FullEnforcement,
			purpose2EnforceVendors: true,
			bidder:                 openrtb_ext.BidderPubmatic,
			consent:                purpose2AndVendorConsent,
			allowBid:               true,
			passGeo:                false,
			passID:                 true,
		},
		{
			description:            "Bid blocked - p2 enabled, user consents to p2 LI but not vendor, vendor consents to p2",
			purpose2EnforcePurpose: config.TCF2FullEnforcement,
			purpose2EnforceVendors: true,
			bidder:                 openrtb_ext.BidderRubicon,
			consent:                purpose2LIWithoutVendorLI,
			allowBid:               false,
			passGeo:                false,
			passID:                 false,
		},
		{
			description:            "Bid allowed - p2 enabled, user consents to p2 LI and vendor, vendor consents to p2",
			purpose2EnforcePurpose: config.TCF2FullEnforcement,
			purpose2EnforceVendors: true,
			bidder:                 openrtb_ext.BidderRubicon,
			consent:                purpose2AndVendorLI,
			allowBid:               true,
			passGeo:                false,
			passID:                 true,
		},
		{
			description:            "Bid allowed - p2 enabled not enforcing vendors, user consents to p2 LI but not vendor, vendor consents to p2",
			purpose2EnforcePurpose: config.TCF2FullEnforcement,
			purpose2EnforceVendors: false,
			bidder:                 openrtb_ext.BidderPubmatic,
			consent:                purpose2AndVendorLI,
			allowBid:               true,
			passGeo:                false,
			passID:                 true,
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
			fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
				tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
					34: parseVendorListDataV2(t, vendorListData),
				}),
			},
		}

		TCF2AggConfig := allPurposesEnabledTCF2Config()
		TCF2AggConfig.HostConfig.Purpose2.EnforcePurpose = td.purpose2EnforcePurpose
		TCF2AggConfig.HostConfig.Purpose2.EnforceVendors = td.purpose2EnforceVendors
		p2Config := TCF2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(2)]
		p2Config.EnforcePurpose = td.purpose2EnforcePurpose
		p2Config.EnforceVendors = td.purpose2EnforceVendors
		TCF2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(2)] = p2Config
		TCF2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(2)] = &TCF2AggConfig.HostConfig.Purpose2

		allowBid, passGeo, passID, err := perms.AuctionActivitiesAllowed(context.Background(), &TCF2AggConfig, td.bidder, "", SignalYes, td.consent)
		assert.NoErrorf(t, err, "Error processing AuctionActivitiesAllowed for %s", td.description)
		assert.EqualValuesf(t, td.allowBid, allowBid, "AllowBid failure on %s", td.description)
		assert.EqualValuesf(t, td.passGeo, passGeo, "PassGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.passID, passID, "PassID failure on %s", td.description)
	}
}

func TestTCF1Consent(t *testing.T) {
	bidderAllowedByConsent := openrtb_ext.BidderAppnexus
	tcf1Consent := "BOS2bx5OS2bx5ABABBAAABoAAAABBwAA"

	perms := permissionsImpl{
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
		},
	}

	bidReq, passGeo, passID, err := perms.AuctionActivitiesAllowed(context.Background(), &TCF2Config{}, bidderAllowedByConsent, "", SignalYes, tcf1Consent)

	assert.Nil(t, err, "TCF1 consent - no error returned")
	assert.Equal(t, false, bidReq, "TCF1 consent - bid request not allowed")
	assert.Equal(t, false, passGeo, "TCF1 consent - passing geo not allowed")
	assert.Equal(t, false, passID, "TCF1 consent - passing id not allowed")
}

func TestAllowActivitiesVendorException(t *testing.T) {
	noPurposeOrVendorConsentAndPubRestrictsP2 := "CPF_61ePF_61eFxAAAENAiCAAAAAAAAAAAAAACEAAAACEAAgAgAA"
	noPurposeOrVendorConsentAndPubRestrictsNone := "CPF_61ePF_61eFxAAAENAiCAAAAAAAAAAAAAACEAAAAA"

	testDefs := []struct {
		description           string
		p2VendorExceptionMap  map[openrtb_ext.BidderName]struct{}
		sf1VendorExceptionMap map[openrtb_ext.BidderName]struct{}
		bidder                openrtb_ext.BidderName
		consent               string
		allowBid              bool
		passGeo               bool
		passID                bool
	}{
		{
			description:          "Bid/ID blocked by publisher - p2 enabled with p2 vendor exception, pub restricts p2 for vendor",
			p2VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderAppnexus: {}},
			bidder:               openrtb_ext.BidderAppnexus,
			consent:              noPurposeOrVendorConsentAndPubRestrictsP2,
			allowBid:             false,
			passGeo:              false,
			passID:               false,
		},
		{
			description:           "Bid/ID allowed by vendor exception - p2 enabled with p2 vendor exception, pub restricts none",
			p2VendorExceptionMap:  map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderAppnexus: {}},
			sf1VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
			bidder:                openrtb_ext.BidderAppnexus,
			consent:               noPurposeOrVendorConsentAndPubRestrictsNone,
			allowBid:              true,
			passGeo:               false,
			passID:                true,
		},
		{
			description:           "Geo blocked - sf1 enabled but no consent",
			p2VendorExceptionMap:  map[openrtb_ext.BidderName]struct{}{},
			sf1VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
			bidder:                openrtb_ext.BidderAppnexus,
			consent:               noPurposeOrVendorConsentAndPubRestrictsNone,
			allowBid:              false,
			passGeo:               false,
			passID:                false,
		},
		{
			description:           "Geo allowed by vendor exception - sf1 enabled with sf1 vendor exception",
			p2VendorExceptionMap:  map[openrtb_ext.BidderName]struct{}{},
			sf1VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderAppnexus: {}},
			bidder:                openrtb_ext.BidderAppnexus,
			consent:               noPurposeOrVendorConsentAndPubRestrictsNone,
			allowBid:              false,
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
			fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
				tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
					34: parseVendorListDataV2(t, vendorListData),
				}),
			},
		}

		TCF2AggConfig := allPurposesEnabledTCF2Config()
		TCF2AggConfig.HostConfig.Purpose2.VendorExceptionMap = td.p2VendorExceptionMap
		TCF2AggConfig.HostConfig.SpecialFeature1.Enforce = true
		TCF2AggConfig.HostConfig.SpecialFeature1.VendorExceptionMap = td.sf1VendorExceptionMap
		TCF2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(2)] = &TCF2AggConfig.HostConfig.Purpose2
		TCF2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(3)] = &TCF2AggConfig.HostConfig.Purpose3

		allowBid, passGeo, passID, err := perms.AuctionActivitiesAllowed(context.Background(), &TCF2AggConfig, td.bidder, "", SignalYes, td.consent)
		assert.NoErrorf(t, err, "Error processing AuctionActivitiesAllowed for %s", td.description)
		assert.EqualValuesf(t, td.allowBid, allowBid, "AllowBid failure on %s", td.description)
		assert.EqualValuesf(t, td.passGeo, passGeo, "PassGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.passID, passID, "PassID failure on %s", td.description)
	}
}

func TestBidderSyncAllowedVendorException(t *testing.T) {
	noPurposeOrVendorConsentAndPubRestrictsP1 := "CPF_61ePF_61eFxAAAENAiCAAAAAAAAAAAAAAQAAAAAAAAAAIIACACA"
	noPurposeOrVendorConsentAndPubRestrictsNone := "CPF_61ePF_61eFxAAAENAiCAAAAAAAAAAAAAACEAAAAA"

	testDefs := []struct {
		description          string
		p1VendorExceptionMap map[openrtb_ext.BidderName]struct{}
		bidder               openrtb_ext.BidderName
		consent              string
		allowSync            bool
	}{
		{
			description:          "Sync blocked by no consent - p1 enabled, no p1 vendor exception, pub restricts none",
			p1VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
			bidder:               openrtb_ext.BidderAppnexus,
			consent:              noPurposeOrVendorConsentAndPubRestrictsNone,
			allowSync:            false,
		},
		{
			description:          "Sync blocked by publisher - p1 enabled with p1 vendor exception, pub restricts p1 for vendor",
			p1VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderAppnexus: {}},
			bidder:               openrtb_ext.BidderAppnexus,
			consent:              noPurposeOrVendorConsentAndPubRestrictsP1,
			allowSync:            false,
		},
		{
			description:          "Sync allowed by vendor exception - p1 enabled with p1 vendor exception, pub restricts none",
			p1VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderAppnexus: {}},
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
			fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
				tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
					34: parseVendorListDataV2(t, vendorListData),
				}),
			},
		}

		TCF2AggConfig := allPurposesEnabledTCF2Config()
		TCF2AggConfig.HostConfig.Purpose1.VendorExceptionMap = td.p1VendorExceptionMap
		TCF2AggConfig.HostConfig.PurposeConfigs[consentconstants.Purpose(1)] = &TCF2AggConfig.HostConfig.Purpose1

		allowSync, err := perms.BidderSyncAllowed(context.Background(), &TCF2AggConfig, td.bidder, SignalYes, td.consent)
		assert.NoErrorf(t, err, "Error processing BidderSyncAllowed for %s", td.description)
		assert.EqualValuesf(t, td.allowSync, allowSync, "AllowSync failure on %s", td.description)
	}
}
