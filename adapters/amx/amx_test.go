package amx

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
)

const (
	amxTestEndpoint  = "http://pbs-dev.amxrtb.com/auction/openrtb"
	sampleVastADM    = "<?xml version=\"1.0\" encoding=\"UTF-8\" ?><VAST version=\"2.0\"><Ad id=\"128a6.44d74.46b3\"><InLine><Error><![CDATA[http://example.net/hbx/verr?e=]]></Error><Impression><![CDATA[http://example.net/hbx/vimp?lid=test&aid=testapp]]></Impression><Creatives><Creative sequence=\"1\"><Linear><Duration>00:00:15</Duration><TrackingEvents><Tracking event=\"firstQuartile\"><![CDATA[https://example.com?event=first_quartile]]></Tracking></TrackingEvents><VideoClicks><ClickThrough><![CDATA[http://example.com]]></ClickThrough></VideoClicks><MediaFiles><MediaFile delivery=\"progressive\" width=\"16\" height=\"9\" type=\"video/mp4\" bitrate=\"800\"><![CDATA[https://example.com/media.mp4]]></MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>"
	sampleDisplayADM = "<img src='https://example.com/300x250.png' height='250' width='300'/>"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAMX, config.Adapter{
		Endpoint: amxTestEndpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "amxtest", bidder)
}

func TestEndpointMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAMX, config.Adapter{
		Endpoint: " http://leading.space.is.invalid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestEndpointQueryStringMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAMX, config.Adapter{
		Endpoint: "http://invalid.query.from.go.docs/page?%gh&%ij"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestMakeRequestsTagID(t *testing.T) {
	var w, h int = 300, 250
	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderAMX, config.Adapter{
		Endpoint: amxTestEndpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

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
		Endpoint: amxTestEndpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

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

func TestMakeBids(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAMX, config.Adapter{
		Endpoint: amxTestEndpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Failed to build bidder: %v", buildErr)
	}

	type testCase struct {
		bidType      openrtb_ext.BidType
		adm          string
		extRaw       string
		seatName     string
		demandSource string
		valid        bool
	}

	tests := []testCase{
		{openrtb_ext.BidTypeNative, `{"assets":[]}`, `{"ct":10}`, "", "", true},
		{openrtb_ext.BidTypeBanner, sampleDisplayADM, `{"ct": 1}`, "", "", true},
		{openrtb_ext.BidTypeBanner, sampleDisplayADM, `{"ct": "invalid"}`, "", "", false},
		{openrtb_ext.BidTypeBanner, sampleDisplayADM, `{}`, "", "", true},
		{openrtb_ext.BidTypeBanner, sampleDisplayADM, `{"bc": "amx-pmp"}`, "amx-pmp", "", true},
		{openrtb_ext.BidTypeBanner, sampleDisplayADM, `{"ds": "pmp-1"}`, "", "pmp-1", true},
		{openrtb_ext.BidTypeBanner, sampleDisplayADM, `{"bc": "amx-pmp", "ds": "pmp-1"}`, "amx-pmp", "pmp-1", true},
		{openrtb_ext.BidTypeVideo, sampleVastADM, `{"startdelay": 1}`, "", "", true},
		{openrtb_ext.BidTypeBanner, sampleVastADM, `{"ct": 1}`, "", "", true}, // the server shouldn't do this
	}

	for _, test := range tests {
		bid := openrtb2.Bid{
			AdM:   test.adm,
			Price: 1,
			Ext:   json.RawMessage(test.extRaw),
		}

		sb := openrtb2.SeatBid{
			Bid: []openrtb2.Bid{bid},
		}

		resp := openrtb2.BidResponse{
			SeatBid: []openrtb2.SeatBid{sb},
		}

		respJson, jsonErr := json.Marshal(resp)
		if jsonErr != nil {
			t.Fatalf("Failed to serialize test bid %v: %v", test, jsonErr)
		}

		bids, errs := bidder.MakeBids(nil, nil, &adapters.ResponseData{
			StatusCode: 200,
			Body:       respJson,
		})

		if !test.valid {
			assert.Len(t, errs, 1)
			continue
		}

		if len(errs) > 0 {
			t.Fatalf("Failed to make bids: %v", errs)
		}

		assert.Len(t, bids.Bids, 1)
		assert.Equal(t, test.bidType, bids.Bids[0].BidType)

		br := bids.Bids[0]
		assert.Equal(t, openrtb_ext.BidderName(test.seatName), br.Seat)
		assert.Equal(t, test.demandSource, br.BidMeta.DemandSource)
	}

}
