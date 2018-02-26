package info

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	yaml "gopkg.in/yaml.v2"
)

func TestBiddersEndpoint(t *testing.T) {
	endpoint := NewBiddersEndpoint()

	req, err := http.NewRequest("GET", "http://prebid-server.com/info/bidders", strings.NewReader(""))
	if err != nil {
		t.Fatalf("Failed to create a GET /info/bidders request: %v", err)
	}

	r := httptest.NewRecorder()
	endpoint(r, req, nil)
	if r.Code != http.StatusOK {
		t.Errorf("GET /info/bidders returned bad status: %d", r.Code)
	}
	if r.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Bad /info/bidders content type. Expected application/json. Got %s", r.Header().Get("Content-Type"))
	}
	bodyBytes := r.Body.Bytes()
	bidderSlice := make([]string, 0, len(openrtb_ext.BidderMap))
	if err := json.Unmarshal(bodyBytes, &bidderSlice); err != nil {
		t.Errorf("Failed to unmarshal /info/bidders response: %v", err)
	}
	for _, bidderName := range bidderSlice {
		if _, ok := openrtb_ext.BidderMap[bidderName]; !ok {
			t.Errorf("Response from /info/bidders contained unexpected BidderName: %s", bidderName)
		}
	}
	if len(bidderSlice) != len(openrtb_ext.BidderMap) {
		t.Errorf("Response from /info/bidders did not match BidderMap. Expected %d elements. Got %d", len(openrtb_ext.BidderMap), len(bidderSlice))
	}
}

// TestInfoFiles makes sure that every bidder contains a valid info.yaml file.
func TestInfoFiles(t *testing.T) {
	fileInfos, err := ioutil.ReadDir("../../static/bidder-info")
	if err != nil {
		t.Fatalf("Error reading the ../static/bidder-info directory: %v", err)
	}

	// Make sure that files exist for each BidderName
	for bidderName, _ := range openrtb_ext.BidderMap {
		if _, err := os.Stat(fmt.Sprintf("../../static/bidder-info/%s.yaml", bidderName)); os.IsNotExist(err) {
			t.Errorf("static/bidder-info/%s.yaml not found. Did you forget to create it?", bidderName)
		}
	}

	// Make sure that all the files have valid content
	for _, fileInfo := range fileInfos {
		infoFileData, err := os.Open(fmt.Sprintf("../../static/bidder-info/%s", fileInfo.Name()))
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
			continue
		}

		content, err := ioutil.ReadAll(infoFileData)
		if err != nil {
			t.Errorf("Failed to read static/bidder-info/%s: %v", fileInfo.Name(), err)
			continue
		}
		var fileInfoContent infoFile
		if err := yaml.Unmarshal(content, &fileInfoContent); err != nil {
			t.Errorf("Error interpreting content from static/bidder-info/%s: %v", fileInfo.Name(), err)
			continue
		}
		if err := validateInfo(&fileInfoContent); err != nil {
			t.Errorf("Invalid content in static/bidder-info/%s: %v", fileInfo.Name(), err)
		}
	}
}

func validateInfo(info *infoFile) error {
	if err := validateMaintainer(info.Maintainer); err != nil {
		return err
	}

	if err := validateCapabilities(info.Capabilities); err != nil {
		return err
	}

	return nil
}

func validateCapabilities(info *capabilitiesInfo) error {
	if info == nil {
		return errors.New("missing required field: capabilities")

	}
	if info.App == nil && info.Site == nil {
		return errors.New("at least one of capabilities.site or capabilities.app should exist")
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

func validatePlatformInfo(info *platformInfo) error {
	if info == nil {
		return errors.New("we can't validate a nil platformInfo")
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

func validateMaintainer(info *maintainerInfo) error {
	if info == nil || info.Email == "" {
		return errors.New("missing required field: maintainer.email")
	}
	return nil
}
