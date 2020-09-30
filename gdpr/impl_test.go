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

func TestNoConsentButAllowByDefault(t *testing.T) {
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
	allowSync, err := perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderAppnexus, "")
	assertBoolsEqual(t, true, allowSync)
	assertNilErr(t, err)
	allowSync, err = perms.HostCookiesAllowed(context.Background(), "")
	assertBoolsEqual(t, true, allowSync)
	assertNilErr(t, err)
}

func TestNoConsentAndRejectByDefault(t *testing.T) {
	perms := permissionsImpl{
		cfg: config.GDPR{
			HostVendorID:        3,
			UsersyncIfAmbiguous: false,
		},
		vendorIDs: nil,
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tcf1SpecVersion: failedListFetcher,
			tcf2SpecVersion: failedListFetcher,
		},
	}
	allowSync, err := perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderAppnexus, "")
	assertBoolsEqual(t, false, allowSync)
	assertNilErr(t, err)
	allowSync, err = perms.HostCookiesAllowed(context.Background(), "")
	assertBoolsEqual(t, false, allowSync)
	assertNilErr(t, err)
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

	allowSync, err := perms.HostCookiesAllowed(context.Background(), "BON3PCUON3PCUABABBAAABoAAAAAMw")
	assertNilErr(t, err)
	assertBoolsEqual(t, true, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderPubmatic, "BON3PCUON3PCUABABBAAABoAAAAAMw")
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

	allowSync, err := perms.HostCookiesAllowed(context.Background(), "BON3PCUON3PCUABABBAAABAAAAAAMw")
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderPubmatic, "BON3PCUON3PCUABABBAAABAAAAAAMw")
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

	allowSync, err := perms.HostCookiesAllowed(context.Background(), "BOS2bx5OS2bx5ABABBAAABoAAAAAFA")
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowSync)

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderPubmatic, "BOS2bx5OS2bx5ABABBAAABoAAAAAFA")
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

	sync, err := perms.HostCookiesAllowed(context.Background(), "BON")
	assertErr(t, err, true)
	assertBoolsEqual(t, false, sync)
}

func TestAllowPersonalInfo(t *testing.T) {
	vendorListData := tcf1MarshalVendorList(tcf1VendorList{
		VendorListVersion: 1,
		Vendors: []tcf1Vendor{
			{ID: 2, Purposes: []int{1}},    // cookie reads/writes
			{ID: 3, Purposes: []int{1, 3}}, // ad personalization
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

	// PI needs both purposes to succeed
	allowPI, _, _, err := perms.PersonalInfoAllowed(context.Background(), openrtb_ext.BidderAppnexus, "", "BOS2bx5OS2bx5ABABBAAABoAAAABBwAA")
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowPI)

	allowPI, _, _, err = perms.PersonalInfoAllowed(context.Background(), openrtb_ext.BidderPubmatic, "", "BOS2bx5OS2bx5ABABBAAABoAAAABBwAA")
	assertNilErr(t, err)
	assertBoolsEqual(t, true, allowPI)

	// Assert that an item that otherwise would not be allowed PI access, gets approved because it is found in the GDPR.NonStandardPublishers array
	perms.cfg.NonStandardPublisherMap = map[string]struct{}{"appNexusAppID": {}}
	allowPI, _, _, err = perms.PersonalInfoAllowed(context.Background(), openrtb_ext.BidderAppnexus, "appNexusAppID", "BOS2bx5OS2bx5ABABBAAABoAAAABBwAA")
	assertNilErr(t, err)
	assertBoolsEqual(t, true, allowPI)
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
	description string
	bidder      openrtb_ext.BidderName
	consent     string
	allowPI     bool
	allowGeo    bool
	allowID     bool
}

func TestAllowPersonalInfoTCF2(t *testing.T) {
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
	}

	for _, td := range testDefs {
		allowPI, allowGeo, allowID, err := perms.PersonalInfoAllowed(context.Background(), td.bidder, "", td.consent)
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
	allowPI, allowGeo, allowID, err := perms.PersonalInfoAllowed(context.Background(), openrtb_ext.BidderAppnexus, "appNexusAppID", "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
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
		allowPI, allowGeo, allowID, err := perms.PersonalInfoAllowed(context.Background(), td.bidder, "", td.consent)
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
		allowPI, allowGeo, allowID, err := perms.PersonalInfoAllowed(context.Background(), td.bidder, "", td.consent)
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
		allowPI, allowGeo, allowID, err := perms.PersonalInfoAllowed(context.Background(), td.bidder, "", td.consent)
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
	allowSync, err := perms.HostCookiesAllowed(context.Background(), "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, true, allowSync, "HostCookiesAllowed failure")

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderRubicon, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
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
	allowSync, err := perms.HostCookiesAllowed(context.Background(), "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, false, allowSync, "HostCookiesAllowed failure")

	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderRubicon, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
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
	allowSync, err := perms.HostCookiesAllowed(context.Background(), "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
	assert.NoErrorf(t, err, "Error processing HostCookiesAllowed")
	assert.EqualValuesf(t, false, allowSync, "HostCookiesAllowed failure")

	// Permission disallowed due to consent string not including vendor 10.
	allowSync, err = perms.BidderSyncAllowed(context.Background(), openrtb_ext.BidderOpenx, "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA")
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
