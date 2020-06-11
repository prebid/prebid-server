package gdpr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
)

func TestVendorFetch(t *testing.T) {
	vendorListOne := mockVendorListData(t, 1, map[uint16]*purposes{
		32: {
			purposes: []int{1, 2},
		},
	})
	vendorListTwo := mockVendorListData(t, 2, map[uint16]*purposes{
		32: {
			purposes: []int{1, 2, 3},
		},
	})
	server := httptest.NewServer(http.HandlerFunc(mockServer(2, map[int]string{
		1: vendorListOne,
		2: vendorListTwo,
	})))
	defer server.Close()

	fetcher := newVendorListFetcher(context.Background(), testConfig(), server.Client(), testURLMaker(server), 1)
	list, err := fetcher(context.Background(), 1)
	assertNilErr(t, err)
	vendor := list.Vendor(32)
	assertBoolsEqual(t, true, vendor.Purpose(1))
	assertBoolsEqual(t, false, vendor.Purpose(3))
	assertBoolsEqual(t, false, vendor.Purpose(4))

	list, err = fetcher(context.Background(), 2)
	assertNilErr(t, err)
	vendor = list.Vendor(32)
	assertBoolsEqual(t, true, vendor.Purpose(1))
	assertBoolsEqual(t, true, vendor.Purpose(3))
}

func TestLazyFetch(t *testing.T) {
	firstVendorList := mockVendorListData(t, 1, map[uint16]*purposes{
		32: {
			purposes: []int{1, 2},
		},
	})
	secondVendorList := mockVendorListData(t, 2, map[uint16]*purposes{
		3: {
			purposes: []int{1},
		},
	})
	server := httptest.NewServer(http.HandlerFunc(mockServer(1, map[int]string{
		1: firstVendorList,
		2: secondVendorList,
	})))
	defer server.Close()

	fetcher := newVendorListFetcher(context.Background(), testConfig(), server.Client(), testURLMaker(server), 1)
	list, err := fetcher(context.Background(), 2)
	assertNilErr(t, err)

	vendor := list.Vendor(3)
	assertBoolsEqual(t, true, vendor.Purpose(1))
	assertBoolsEqual(t, false, vendor.Purpose(2))
}

func TestInitialTimeout(t *testing.T) {
	list := mockVendorListData(t, 1, map[uint16]*purposes{
		32: {
			purposes: []int{1, 2},
		},
	})
	server := httptest.NewServer(http.HandlerFunc(mockServer(1, map[int]string{
		1: list,
	})))
	defer server.Close()

	ctx, cancel := context.WithDeadline(context.Background(), time.Time{})
	defer cancel()
	fetcher := newVendorListFetcher(ctx, testConfig(), server.Client(), testURLMaker(server), 1)
	_, err := fetcher(context.Background(), 1) // This should do a lazy fetch, even though the initial call failed
	assertNilErr(t, err)
}

func TestFetchThrottling(t *testing.T) {
	vendorListTwo := mockVendorListData(t, 2, map[uint16]*purposes{
		32: {
			purposes: []int{1, 2},
		},
	})
	vendorListThree := mockVendorListData(t, 3, map[uint16]*purposes{
		32: {
			purposes: []int{1, 2},
		},
	})
	server := httptest.NewServer(http.HandlerFunc(mockServer(1, map[int]string{
		1: "{}",
		2: vendorListTwo,
		3: vendorListThree,
	})))
	defer server.Close()

	fetcher := newVendorListFetcher(context.Background(), testConfig(), server.Client(), testURLMaker(server), 1)
	_, err := fetcher(context.Background(), 2)
	assertNilErr(t, err)
	_, err = fetcher(context.Background(), 3)
	assertErr(t, err, false)
}

func TestMalformedVendorlistFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mockServer(1, map[int]string{1: "{}"})))
	defer server.Close()

	fetcher := newVendorListFetcher(context.Background(), testConfig(), server.Client(), testURLMaker(server), 1)
	_, err := fetcher(context.Background(), 1)
	assertErr(t, err, false)
}

func TestMissingVendorlistFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mockServer(1, map[int]string{1: "{}"})))
	defer server.Close()

	fetcher := newVendorListFetcher(context.Background(), testConfig(), server.Client(), testURLMaker(server), 1)
	_, err := fetcher(context.Background(), 2)
	assertErr(t, err, false)
}

func TestVendorListMaker(t *testing.T) {
	assertStringsEqual(t, "https://vendorlist.consensu.org/vendorlist.json", vendorListURLMaker(0, 1))
	assertStringsEqual(t, "https://vendorlist.consensu.org/v-2/vendorlist.json", vendorListURLMaker(2, 1))
	assertStringsEqual(t, "https://vendorlist.consensu.org/v-12/vendorlist.json", vendorListURLMaker(12, 1))
	assertStringsEqual(t, "https://vendorlist.consensu.org/v2/vendor-list.json", vendorListURLMaker(0, 2))
	assertStringsEqual(t, "https://vendorlist.consensu.org/v2/archives/vendor-list-v7.json", vendorListURLMaker(7, 2))
}

// mockServer returns a handler which returns the given response for each global vendor list version.
// The latestVersion param can be used to mock "updates" which occur after PBS has been turned on.
// For example, if latestVersion is 3, but the responses map has data at "4", the server will return
// version "3" when asked for the latest version.
//
// This will help test lazy-fetches for versions which aren't there on app startup.
//
// If the "version" query param doesn't exist, it returns a 400.
//
// If the "version" query param points to a version which doesn't exist, it returns a 403.
// Don't ask why... that's just what the official page is doing. See https://vendorlist.consensu.org/v-9999/vendorlist.json
func mockServer(latestVersion int, responses map[int]string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		version := req.URL.Query().Get("version")
		versionInt, err := strconv.Atoi(version)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Request had invalid version: " + version))
			return
		}
		if versionInt == 0 {
			versionInt = latestVersion
		}
		response, ok := responses[versionInt]
		if !ok {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Version not found: " + version))
			return
		}
		w.Write([]byte(response))
	}
}

func mockVendorListData(t *testing.T, version uint16, vendors map[uint16]*purposes) string {
	type vendorContract struct {
		ID       uint16 `json:"id"`
		Purposes []int  `json:"purposeIds"`
	}

	type vendorListContract struct {
		Version uint16           `json:"vendorListVersion"`
		Vendors []vendorContract `json:"vendors"`
	}

	buildVendors := func(input map[uint16]*purposes) []vendorContract {
		vendors := make([]vendorContract, 0, len(input))
		for id, purpose := range input {
			vendors = append(vendors, vendorContract{
				ID:       id,
				Purposes: purpose.purposes,
			})
		}
		return vendors
	}

	obj := vendorListContract{
		Version: version,
		Vendors: buildVendors(vendors),
	}
	data, err := json.Marshal(obj)
	assertNilErr(t, err)
	return string(data)
}

type purposeMap map[uint16]*purposes

func mockVendorListDataTCF2(t *testing.T, version uint16, basicPurposes purposeMap, legitInterests purposeMap, flexPurposes purposeMap, specialPurposes purposeMap) string {
	type vendorContract struct {
		ID               uint16 `json:"id"`
		Purposes         []int  `json:"purposes"`
		LegIntPurposes   []int  `json:"legIntPurposes"`
		FlexiblePurposes []int  `json:"flexiblePurposes"`
		SpecialPurposes  []int  `json:"specialPurposes"`
	}

	type vendorListContract struct {
		Version uint16                    `json:"vendorListVersion"`
		Vendors map[string]vendorContract `json:"vendors"`
	}

	vendors := make(map[string]vendorContract, len(basicPurposes))
	for id, purpose := range basicPurposes {
		sid := strconv.Itoa(int(id))
		vendor, ok := vendors[sid]
		if !ok {
			vendor = vendorContract{ID: id}
		}
		vendor.Purposes = purpose.purposes
		vendors[sid] = vendor
	}

	for id, purpose := range legitInterests {
		sid := strconv.Itoa(int(id))
		vendor, ok := vendors[sid]
		if !ok {
			vendor = vendorContract{ID: id}
		}
		vendor.LegIntPurposes = purpose.purposes
		vendors[sid] = vendor
	}

	for id, purpose := range flexPurposes {
		sid := strconv.Itoa(int(id))
		vendor, ok := vendors[sid]
		if !ok {
			vendor = vendorContract{ID: id}
		}
		vendor.FlexiblePurposes = purpose.purposes
		vendors[sid] = vendor
	}

	for id, purpose := range specialPurposes {
		sid := strconv.Itoa(int(id))
		vendor, ok := vendors[sid]
		if !ok {
			vendor = vendorContract{ID: id}
		}
		vendor.SpecialPurposes = purpose.purposes
		vendors[sid] = vendor
	}

	obj := vendorListContract{
		Version: version,
		Vendors: vendors,
	}
	data, err := json.Marshal(obj)
	assertNilErr(t, err)
	return string(data)
}

func testURLMaker(server *httptest.Server) func(uint16, uint8) string {
	url := server.URL
	return func(version uint16, TCFVer uint8) string {
		return url + "?version=" + strconv.Itoa(int(version))
	}
}

func testConfig() config.GDPR {
	return config.GDPR{
		Timeouts: config.GDPRTimeouts{
			InitVendorlistFetch:   60 * 1000,
			ActiveVendorlistFetch: 1000 * 5,
		},
	}
}

type purposes struct {
	purposes []int
}
