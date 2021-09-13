package amx

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const (
	amxTestEndpoint  = "http://pbs-dev.amxrtb.com/auction/openrtb"
	sampleVastADM    = "<?xml version=\"1.0\" encoding=\"UTF-8\" ?><VAST version=\"2.0\"><Ad id=\"128a6.44d74.46b3\"><InLine><Error><![CDATA[http://example.net/hbx/verr?e=]]></Error><Impression><![CDATA[http://example.net/hbx/vimp?lid=test&aid=testapp]]></Impression><Creatives><Creative sequence=\"1\"><Linear><Duration>00:00:15</Duration><TrackingEvents><Tracking event=\"firstQuartile\"><![CDATA[https://example.com?event=first_quartile]]></Tracking></TrackingEvents><VideoClicks><ClickThrough><![CDATA[http://example.com]]></ClickThrough></VideoClicks><MediaFiles><MediaFile delivery=\"progressive\" width=\"16\" height=\"9\" type=\"video/mp4\" bitrate=\"800\"><![CDATA[https://example.com/media.mp4]]></MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>"
	sampleDisplayADM = "<img src='https://example.com/300x250.png' height='250' width='300'/>"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAMX, config.Adapter{
		Endpoint: amxTestEndpoint})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "amxtest", bidder)
}

func TestEndpointMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAMX, config.Adapter{
		Endpoint: " http://leading.space.is.invalid"})

	assert.Error(t, buildErr)
}

func TestEndpointQueryStringMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAMX, config.Adapter{
		Endpoint: "http://invalid.query.from.go.docs/page?%gh&%ij"})

	assert.Error(t, buildErr)
}

func TestMakeRequestsTagID(t *testing.T) {
	var w, h int = 300, 250
	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderAMX, config.Adapter{
		Endpoint: amxTestEndpoint})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	type testCase struct {
		tagID         string
		extAdUnitID   string
		expectedTagID string
		blankNil      bool
	}

	tests := []testCase{
		{tagID: "tag-id", extAdUnitID: "ext.adUnitID", expectedTagID: "ext.adUnitID", blankNil: false},
		{tagID: "tag-id", extAdUnitID: "", expectedTagID: "tag-id", blankNil: false},
		{tagID: "tag-id", extAdUnitID: "", expectedTagID: "tag-id", blankNil: true},
		{tagID: "", extAdUnitID: "", expectedTagID: "", blankNil: true},
		{tagID: "", extAdUnitID: "", expectedTagID: "", blankNil: false},
		{tagID: "", extAdUnitID: "ext.adUnitID", expectedTagID: "ext.adUnitID", blankNil: true},
		{tagID: "", extAdUnitID: "ext.adUnitID", expectedTagID: "ext.adUnitID", blankNil: false},
	}

	for _, tc := range tests {
		imp1 := openrtb2.Imp{
			ID: "sample_imp_1",
			Banner: &openrtb2.Banner{
				W: &width,
				H: &height,
				Format: []openrtb2.Format{
					{W: 300, H: 250},
				},
			}}

		if tc.extAdUnitID != "" || !tc.blankNil {
			imp1.Ext = json.RawMessage(
				fmt.Sprintf(`{"bidder":{"adUnitId":"%s"}}`, tc.extAdUnitID))
		}

		if tc.tagID != "" || !tc.blankNil {
			imp1.TagID = tc.tagID
		}

		inputRequest := openrtb2.BidRequest{
			User: &openrtb2.User{},
			Imp:  []openrtb2.Imp{imp1},
			Site: &openrtb2.Site{},
		}

		actualAdapterRequests, err := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
		assert.Len(t, actualAdapterRequests, 1)
		assert.Empty(t, err)
		var body openrtb2.BidRequest
		assert.Nil(t, json.Unmarshal(actualAdapterRequests[0].Body, &body))
		assert.Equal(t, tc.expectedTagID, body.Imp[0].TagID)
	}
}

func TestMakeRequestsPublisherId(t *testing.T) {
	var w, h int = 300, 250
	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderAMX, config.Adapter{
		Endpoint: amxTestEndpoint})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	type testCase struct {
		publisherID         string
		extTagID            string
		expectedPublisherID string
		blankNil            bool
	}

	tests := []testCase{
		{publisherID: "publisher.id", extTagID: "ext.tagId", expectedPublisherID: "ext.tagId", blankNil: false},
		{publisherID: "publisher.id", extTagID: "", expectedPublisherID: "publisher.id", blankNil: false},
		{publisherID: "", extTagID: "ext.tagId", expectedPublisherID: "ext.tagId", blankNil: false},
		{publisherID: "", extTagID: "ext.tagId", expectedPublisherID: "ext.tagId", blankNil: true},
		{publisherID: "publisher.id", extTagID: "", expectedPublisherID: "publisher.id", blankNil: false},
		{publisherID: "publisher.id", extTagID: "", expectedPublisherID: "publisher.id", blankNil: true},
	}

	for _, tc := range tests {
		imp1 := openrtb2.Imp{
			ID: "sample_imp_1",
			Banner: &openrtb2.Banner{
				W: &width,
				H: &height,
				Format: []openrtb2.Format{
					{W: 300, H: 250},
				},
			}}

		if tc.extTagID != "" || !tc.blankNil {
			imp1.Ext = json.RawMessage(
				fmt.Sprintf(`{"bidder":{"tagId":"%s"}}`, tc.extTagID))
		}

		inputRequest := openrtb2.BidRequest{
			User: &openrtb2.User{ID: "example_user_id"},
			Imp:  []openrtb2.Imp{imp1},
			Site: &openrtb2.Site{},
			ID:   "1234",
		}

		if tc.publisherID != "" || !tc.blankNil {
			inputRequest.Site.Publisher = &openrtb2.Publisher{
				ID: tc.publisherID,
			}
		}

		actualAdapterRequests, err := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
		assert.Len(t, actualAdapterRequests, 1)
		assert.Empty(t, err)
		var body openrtb2.BidRequest
		assert.Nil(t, json.Unmarshal(actualAdapterRequests[0].Body, &body))
		assert.Equal(t, tc.expectedPublisherID, body.Site.Publisher.ID)
	}
}

var vastImpressionRXP = regexp.MustCompile(`<Impression><!\[CDATA\[[^\]]*\]\]></Impression>`)

func countImpressionPixels(vast string) int {
	matches := vastImpressionRXP.FindAllIndex([]byte(vast), -1)
	return len(matches)
}

func TestVideoImpInsertion(t *testing.T) {
	markup := interpolateImpressions(openrtb2.Bid{
		AdM:  sampleVastADM,
		NURL: "https://example2.com/nurl",
	}, amxBidExt{Himp: []string{"https://example.com/pixel.png"}})
	assert.Contains(t, markup, "example2.com/nurl")
	assert.Contains(t, markup, "example.com/pixel.png")
	assert.Equal(t, 3, countImpressionPixels(markup), "should have 3 Impression pixels")

	// make sure that a blank NURL won't result in a blank impression tag
	markup = interpolateImpressions(openrtb2.Bid{
		AdM:  sampleVastADM,
		NURL: "",
	}, amxBidExt{})
	assert.Equal(t, 1, countImpressionPixels(markup), "should have 1 impression pixels")

	// we should also ignore blank ext.Himp pixels
	markup = interpolateImpressions(openrtb2.Bid{
		AdM:  sampleVastADM,
		NURL: "https://example-nurl.com/nurl",
	}, amxBidExt{Himp: []string{"", "", ""}})
	assert.Equal(t, 2, countImpressionPixels(markup), "should have 2 impression pixels")
}

func TestNoDisplayImpInsertion(t *testing.T) {
	data := interpolateImpressions(openrtb2.Bid{
		AdM:  sampleDisplayADM,
		NURL: "https://example2.com/nurl",
	}, amxBidExt{Himp: []string{"https://example.com/pixel.png"}})
	assert.NotContains(t, data, "example2.com/nurl")
	assert.NotContains(t, data, "example.com/pixel.png")
}
