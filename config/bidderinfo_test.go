package config

import (
	"strings"
	"testing"

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
	r := infoReaderFromDisk{testInvalidInfoFilesPath}
	_, err := loadBidderInfo(r, testInvalidInfoFilesPath)
	assert.EqualError(t, err, expectedError, "incorrect error message returned while loading invalid bidder config")
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
