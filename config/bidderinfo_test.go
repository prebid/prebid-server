package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const testInfoFilesPath = "./test/bidder-info"
const testSimpleYAML = `
maintainer:
  email: "some-email@domain.com"
gvlVendorID: 42
`
const fullBidderYAMLConfig = `
maintainer:
  email: "some-email@domain.com"
capabilities:
  app:
    mediaTypes:
      - banner
      - video
      - native
  site:
    mediaTypes:
      - banner
      - video
      - native
modifyingVastXmlAllowed: true
debug:
  allow: true
gvlVendorID: 42
userSync:
  iframe:
    url: "someURL"
    userMacro: "aValue"
  redirect:
    url: "anotherURL"
    userMacro: "anotherValue"
  key: "aKey"
  supports:
    - item
    - item2
  externalUrl: oneMoreUrl
  supportCors: true
experiment:
  adsCert:
    enabled: true
endpointCompression: "GZIP"
`

func TestLoadBidderInfoFromDisk(t *testing.T) {
	bidder := "someBidder"
	trueValue := true

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
			Syncer: &Syncer{
				Key: "foo",
				IFrame: &SyncerEndpoint{
					URL:         "https://foo.com/sync?mode=iframe&r={{.RedirectURL}}",
					RedirectURL: "{{.ExternalURL}}/setuid/iframe",
					ExternalURL: "https://iframe.host",
					UserMacro:   "%UID",
				},
				Redirect: &SyncerEndpoint{
					URL:         "https://foo.com/sync?mode=redirect&r={{.RedirectURL}}",
					RedirectURL: "{{.ExternalURL}}/setuid/redirect",
					ExternalURL: "https://redirect.host",
					UserMacro:   "#UID",
				},
				SupportCORS: &trueValue,
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
			givenContent: testSimpleYAML,
			expectedInfo: map[string]BidderInfo{
				bidder: {
					Enabled: true,
					Maintainer: &MaintainerInfo{
						Email: "some-email@domain.com",
					},
					GVLVendorID: 42,
				},
			},
		},
		{
			description:  "Disabled - Bidder Not Configured",
			givenConfigs: map[string]Adapter{},
			givenContent: testSimpleYAML,
			expectedInfo: map[string]BidderInfo{
				bidder: {
					Enabled: false,
					Maintainer: &MaintainerInfo{
						Email: "some-email@domain.com",
					},
					GVLVendorID: 42,
				},
			},
		},
		{
			description:  "Disabled - Bidder Wrong Case",
			givenConfigs: map[string]Adapter{bidder: {}},
			givenContent: testSimpleYAML,
			expectedInfo: map[string]BidderInfo{
				bidder: {
					Enabled: false,
					Maintainer: &MaintainerInfo{
						Email: "some-email@domain.com",
					},
					GVLVendorID: 42,
				},
			},
		},
		{
			description:  "Disabled - Explicitly Configured",
			givenConfigs: map[string]Adapter{strings.ToLower(bidder): {Disabled: false}},
			givenContent: testSimpleYAML,
			expectedInfo: map[string]BidderInfo{
				bidder: {
					Enabled: true,
					Maintainer: &MaintainerInfo{
						Email: "some-email@domain.com",
					},
					GVLVendorID: 42,
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

func TestSyncerOverride(t *testing.T) {
	var (
		trueValue  = true
		falseValue = false
	)

	testCases := []struct {
		description   string
		givenOriginal *Syncer
		givenOverride *Syncer
		expected      *Syncer
	}{
		{
			description:   "Nil",
			givenOriginal: nil,
			givenOverride: nil,
			expected:      nil,
		},
		{
			description:   "Original Nil",
			givenOriginal: nil,
			givenOverride: &Syncer{Key: "anyKey"},
			expected:      &Syncer{Key: "anyKey"},
		},
		{
			description:   "Original Empty",
			givenOriginal: &Syncer{},
			givenOverride: &Syncer{Key: "anyKey"},
			expected:      &Syncer{Key: "anyKey"},
		},
		{
			description:   "Override Nil",
			givenOriginal: &Syncer{Key: "anyKey"},
			givenOverride: nil,
			expected:      &Syncer{Key: "anyKey"},
		},
		{
			description:   "Override Empty",
			givenOriginal: &Syncer{Key: "anyKey"},
			givenOverride: &Syncer{},
			expected:      &Syncer{Key: "anyKey"},
		},
		{
			description:   "Override Key",
			givenOriginal: &Syncer{Key: "original"},
			givenOverride: &Syncer{Key: "override"},
			expected:      &Syncer{Key: "override"},
		},
		{
			description:   "Override IFrame",
			givenOriginal: &Syncer{IFrame: &SyncerEndpoint{URL: "original"}},
			givenOverride: &Syncer{IFrame: &SyncerEndpoint{URL: "override"}},
			expected:      &Syncer{IFrame: &SyncerEndpoint{URL: "override"}},
		},
		{
			description:   "Override Redirect",
			givenOriginal: &Syncer{Redirect: &SyncerEndpoint{URL: "original"}},
			givenOverride: &Syncer{Redirect: &SyncerEndpoint{URL: "override"}},
			expected:      &Syncer{Redirect: &SyncerEndpoint{URL: "override"}},
		},
		{
			description:   "Override ExternalURL",
			givenOriginal: &Syncer{ExternalURL: "original"},
			givenOverride: &Syncer{ExternalURL: "override"},
			expected:      &Syncer{ExternalURL: "override"},
		},
		{
			description:   "Override SupportCORS",
			givenOriginal: &Syncer{SupportCORS: &trueValue},
			givenOverride: &Syncer{SupportCORS: &falseValue},
			expected:      &Syncer{SupportCORS: &falseValue},
		},
		{
			description:   "Override Partial - Other Fields Untouched",
			givenOriginal: &Syncer{Key: "originalKey", ExternalURL: "originalExternalURL"},
			givenOverride: &Syncer{ExternalURL: "overrideExternalURL"},
			expected:      &Syncer{Key: "originalKey", ExternalURL: "overrideExternalURL"},
		},
	}

	for _, test := range testCases {
		result := test.givenOverride.Override(test.givenOriginal)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestSyncerEndpointOverride(t *testing.T) {
	testCases := []struct {
		description   string
		givenOriginal *SyncerEndpoint
		givenOverride *SyncerEndpoint
		expected      *SyncerEndpoint
	}{
		{
			description:   "Nil",
			givenOriginal: nil,
			givenOverride: nil,
			expected:      nil,
		},
		{
			description:   "Original Nil",
			givenOriginal: nil,
			givenOverride: &SyncerEndpoint{URL: "anyURL"},
			expected:      &SyncerEndpoint{URL: "anyURL"},
		},
		{
			description:   "Original Empty",
			givenOriginal: &SyncerEndpoint{},
			givenOverride: &SyncerEndpoint{URL: "anyURL"},
			expected:      &SyncerEndpoint{URL: "anyURL"},
		},
		{
			description:   "Override Nil",
			givenOriginal: &SyncerEndpoint{URL: "anyURL"},
			givenOverride: nil,
			expected:      &SyncerEndpoint{URL: "anyURL"},
		},
		{
			description:   "Override Empty",
			givenOriginal: &SyncerEndpoint{URL: "anyURL"},
			givenOverride: &SyncerEndpoint{},
			expected:      &SyncerEndpoint{URL: "anyURL"},
		},
		{
			description:   "Override URL",
			givenOriginal: &SyncerEndpoint{URL: "original"},
			givenOverride: &SyncerEndpoint{URL: "override"},
			expected:      &SyncerEndpoint{URL: "override"},
		},
		{
			description:   "Override RedirectURL",
			givenOriginal: &SyncerEndpoint{RedirectURL: "original"},
			givenOverride: &SyncerEndpoint{RedirectURL: "override"},
			expected:      &SyncerEndpoint{RedirectURL: "override"},
		},
		{
			description:   "Override ExternalURL",
			givenOriginal: &SyncerEndpoint{ExternalURL: "original"},
			givenOverride: &SyncerEndpoint{ExternalURL: "override"},
			expected:      &SyncerEndpoint{ExternalURL: "override"},
		},
		{
			description:   "Override UserMacro",
			givenOriginal: &SyncerEndpoint{UserMacro: "original"},
			givenOverride: &SyncerEndpoint{UserMacro: "override"},
			expected:      &SyncerEndpoint{UserMacro: "override"},
		},
		{
			description:   "Override",
			givenOriginal: &SyncerEndpoint{URL: "originalURL", RedirectURL: "originalRedirectURL", ExternalURL: "originalExternalURL", UserMacro: "originalUserMacro"},
			givenOverride: &SyncerEndpoint{URL: "overideURL", RedirectURL: "overideRedirectURL", ExternalURL: "overideExternalURL", UserMacro: "overideUserMacro"},
			expected:      &SyncerEndpoint{URL: "overideURL", RedirectURL: "overideRedirectURL", ExternalURL: "overideExternalURL", UserMacro: "overideUserMacro"},
		},
	}

	for _, test := range testCases {
		result := test.givenOverride.Override(test.givenOriginal)
		assert.Equal(t, test.expected, result, test.description)
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

func TestReadFullYamlBidderConfig(t *testing.T) {
	bidder := "someBidder"
	trueValue := true
	r := fakeInfoReader{fullBidderYAMLConfig, nil}
	actualBidderInfo, err := loadBidderInfo(r, map[string]Adapter{strings.ToLower(bidder): {}}, []string{bidder})
	assert.NoError(t, err, "Error wasn't expected")

	expectedBidderInfo := BidderInfos{
		bidder: {
			Enabled: true,
			Maintainer: &MaintainerInfo{
				Email: "some-email@domain.com",
			},
			GVLVendorID: 42,
			Capabilities: &CapabilitiesInfo{
				App: &PlatformInfo{
					MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
				},
				Site: &PlatformInfo{
					MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
				},
			},
			Debug:                   &DebugInfo{Allow: true},
			ModifyingVastXmlAllowed: true,
			Syncer: &Syncer{
				Key: "aKey",
				IFrame: &SyncerEndpoint{
					URL:       "someURL",
					UserMacro: "aValue",
				},
				Redirect: &SyncerEndpoint{
					URL:       "anotherURL",
					UserMacro: "anotherValue",
				},
				Supports:    []string{"item", "item2"},
				SupportCORS: &trueValue,
				ExternalURL: "oneMoreUrl",
			},
			Experiment:          BidderInfoExperiment{AdsCert: BidderAdsCert{Enabled: true}},
			EndpointCompression: "GZIP",
		},
	}
	assert.Equalf(t, expectedBidderInfo, actualBidderInfo, "Bidder info objects aren't matching")
}
