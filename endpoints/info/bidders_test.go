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
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/endpoints/info"
	"github.com/prebid/prebid-server/openrtb_ext"
	yaml "gopkg.in/yaml.v2"
)

func TestGetBiddersNoAliases(t *testing.T) {
	testGetBidders(t, map[string]string{})
}

func TestGetBiddersWithAliases(t *testing.T) {
	aliases := map[string]string{
		"test1": "appnexus",
		"test2": "rubicon",
		"test3": "openx",
	}
	testGetBidders(t, aliases)
}

func testGetBidders(t *testing.T, aliases map[string]string) {
	endpoint := info.NewBiddersEndpoint(aliases)

	req, err := http.NewRequest("GET", "http://prebid-server.com/info/bidders", strings.NewReader(""))
	if err != nil {
		assert.FailNow(t, "Failed to create a GET /info/bidders request: %v", err)
	}

	r := httptest.NewRecorder()
	endpoint(r, req, nil)

	assert.Equal(t, http.StatusOK, r.Code, "GET /info/bidders returned bad status: %d", r.Code)
	assert.Equal(t, "application/json", r.Header().Get("Content-Type"), "Bad /info/bidders content type. Expected application/json. Got %s", r.Header().Get("Content-Type"))

	bodyBytes := r.Body.Bytes()
	bidderSlice := make([]string, 0, len(openrtb_ext.BidderMap)+len(aliases))
	err = json.Unmarshal(bodyBytes, &bidderSlice)
	assert.NoError(t, err, "Failed to unmarshal /info/bidders response: %v", err)

	for _, bidderName := range bidderSlice {
		if _, ok := openrtb_ext.BidderMap[bidderName]; !ok {
			assert.Contains(t, aliases, bidderName, "Response from /info/bidders contained unexpected BidderName: %s", bidderName)
		}
	}

	assert.Len(t, bidderSlice, len(openrtb_ext.BidderMap)+len(aliases),
		"Response from /info/bidders did not match BidderMap. Expected %d elements. Got %d",
		len(openrtb_ext.BidderMap)+len(aliases), len(bidderSlice))
}

// TestGetSpecificBidders validates all the GET /info/bidders/{bidderName} endpoints
func TestGetSpecificBidders(t *testing.T) {
	// Setup:
	testCases := []struct {
		status      adapters.BidderStatus
		description string
	}{
		{
			status:      adapters.StatusActive,
			description: "case 1 - bidder status is active",
		},
		{
			status:      adapters.StatusDisabled,
			description: "case 2 - bidder status is disabled",
		},
	}

	for _, tc := range testCases {
		bidderDisabled := false
		if tc.status == adapters.StatusDisabled {
			bidderDisabled = true
		}
		cfg := blankAdapterConfigWithStatus(openrtb_ext.BidderList(), bidderDisabled)
		bidderInfos := adapters.ParseBidderInfos(cfg, "../../static/bidder-info", openrtb_ext.BidderList())
		endpoint := info.NewBidderDetailsEndpoint(bidderInfos, map[string]string{})

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

			// Execute:
			endpoint(r, req, params)

			// Verify:
			assert.Equal(t, http.StatusOK, r.Code, "GET /info/bidders/"+bidderName+" returned a %d. Expected 200", r.Code, tc.description)
			assert.Equal(t, "application/json", r.HeaderMap.Get("Content-Type"), "GET /info/bidders/"+bidderName+" returned Content-Type %s. Expected application/json", r.HeaderMap.Get("Content-Type"), tc.description)

			var resBidderInfo adapters.BidderInfo
			if err := json.Unmarshal(r.Body.Bytes(), &resBidderInfo); err != nil {
				assert.FailNow(t, "Failed to unmarshal JSON from endpoints/info/bidders/%s: %v", bidderName, err, tc.description)
			}

			assert.Equal(t, tc.status, resBidderInfo.Status, tc.description)
		}
	}
}

func TestGetBidderAccuracyNoAliases(t *testing.T) {
	testGetBidderAccuracy(t, "")
}

func TestGetBidderAccuracyAliases(t *testing.T) {
	testGetBidderAccuracy(t, "aliasedBidder")
}

// TestGetBidderAccuracyAlias validates the output for an alias of a known file.
func testGetBidderAccuracy(t *testing.T, alias string) {
	cfg := blankAdapterConfig(openrtb_ext.BidderList())
	bidderInfos := adapters.ParseBidderInfos(cfg, "../../adapters/adapterstest/bidder-info", []openrtb_ext.BidderName{openrtb_ext.BidderName("someBidder")})

	aliases := map[string]string{}
	bidder := "someBidder"
	if len(alias) > 0 {
		aliases[alias] = bidder
		bidder = alias
	}

	endpoint := info.NewBidderDetailsEndpoint(bidderInfos, aliases)
	req, err := http.NewRequest("GET", "http://prebid-server.com/info/bidders/"+bidder, strings.NewReader(""))
	assert.NoError(t, err, "Failed to create a GET /info/bidders request: %v", err)
	params := []httprouter.Param{{
		Key:   "bidderName",
		Value: bidder,
	}}

	r := httptest.NewRecorder()
	endpoint(r, req, params)

	var fileData adapters.BidderInfo
	if err := json.Unmarshal(r.Body.Bytes(), &fileData); err != nil {
		assert.FailNow(t, "Failed to unmarshal JSON from endpoints/info/sample/someBidder.yaml: %v", err)
	}

	assert.Equal(t, "some-email@domain.com", fileData.Maintainer.Email, "maintainer.email should be some-email@domain.com. Got %s", fileData.Maintainer.Email)
	assert.Len(t, fileData.Capabilities.App.MediaTypes, 2, "Expected 2 supported mediaTypes on app. Got %d", len(fileData.Capabilities.App.MediaTypes))
	assert.Equal(t, openrtb_ext.BidType("banner"), fileData.Capabilities.App.MediaTypes[0], "capabilities.app.mediaTypes[0] should be banner. Got %s", fileData.Capabilities.App.MediaTypes[0])
	assert.Equal(t, openrtb_ext.BidType("native"), fileData.Capabilities.App.MediaTypes[1], "capabilities.app.mediaTypes[1] should be native. Got %s", fileData.Capabilities.App.MediaTypes[1])
	assert.Len(t, fileData.Capabilities.Site.MediaTypes, 3, "Expected 3 supported mediaTypes on app. Got %d", len(fileData.Capabilities.Site.MediaTypes))
	assert.Equal(t, openrtb_ext.BidType("banner"), fileData.Capabilities.Site.MediaTypes[0], "capabilities.app.mediaTypes[0] should be banner. Got %s", fileData.Capabilities.Site.MediaTypes[0])
	assert.Equal(t, openrtb_ext.BidType("video"), fileData.Capabilities.Site.MediaTypes[1], "capabilities.app.mediaTypes[1] should be video. Got %s", fileData.Capabilities.Site.MediaTypes[1])
	assert.Equal(t, openrtb_ext.BidType("native"), fileData.Capabilities.Site.MediaTypes[2], "capabilities.app.mediaTypes[2] should be native. Got %s", fileData.Capabilities.Site.MediaTypes[2])
	if len(alias) > 0 {
		assert.Equal(t, "someBidder", fileData.AliasOf, "aliasOf should be \"someBidder\". Got \"%s\"", fileData.AliasOf)
	} else {
		assert.Zero(t, len(fileData.AliasOf), "aliasOf should be empty. Got \"%s\"", fileData.AliasOf)
	}
}

func TestGetUnknownBidder(t *testing.T) {
	bidderInfos := adapters.BidderInfos(make(map[string]adapters.BidderInfo))
	endpoint := info.NewBidderDetailsEndpoint(bidderInfos, map[string]string{})
	req, err := http.NewRequest("GET", "http://prebid-server.com/info/bidders/someUnknownBidder", strings.NewReader(""))
	if err != nil {
		assert.FailNow(t, "Failed to create a GET /info/bidders/someUnknownBidder request: %v", err)
	}

	params := []httprouter.Param{{
		Key:   "bidderName",
		Value: "someUnknownBidder",
	}}
	r := httptest.NewRecorder()

	endpoint(r, req, params)
	assert.Equal(t, http.StatusNotFound, r.Code, "GET /info/bidders/* should return a 404 on unknown bidders. Got %d", r.Code)
}
func TestGetAllBidders(t *testing.T) {
	cfg := blankAdapterConfig(openrtb_ext.BidderList())
	bidderInfos := adapters.ParseBidderInfos(cfg, "../../static/bidder-info", openrtb_ext.BidderList())
	endpoint := info.NewBidderDetailsEndpoint(bidderInfos, map[string]string{})
	req, err := http.NewRequest("GET", "http://prebid-server.com/info/bidders/all", strings.NewReader(""))
	if err != nil {
		assert.FailNow(t, "Failed to create a GET /info/bidders/someUnknownBidder request: %v", err)
	}
	params := []httprouter.Param{{
		Key:   "bidderName",
		Value: "all",
	}}
	r := httptest.NewRecorder()

	endpoint(r, req, params)
	assert.Equal(t, http.StatusOK, r.Code, "GET /info/bidders/all returned a %d. Expected 200", r.Code)
	assert.Equal(t, "application/json", r.HeaderMap.Get("Content-Type"), "GET /info/bidders/all returned Content-Type %s. Expected application/json", r.HeaderMap.Get("Content-Type"))

	var resBidderInfos map[string]adapters.BidderInfo

	if err := json.Unmarshal(r.Body.Bytes(), &resBidderInfos); err != nil {
		assert.FailNow(t, "Failed to unmarshal JSON from endpoints/info/sample/someBidder.yaml: %v", err)
	}

	assert.Len(t, resBidderInfos, len(bidderInfos), "GET /info/bidders/all should respond with all bidders info")
}

// TestInfoFiles makes sure that static/bidder-info contains a .yaml file for every BidderName.
func TestInfoFiles(t *testing.T) {
	fileInfos, err := ioutil.ReadDir("../../static/bidder-info")
	if err != nil {
		assert.FailNow(t, "Error reading the static/bidder-info directory: %v", err)
	}

	// Make sure that files exist for each BidderName
	for bidderName := range openrtb_ext.BidderMap {
		_, err := os.Stat(fmt.Sprintf("../../static/bidder-info/%s.yaml", bidderName))
		assert.False(t, os.IsNotExist(err), "static/bidder-info/%s.yaml not found. Did you forget to create it?", bidderName)
	}

	assert.Len(t, fileInfos, len(openrtb_ext.BidderMap), "static/bidder-info contains %d files, but the BidderMap has %d entries. These two should be in sync.", len(fileInfos), len(openrtb_ext.BidderMap))

	// Make sure that all the files have valid content
	for _, fileInfo := range fileInfos {
		infoFileData, err := os.Open(fmt.Sprintf("../../static/bidder-info/%s", fileInfo.Name()))
		assert.NoError(t, err, "Unexpected error: %v", err)

		content, err := ioutil.ReadAll(infoFileData)
		assert.NoError(t, err, "Failed to read static/bidder-info/%s: %v", fileInfo.Name(), err)

		var fileInfoContent adapters.BidderInfo
		err = yaml.Unmarshal(content, &fileInfoContent)
		assert.NoError(t, err, "Error interpreting content from static/bidder-info/%s: %v", fileInfo.Name(), err)

		err = validateInfo(&fileInfoContent)
		assert.NoError(t, err, "Invalid content in static/bidder-info/%s: %v", fileInfo.Name(), err)

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

func blankAdapterConfig(bidderList []openrtb_ext.BidderName) map[string]config.Adapter {
	return blankAdapterConfigWithStatus(bidderList, false)
}

func blankAdapterConfigWithStatus(bidderList []openrtb_ext.BidderName, biddersAreDisabled bool) map[string]config.Adapter {
	adapters := make(map[string]config.Adapter)
	for _, b := range bidderList {
		adapters[strings.ToLower(string(b))] = config.Adapter{
			Disabled: biddersAreDisabled,
		}
	}
	return adapters
}
