package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const testInfoFilesPath = "./test/bidder-info"
const testInvalidInfoFilesPath = "./test/bidder-info-invalid"

func TestLoadBidderInfoFromDisk(t *testing.T) {
	bidder := "someBidder"
	trueValue := true

	adapterConfigs := make(map[string]Adapter)
	adapterConfigs[strings.ToLower(bidder)] = Adapter{}

	infos, err := LoadBidderInfoFromDisk(testInfoFilesPath)
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

func TestLoadBidderInfoInvalid(t *testing.T) {
	expectedError := "error parsing yaml for bidder someBidder-invalid.yaml: yaml: unmarshal errors:\n  line 3: cannot unmarshal !!str `42` into uint16"
	_, err := loadBidderInfo(testInvalidInfoFilesPath)
	assert.EqualError(t, err, expectedError, "incorrect error message returned while loading invalid bidder config")
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
// against the file system with separate yaml unmarshalling from the LoadBidderInfoFromDisk func.
func TestBidderInfoFiles(t *testing.T) {
	_, errs := ProcessBidderInfos(bidderInfoRelativePath, nil)
	if len(errs) > 0 {
		errorMsg := errortypes.NewAggregateError("bidder infos", errs)
		assert.Fail(t, errorMsg.Message, "Errors in bidder info files")
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
				Email: "maintainer@bidderA.com",
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
			},
		},
	}
	errs := validateBidderInfos(bidderInfos)
	assert.Len(t, errs, 0, "All bidder infos should be correct")
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
			"One bidder missing capabilities site and app",
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
				errors.New("at least one of capabilities.site or capabilities.app must exist for adapter: bidderA"),
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
			"One bidder empty capabilities for app",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						App: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{},
						},
					},
				},
			},
			[]error{
				errors.New("capabilities.app failed validation: mediaTypes should be an array with at least one string element for adapter: bidderA"),
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
			"One bidder empty capabilities for site",
			BidderInfos{
				"bidderA": BidderInfo{
					Endpoint: "http://bidderA.com/openrtb2",
					Maintainer: &MaintainerInfo{
						Email: "maintainer@bidderA.com",
					},
					Capabilities: &CapabilitiesInfo{
						Site: &PlatformInfo{
							MediaTypes: []openrtb_ext.BidType{},
						},
					},
				},
			},
			[]error{
				errors.New("capabilities.site failed validation: mediaTypes should be an array with at least one string element, for adapter: bidderA"),
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
	}

	for _, test := range testCases {
		errs := validateBidderInfos(test.bidderInfos)
		assert.ElementsMatch(t, errs, test.expectErrors, "incorrect errors returned for test: %s", test.description)
	}
}
