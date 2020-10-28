package amx

import (
	"encoding/json"
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
	t.Logf("TESTING JSON SAMPLES!!")
	adapterstest.RunJSONBidderTest(t, "amxtest", new(AMXAdapter))
}

func TestMakeRequestsPublisherIdOverride(t *testing.T) {
	var w, h int = 300, 250

	var width, height uint64 = uint64(w), uint64(h)

	adapter := NewAMXBidder(amxTestEndpoint)
	imp1 := openrtb.Imp{
		ID:  "sample_imp_1",
		Ext: json.RawMessage(defaultImpExt),
		Banner: &openrtb.Banner{
			W: &width,
			H: &height,
			Format: []openrtb.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb.BidRequest{
		User: &openrtb.User{ID: "example_user_id"},
		Imp:  []openrtb.Imp{imp1},
		Site: &openrtb.Site{
			Publisher: &openrtb.Publisher{
				ID: "1234567",
			},
		},
		ID: "1234",
	}

	actualAdapterRequests, err := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
	assert.Len(t, actualAdapterRequests, 1)

	assert.Empty(t, err)

	// check that the publisher ID overrides the site.publisher.id if provided
	var body openrtb.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &body); err != nil {
		t.Errorf("failed to read bid request")
	}

	assert.Equal(t, "publisher_id_example", body.Site.Publisher.ID)
	assert.Equal(t, "http://pbs-dev.amxrtb.com/auction/openrtb?v=pbs1.0", actualAdapterRequests[0].Uri)
}

func TestWillEnsurePublisher(t *testing.T) {
	adapter := NewAMXBidder(amxTestEndpoint)
	var width, height uint64 = 300, 250

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
		Imp:  []openrtb.Imp{imp1},
		Site: &openrtb.Site{Publisher: nil},
		ID:   "1234",
	}

	actualAdapterRequests, _ := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
	var body openrtb.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &body); err != nil {
		t.Errorf("failed to read bid request")
	}

	assert.NotNil(t, body.Site.Publisher)

	assert.Equal(t, "site_publisher_id", body.Site.Publisher.ID)
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

func getRequestBody(requests []*adapters.RequestData) *openrtb.BidRequest {
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

func TestMakeBidVideo(t *testing.T) {
	var w, h int = 640, 480

	var width, height uint64 = uint64(w), uint64(h)

	adapter := NewAMXBidder(amxTestEndpoint)
	imp1 := openrtb.Imp{
		ID:  "video_imp_1",
		Ext: json.RawMessage(defaultImpExt),
		Video: &openrtb.Video{
			W:     width,
			H:     height,
			MIMEs: []string{"video/mp4"},
		}}

	inputRequest := openrtb.BidRequest{
		Imp: []openrtb.Imp{imp1},
		Site: &openrtb.Site{
			Publisher: &openrtb.Publisher{
				ID: "1234567",
			},
		},
		User: &openrtb.User{ID: "amx_uid", BuyerUID: "amx_buid"},
		ID:   "1234",
	}

	actualAdapterRequests, _ := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
	assert.Len(t, actualAdapterRequests, 1, "expecting 1 request")

	body := getRequestBody(actualAdapterRequests)
	assert.NotNil(t, body, "body should be valid JSON")

	assert.Len(t, body.Imp, 1, "expecting 1 bid")

	resps, errs := adapter.MakeBids(body, &adapters.RequestData{}, &adapters.ResponseData{
		StatusCode: 200,
		Body: []byte(`{
			"id": "WQ5V2DWVTMNXABDD",
			"seatbid": [{
				"bid": [{
					"id": "TEST",
					"impid": "1",
					"price": 10.0,
					"adid": "1",
					"adm": "<?xml version=\"1.0\" encoding=\"UTF-8\" ?><VAST version=\"2.0\"><Ad id=\"128a6.44d74.46b3\"><InLine><Error><![CDATA[http://example.net/hbx/verr?e=]]></Error><Impression><![CDATA[http://example.net/hbx/vimp?lid=test&aid=testapp]]></Impression><Creatives><Creative sequence=\"1\"><Linear><Duration>00:00:15</Duration><TrackingEvents><Tracking event=\"firstQuartile\"><![CDATA[https://example.com?event=first_quartile]]></Tracking></TrackingEvents><VideoClicks><ClickThrough><![CDATA[http://example.com]]></ClickThrough></VideoClicks><MediaFiles><MediaFile delivery=\"progressive\" width=\"16\" height=\"9\" type=\"video/mp4\" bitrate=\"800\"><![CDATA[https://example.com/media.mp4]]></MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>",
					"adomain": ["amxrtb.com"],
					"iurl": "https://assets.a-mo.net/300x250.v2.png",
					"cid": "1",
					"crid": "1",
					"h": 600,
					"w": 300,
					"ext": {
						"himp": ["https://example.com/imp-tracker/pixel.gif?param=1&param2=2"],
						"startdelay": 0
					}
				}]
			}],
			"cur": "USD"
		}`),
	})

	assert.Empty(t, errs, "unexpected errors in response")
	assert.Len(t, resps.Bids, 1, "there should only be 1 bid")

	// it should be a video bid
	assert.Equal(t, openrtb_ext.BidTypeVideo, resps.Bids[0].BidType, "the bid should be video type")
}

func TestUserEidsOnly(t *testing.T) {
	var w, h int = 300, 250

	var width, height uint64 = uint64(w), uint64(h)

	adapter := NewAMXBidder(amxTestEndpoint)
	imp1 := openrtb.Imp{
		ID:  "imp1",
		Ext: json.RawMessage(defaultImpExt),
		Banner: &openrtb.Banner{
			W: &width,
			H: &height,
			Format: []openrtb.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb.BidRequest{
		Imp: []openrtb.Imp{imp1, imp1, imp1},
		Site: &openrtb.Site{
			Publisher: &openrtb.Publisher{
				ID: "10007",
			},
		},
		User: &openrtb.User{Ext: json.RawMessage(
			`{
				"eids": [{
				"source": "adserver.org",
					"uids": [{
						"id": "111111111111",
						"ext": {
							"rtiPartner": "TDID"
						}
					}]
				},{
					"source": "example.buid",
					"uids": [{
						"id": "123456"
					}]
				}]
			}`)},
		ID: "1234",
	}

	actualAdapterRequests, _ := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
	assert.Len(t, actualAdapterRequests, 1, "there should be 1 request")

	body := getRequestBody(actualAdapterRequests)
	assert.NotNil(t, body, "the generated OpenRTB request is not valid JSON")

	userExt := getUserExt(body.User)
	assert.NotNil(t, userExt, "the generated user.ext is invalid")

	assert.Len(t, userExt.Eids, 2)
	assert.Equal(t, "adserver.org", userExt.Eids[0].Source, "the eid source does is incorrect")
}

func TestVideoImpInsertion(t *testing.T) {
	var bidResp openrtb.BidResponse
	var bid openrtb.Bid
	payload := []byte(`{
    "id": "amx_request_id",
    "seatbid": [
        {
            "bid": [
                {
                    "id": "video_bid",
                    "impid": "video_imp_id",
                    "price": 6.11,
                    "nurl": "https://example2.com/nurl",
                    "adm": "<?xml version=\"1.0\" encoding=\"UTF-8\" ?><VAST version=\"2.0\"><Ad id=\"128a6.44d74.46b3\"><InLine><Error><![CDATA[http://example.net/hbx/verr?e=]]></Error><Impression><![CDATA[http://example.net/hbx/vimp?lid=test&aid=testapp]]></Impression><Creatives><Creative sequence=\"1\"><Linear><Duration>00:00:15</Duration><TrackingEvents><Tracking event=\"firstQuartile\"><![CDATA[https://example.com?event=first_quartile]]></Tracking></TrackingEvents><VideoClicks><ClickThrough><![CDATA[http://example.com]]></ClickThrough></VideoClicks><MediaFiles><MediaFile delivery=\"progressive\" width=\"16\" height=\"9\" type=\"video/mp4\" bitrate=\"800\"><![CDATA[https://example.com/media.mp4]]></MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>",
                    "crid": "123456789",
                    "w": 640,
                    "h": 480,
                    "ext": {
                        "himp": ["https://example.com/pixel.png"]
                    }
                },
                {
                    "id": "display_bid",
                    "impid": "display_imp_id",
                    "price": 1.23,
                    "adm": "<img src='https://example.com/300x250.png' height='250' width='300'/>",
                    "crid": "123456789",
                    "w": 300,
                    "h": 250,
                    "ext": {
                        "himp": ["https://example.com/pixel.png"]
                    }
                }
            ]
        }
    ]
}`)

	err := json.Unmarshal(payload, &bidResp)
	assert.Nil(t, err)
	bid = openrtb.Bid(bidResp.SeatBid[0].Bid[0])

	// get the EXT from it too..
	var bidExt amxBidExt
	err = json.Unmarshal(bid.Ext, &bidExt)
	assert.Nil(t, err)

	data := interpolateImpressions(bid, bidExt)
	assert.Contains(t, data, "example2.com/nurl")
	assert.Contains(t, data, "example.com/pixel.png")
}
