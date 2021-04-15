package gdpr

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/go-gdpr/vendorlist2"

	"github.com/stretchr/testify/assert"
)

func TestDisallowOnEmptyConsent(t *testing.T) {
	perms := permissionsImpl{
		cfg: config.GDPR{
			HostVendorID:        3,
			UsersyncIfAmbiguous: true,
		},
		vendorIDs: nil,
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: failedListFetcher,
			tcf2SpecVersion: failedListFetcher,
		},
	}
	allowSync, err := perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderAppnexus, SignalYes, "")
	assertBoolsEqual(t, false, allowSync)
	assertNilErr(t, err)
	allowSync, err = perms.HostCookiesAllowed(context.Background(), SignalYes, "")
	assertBoolsEqual(t, false, allowSync)
	assertNilErr(t, err)
}

func TestAllowOnSignalNo(t *testing.T) {
	perms := permissionsImpl{}
	emptyConsent := ""

	allowSync, err := perms.HostCookiesAllowed(context.Background(), SignalNo, emptyConsent)
	assert.Equal(t, true, allowSync)
	assert.Nil(t, err)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderAppnexus, SignalNo, emptyConsent)
	assert.Equal(t, true, allowSync)
	assert.Nil(t, err)
}

func TestAllowedSyncs(t *testing.T) {
	vendorListData := tcf1MarshalVendorList(tcf1VendorList{
		VendorListVersion: 1,
		Vendors: []tcf1Vendor{
			{ID: 2, Purposes: []int{1}},
			{ID: 3, Purposes: []int{1}},
		},
	})
	perms := permissionsImpl{
		cfg: config.GDPR{
			HostVendorID: 2,
		},
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 3,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
		},
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background(), SignalYes, "BON3PCUON3PCUABABBAAABoAAAAAMw")
	assertNilErr(t, err)
	assertBoolsEqual(t, true, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderPubmatic, SignalYes, "BON3PCUON3PCUABABBAAABoAAAAAMw")
	assertNilErr(t, err)
	assertBoolsEqual(t, true, allowSync)
}

func TestProhibitedPurposes(t *testing.T) {
	vendorListData := tcf1MarshalVendorList(tcf1VendorList{
		VendorListVersion: 1,
		Vendors: []tcf1Vendor{
			{ID: 2, Purposes: []int{1}}, // cookie reads/writes
			{ID: 3, Purposes: []int{3}}, // ad personalization
		},
	})
	perms := permissionsImpl{
		cfg: config.GDPR{
			HostVendorID: 2,
		},
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 3,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
		},
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background(), SignalYes, "BON3PCUON3PCUABABBAAABAAAAAAMw")
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderPubmatic, SignalYes, "BON3PCUON3PCUABABBAAABAAAAAAMw")
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)
}

func TestProhibitedVendors(t *testing.T) {
	vendorListData := tcf1MarshalVendorList(tcf1VendorList{
		VendorListVersion: 1,
		Vendors: []tcf1Vendor{
			{ID: 2, Purposes: []int{1}}, // cookie reads/writes
			{ID: 3, Purposes: []int{3}}, // ad personalization
		},
	})
	perms := permissionsImpl{
		cfg: config.GDPR{
			HostVendorID: 2,
		},
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 3,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
		},
	}

	allowSync, err := perms.HostCookiesAllowed(context.Background(), SignalYes, "BOS2bx5OS2bx5ABABBAAABoAAAAAFA")
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderPubmatic, SignalYes, "BOS2bx5OS2bx5ABABBAAABoAAAAAFA")
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)
}

func TestMalformedConsent(t *testing.T) {
	perms := permissionsImpl{
		cfg: config.GDPR{
			HostVendorID: 2,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: listFetcher(nil),
			tcf2SpecVersion: listFetcher(nil),
		},
	}

	sync, err := perms.HostCookiesAllowed(context.Background(), SignalYes, "BON")
	assertErr(t, err, true)
	assertBoolsEqual(t, false, sync)
}

func TestAllowPersonalInfo(t *testing.T) {
	bidderAllowedByConsent := openrtb_ext.BidderAppnexus
	bidderBlockedByConsent := openrtb_ext.BidderRubicon
	consent := "BOS2bx5OS2bx5ABABBAAABoAAAABBwAA"

	tests := []struct {
		description           string
		bidderName            openrtb_ext.BidderName
		publisherID           string
		userSyncIfAmbiguous   bool
		gdpr                  Signal
		consent               string
		allowPI               bool
		weakVendorEnforcement bool
	}{
		{
			description:         "Allow PI - Non standard publisher",
			bidderName:          bidderBlockedByConsent,
			publisherID:         "appNexusAppID",
			userSyncIfAmbiguous: false,
			gdpr:                SignalYes,
			consent:             consent,
			allowPI:             true,
		},
		{
			description:         "Allow PI - known vendor with No GDPR",
			bidderName:          bidderBlockedByConsent,
			userSyncIfAmbiguous: false,
			gdpr:                SignalNo,
			consent:             consent,
			allowPI:             true,
		},
		{
			description:         "Allow PI - known vendor with Yes GDPR",
			bidderName:          bidderAllowedByConsent,
			userSyncIfAmbiguous: false,
			gdpr:                SignalYes,
			consent:             consent,
			allowPI:             true,
		},
		{
			description:         "PI allowed according to host setting UserSyncIfAmbiguous true - known vendor with ambiguous GDPR and empty consent",
			bidderName:          bidderAllowedByConsent,
			userSyncIfAmbiguous: true,
			gdpr:                SignalAmbiguous,
			consent:             "",
			allowPI:             true,
		},
		{
			description:         "PI allowed according to host setting UserSyncIfAmbiguous true - known vendor with ambiguous GDPR and non-empty consent",
			bidderName:          bidderAllowedByConsent,
			userSyncIfAmbiguous: true,
			gdpr:                SignalAmbiguous,
			consent:             consent,
			allowPI:             true,
		},
		{
			description:         "PI allowed according to host setting UserSyncIfAmbiguous false - known vendor with ambiguous GDPR and empty consent",
			bidderName:          bidderAllowedByConsent,
			userSyncIfAmbiguous: false,
			gdpr:                SignalAmbiguous,
			consent:             "",
			allowPI:             false,
		},
		{
			description:         "PI allowed according to host setting UserSyncIfAmbiguous false - known vendor with ambiguous GDPR and non-empty consent",
			bidderName:          bidderAllowedByConsent,
			userSyncIfAmbiguous: false,
			gdpr:                SignalAmbiguous,
			consent:             consent,
			allowPI:             true,
		},
		{
			description:         "Don't allow PI - known vendor with Yes GDPR and empty consent",
			bidderName:          bidderAllowedByConsent,
			userSyncIfAmbiguous: false,
			gdpr:                SignalYes,
			consent:             "",
			allowPI:             false,
		},
		{
			description:         "Don't allow PI - default vendor with Yes GDPR and non-empty consent",
			bidderName:          bidderBlockedByConsent,
			userSyncIfAmbiguous: false,
			gdpr:                SignalYes,
			consent:             consent,
			allowPI:             false,
		},
	}

	vendorListData := tcf1MarshalVendorList(tcf1VendorList{
		VendorListVersion: 1,
		Vendors: []tcf1Vendor{
			{ID: 2, Purposes: []int{1, 3}},
		},
	})
	perms := permissionsImpl{
		cfg: config.GDPR{
			HostVendorID:            2,
			NonStandardPublisherMap: map[string]struct{}{"appNexusAppID": {}},
		},
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
		},
	}

	for _, tt := range tests {
		perms.cfg.UsersyncIfAmbiguous = tt.userSyncIfAmbiguous

		allowPI, _, _, err := perms.PersonalInfoAllowed(context.Background(), tt.bidderName, tt.publisherID, tt.gdpr, tt.consent, tt.weakVendorEnforcement)

		assert.Nil(t, err, tt.description)
		assert.Equal(t, tt.allowPI, allowPI, tt.description)
	}
}

func buildTCF2VendorList34() tcf2VendorList {
	return tcf2VendorList{
		VendorListVersion: 2,
		Vendors: map[string]*tcf2Vendor{
			"2": {
				ID:       2,
				Purposes: []int{1},
			},
			"6": {
				ID:               6,
				Purposes:         []int{1, 2, 4},
				LegIntPurposes:   []int{7},
				SpecialPurposes:  []int{1},
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
				SpecialPurposes: []int{1},
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

var tcf2Config = config.GDPR{
	HostVendorID: 2,
	TCF2: config.TCF2{
		Enabled:         true,
		Purpose1:        config.PurposeDetail{Enabled: true},
		Purpose2:        config.PurposeDetail{Enabled: true},
		Purpose7:        config.PurposeDetail{Enabled: true},
		SpecialPurpose1: config.PurposeDetail{Enabled: true},
	},
}

type tcf2TestDef struct {
	description           string
	bidder                openrtb_ext.BidderName
	consent               string
	allowPI               bool
	allowGeo              bool
	allowID               bool
	weakVendorEnforcement bool
}

func TestAllowPersonalInfoTCF2(t *testing.T) {
	vendorListData := tcf2MarshalVendorList(buildTCF2VendorList34())
	perms := permissionsImpl{
		cfg: tcf2Config,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
			openrtb_ext.BidderOpenx:    20,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: nil,
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
				74: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	// COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA : TCF2 with full consents to purposes and vendors 2, 6, 8
	// PI needs all purposes to succeed
	testDefs := []tcf2TestDef{
		{
			description: "Appnexus vendor test, insufficient purposes claimed",
			bidder:      openrtb_ext.BidderAppnexus,
			consent:     "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowPI:     false,
			allowGeo:    false,
			allowID:     false,
		},
		{
			description:           "Appnexus vendor test, insufficient purposes claimed, basic enforcement",
			bidder:                openrtb_ext.BidderAppnexus,
			consent:               "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowPI:               true,
			allowGeo:              true,
			allowID:               true,
			weakVendorEnforcement: true,
		},
		{
			description: "Pubmatic vendor test, flex purposes claimed",
			bidder:      openrtb_ext.BidderPubmatic,
			consent:     "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowPI:     true,
			allowGeo:    true,
			allowID:     true,
		},
		{
			description: "Rubicon vendor test, Specific purposes/LIs claimed, no geo claimed",
			bidder:      openrtb_ext.BidderRubicon,
			consent:     "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			allowPI:     true,
			allowGeo:    false,
			allowID:     true,
		},
		{
			// This requires publisher restrictions on any claimed purposes, 2-10. Vendor must declare all claimed purposes
			// as flex with legit interest as primary.
			// Using vendor 20 for this.
			description: "OpenX vendor test, Specific purposes/LIs claimed, no geo claimed, Publisher restrictions apply",
			bidder:      openrtb_ext.BidderOpenx,
			consent:     "CPAavcCPAavcCAGABCFRBKCsAP_AAH_AAAqIHFNf_X_fb3_j-_59_9t0eY1f9_7_v-0zjgeds-8Nyd_X_L8X5mM7vB36pq4KuR4Eu3LBAQdlHOHcTUmw6IkVqTPsbk2Mr7NKJ7PEinMbe2dYGH9_n9XT_ZKY79_____7__-_____7_f__-__3_vp9V---wOJAIMBAUAgAEMAAQIFCIQAAQhiQAAAABBCIBQJIAEqgAWVwEdoIEACAxAQgQAgBBQgwCAAQAAJKAgBACwQCAAiAQAAgAEAIAAEIAILACQEAAAEAJCAAiACECAgiAAg5DAgIgCCAFABAAAuJDACAMooASBAPGQGAAKAAqACGAEwALgAjgBlgDUAHZAPsA_ACMAFLAK2AbwBMQCbAFogLYAYEAw8BkQDOQGeAM-EQHwAVABWAC4AIYAZAAywBqADZAHYAPwAgABGAClgFPANYAdUA-QCGwEOgIvASIAmwBOwCkQFyAMCAYSAw8Bk4DOQGfCQAYADgBzgN_CQTgAEAALgAoACoAGQAOAAeABAACIAFQAMIAaABqADyAIYAigBMgCqAKwAWAAuABvADmAHoAQ0AiACJgEsAS4AmgBSgC3AGGAMgAZcA1ADVAGyAO8AewA-IB9gH6AQAAjABQQClgFPAL8AYoA1gBtADcAG8AOIAegA-QCGwEOgIqAReAkQBMQCZQE2AJ2AUOApEBYoC2AFyALvAYEAwYBhIDDQGHgMiAZIAycBlwDOQGfANIAadA1gDWQoAEAYQaBIACoAKwAXABDADIAGWANQAbIA7AB-AEAAIKARgApYBT4C0ALSAawA3gB1QD5AIbAQ6Ai8BIgCbAE7AKRAXIAwIBhIDDwGMAMnAZyAzwBnwcAEAA4Bv4qA2ABQAFQAQwAmABcAEcAMsAagA7AB-AEYAKXAWgBaQDeAJBATEAmwBTYC2AFyAMCAYeAyIBnIDPAGfANyHQWQAFwAUABUADIAHAAQAAiABdADAAMYAaABqADwAH0AQwBFACZAFUAVgAsABcADEAGYAN4AcwA9ACGAERAJYAmABNACjAFKALEAW4AwwBkADKAGiANQAbIA3wB3gD2gH2AfoBGACVAFBAKeAWKAtAC0gFzALyAX4AxQBuADiQHTAdQA9ACGwEOgIiAReAkEBIgCbAE7AKHAU0AqwBYsC2ALZAXAAuQBdoC7wGEgMNAYeAxIBjADHgGSAMnAZUAywBlwDOQGfANEgaQBpIDSwGnANYAbGPABAIqAb-QgZgALAAoABkAEQALgAYgBDACYAFUALgAYgAzABvAD0AI4AWIAygBqADfAHfAPsA_ACMAFBAKGAU-AtAC0gF-AMUAdQA9ACQQEiAJsAU0AsUBaMC2ALaAXAAuQBdoDDwGJAMiAZOAzkBngDPgGiANJAaWA4AlAyAAQAAsACgAGQAOAAigBgAGIAPAAiABMACqAFwAMQAZgA2gCGgEQARIAowBSgC3AGEAMoAaoA2QB3gD8AIwAU-AtAC0gGKANwAcQA6gCHQEXgJEATYAsUBbAC7QGHgMiAZOAywBnIDPAGfANIAawA4AmACARUA38pBBAAXABQAFQAMgAcABAACKAGAAYwA0ADUAHkAQwBFACYAFIAKoAWAAuABiADMAHMAQwAiABRgClAFiALcAZQA0QBqgDZAHfAPsA_ACMAFBAKGAVsAuYBeQDaAG4APQAh0BF4CRAE2AJ2AUOApoBWwCxQFsALgAXIAu0BhoDDwGMAMiAZIAycBlwDOQGeAM-gaQBpMDWANZAbGVABAA-Ab-A.YAAAAAAAAAAA",
			allowPI:     true,
			allowGeo:    false,
			allowID:     true,
		},
	}

	for _, td := range testDefs {
		allowPI, allowGeo, allowID, err := perms.PersonalInfoAllowed(context.Background(), td.bidder, "", SignalYes, td.consent, td.weakVendorEnforcement)
		assert.NoErrorf(t, err, "Error processing PersonalInfoAllowed for %s", td.description)
		assert.EqualValuesf(t, td.allowPI, allowPI, "AllowPI failure on %s", td.description)
		assert.EqualValuesf(t, td.allowGeo, allowGeo, "AllowGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.allowID, allowID, "AllowGeo failure on %s", td.description)
	}
}

func TestAllowPersonalInfoWhitelistTCF2(t *testing.T) {
	vendorListData := tcf2MarshalVendorList(buildTCF2VendorList34())
	perms := permissionsImpl{
		cfg: tcf2Config,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: nil,
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}
	// Assert that an item that otherwise would not be allowed PI access, gets approved because it is found in the GDPR.NonStandardPublishers array
	perms.cfg.NonStandardPublisherMap = map[string]struct{}{"appNexusAppID": {}}
	allowPI, allowGeo, allowID, err := perms.PersonalInfoAllowed(context.Background(), openrtb_ext.BidderAppnexus, "appNexusAppID", SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA", false)
	assert.NoErrorf(t, err, "Error processing PersonalInfoAllowed")
	assert.EqualValuesf(t, true, allowPI, "AllowPI failure")
	assert.EqualValuesf(t, true, allowGeo, "AllowGeo failure")
	assert.EqualValuesf(t, true, allowID, "AllowID failure")
}

func TestAllowPersonalInfoTCF2PubRestrict(t *testing.T) {
	vendorListData := tcf2MarshalVendorList(buildTCF2VendorList34())
	perms := permissionsImpl{
		cfg: tcf2Config,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 32,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: nil,
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				15: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	// COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA - vendors 1-10 legit interest only,
	// Pub restriction on purpose 7, consent only ... no allowPI will pass, no Special purpose 1 consent
	testDefs := []tcf2TestDef{
		{
			description: "Appnexus vendor test, insufficient purposes claimed",
			bidder:      openrtb_ext.BidderAppnexus,
			consent:     "COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA",
			allowPI:     false,
			allowGeo:    false,
			allowID:     false,
		},
		{
			description: "Pubmatic vendor test, flex purposes claimed",
			bidder:      openrtb_ext.BidderPubmatic,
			consent:     "COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA",
			allowPI:     false,
			allowGeo:    false,
			allowID:     false,
		},
		{
			description: "Rubicon vendor test, Specific purposes/LIs claimed, no geo claimed",
			bidder:      openrtb_ext.BidderRubicon,
			consent:     "COwAdDhOwAdDhN4ABAENAPCgAAQAAv___wAAAFP_AAp_4AI6ACACAA",
			allowPI:     false,
			allowGeo:    false,
			allowID:     true,
		},
	}

	for _, td := range testDefs {
		allowPI, allowGeo, allowID, err := perms.PersonalInfoAllowed(context.Background(), td.bidder, "", SignalYes, td.consent, td.weakVendorEnforcement)
		assert.NoErrorf(t, err, "Error processing PersonalInfoAllowed for %s", td.description)
		assert.EqualValuesf(t, td.allowPI, allowPI, "AllowPI failure on %s", td.description)
		assert.EqualValuesf(t, td.allowGeo, allowGeo, "AllowGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.allowID, allowID, "AllowPI failure on %s", td.description)
	}
}

func TestAllowPersonalInfoTCF2PurposeOneTrue(t *testing.T) {
	vendorListData := tcf2MarshalVendorList(buildTCF2VendorList34())
	perms := permissionsImpl{
		cfg: tcf2Config,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 10,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: nil,
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}
	perms.cfg.TCF2.PurposeOneTreatment.Enabled = true
	perms.cfg.TCF2.PurposeOneTreatment.AccessAllowed = true

	// COzqiL3OzqiL3NIAAAENAiCMAP_AAH_AAIAAAQEX2S5MAICL7JcmAAA Purpose one flag set
	testDefs := []tcf2TestDef{
		{
			description: "Appnexus vendor test, insufficient purposes claimed",
			bidder:      openrtb_ext.BidderAppnexus,
			consent:     "COzqiL3OzqiL3NIAAAENAiCMAP_AAH_AAIAAAQEX2S5MAICL7JcmAAA",
			allowPI:     false,
			allowGeo:    false,
			allowID:     false,
		},
		{
			description: "Pubmatic vendor test, flex purposes claimed",
			bidder:      openrtb_ext.BidderPubmatic,
			consent:     "COzqiL3OzqiL3NIAAAENAiCMAP_AAH_AAIAAAQEX2S5MAICL7JcmAAA",
			allowPI:     true,
			allowGeo:    true,
			allowID:     true,
		},
		{
			description: "Rubicon vendor test, Specific purposes/LIs claimed, no geo claimed",
			bidder:      openrtb_ext.BidderRubicon,
			consent:     "COzqiL3OzqiL3NIAAAENAiCMAP_AAH_AAIAAAQEX2S5MAICL7JcmAAA",
			allowPI:     true,
			allowGeo:    false,
			allowID:     true,
		},
	}

	for _, td := range testDefs {
		allowPI, allowGeo, allowID, err := perms.PersonalInfoAllowed(context.Background(), td.bidder, "", SignalYes, td.consent, td.weakVendorEnforcement)
		assert.NoErrorf(t, err, "Error processing PersonalInfoAllowed for %s", td.description)
		assert.EqualValuesf(t, td.allowPI, allowPI, "AllowPI failure on %s", td.description)
		assert.EqualValuesf(t, td.allowGeo, allowGeo, "AllowGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.allowID, allowID, "AllowID failure on %s", td.description)
	}
}

func TestAllowPersonalInfoTCF2PurposeOneFalse(t *testing.T) {
	vendorListData := tcf2MarshalVendorList(buildTCF2VendorList34())
	perms := permissionsImpl{
		cfg: tcf2Config,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 10,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: nil,
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}
	perms.cfg.TCF2.PurposeOneTreatment.Enabled = true
	perms.cfg.TCF2.PurposeOneTreatment.AccessAllowed = false

	// COzqiL3OzqiL3NIAAAENAiCMAP_AAH_AAIAAAQEX2S5MAICL7JcmAAA Purpose one flag set
	// Purpose one treatment will fail PI, but allow passing the IDs.
	testDefs := []tcf2TestDef{
		{
			description: "Appnexus vendor test, insufficient purposes claimed",
			bidder:      openrtb_ext.BidderAppnexus,
			consent:     "COzqiL3OzqiL3NIAAAENAiCMAP_AAH_AAIAAAQEX2S5MAICL7JcmAAA",
			allowPI:     false,
			allowGeo:    false,
			allowID:     false,
		},
		{
			description: "Pubmatic vendor test, flex purposes claimed",
			bidder:      openrtb_ext.BidderPubmatic,
			consent:     "COzqiL3OzqiL3NIAAAENAiCMAP_AAH_AAIAAAQEX2S5MAICL7JcmAAA",
			allowPI:     false,
			allowGeo:    true,
			allowID:     true,
		},
		{
			description: "Rubicon vendor test, Specific purposes/LIs claimed, no geo claimed",
			bidder:      openrtb_ext.BidderRubicon,
			consent:     "COzqiL3OzqiL3NIAAAENAiCMAP_AAH_AAIAAAQEX2S5MAICL7JcmAAA",
			allowPI:     false,
			allowGeo:    false,
			allowID:     true,
		},
	}

	for _, td := range testDefs {
		allowPI, allowGeo, allowID, err := perms.PersonalInfoAllowed(context.Background(), td.bidder, "", SignalYes, td.consent, td.weakVendorEnforcement)
		assert.NoErrorf(t, err, "Error processing PersonalInfoAllowed for %s", td.description)
		assert.EqualValuesf(t, td.allowPI, allowPI, "AllowPI failure on %s", td.description)
		assert.EqualValuesf(t, td.allowGeo, allowGeo, "AllowGeo failure on %s", td.description)
		assert.EqualValuesf(t, td.allowID, allowID, "AllowID failure on %s", td.description)
	}
}

func TestAllowSyncTCF2(t *testing.T) {
	vendorListData := tcf2MarshalVendorList(buildTCF2VendorList34())
	perms := permissionsImpl{
		cfg: tcf2Config,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: nil,
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}

	// COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA : TCF2 with full consensts to purposes and vendors 2, 6, 8
	allowSync, err := perms.HostCookiesAllowed(context.Background(), SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, true, allowSync, "HostCookiesAllowed failure")

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderRubicon, SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing BidderSyncAllowed")
	assert.EqualValuesf(t, true, allowSync, "BidderSyncAllowed failure")
}

func TestProhibitedPurposeSyncTCF2(t *testing.T) {
	tcf2VendorList34 := buildTCF2VendorList34()
	tcf2VendorList34.Vendors["8"].Purposes = []int{7}
	vendorListData := tcf2MarshalVendorList(tcf2VendorList34)
	perms := permissionsImpl{
		cfg: tcf2Config,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: nil,
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}
	perms.cfg.HostVendorID = 8

	// COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA : TCF2 with full consents to purposes for vendors 2, 6, 8
	allowSync, err := perms.HostCookiesAllowed(context.Background(), SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, false, allowSync, "HostCookiesAllowed failure")

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderRubicon, SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing BidderSyncAllowed")
	assert.EqualValuesf(t, false, allowSync, "BidderSyncAllowed failure")
}

func TestProhibitedVendorSyncTCF2(t *testing.T) {
	vendorListData := tcf2MarshalVendorList(buildTCF2VendorList34())
	perms := permissionsImpl{
		cfg: tcf2Config,
		vendorIDs: map[openrtb_ext.BidderName]uint16{
			openrtb_ext.BidderAppnexus: 2,
			openrtb_ext.BidderPubmatic: 6,
			openrtb_ext.BidderRubicon:  8,
			openrtb_ext.BidderOpenx:    10,
		},
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: nil,
			tcf2SpecVersion: listFetcher(map[uint16]vendorlist.VendorList{
				34: parseVendorListDataV2(t, vendorListData),
			}),
		},
	}
	perms.cfg.HostVendorID = 10

	// COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA : TCF2 with full consents to purposes for vendors 2, 6, 8
	allowSync, err := perms.HostCookiesAllowed(context.Background(), SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, false, allowSync, "HostCookiesAllowed failure")

	// Permission disallowed due to consent string not including vendor 10.
	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderOpenx, SignalYes, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing BidderSyncAllowed")
	assert.EqualValuesf(t, false, allowSync, "BidderSyncAllowed failure")
}

func parseVendorListData(t *testing.T, data string) vendorlist.VendorList {
	t.Helper()
	parsed, err := vendorlist.ParseEagerly([]byte(data))
	if err != nil {
		t.Fatalf("Failed to parse vendor list data. %v", err)
	}
	return parsed
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

func assertStringsEqual(t *testing.T, expected string, actual string) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestNormalizeGDPR(t *testing.T) {
	tests := []struct {
		description         string
		userSyncIfAmbiguous bool
		giveSignal          Signal
		wantSignal          Signal
	}{
		{
			description:         "Don't normalize - Signal No and userSyncIfAmbiguous false",
			userSyncIfAmbiguous: false,
			giveSignal:          SignalNo,
			wantSignal:          SignalNo,
		},
		{
			description:         "Don't normalize - Signal No and userSyncIfAmbiguous true",
			userSyncIfAmbiguous: true,
			giveSignal:          SignalNo,
			wantSignal:          SignalNo,
		},
		{
			description:         "Don't normalize - Signal Yes and userSyncIfAmbiguous false",
			userSyncIfAmbiguous: false,
			giveSignal:          SignalYes,
			wantSignal:          SignalYes,
		},
		{
			description:         "Don't normalize - Signal Yes and userSyncIfAmbiguous true",
			userSyncIfAmbiguous: true,
			giveSignal:          SignalYes,
			wantSignal:          SignalYes,
		},
		{
			description:         "Normalize - Signal Ambiguous and userSyncIfAmbiguous false",
			userSyncIfAmbiguous: false,
			giveSignal:          SignalAmbiguous,
			wantSignal:          SignalYes,
		},
		{
			description:         "Normalize - Signal Ambiguous and userSyncIfAmbiguous true",
			userSyncIfAmbiguous: true,
			giveSignal:          SignalAmbiguous,
			wantSignal:          SignalNo,
		},
	}

	for _, tt := range tests {
		perms := permissionsImpl{
			cfg: config.GDPR{
				UsersyncIfAmbiguous: tt.userSyncIfAmbiguous,
			},
		}

		normalizedSignal := perms.normalizeGDPR(tt.giveSignal)

		assert.Equal(t, tt.wantSignal, normalizedSignal, tt.description)
	}
}
