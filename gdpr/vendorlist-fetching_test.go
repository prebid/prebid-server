package gdpr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

func TestFetcherDynamicLoadListExists(t *testing.T) {
	// Loads the first vendor list during initialization by setting the latest vendor list version to 1.
	// All other vendor lists will be dynamically loaded.

	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]map[int]string{
			3: {
				1: vendorList1,
				2: vendorList2,
			},
		},
	})))
	defer server.Close()

	test := test{
		description: "Dynamic Load - List Exists",
		setup: testSetup{
			specVersion: 3,
			listVersion: 2,
		},
		expected: vendorList2Expected,
	}

	runTest(t, test, server)
}

func TestFetcherDynamicLoadListDoesntExist(t *testing.T) {
	// Loads the first vendor list during initialization by setting the latest vendor list version to 1.
	// All other vendor list load attempts will be done dynamically.

	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]map[int]string{
			3: {
				1: vendorList1,
			},
		},
	})))
	defer server.Close()

	test := test{
		description: "No Fallback - Vendor Doesn't Exist",
		setup: testSetup{
			specVersion: 3,
			listVersion: 2,
		},
		expected: testExpected{
			errorMessage: "gdpr vendor list spec version 3 list version 2 does not exist, or has not been loaded yet. Try again in a few minutes",
		},
	}

	runTest(t, test, server)
}

func TestFetcherThrottling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]map[int]string{
			3: {
				1: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 3,
					VendorListVersion:       1,
					Vendors:                 map[string]*vendor{"12": {ID: 12, Purposes: []int{1}}},
				}),
				2: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 3,
					VendorListVersion:       2,
					Vendors:                 map[string]*vendor{"12": {ID: 12, Purposes: []int{1, 2}}},
				}),
				3: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 3,
					VendorListVersion:       3,
					Vendors:                 map[string]*vendor{"12": {ID: 12, Purposes: []int{1, 2, 3}}},
				}),
			},
		},
	})))
	defer server.Close()

	fetcher := NewVendorListFetcher(context.Background(), testConfig(), server.Client(), testURLMaker(server))

	// Dynamically Load List 2 Successfully
	_, errList1 := fetcher(context.Background(), 3, 2)
	assert.NoError(t, errList1)

	// Fail To Load List 3 Due To Rate Limiting
	// - The request is rate limited after dynamically list 2.
	_, errList2 := fetcher(context.Background(), 3, 3)
	assert.EqualError(t, errList2, "gdpr vendor list spec version 3 list version 3 does not exist, or has not been loaded yet. Try again in a few minutes")
}

func TestMalformedVendorlist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 1,
		vendorLists: map[int]map[int]string{
			3: {
				1: "malformed",
			},
		},
	})))
	defer server.Close()

	fetcher := NewVendorListFetcher(context.Background(), testConfig(), server.Client(), testURLMaker(server))
	_, err := fetcher(context.Background(), 3, 1)

	// Fetching should fail since vendor list could not be unmarshalled.
	assert.Error(t, err)
}

func TestServerUrlInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	invalidURLGenerator := func(uint16, uint16) string { return " http://invalid-url-has-leading-whitespace" }

	fetcher := NewVendorListFetcher(context.Background(), testConfig(), server.Client(), invalidURLGenerator)
	_, err := fetcher(context.Background(), 3, 1)

	assert.EqualError(t, err, "gdpr vendor list spec version 3 list version 1 does not exist, or has not been loaded yet. Try again in a few minutes")
}

func TestServerUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	fetcher := NewVendorListFetcher(context.Background(), testConfig(), server.Client(), testURLMaker(server))
	_, err := fetcher(context.Background(), 3, 1)

	assert.EqualError(t, err, "gdpr vendor list spec version 3 list version 1 does not exist, or has not been loaded yet. Try again in a few minutes")
}

func TestVendorListURLMaker(t *testing.T) {
	testCases := []struct {
		description string
		specVersion uint16
		listVersion uint16
		expectedURL string
	}{
		{
			description: "Spec version 2 latest list",
			specVersion: 2,
			listVersion: 0,
			expectedURL: "https://vendor-list.consensu.org/v2/vendor-list.json",
		},
		{
			description: "Spec version 2 specific list",
			specVersion: 2,
			listVersion: 42,
			expectedURL: "https://vendor-list.consensu.org/v2/archives/vendor-list-v42.json",
		},
		{
			description: "Spec version 3 latest list",
			specVersion: 3,
			listVersion: 0,
			expectedURL: "https://vendor-list.consensu.org/v3/vendor-list.json",
		},
		{
			description: "Spec version 3 specific list",
			specVersion: 3,
			listVersion: 42,
			expectedURL: "https://vendor-list.consensu.org/v3/archives/vendor-list-v42.json",
		},
	}

	for _, test := range testCases {
		result := VendorListURLMaker(test.specVersion, test.listVersion)
		assert.Equal(t, test.expectedURL, result)
	}
}

type versionInfo struct {
	specVersion uint16
	listVersion uint16
}
type saver []versionInfo

func (s *saver) saveVendorLists(specVersion uint16, listVersion uint16, gvl api.VendorList) {
	vi := versionInfo{
		specVersion: specVersion,
		listVersion: listVersion,
	}
	*s = append(*s, vi)
}

func TestPreloadCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mockServer(serverSettings{
		vendorListLatestVersion: 3,
		vendorLists: map[int]map[int]string{
			1: {
				1: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 1, VendorListVersion: 1,
				}),
				2: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 1, VendorListVersion: 2,
				}),
				3: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 1, VendorListVersion: 3,
				}),
			},
			2: {
				1: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 2, VendorListVersion: 1,
				}),
				2: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 2, VendorListVersion: 2,
				}),
				3: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 2, VendorListVersion: 3,
				}),
			},
			3: {
				1: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 3, VendorListVersion: 1,
				}),
				2: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 3, VendorListVersion: 2,
				}),
				3: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 3, VendorListVersion: 3,
				}),
			},
			4: {
				1: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 4, VendorListVersion: 1,
				}),
				2: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 4, VendorListVersion: 2,
				}),
				3: MarshalVendorList(vendorList{
					GVLSpecificationVersion: 4, VendorListVersion: 3,
				}),
			},
		},
	})))
	defer server.Close()

	s := make(saver, 0, 5)
	preloadCache(context.Background(), server.Client(), testURLMaker(server), s.saveVendorLists)

	expectedLoadedVersions := []versionInfo{
		{specVersion: 2, listVersion: 2},
		{specVersion: 2, listVersion: 3},
		{specVersion: 3, listVersion: 1},
		{specVersion: 3, listVersion: 2},
		{specVersion: 3, listVersion: 3},
	}

	assert.ElementsMatch(t, expectedLoadedVersions, s)
}

var vendorList1 = MarshalVendorList(vendorList{
	GVLSpecificationVersion: 3,
	VendorListVersion:       1,
	Vendors:                 map[string]*vendor{"12": {ID: 12, Purposes: []int{2}}},
})

var vendorList2 = MarshalVendorList(vendorList{
	GVLSpecificationVersion: 3,
	VendorListVersion:       2,
	Vendors:                 map[string]*vendor{"12": {ID: 12, Purposes: []int{2, 3}}},
})

var vendorList2Expected = testExpected{
	vendorListVersion: 2,
	vendorID:          12,
	vendorPurposes:    map[int]bool{1: false, 2: true, 3: true},
}

var vendorListFallbackExpected = testExpected{
	vendorListVersion: 215, // Values from hardcoded fallback file.
	vendorID:          12,
	vendorPurposes:    map[int]bool{1: true, 2: false, 3: true},
}

type vendorList struct {
	GVLSpecificationVersion uint16             `json:"gvlSpecificationVersion"`
	VendorListVersion       uint16             `json:"vendorListVersion"`
	Vendors                 map[string]*vendor `json:"vendors"`
}

type vendor struct {
	ID               uint16 `json:"id"`
	Purposes         []int  `json:"purposes"`
	LegIntPurposes   []int  `json:"legIntPurposes"`
	FlexiblePurposes []int  `json:"flexiblePurposes"`
	SpecialFeatures  []int  `json:"specialFeatures"`
}

func MarshalVendorList(vendorList vendorList) string {
	json, _ := jsonutil.Marshal(vendorList)
	return string(json)
}

type serverSettings struct {
	vendorListLatestVersion int
	vendorLists             map[int]map[int]string
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
// Don't ask why... that's just what the official page is doing. See https://vendor-list.consensu.org/v-9999/vendorlist.json
func mockServer(settings serverSettings) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		specVersion := req.URL.Query().Get("specversion")
		specVersionInt, err := strconv.Atoi(specVersion)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Request had invalid spec version: " + specVersion))
			return
		}
		listVersion := req.URL.Query().Get("listversion")
		listVersionInt, err := strconv.Atoi(listVersion)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Request had invalid version: " + listVersion))
			return
		}
		if listVersionInt == 0 {
			listVersionInt = settings.vendorListLatestVersion
		}
		specVersionVendorLists, ok := settings.vendorLists[specVersionInt]
		if !ok {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Version not found: spec version " + specVersion + " list version " + listVersion))
			return
		}
		response, ok := specVersionVendorLists[listVersionInt]
		if !ok {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Version not found: " + listVersion))
			return
		}
		w.Write([]byte(response))
	}
}

type test struct {
	description string
	setup       testSetup
	expected    testExpected
}

type testSetup struct {
	specVersion uint16
	listVersion uint16
}

type testExpected struct {
	errorMessage      string
	vendorListVersion uint16
	vendorID          uint16
	vendorPurposes    map[int]bool
}

func runTest(t *testing.T, test test, server *httptest.Server) {
	config := testConfig()
	fetcher := NewVendorListFetcher(context.Background(), config, server.Client(), testURLMaker(server))
	vendorList, err := fetcher(context.Background(), test.setup.specVersion, test.setup.listVersion)

	if test.expected.errorMessage != "" {
		assert.EqualError(t, err, test.expected.errorMessage, test.description+":error")
	} else {
		assert.NoError(t, err, test.description+":vendorlist")
		assert.Equal(t, test.expected.vendorListVersion, vendorList.Version(), test.description+":vendorlistid")
		vendor := vendorList.Vendor(test.expected.vendorID)
		for id, expected := range test.expected.vendorPurposes {
			result := vendor.Purpose(consentconstants.Purpose(id))
			assert.Equalf(t, expected, result, "%s:vendor-%d:purpose-%d", test.description, vendorList.Version(), id)
		}
	}
}

func testURLMaker(server *httptest.Server) func(uint16, uint16) string {
	url := server.URL
	return func(specVersion, listVersion uint16) string {
		return url + "?specversion=" + strconv.Itoa(int(specVersion)) + "&listversion=" + strconv.Itoa(int(listVersion))
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
