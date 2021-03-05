package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

const bidderInfoRelativePath = "../static/bidder-info"

// TestBidderInfoFiles ensures each bidder has a valid static/bidder-info/bidder.yaml file. Validation is performed directly
// against the file system with separate yaml unmarshalling from the LoadBidderInfoFromDisk func.
func TestBidderInfoFiles(t *testing.T) {
	fileInfos, err := ioutil.ReadDir(bidderInfoRelativePath)
	if err != nil {
		assert.FailNow(t, "Error reading the static/bidder-info directory: %v", err)
	}

	// Ensure YAML Files Are For A Known Core Bidder
	for _, fileInfo := range fileInfos {
		bidder := strings.TrimSuffix(fileInfo.Name(), ".yaml")

		_, isKnown := openrtb_ext.NormalizeBidderName(bidder)
		assert.True(t, isKnown, "unexpected bidder info yaml file %s", fileInfo.Name())
	}

	// Ensure YAML Files Are Defined For Each Core Bidder
	expectedFileInfosLength := len(openrtb_ext.CoreBidderNames())
	assert.Len(t, fileInfos, expectedFileInfosLength, "static/bidder-info contains %d files, but there are %d known bidders. Did you forget to add a YAML file for your bidder?", len(fileInfos), expectedFileInfosLength)

	// Validate Contents
	for _, fileInfo := range fileInfos {
		path := fmt.Sprintf(bidderInfoRelativePath + "/" + fileInfo.Name())

		infoFileData, err := os.Open(path)
		assert.NoError(t, err, "Unexpected error: %v", err)

		content, err := ioutil.ReadAll(infoFileData)
		assert.NoError(t, err, "Failed to read static/bidder-info/%s: %v", fileInfo.Name(), err)

		var fileInfoContent BidderInfo
		err = yaml.Unmarshal(content, &fileInfoContent)
		assert.NoError(t, err, "Error interpreting content from static/bidder-info/%s: %v", fileInfo.Name(), err)

		err = validateInfo(&fileInfoContent)
		assert.NoError(t, err, "Invalid content in static/bidder-info/%s: %v", fileInfo.Name(), err)
	}
}

func validateInfo(info *BidderInfo) error {
	if err := validateMaintainer(info.Maintainer); err != nil {
		return err
	}

	if err := validateCapabilities(info.Capabilities); err != nil {
		return err
	}

	return nil
}

func validateMaintainer(info *MaintainerInfo) error {
	if info == nil || info.Email == "" {
		return errors.New("missing required field: maintainer.email")
	}
	return nil
}

func validateCapabilities(info *CapabilitiesInfo) error {
	if info == nil {
		return errors.New("missing required field: capabilities")
	}

	if info.App == nil && info.Site == nil {
		return errors.New("at least one of capabilities.site or capabilities.app must exist")
	}

	if info.App != nil {
		if err := validatePlatformInfo(info.App); err != nil {
			return fmt.Errorf("capabilities.app failed validation: %v", err)
		}
	}

	if info.Site != nil {
		if err := validatePlatformInfo(info.Site); err != nil {
			return fmt.Errorf("capabilities.site failed validation: %v", err)
		}
	}
	return nil
}

func validatePlatformInfo(info *PlatformInfo) error {
	if info == nil {
		return errors.New("object cannot be empty")
	}

	if len(info.MediaTypes) == 0 {
		return errors.New("mediaTypes should be an array with at least one string element")
	}

	for index, mediaType := range info.MediaTypes {
		if mediaType != "banner" && mediaType != "video" && mediaType != "native" && mediaType != "audio" {
			return fmt.Errorf("unrecognized media type at index %d: %s", index, mediaType)
		}
	}

	return nil
}
