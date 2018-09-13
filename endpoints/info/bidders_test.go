package info_test

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

	"github.com/julienschmidt/httprouter"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/endpoints/info"
	"github.com/prebid/prebid-server/openrtb_ext"
	yaml "gopkg.in/yaml.v2"
)

func TestGetBidders(t *testing.T) {
	endpoint := info.NewBiddersEndpoint()

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

// TestGetSpecificBidders validates all the GET /info/bidders/{bidderName} endpoints
func TestGetSpecificBidders(t *testing.T) {
	bidderInfos := adapters.ParseBidderInfos("../../static/bidder-info", openrtb_ext.BidderList())
	endpoint := info.NewBidderDetailsEndpoint(bidderInfos)

	for bidderName := range openrtb_ext.BidderMap {
		req, err := http.NewRequest("GET", "http://prebid-server.com/info/bidders/"+bidderName, strings.NewReader(""))
		if err != nil {
			t.Errorf("Failed to create a GET /info/bidders request: %v", err)
			continue
		}
		params := []httprouter.Param{{
			Key:   "bidderName",
			Value: bidderName,
		}}
		r := httptest.NewRecorder()

		endpoint(r, req, params)

		if r.Code != http.StatusOK {
			t.Errorf("GET /info/bidders/"+bidderName+" returned a %d. Expected 200", r.Code)
		}
		if r.HeaderMap.Get("Content-Type") != "application/json" {
			t.Errorf("GET /info/bidders/"+bidderName+" returned Content-Type %s. Expected application/json", r.HeaderMap.Get("Content-Type"))
		}
	}
}

// TestGetBidderAccuracy validates the output for a known file.
func TestGetBidderAccuracy(t *testing.T) {
	bidderInfos := adapters.ParseBidderInfos("../../adapters/adapterstest/bidder-info", []openrtb_ext.BidderName{openrtb_ext.BidderName("someBidder")})

	endpoint := info.NewBidderDetailsEndpoint(bidderInfos)
	req, err := http.NewRequest("GET", "http://prebid-server.com/info/bidders/someBidder", strings.NewReader(""))
	if err != nil {
		t.Fatalf("Failed to create a GET /info/bidders request: %v", err)
	}
	params := []httprouter.Param{{
		Key:   "bidderName",
		Value: "someBidder",
	}}

	r := httptest.NewRecorder()
	endpoint(r, req, params)

	var fileData adapters.BidderInfo
	if err := json.Unmarshal(r.Body.Bytes(), &fileData); err != nil {
		t.Fatalf("Failed to unmarshal JSON from endpoints/info/sample/someBidder.yaml: %v", err)
	}

	if fileData.Maintainer.Email != "some-email@domain.com" {
		t.Errorf("maintainer.email should be some-email@domain.com. Got %s", fileData.Maintainer.Email)
	}

	if len(fileData.Capabilities.App.MediaTypes) != 2 {
		t.Fatalf("Expected 2 supported mediaTypes on app. Got %d", len(fileData.Capabilities.App.MediaTypes))
	}
	if fileData.Capabilities.App.MediaTypes[0] != "banner" {
		t.Errorf("capabilities.app.mediaTypes[0] should be banner. Got %s", fileData.Capabilities.App.MediaTypes[0])
	}
	if fileData.Capabilities.App.MediaTypes[1] != "native" {
		t.Errorf("capabilities.app.mediaTypes[1] should be native. Got %s", fileData.Capabilities.App.MediaTypes[1])
	}

	if len(fileData.Capabilities.Site.MediaTypes) != 3 {
		t.Fatalf("Expected 3 supported mediaTypes on app. Got %d", len(fileData.Capabilities.Site.MediaTypes))
	}
	if fileData.Capabilities.Site.MediaTypes[0] != "banner" {
		t.Errorf("capabilities.app.mediaTypes[0] should be banner. Got %s", fileData.Capabilities.Site.MediaTypes[0])
	}
	if fileData.Capabilities.Site.MediaTypes[1] != "video" {
		t.Errorf("capabilities.app.mediaTypes[1] should be video. Got %s", fileData.Capabilities.Site.MediaTypes[1])
	}
	if fileData.Capabilities.Site.MediaTypes[2] != "native" {
		t.Errorf("capabilities.app.mediaTypes[2] should be native. Got %s", fileData.Capabilities.Site.MediaTypes[2])
	}
}

func TestGetUnknownBidder(t *testing.T) {
	bidderInfos := adapters.BidderInfos(make(map[string]adapters.BidderInfo))
	endpoint := info.NewBidderDetailsEndpoint(bidderInfos)
	req, err := http.NewRequest("GET", "http://prebid-server.com/info/bidders/someUnknownBidder", strings.NewReader(""))
	if err != nil {
		t.Fatalf("Failed to create a GET /info/bidders/someUnknownBidder request: %v", err)
	}

	params := []httprouter.Param{{
		Key:   "bidderName",
		Value: "someUnknownBidder",
	}}
	r := httptest.NewRecorder()

	endpoint(r, req, params)
	if r.Code != http.StatusNotFound {
		t.Errorf("GET /info/bidders/* should return a 404 on unknown bidders. Got %d", r.Code)
	}
}

// TestInfoFiles makes sure that static/bidder-info contains a .yaml file for every BidderName.
func TestInfoFiles(t *testing.T) {
	fileInfos, err := ioutil.ReadDir("../../static/bidder-info")
	if err != nil {
		t.Fatalf("Error reading the static/bidder-info directory: %v", err)
	}

	// Make sure that files exist for each BidderName
	for bidderName := range openrtb_ext.BidderMap {
		if _, err := os.Stat(fmt.Sprintf("../../static/bidder-info/%s.yaml", bidderName)); os.IsNotExist(err) {
			t.Errorf("static/bidder-info/%s.yaml not found. Did you forget to create it?", bidderName)
		}
	}

	if len(fileInfos) != len(openrtb_ext.BidderMap) {
		t.Errorf("static/bidder-info contains %d files, but the BidderMap has %d entries. These two should be in sync.", len(fileInfos), len(openrtb_ext.BidderMap))
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
		var fileInfoContent adapters.BidderInfo
		if err := yaml.Unmarshal(content, &fileInfoContent); err != nil {
			t.Errorf("Error interpreting content from static/bidder-info/%s: %v", fileInfo.Name(), err)
			continue
		}
		if err := validateInfo(&fileInfoContent); err != nil {
			t.Errorf("Invalid content in static/bidder-info/%s: %v", fileInfo.Name(), err)
		}
	}
}

func validateInfo(info *adapters.BidderInfo) error {
	if err := validateMaintainer(info.Maintainer); err != nil {
		return err
	}

	if err := validateCapabilities(info.Capabilities); err != nil {
		return err
	}

	return nil
}

func validateCapabilities(info *adapters.CapabilitiesInfo) error {
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

func validatePlatformInfo(info *adapters.PlatformInfo) error {
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

func validateMaintainer(info *adapters.MaintainerInfo) error {
	if info == nil || info.Email == "" {
		return errors.New("missing required field: maintainer.email")
	}
	return nil
}
