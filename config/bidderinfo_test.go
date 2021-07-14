package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const testInfoFilesPath = "./test/bidder-info"
const testYAML = `
maintainer:
  email: "some-email@domain.com"
gvlVendorID: 42
capabilities:
  app:
    mediaTypes:
      - banner
      - native
  site:
    mediaTypes:
      - banner
      - video
      - native
`

func TestLoadBidderInfoFromDisk(t *testing.T) {
	bidder := "someBidder"

	adapterConfigs := make(map[string]Adapter)
	adapterConfigs[strings.ToLower(bidder)] = Adapter{}

	infos, err := LoadBidderInfoFromDisk(testInfoFilesPath, adapterConfigs, []string{bidder})
	if err != nil {
		t.Fatal(err)
	}

	expected := BidderInfos{
		bidder: {
			Enabled: true,
			Maintainer: &MaintainerInfo{
				Email: "some-email@domain.com",
			},
			GVLVendorID: 42,
			Capabilities: &CapabilitiesInfo{
				App: &PlatformInfo{
					MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeNative},
				},
				Site: &PlatformInfo{
					MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
				},
			},
		},
	}
	assert.Equal(t, expected, infos)
}

func TestLoadBidderInfo(t *testing.T) {
	bidder := "someBidder" // important to be mixed case for tests

	testCases := []struct {
		description   string
		givenConfigs  map[string]Adapter
		givenContent  string
		givenError    error
		expectedInfo  BidderInfos
		expectedError string
	}{
		{
			description:  "Enabled",
			givenConfigs: map[string]Adapter{strings.ToLower(bidder): {}},
			givenContent: testYAML,
			expectedInfo: map[string]BidderInfo{
				bidder: {
					Enabled: true,
					Maintainer: &MaintainerInfo{
						Email: "some-email@domain.com",
					},
					GVLVendorID: 42,
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeNative},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
						},
					},
				},
			},
		},
		{
			description:  "Disabled - Bidder Not Configured",
			givenConfigs: map[string]Adapter{},
			givenContent: testYAML,
			expectedInfo: map[string]BidderInfo{
				bidder: {
					Enabled: false,
					Maintainer: &MaintainerInfo{
						Email: "some-email@domain.com",
					},
					GVLVendorID: 42,
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeNative},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
						},
					},
				},
			},
		},
		{
			description:  "Disabled - Bidder Wrong Case",
			givenConfigs: map[string]Adapter{bidder: {}},
			givenContent: testYAML,
			expectedInfo: map[string]BidderInfo{
				bidder: {
					Enabled: false,
					Maintainer: &MaintainerInfo{
						Email: "some-email@domain.com",
					},
					GVLVendorID: 42,
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeNative},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
						},
					},
				},
			},
		},
		{
			description:  "Disabled - Explicitly Configured",
			givenConfigs: map[string]Adapter{strings.ToLower(bidder): {Disabled: false}},
			givenContent: testYAML,
			expectedInfo: map[string]BidderInfo{
				bidder: {
					Enabled: true,
					Maintainer: &MaintainerInfo{
						Email: "some-email@domain.com",
					},
					GVLVendorID: 42,
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeNative},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
						},
					},
				},
			},
		},
		{
			description:   "Read Error",
			givenConfigs:  map[string]Adapter{strings.ToLower(bidder): {}},
			givenError:    errors.New("any read error"),
			expectedError: "any read error",
		},
		{
			description:   "Unmarshal Error",
			givenConfigs:  map[string]Adapter{strings.ToLower(bidder): {}},
			givenContent:  "invalid yaml",
			expectedError: "error parsing yaml for bidder someBidder: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid...` into config.BidderInfo",
		},
	}

	for _, test := range testCases {
		r := fakeInfoReader{test.givenContent, test.givenError}
		info, err := loadBidderInfo(r, test.givenConfigs, []string{bidder})

		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}

		assert.Equal(t, test.expectedInfo, info, test.description)
	}
}

type fakeInfoReader struct {
	content string
	err     error
}

func (r fakeInfoReader) Read(bidder string) ([]byte, error) {
	return []byte(r.content), r.err
}

func TestToGVLVendorIDMap(t *testing.T) {
	givenBidderInfos := BidderInfos{
		"bidderA": BidderInfo{Enabled: true, GVLVendorID: 0},
		"bidderB": BidderInfo{Enabled: true, GVLVendorID: 100},
		"bidderC": BidderInfo{Enabled: false, GVLVendorID: 0},
		"bidderD": BidderInfo{Enabled: false, GVLVendorID: 200},
	}

	expectedGVLVendorIDMap := map[openrtb_ext.BidderName]uint16{
		"bidderB": 100,
	}

	result := givenBidderInfos.ToGVLVendorIDMap()
	assert.Equal(t, expectedGVLVendorIDMap, result)
}
