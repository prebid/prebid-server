package gdpr

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/prebid/go-gdpr/vendorlist"
)

func TestNoConsentButAllowByDefault(t *testing.T) {
	perms := permissionsImpl{
		cfg: config.GDPR{
			HostVendorID:        3,
			UsersyncIfAmbiguous: true,
		},
		vendorIDs: nil,
		fetchVendorList: map[uint8]func(ctx context.Context, id uint16) (vendorlist.VendorList, error){
			tCF1: failedListFetcher,
			tCF2: failedListFetcher,
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
			tCF1: failedListFetcher,
			tCF2: failedListFetcher,
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
	vendorListData := mockVendorListData(t, 1, map[uint16]*purposes{
		2: {
			purposes: []uint8{1},
		},
		3: {
			purposes: []uint8{1},
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
			tCF1: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
			tCF2: listFetcher(map[uint16]vendorlist.VendorList{
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
	vendorListData := mockVendorListData(t, 1, map[uint16]*purposes{
		2: {
			purposes: []uint8{1}, // cookie reads/writes
		},
		3: {
			purposes: []uint8{3}, // ad personalization
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
			tCF1: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
			tCF2: listFetcher(map[uint16]vendorlist.VendorList{
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
	vendorListData := mockVendorListData(t, 1, map[uint16]*purposes{
		2: {
			purposes: []uint8{1}, // cookie reads/writes
		},
		3: {
			purposes: []uint8{3}, // ad personalization
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
			tCF1: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
			tCF2: listFetcher(map[uint16]vendorlist.VendorList{
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
			tCF1: listFetcher(nil),
			tCF2: listFetcher(nil),
		},
	}

	sync, err := perms.HostCookiesAllowed(context.Background(), "BON")
	assertErr(t, err, true)
	assertBoolsEqual(t, false, sync)
}

func TestAllowPersonalInfo(t *testing.T) {
	vendorListData := mockVendorListData(t, 1, map[uint16]*purposes{
		2: {
			purposes: []uint8{1}, // cookie reads/writes
		},
		3: {
			purposes: []uint8{1, 3}, // ad personalization
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
			tCF1: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
			tCF2: listFetcher(map[uint16]vendorlist.VendorList{
				1: parseVendorListData(t, vendorListData),
			}),
		},
	}

	// PI needs both purposes to succeed
	allowPI, err := perms.PersonalInfoAllowed(context.Background(), openrtb_ext.BidderAppnexus, "", "BOS2bx5OS2bx5ABABBAAABoAAAABBwAA")
	assertNilErr(t, err)
	assertBoolsEqual(t, false, allowPI)

	allowPI, err = perms.PersonalInfoAllowed(context.Background(), openrtb_ext.BidderPubmatic, "", "BOS2bx5OS2bx5ABABBAAABoAAAABBwAA")
	assertNilErr(t, err)
	assertBoolsEqual(t, true, allowPI)

	// Assert that an item that otherwise would not be allowed PI access, gets approved because it is found in the GDPR.NonStandardPublishers array
	perms.cfg.NonStandardPublisherMap = map[string]int{"appNexusAppID": 1}
	allowPI, err = perms.PersonalInfoAllowed(context.Background(), openrtb_ext.BidderAppnexus, "appNexusAppID", "BOS2bx5OS2bx5ABABBAAABoAAAABBwAA")
	assertNilErr(t, err)
	assertBoolsEqual(t, true, allowPI)
}

func parseVendorListData(t *testing.T, data string) vendorlist.VendorList {
	t.Helper()
	parsed, err := vendorlist.ParseEagerly([]byte(data))
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
