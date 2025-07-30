package config

import (
	"errors"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testInfoFilesPathValid = "./test/bidder-info-valid"
const testSimpleYAML = `
maintainer:
  email: some-email@domain.com
gvlVendorID: 42
`
const fullBidderYAMLConfig = `
maintainer:
  email: some-email@domain.com
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
  dooh:
    mediaTypes:
      - banner
modifyingVastXmlAllowed: true
debug:
  allow: true
gvlVendorID: 42
experiment:
  adsCert:
    enabled: true
endpointCompression: GZIP
openrtb:
  version: 2.6
  gpp-supported: true
  multiformat-supported: false
endpoint: https://endpoint.com
disabled: false
extra_info: extra-info
app_secret: app-secret
platform_id: 123
userSync:
  key: foo
  default: iframe
  iframe:
    url: https://foo.com/sync?mode=iframe&r={{.RedirectURL}}
    redirectUrl: https://redirect/setuid/iframe
    externalUrl: https://iframe.host
    userMacro: UID
xapi:
  username: uname
  password: pwd
  tracker: tracker
`
const testSimpleAliasYAML = `
aliasOf: bidderA
`

func TestLoadBidderInfoFromDisk(t *testing.T) {
	// should appear in result in mixed case
	bidder := "stroeerCore"
	trueValue := true

	adapterConfigs := make(map[string]Adapter)
	adapterConfigs[strings.ToLower(bidder)] = Adapter{}

	infos, err := LoadBidderInfoFromDisk(testInfoFilesPathValid)
	if err != nil {
		t.Fatal(err)
	}

	expected := BidderInfos{
		bidder: {
			Disabled: false,
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
				DOOH: &PlatformInfo{
					MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo},
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

func TestProcessBidderInfo(t *testing.T) {
	falseValue := false

	testCases := []struct {
		description         string
		bidderInfos         map[string][]byte
		expectedBidderInfos BidderInfos
		expectError         string
	}{

		{
			description: "Valid bidder info",
			bidderInfos: map[string][]byte{
				"bidderA.yaml": []byte(testSimpleYAML),
			},
			expectedBidderInfos: BidderInfos{
				"bidderA": BidderInfo{
					Maintainer: &MaintainerInfo{
						Email: "some-email@domain.com",
					},
					GVLVendorID: 42,
				},
			},
			expectError: "",
		},
		{
			description: "Bidder doesn't exist in bidder info list",
			bidderInfos: map[string][]byte{
				"unknown.yaml": []byte(testSimpleYAML),
			},
			expectedBidderInfos: nil,
			expectError:         "error parsing config for bidder unknown.yaml",
		},
		{
			description: "Invalid bidder config",
			bidderInfos: map[string][]byte{
				"bidderA.yaml": []byte("invalid bidder config"),
			},
			expectedBidderInfos: nil,
			expectError:         "error parsing config for bidder bidderA.yaml",
		},
		{
			description: "Invalid alias name",
			bidderInfos: map[string][]byte{
				"all.yaml": []byte(testSimpleAliasYAML),
			},
			expectedBidderInfos: nil,
			expectError:         "alias all is a reserved bidder name and cannot be used",
		},
		{
			description: "Valid aliases",
			bidderInfos: map[string][]byte{
				"bidderA.yaml": []byte(fullBidderYAMLConfig),
				"bidderB.yaml": []byte(testSimpleAliasYAML),
			},
			expectedBidderInfos: BidderInfos{
				"bidderA": BidderInfo{
					AppSecret: "app-secret",
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
						},
						DOOH: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
						},
					},
					Debug: &DebugInfo{
						Allow: true,
					},
					Disabled:            false,
					Endpoint:            "https://endpoint.com",
					EndpointCompression: "GZIP",
					Experiment: BidderInfoExperiment{
						AdsCert: BidderAdsCert{
							Enabled: true,
						},
					},
					ExtraAdapterInfo: "extra-info",
					GVLVendorID:      42,
					Maintainer: &MaintainerInfo{
						Email: "some-email@domain.com",
					},
					ModifyingVastXmlAllowed: true,
					OpenRTB: &OpenRTBInfo{
						GPPSupported:         true,
						Version:              "2.6",
						MultiformatSupported: &falseValue,
					},
					PlatformID: "123",
					Syncer: &Syncer{
						Key: "foo",
						IFrame: &SyncerEndpoint{
							URL:         "https://foo.com/sync?mode=iframe&r={{.RedirectURL}}",
							RedirectURL: "https://redirect/setuid/iframe",
							ExternalURL: "https://iframe.host",
							UserMacro:   "UID",
						},
					},
					XAPI: AdapterXAPI{
						Username: "uname",
						Password: "pwd",
						Tracker:  "tracker",
					},
				},
				"bidderB": BidderInfo{
					AliasOf:   "bidderA",
					AppSecret: "app-secret",
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
						},
						DOOH: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
						},
					},
					Debug: &DebugInfo{
						Allow: true,
					},
					Disabled:            false,
					Endpoint:            "https://endpoint.com",
					EndpointCompression: "GZIP",
					Experiment: BidderInfoExperiment{
						AdsCert: BidderAdsCert{
							Enabled: true,
						},
					},
					ExtraAdapterInfo: "extra-info",
					GVLVendorID:      42,
					Maintainer: &MaintainerInfo{
						Email: "some-email@domain.com",
					},
					ModifyingVastXmlAllowed: true,
					OpenRTB: &OpenRTBInfo{
						GPPSupported:         true,
						Version:              "2.6",
						MultiformatSupported: &falseValue,
					},
					PlatformID: "123",
					Syncer: &Syncer{
						Key: "foo",
					},
					XAPI: AdapterXAPI{
						Username: "uname",
						Password: "pwd",
						Tracker:  "tracker",
					},
				},
			},
		},
	}

	for _, test := range testCases {
		reader := StubInfoReader{test.bidderInfos}
		bidderInfos, err := processBidderInfos(reader, mockNormalizeBidderName)
		if test.expectError != "" {
			assert.ErrorContains(t, err, test.expectError, "")
		} else {
			assert.Equal(t, test.expectedBidderInfos, bidderInfos, "incorrect bidder infos for test case: %s", test.description)
		}
	}
}

func TestProcessAliasBidderInfo(t *testing.T) {

	trueValue := true

	parentWithSyncerKey := BidderInfo{
		AppSecret: "app-secret",
		Capabilities: &CapabilitiesInfo{
			App: &PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
			},
			Site: &PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative},
			},
		},
		Debug: &DebugInfo{
			Allow: true,
		},
		Disabled:            false,
		Endpoint:            "https://endpoint.com",
		EndpointCompression: "GZIP",
		Experiment: BidderInfoExperiment{
			AdsCert: BidderAdsCert{
				Enabled: true,
			},
		},
		ExtraAdapterInfo: "extra-info",
		GVLVendorID:      42,
		Maintainer: &MaintainerInfo{
			Email: "some-email@domain.com",
		},
		ModifyingVastXmlAllowed: true,
		OpenRTB: &OpenRTBInfo{
			GPPSupported:         true,
			Version:              "2.6",
			MultiformatSupported: &trueValue,
		},
		PlatformID: "123",
		Syncer: &Syncer{
			Key: "foo",
			IFrame: &SyncerEndpoint{
				URL:         "https://foo.com/sync?mode=iframe&r={{.RedirectURL}}",
				RedirectURL: "https://redirect/setuid/iframe",
				ExternalURL: "https://iframe.host",
				UserMacro:   "UID",
			},
		},
		XAPI: AdapterXAPI{
			Username: "uname",
			Password: "pwd",
			Tracker:  "tracker",
		},
	}
	aliasBidderInfo := BidderInfo{
		AppSecret: "alias-app-secret",
		Capabilities: &CapabilitiesInfo{
			App: &PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
			Site: &PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
		},
		Debug: &DebugInfo{
			Allow: false,
		},
		Disabled:            true,
		Endpoint:            "https://alias-endpoint.com",
		EndpointCompression: "DEFAULT",
		Experiment: BidderInfoExperiment{
			AdsCert: BidderAdsCert{
				Enabled: false,
			},
		},
		ExtraAdapterInfo: "alias-extra-info",
		GVLVendorID:      43,
		Maintainer: &MaintainerInfo{
			Email: "alias-email@domain.com",
		},
		ModifyingVastXmlAllowed: false,
		OpenRTB: &OpenRTBInfo{
			GPPSupported:         false,
			Version:              "2.5",
			MultiformatSupported: &trueValue,
		},
		PlatformID: "456",
		Syncer: &Syncer{
			Key: "alias",
			IFrame: &SyncerEndpoint{
				URL:         "https://alias.com/sync?mode=iframe&r={{.RedirectURL}}",
				RedirectURL: "https://alias-redirect/setuid/iframe",
				ExternalURL: "https://alias-iframe.host",
				UserMacro:   "alias-UID",
			},
		},
		XAPI: AdapterXAPI{
			Username: "alias-uname",
			Password: "alias-pwd",
			Tracker:  "alias-tracker",
		},
	}
	bidderB := parentWithSyncerKey
	bidderB.AliasOf = "bidderA"
	bidderB.Syncer = &Syncer{
		Key: bidderB.Syncer.Key,
	}

	parentWithoutSyncerKey := BidderInfo{
		Syncer: &Syncer{
			IFrame: &SyncerEndpoint{
				URL:         "https://foo.com/sync?mode=iframe&r={{.RedirectURL}}",
				RedirectURL: "https://redirect/setuid/iframe",
				ExternalURL: "https://iframe.host",
				UserMacro:   "UID",
			},
		},
	}

	bidderC := parentWithoutSyncerKey
	bidderC.AliasOf = "bidderA"
	bidderC.Syncer = &Syncer{
		Key: "bidderA",
	}

	parentWithSyncerSupports := parentWithoutSyncerKey
	parentWithSyncerSupports.Syncer = &Syncer{
		Supports: []string{"iframe"},
	}

	aliasWithoutSyncer := parentWithoutSyncerKey
	aliasWithoutSyncer.AliasOf = "bidderA"
	aliasWithoutSyncer.Syncer = nil

	testCases := []struct {
		description         string
		aliasInfos          map[string]aliasNillableFields
		bidderInfos         BidderInfos
		expectedBidderInfos BidderInfos
		expectedErr         error
	}{
		{
			description: "inherit all parent info in alias bidder, use parent syncer key as syncer alias key",
			aliasInfos: map[string]aliasNillableFields{
				"bidderB": {
					Disabled:                nil,
					ModifyingVastXmlAllowed: nil,
					Experiment:              nil,
					XAPI:                    nil,
				},
			},
			bidderInfos: BidderInfos{
				"bidderA": parentWithSyncerKey,
				"bidderB": BidderInfo{
					AliasOf: "bidderA",
					// all other fields should be inherited from parent bidder
				},
			},
			expectedErr:         nil,
			expectedBidderInfos: BidderInfos{"bidderA": parentWithSyncerKey, "bidderB": bidderB},
		},
		{
			description: "inherit all parent info in alias bidder, except for syncer is parent only defines supports",
			aliasInfos: map[string]aliasNillableFields{
				"bidderB": {
					Disabled:                nil,
					ModifyingVastXmlAllowed: nil,
					Experiment:              nil,
					XAPI:                    nil,
				},
			},
			bidderInfos: BidderInfos{
				"bidderA": parentWithSyncerSupports,
				"bidderB": BidderInfo{
					AliasOf: "bidderA",
					// all other fields should be inherited from parent bidder, except for syncer
				},
			},
			expectedErr:         nil,
			expectedBidderInfos: BidderInfos{"bidderA": parentWithSyncerSupports, "bidderB": aliasWithoutSyncer},
		},
		{
			description: "inherit all parent info in alias bidder, use parent name as syncer alias key",
			aliasInfos: map[string]aliasNillableFields{
				"bidderC": {
					Disabled:                nil,
					ModifyingVastXmlAllowed: nil,
					Experiment:              nil,
					XAPI:                    nil,
				},
			},
			bidderInfos: BidderInfos{
				"bidderA": parentWithoutSyncerKey,
				"bidderC": BidderInfo{
					AliasOf: "bidderA",
					// all other fields should be inherited from parent bidder
				},
			},
			expectedErr:         nil,
			expectedBidderInfos: BidderInfos{"bidderA": parentWithoutSyncerKey, "bidderC": bidderC},
		},
		{
			description: "all bidder info specified for alias, do not inherit from parent bidder",
			aliasInfos: map[string]aliasNillableFields{
				"bidderB": {
					Disabled:                &aliasBidderInfo.Disabled,
					ModifyingVastXmlAllowed: &aliasBidderInfo.ModifyingVastXmlAllowed,
					Experiment:              &aliasBidderInfo.Experiment,
					XAPI:                    &aliasBidderInfo.XAPI,
				},
			},
			bidderInfos: BidderInfos{
				"bidderA": parentWithSyncerKey,
				"bidderB": aliasBidderInfo,
			},
			expectedErr:         nil,
			expectedBidderInfos: BidderInfos{"bidderA": parentWithSyncerKey, "bidderB": aliasBidderInfo},
		},
		{
			description: "invalid alias",
			aliasInfos: map[string]aliasNillableFields{
				"bidderB": {},
			},
			bidderInfos: BidderInfos{
				"bidderB": BidderInfo{
					AliasOf: "bidderA",
				},
			},
			expectedErr: errors.New("bidder: bidderA not found for an alias: bidderB"),
		},
		{
			description: "bidder info not found for an alias",
			aliasInfos: map[string]aliasNillableFields{
				"bidderB": {},
			},
			expectedErr: errors.New("bidder info not found for an alias: bidderB"),
		},
	}

	for _, test := range testCases {
		bidderInfos, err := processBidderAliases(test.aliasInfos, test.bidderInfos)
		if test.expectedErr != nil {
			assert.Equal(t, test.expectedErr, err)
		} else {
			assert.Equal(t, test.expectedBidderInfos, bidderInfos, test.description)
		}
	}
}

type StubInfoReader struct {
	mockBidderInfos map[string][]byte
}

func (r StubInfoReader) Read() (map[string][]byte, error) {
	return r.mockBidderInfos, nil
}

var testBidderNames = map[string]openrtb_ext.BidderName{
	"biddera": openrtb_ext.BidderName("bidderA"),
	"bidderb": openrtb_ext.BidderName("bidderB"),
	"bidder1": openrtb_ext.BidderName("bidder1"),
	"bidder2": openrtb_ext.BidderName("bidder2"),
	"a":       openrtb_ext.BidderName("a"),
}

func mockNormalizeBidderName(name string) (openrtb_ext.BidderName, bool) {
	nameLower := strings.ToLower(name)
	bidderName, exists := testBidderNames[nameLower]
	return bidderName, exists
}

func TestToGVLVendorIDMap(t *testing.T) {
	givenBidderInfos := BidderInfos{
		"bidderA": BidderInfo{Disabled: false, GVLVendorID: 0},
		"bidderB": BidderInfo{Disabled: false, GVLVendorID: 100},
		"bidderC": BidderInfo{Disabled: true, GVLVendorID: 0},
		"bidderD": BidderInfo{Disabled: true, GVLVendorID: 200},
	}

	expectedGVLVendorIDMap := map[openrtb_ext.BidderName]uint16{
		"bidderB": 100,
	}

	result := givenBidderInfos.ToGVLVendorIDMap()
	assert.Equal(t, expectedGVLVendorIDMap, result)
}

const bidderInfoRelativePath = "../static/bidder-info"

// TestBidderInfoFiles ensures each bidder has a valid static/bidder-info/bidder.yaml file. Validation is performed directly
// against the file system with separate yaml unmarshalling from the LoadBidderInfo func.
func TestBidderInfoFiles(t *testing.T) {
	_, err := LoadBidderInfoFromDisk(bidderInfoRelativePath)
	if err != nil {
		assert.Fail(t, err.Error(), "Errors in bidder info files")
	}
}

func TestBidderInfoValidationPositive(t *testing.T) {
	bidderInfos := BidderInfos{
		"bidderA": BidderInfo{
			Endpoint:   "http://bidderA.com/openrtb2",
			PlatformID: "A",
			Maintainer: &MaintainerInfo{
				Email: "maintainer@bidderA.com",
			},
			GVLVendorID: 1,
			Capabilities: &CapabilitiesInfo{
				App: &PlatformInfo{
					MediaTypes: []openrtb_ext.BidType{
						openrtb_ext.BidTypeVideo,
						openrtb_ext.BidTypeNative,
						openrtb_ext.BidTypeBanner,
					},
				},
			},
			Syncer: &Syncer{
				Key: "bidderAkey",
				Redirect: &SyncerEndpoint{
					URL:       "http://bidderA.com/usersync",
					UserMacro: "UID",
				},
			},
		},
		"bidderB": BidderInfo{
			Endpoint:   "http://bidderB.com/openrtb2",
			PlatformID: "B",
			Maintainer: &MaintainerInfo{
				Email: "maintainer@bidderB.com",
			},
			GVLVendorID: 2,
			Capabilities: &CapabilitiesInfo{
				Site: &PlatformInfo{
					MediaTypes: []openrtb_ext.BidType{
						openrtb_ext.BidTypeVideo,
						openrtb_ext.BidTypeNative,
						openrtb_ext.BidTypeBanner,
					},
				},
			},
			Syncer: &Syncer{
				Key: "bidderBkey",
				Redirect: &SyncerEndpoint{
					URL:       "http://bidderB.com/usersync",
					UserMacro: "UID",
				},
				FormatOverride: SyncResponseFormatRedirect,
			},
		},
		"bidderC": BidderInfo{
			Endpoint: "http://bidderB.com/openrtb2",
			Maintainer: &MaintainerInfo{
				Email: "maintainer@bidderA.com",
			},
			Capabilities: &CapabilitiesInfo{
				Site: &PlatformInfo{
					MediaTypes: []openrtb_ext.BidType{
						openrtb_ext.BidTypeVideo,
						openrtb_ext.BidTypeNative,
						openrtb_ext.BidTypeBanner,
					},
				},
			},
			AliasOf: "bidderB",
		},
		"bidderD": BidderInfo{
			Endpoint:   "http://bidderD.com/openrtb2",
			PlatformID: "D",
			Maintainer: &MaintainerInfo{
				Email: "maintainer@bidderD.com",
			},
			GVLVendorID: 3,
			Capabilities: &CapabilitiesInfo{
				DOOH: &PlatformInfo{
					MediaTypes: []openrtb_ext.BidType{
						openrtb_ext.BidTypeVideo,
						openrtb_ext.BidTypeNative,
						openrtb_ext.BidTypeBanner,
					},
				},
			},
			Syncer: &Syncer{
				FormatOverride: SyncResponseFormatIFrame,
			},
		},
	}
	errs := bidderInfos.validate(make([]error, 0))
	assert.Len(t, errs, 0, "All bidder infos should be correct")
}

func TestValidateAliases(t *testing.T) {
	testCase := struct {
		description  string
		bidderInfos  BidderInfos
		expectErrors []error
	}{
		description: "invalid aliases",
		bidderInfos: BidderInfos{
			"bidderA": BidderInfo{
				Endpoint: "http://bidderA.com/openrtb2",
				Maintainer: &MaintainerInfo{
					Email: "maintainer@bidderA.com",
				},
				Capabilities: &CapabilitiesInfo{
					Site: &PlatformInfo{
						MediaTypes: []openrtb_ext.BidType{
							openrtb_ext.BidTypeVideo,
						},
					},
				},
				AliasOf: "bidderB",
			},
			"bidderB": BidderInfo{
				Endpoint: "http://bidderA.com/openrtb2",
				Maintainer: &MaintainerInfo{
					Email: "maintainer@bidderA.com",
				},
				Capabilities: &CapabilitiesInfo{
					Site: &PlatformInfo{
						MediaTypes: []openrtb_ext.BidType{
							openrtb_ext.BidTypeVideo,
						},
					},
				},
				AliasOf: "bidderC",
			},
		},
		expectErrors: []error{
			errors.New("bidder: bidderB cannot be an alias of an alias: bidderA"),
			errors.New("bidder: bidderC not found for an alias: bidderB"),
		},
	}

	var errs []error
	for bidderName, bidderInfo := range testCase.bidderInfos {
		errs = append(errs, validateAliases(bidderInfo, testCase.bidderInfos, bidderName))
	}

	assert.ElementsMatch(t, errs, testCase.expectErrors)
}

func TestBidderInfoValidationNegative(t *testing.T) {
	testCases := []struct {
		description  string
		bidderInfos  BidderInfos
		expectErrors []error
	}{
		{
			"One bidder incorrect url",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "incorrect",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
			},
			[]error{
				errors.New("The endpoint: incorrect for bidderA is not a valid URL"),
			},
		},
		{
			"One bidder empty url",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
			},
			[]error{
				errors.New("There's no default endpoint available for bidderA. Calls to this bidder/exchange will fail. Please set adapters.bidderA.endpoint in your app config"),
			},
		},
		{
			"One bidder incorrect url template",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2/getuid?{{.incorrect}}",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
			},
			[]error{
				errors.New("Unable to resolve endpoint: http://bidderA.com/openrtb2/getuid?{{.incorrect}} for adapter: bidderA. template: endpointTemplate:1:37: executing \"endpointTemplate\" at <.incorrect>: can't evaluate field incorrect in type macros.EndpointTemplateParams"),
			},
		},
		{
			"One bidder incorrect url template parameters",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2/getuid?r=[{{.]RedirectURL}}",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
			},
			[]error{
				errors.New("Invalid endpoint template: http://bidderA.com/openrtb2/getuid?r=[{{.]RedirectURL}} for adapter: bidderA. template: endpointTemplate:1: bad character U+005D ']'"),
			},
		},
		{
			"One bidder no maintainer",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
			},
			[]error{
				errors.New("missing required field: maintainer.email for adapter: bidderA"),
			},
		},
		{
			"One bidder missing maintainer email",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
			},
			[]error{
				errors.New("missing required field: maintainer.email for adapter: bidderA"),
			},
		},
		{
			"One bidder missing capabilities",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
				},
			},
			[]error{
				errors.New("missing required field: capabilities for adapter: bidderA"),
			},
		},
		{
			"One bidder missing capabilities site and app and dooh",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{},
				},
			},
			[]error{
				errors.New("at least one of capabilities.site, capabilities.app, or capabilities.dooh must exist for adapter: bidderA"),
			},
		},
		{
			"One bidder incorrect capabilities for app",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								"incorrect",
							},
						},
					},
				},
			},
			[]error{
				errors.New("capabilities.app failed validation: unrecognized media type at index 0: incorrect for adapter: bidderA"),
			},
		},
		{
			"One bidder incorrect capabilities for dooh",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						DOOH: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								"incorrect",
							},
						},
					},
				},
			},
			[]error{
				errors.New("capabilities.dooh failed validation: unrecognized media type at index 0: incorrect for adapter: bidderA"),
			},
		},
		{
			"One bidder nil capabilities",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: nil,
				},
			},
			[]error{
				errors.New("missing required field: capabilities for adapter: bidderA"),
			},
		},
		{
			"One bidder invalid syncer",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
					Syncer: &Syncer{
						Supports: []string{"incorrect"},
					},
				},
			},
			[]error{
				errors.New("syncer could not be created, invalid supported endpoint: incorrect"),
			},
		},
		{
			"Two bidders, one with incorrect url",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "incorrect",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
				"bidderB": BidderInfo{
					Endpoint: "http://bidderB.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderB.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
			},
			[]error{
				errors.New("The endpoint: incorrect for bidderA is not a valid URL"),
			},
		},
		{
			"Two bidders, both with incorrect url",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "incorrect",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
				"bidderB": BidderInfo{
					Endpoint: "incorrect",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderB.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
			},
			[]error{
				errors.New("The endpoint: incorrect for bidderA is not a valid URL"),
				errors.New("The endpoint: incorrect for bidderB is not a valid URL"),
			},
		},
		{
			"Invalid alias Site capabilities",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
				"bidderB": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
					AliasOf: "bidderA",
				},
			},
			[]error{
				errors.New("capabilities for alias: bidderB should be a subset of capabilities for parent bidder: bidderA"),
			},
		},
		{
			"Invalid alias App capabilities",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
				"bidderB": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
					AliasOf: "bidderA",
				},
			},
			[]error{
				errors.New("capabilities for alias: bidderB should be a subset of capabilities for parent bidder: bidderA"),
			},
		},
		{
			"Invalid alias capabilities",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{},
				},
				"bidderB": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
					AliasOf: "bidderA",
				},
			},
			[]error{
				errors.New("at least one of capabilities.site, capabilities.app, or capabilities.dooh must exist for adapter: bidderA"),
				errors.New("capabilities for alias: bidderB should be a subset of capabilities for parent bidder: bidderA"),
			},
		},
		{
			"Invalid alias MediaTypes for site",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
				"bidderB": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
					},
					AliasOf: "bidderA",
				},
			},
			[]error{
				errors.New("mediaTypes for alias: bidderB should be a subset of MediaTypes for parent bidder: bidderA"),
			},
		},
		{
			"Invalid alias MediaTypes for app",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
				"bidderB": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
					},
					AliasOf: "bidderA",
				},
			},
			[]error{
				errors.New("mediaTypes for alias: bidderB should be a subset of MediaTypes for parent bidder: bidderA"),
			},
		},
		{
			"Invalid parent bidder capabilities",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
				},
				"bidderB": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
							},
						},
					},
					AliasOf: "bidderA",
				},
			},
			[]error{
				errors.New("missing required field: capabilities for adapter: bidderA"),
				errors.New("capabilities for alias: bidderB should be a subset of capabilities for parent bidder: bidderA"),
			},
		},
		{
			"Invalid site alias capabilities with both site and app",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeNative,
							},
						},
					},
				},
				"bidderB": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
					},
					AliasOf: "bidderA",
				},
			},
			[]error{
				errors.New("mediaTypes for alias: bidderB should be a subset of MediaTypes for parent bidder: bidderA"),
			},
		},
		{
			"Invalid app alias capabilities with both site and app",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeNative,
							},
						},
					},
				},
				"bidderB": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
					},
					AliasOf: "bidderA",
				},
			},
			[]error{
				errors.New("mediaTypes for alias: bidderB should be a subset of MediaTypes for parent bidder: bidderA"),
			},
		},
		{
			"Invalid parent bidder for alias",
			BidderInfos{
				"bidderB": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
					},
					AliasOf: "bidderC",
				},
			},
			[]error{
				errors.New("parent bidder: bidderC not found for an alias: bidderB"),
			},
		},
		{
			"Invalid format override value",
			BidderInfos{
				"bidderB": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{
								openrtb_ext.BidTypeBanner,
								openrtb_ext.BidTypeNative,
							},
						},
					},
					Syncer: &Syncer{
						FormatOverride: "x",
					},
				},
			},
			[]error{
				errors.New("syncer could not be created, invalid format override value: x"),
			},
		},
	}

	for _, test := range testCases {
		errs := test.bidderInfos.validate(make([]error, 0))
		assert.ElementsMatch(t, errs, test.expectErrors, "incorrect errors returned for test: %s", test.description)
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

func TestSyncerDefined(t *testing.T) {
	testCases := []struct {
		name        string
		givenSyncer *Syncer
		expected    bool
	}{
		{
			name:        "nil",
			givenSyncer: nil,
			expected:    false,
		},
		{
			name:        "empty",
			givenSyncer: &Syncer{},
			expected:    false,
		},
		{
			name:        "key-only",
			givenSyncer: &Syncer{Key: "anyKey"},
			expected:    true,
		},
		{
			name:        "iframe-only",
			givenSyncer: &Syncer{IFrame: &SyncerEndpoint{}},
			expected:    true,
		},
		{
			name:        "redirect-only",
			givenSyncer: &Syncer{Redirect: &SyncerEndpoint{}},
			expected:    true,
		},
		{
			name:        "externalurl-only",
			givenSyncer: &Syncer{ExternalURL: "anyURL"},
			expected:    true,
		},
		{
			name:        "supportscors-only",
			givenSyncer: &Syncer{SupportCORS: ptrutil.ToPtr(false)},
			expected:    true,
		},
		{
			name:        "formatoverride-only",
			givenSyncer: &Syncer{FormatOverride: "anyFormat"},
			expected:    true,
		},
		{
			name:        "skipwhen-only",
			givenSyncer: &Syncer{SkipWhen: &SkipWhen{}},
			expected:    true,
		},
		{
			name:        "supports-only",
			givenSyncer: &Syncer{Supports: []string{"anySupports"}},
			expected:    false,
		},
		{
			name:        "supports-with-other",
			givenSyncer: &Syncer{Key: "anyKey", Supports: []string{"anySupports"}},
			expected:    true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := test.givenSyncer.Defined()
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestApplyBidderInfoConfigSyncerOverrides(t *testing.T) {
	var (
		givenFileSystem = BidderInfos{"a": {Syncer: &Syncer{Key: "original"}}}
		givenConfig     = nillableFieldBidderInfos{
			"a": {
				bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}},
			},
		}
		expected = BidderInfos{"a": {Syncer: &Syncer{Key: "override"}}}
	)

	result, resultErr := applyBidderInfoConfigOverrides(givenConfig, givenFileSystem, mockNormalizeBidderName)
	assert.NoError(t, resultErr)
	assert.Equal(t, expected, result)
}

func TestApplyBidderInfoConfigOverrides(t *testing.T) {
	falseValue := false

	var testCases = []struct {
		description            string
		givenFsBidderInfos     BidderInfos
		givenConfigBidderInfos nillableFieldBidderInfos
		expectedError          string
		expectedBidderInfos    BidderInfos
	}{
		{
			description:            "Don't override endpoint",
			givenFsBidderInfos:     BidderInfos{"a": {Endpoint: "original"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {Endpoint: "original", Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Override endpoint",
			givenFsBidderInfos:     BidderInfos{"a": {Endpoint: "original"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Endpoint: "override", Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {Endpoint: "override", Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Don't override ExtraAdapterInfo",
			givenFsBidderInfos:     BidderInfos{"a": {ExtraAdapterInfo: "original"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {ExtraAdapterInfo: "original", Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Override ExtraAdapterInfo",
			givenFsBidderInfos:     BidderInfos{"a": {ExtraAdapterInfo: "original"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{ExtraAdapterInfo: "override", Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {ExtraAdapterInfo: "override", Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Don't override Maintainer",
			givenFsBidderInfos:     BidderInfos{"a": {Maintainer: &MaintainerInfo{Email: "original"}}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {Maintainer: &MaintainerInfo{Email: "original"}, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Override maintainer",
			givenFsBidderInfos:     BidderInfos{"a": {Maintainer: &MaintainerInfo{Email: "original"}}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Maintainer: &MaintainerInfo{Email: "override"}, Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {Maintainer: &MaintainerInfo{Email: "override"}, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description: "Don't override Capabilities",
			givenFsBidderInfos: BidderInfos{"a": {
				Capabilities: &CapabilitiesInfo{App: &PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeVideo}}},
			}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos: BidderInfos{"a": {
				Syncer:       &Syncer{Key: "override"},
				Capabilities: &CapabilitiesInfo{App: &PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeVideo}}},
			}},
		},
		{
			description: "Override Capabilities",
			givenFsBidderInfos: BidderInfos{"a": {
				Capabilities: &CapabilitiesInfo{App: &PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeVideo}}},
			}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{
				Syncer:       &Syncer{Key: "override"},
				Capabilities: &CapabilitiesInfo{App: &PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner}}},
			}}},
			expectedBidderInfos: BidderInfos{"a": {
				Syncer:       &Syncer{Key: "override"},
				Capabilities: &CapabilitiesInfo{App: &PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner}}},
			}},
		},
		{
			description:            "Don't override Debug",
			givenFsBidderInfos:     BidderInfos{"a": {Debug: &DebugInfo{Allow: true}}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {Debug: &DebugInfo{Allow: true}, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Override Debug",
			givenFsBidderInfos:     BidderInfos{"a": {Debug: &DebugInfo{Allow: true}}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Debug: &DebugInfo{Allow: false}, Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {Debug: &DebugInfo{Allow: false}, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Don't override GVLVendorID",
			givenFsBidderInfos:     BidderInfos{"a": {GVLVendorID: 5}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {GVLVendorID: 5, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Override GVLVendorID",
			givenFsBidderInfos:     BidderInfos{"a": {}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{GVLVendorID: 5, Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {GVLVendorID: 5, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description: "Don't override XAPI",
			givenFsBidderInfos: BidderInfos{"a": {
				XAPI: AdapterXAPI{Username: "username1", Password: "password2", Tracker: "tracker3"},
			}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos: BidderInfos{"a": {
				XAPI:   AdapterXAPI{Username: "username1", Password: "password2", Tracker: "tracker3"},
				Syncer: &Syncer{Key: "override"}}},
		},
		{
			description: "Override XAPI",
			givenFsBidderInfos: BidderInfos{"a": {
				XAPI: AdapterXAPI{Username: "username", Password: "password", Tracker: "tracker"}}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{
				XAPI:   AdapterXAPI{Username: "username1", Password: "password2", Tracker: "tracker3"},
				Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos: BidderInfos{"a": {
				XAPI:   AdapterXAPI{Username: "username1", Password: "password2", Tracker: "tracker3"},
				Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Don't override PlatformID",
			givenFsBidderInfos:     BidderInfos{"a": {PlatformID: "PlatformID"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {PlatformID: "PlatformID", Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Override PlatformID",
			givenFsBidderInfos:     BidderInfos{"a": {PlatformID: "PlatformID1"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{PlatformID: "PlatformID2", Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {PlatformID: "PlatformID2", Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Don't override AppSecret",
			givenFsBidderInfos:     BidderInfos{"a": {AppSecret: "AppSecret"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {AppSecret: "AppSecret", Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Override AppSecret",
			givenFsBidderInfos:     BidderInfos{"a": {AppSecret: "AppSecret1"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{AppSecret: "AppSecret2", Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {AppSecret: "AppSecret2", Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Don't override EndpointCompression",
			givenFsBidderInfos:     BidderInfos{"a": {EndpointCompression: "GZIP"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {EndpointCompression: "GZIP", Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Override EndpointCompression",
			givenFsBidderInfos:     BidderInfos{"a": {EndpointCompression: "GZIP"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{EndpointCompression: "LZ77", Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {EndpointCompression: "LZ77", Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Don't override Disabled",
			givenFsBidderInfos:     BidderInfos{"a": {Disabled: true}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Disabled: false, Syncer: &Syncer{Key: "override"}}, nillableFields: bidderInfoNillableFields{Disabled: nil}}},
			expectedBidderInfos:    BidderInfos{"a": {Disabled: true, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Override Disabled",
			givenFsBidderInfos:     BidderInfos{"a": {Disabled: true}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Disabled: false, Syncer: &Syncer{Key: "override"}}, nillableFields: bidderInfoNillableFields{Disabled: &falseValue}}},
			expectedBidderInfos:    BidderInfos{"a": {Disabled: false, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Don't override ModifyingVastXmlAllowed",
			givenFsBidderInfos:     BidderInfos{"a": {ModifyingVastXmlAllowed: true}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{ModifyingVastXmlAllowed: false, Syncer: &Syncer{Key: "override"}}, nillableFields: bidderInfoNillableFields{ModifyingVastXmlAllowed: nil}}},
			expectedBidderInfos:    BidderInfos{"a": {ModifyingVastXmlAllowed: true, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Override ModifyingVastXmlAllowed",
			givenFsBidderInfos:     BidderInfos{"a": {ModifyingVastXmlAllowed: true}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{ModifyingVastXmlAllowed: false, Syncer: &Syncer{Key: "override"}}, nillableFields: bidderInfoNillableFields{ModifyingVastXmlAllowed: &falseValue}}},
			expectedBidderInfos:    BidderInfos{"a": {ModifyingVastXmlAllowed: false, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Don't override OpenRTB",
			givenFsBidderInfos:     BidderInfos{"a": {OpenRTB: &OpenRTBInfo{Version: "1"}}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {OpenRTB: &OpenRTBInfo{Version: "1"}, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Override OpenRTB",
			givenFsBidderInfos:     BidderInfos{"a": {OpenRTB: &OpenRTBInfo{Version: "1"}}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{OpenRTB: &OpenRTBInfo{Version: "2"}, Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {OpenRTB: &OpenRTBInfo{Version: "2"}, Syncer: &Syncer{Key: "override"}}},
		},
		{
			description:            "Don't override AliasOf",
			givenFsBidderInfos:     BidderInfos{"a": {AliasOf: "Alias1"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{}}},
			expectedBidderInfos:    BidderInfos{"a": {AliasOf: "Alias1"}},
		},
		{
			description:            "Attempt override AliasOf but ignored",
			givenFsBidderInfos:     BidderInfos{"a": {AliasOf: "Alias1"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{AliasOf: "Alias2"}}},
			expectedBidderInfos:    BidderInfos{"a": {AliasOf: "Alias1"}},
		},
		{
			description:            "Two bidder infos: One with overrides and one without",
			givenFsBidderInfos:     BidderInfos{"a": {Endpoint: "original"}, "b": {Endpoint: "b endpoint"}},
			givenConfigBidderInfos: nillableFieldBidderInfos{"a": {bidderInfo: BidderInfo{Endpoint: "override", Syncer: &Syncer{Key: "override"}}}},
			expectedBidderInfos:    BidderInfos{"a": {Endpoint: "override", Syncer: &Syncer{Key: "override"}}, "b": {Endpoint: "b endpoint"}},
		},
	}
	for _, test := range testCases {
		bidderInfos, resultErr := applyBidderInfoConfigOverrides(test.givenConfigBidderInfos, test.givenFsBidderInfos, mockNormalizeBidderName)
		assert.NoError(t, resultErr, test.description+":err")
		assert.Equal(t, test.expectedBidderInfos, bidderInfos, test.description+":result")
	}
}

func TestApplyBidderInfoConfigOverridesInvalid(t *testing.T) {
	var testCases = []struct {
		description                   string
		givenFsBidderInfos            BidderInfos
		givenNillableFieldBidderInfos nillableFieldBidderInfos
		expectedError                 string
		expectedBidderInfos           BidderInfos
	}{
		{
			description: "Bidder doesn't exists in bidder list",
			givenNillableFieldBidderInfos: nillableFieldBidderInfos{"unknown": nillableFieldBidderInfo{
				bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}},
			}},
			expectedError: "error setting configuration for bidder unknown: unknown bidder",
		},
		{
			description:        "Bidder doesn't exists in file system",
			givenFsBidderInfos: BidderInfos{"unknown": {Endpoint: "original"}},
			givenNillableFieldBidderInfos: nillableFieldBidderInfos{"bidderA": nillableFieldBidderInfo{
				bidderInfo: BidderInfo{Syncer: &Syncer{Key: "override"}},
			}},
			expectedError: "error finding configuration for bidder bidderA: unknown bidder",
		},
	}
	for _, test := range testCases {

		_, err := applyBidderInfoConfigOverrides(test.givenNillableFieldBidderInfos, test.givenFsBidderInfos, mockNormalizeBidderName)
		assert.ErrorContains(t, err, test.expectedError, test.description+":err")
	}
}

func TestReadFullYamlBidderConfig(t *testing.T) {
	bidder := "bidderA"
	bidderInf := BidderInfo{}
	falseValue := false
	err := yaml.Unmarshal([]byte(fullBidderYAMLConfig), &bidderInf)
	require.NoError(t, err)

	bidderInfoOverrides := nillableFieldBidderInfos{
		bidder: nillableFieldBidderInfo{
			bidderInfo: bidderInf,
			nillableFields: bidderInfoNillableFields{
				Disabled:                &bidderInf.Disabled,
				ModifyingVastXmlAllowed: &bidderInf.ModifyingVastXmlAllowed,
			},
		},
	}
	bidderInfoBase := BidderInfos{
		bidder: {Syncer: &Syncer{Supports: []string{"iframe"}}},
	}
	actualBidderInfo, err := applyBidderInfoConfigOverrides(bidderInfoOverrides, bidderInfoBase, mockNormalizeBidderName)
	require.NoError(t, err)

	expectedBidderInfo := BidderInfos{
		bidder: {
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
				DOOH: &PlatformInfo{
					MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
				},
			},
			ModifyingVastXmlAllowed: true,
			Debug: &DebugInfo{
				Allow: true,
			},
			Experiment: BidderInfoExperiment{
				AdsCert: BidderAdsCert{
					Enabled: true,
				},
			},
			EndpointCompression: "GZIP",
			OpenRTB: &OpenRTBInfo{
				GPPSupported:         true,
				Version:              "2.6",
				MultiformatSupported: &falseValue,
			},
			Disabled:         false,
			ExtraAdapterInfo: "extra-info",
			AppSecret:        "app-secret",
			PlatformID:       "123",
			Syncer: &Syncer{
				Key: "foo",
				IFrame: &SyncerEndpoint{
					URL:         "https://foo.com/sync?mode=iframe&r={{.RedirectURL}}",
					RedirectURL: "https://redirect/setuid/iframe",
					ExternalURL: "https://iframe.host",
					UserMacro:   "UID",
				},
				Supports: []string{"iframe"},
			},
			XAPI: AdapterXAPI{
				Username: "uname",
				Password: "pwd",
				Tracker:  "tracker",
			},
			Endpoint: "https://endpoint.com",
		},
	}
	assert.Equalf(t, expectedBidderInfo, actualBidderInfo, "Bidder info objects aren't matching")
}
