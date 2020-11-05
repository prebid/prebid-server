package amx

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

var (
	bidRequest string
)

const (
	amxTestEndpoint = "http://pbs-dev.amxrtb.com/auction/openrtb"
	defaultImpExt   = "{\"bidder\":{\"tagId\":\"publisher_id_example\"}}"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "amxtest", NewAMXBidder(amxTestEndpoint))
}

func TestMakeRequestsPublisherId(t *testing.T) {
	var w, h int = 300, 250
	var width, height uint64 = uint64(w), uint64(h)
	adapter := NewAMXBidder(amxTestEndpoint)

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
		imp1 := openrtb.Imp{
			ID: "sample_imp_1",
			Banner: &openrtb.Banner{
				W: &width,
				H: &height,
				Format: []openrtb.Format{
					{W: 300, H: 250},
				},
			}}

		if tc.extTagID != "" || !tc.blankNil {
			imp1.Ext = json.RawMessage(
				fmt.Sprintf(`{"bidder":{"tagId":"%s"}}`, tc.extTagID))
		}

		inputRequest := openrtb.BidRequest{
			User: &openrtb.User{ID: "example_user_id"},
			Imp:  []openrtb.Imp{imp1},
			Site: &openrtb.Site{},
			ID:   "1234",
		}

		if tc.publisherID != "" || !tc.blankNil {
			inputRequest.Site.Publisher = &openrtb.Publisher{
				ID: tc.publisherID,
			}
		}

		actualAdapterRequests, err := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
		assert.Len(t, actualAdapterRequests, 1)
		assert.Empty(t, err)
		var body openrtb.BidRequest
		assert.Nil(t, json.Unmarshal(actualAdapterRequests[0].Body, &body))
		assert.Equal(t, tc.expectedPublisherID, body.Site.Publisher.ID)
	}
}

func TestMakeRequestsApp(t *testing.T) {
	var w, h int = 300, 250
	var width, height uint64 = uint64(w), uint64(h)

	adapter := NewAMXBidder(amxTestEndpoint)
	imp1 := openrtb.Imp{
		ID:  "sample_imp_1",
		Ext: json.RawMessage("{\"bidder\":{\"tagId\":\"site_publisher_id\"}}"),
		Banner: &openrtb.Banner{
			W: &width,
			H: &height,
			Format: []openrtb.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb.BidRequest{
		Imp: []openrtb.Imp{imp1},
		Site: &openrtb.Site{
			Publisher: &openrtb.Publisher{
				ID: "1234567",
			},
		},
		ID: "1234",
	}

	actualAdapterRequests, _ := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
	assert.Len(t, actualAdapterRequests, 1, "expecting 1 request")

	var body openrtb.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &body); err != nil {
		t.Errorf("failed to read bid request")
	}

	assert.Equal(t, "site_publisher_id", body.Site.Publisher.ID)
}

func getRequestBody(t *testing.T, requests []*adapters.RequestData) *openrtb.BidRequest {
	assert.GreaterOrEqual(t, 1, len(requests))

	var body openrtb.BidRequest
	if err := json.Unmarshal(requests[0].Body, &body); err == nil {
		return &body
	}
	return nil
}

func getUserExt(user *openrtb.User) *openrtb_ext.ExtUser {
	if user == nil {
		return nil
	}
	var userExt openrtb_ext.ExtUser
	if err := json.Unmarshal(user.Ext, &userExt); err == nil {
		return &userExt
	}
	return nil
}

const (
	sampleVastADM    = "<?xml version=\"1.0\" encoding=\"UTF-8\" ?><VAST version=\"2.0\"><Ad id=\"128a6.44d74.46b3\"><InLine><Error><![CDATA[http://example.net/hbx/verr?e=]]></Error><Impression><![CDATA[http://example.net/hbx/vimp?lid=test&aid=testapp]]></Impression><Creatives><Creative sequence=\"1\"><Linear><Duration>00:00:15</Duration><TrackingEvents><Tracking event=\"firstQuartile\"><![CDATA[https://example.com?event=first_quartile]]></Tracking></TrackingEvents><VideoClicks><ClickThrough><![CDATA[http://example.com]]></ClickThrough></VideoClicks><MediaFiles><MediaFile delivery=\"progressive\" width=\"16\" height=\"9\" type=\"video/mp4\" bitrate=\"800\"><![CDATA[https://example.com/media.mp4]]></MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>"
	sampleDisplayADM = "<img src='https://example.com/300x250.png' height='250' width='300'/>"
)

var vastImpressionRXP = regexp.MustCompile(`<Impression><!\[CDATA\[[^\]]*\]\]></Impression>`)

func countImpressionPixels(vast string) int {
	matches := vastImpressionRXP.FindAllIndex([]byte(vast), -1)
	return len(matches)
}

func TestVideoImpInsertion(t *testing.T) {
	markup := interpolateImpressions(openrtb.Bid{
		AdM:  sampleVastADM,
		NURL: "https://example2.com/nurl",
	}, amxBidExt{Himp: []string{"https://example.com/pixel.png"}})
	assert.Contains(t, markup, "example2.com/nurl")
	assert.Contains(t, markup, "example.com/pixel.png")
	assert.Equal(t, 3, countImpressionPixels(markup), "should have 3 Impression pixels")

	// make sure that a blank NURL won't result in a blank impression tag
	markup = interpolateImpressions(openrtb.Bid{
		AdM:  sampleVastADM,
		NURL: "",
	}, amxBidExt{})
	assert.Equal(t, 1, countImpressionPixels(markup), "should have 1 impression pixels")

	// we should also ignore blank ext.Himp pixels
	markup = interpolateImpressions(openrtb.Bid{
		AdM:  sampleVastADM,
		NURL: "https://example-nurl.com/nurl",
	}, amxBidExt{Himp: []string{"", "", ""}})
	assert.Equal(t, 2, countImpressionPixels(markup), "should have 2 impression pixels")
}

func TestNoDisplayImpInsertion(t *testing.T) {
	data := interpolateImpressions(openrtb.Bid{
		AdM:  sampleDisplayADM,
		NURL: "https://example2.com/nurl",
	}, amxBidExt{Himp: []string{"https://example.com/pixel.png"}})
	assert.NotContains(t, data, "example2.com/nurl")
	assert.NotContains(t, data, "example.com/pixel.png")
}
