package amx

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"

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

	if actualAdapterRequests == nil {
		t.Errorf("request should be nil")
	}

	if len(err) != 0 {
		t.Errorf("We should have no error")
	}

	// check that the publisher ID overrides the site.publisher.id if provided
	var body openrtb.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &body); err != nil {
		t.Errorf("failed to read bid request")
	}
	if body.Site.Publisher.ID != "publisher_id_example" {
		t.Errorf("invalid publisher id: %s", body.Site.Publisher.ID)
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
		App: &openrtb.App{ID: "cansanuabnua", Publisher: &openrtb.Publisher{ID: "whatever"}},
		ID:  "1234",
	}

	actualAdapterRequests, _ := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if len(actualAdapterRequests) != 1 {
		t.Errorf("openrtb type should be an Array when it's an App")
	}
	var body openrtb.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &body); err != nil {
		t.Errorf("failed to read bid request")
	}

	if body.App == nil {
		t.Errorf("app property should be populated")
	}

	if body.App.Publisher.ID != "site_publisher_id" {
		t.Errorf("incorrect publisher ID for app")
	}

	if body.Site.Publisher.ID != "site_publisher_id" {
		t.Errorf("incorrect publisher ID for site")
	}
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

	if len(actualAdapterRequests) != 1 {
		t.Errorf("should have 1 request")
	}

	body := getRequestBody(actualAdapterRequests)
	if body == nil {
		t.Errorf("invalid request: cannot parse")
	}

	if len(body.Imp) != 1 {
		t.Errorf("must have 1 bids")
	}

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

	if len(errs) > 0 {
		t.Errorf("unexpected errors in response: %v", errs)
	}

	if len(resps.Bids) != 1 {
		t.Errorf("got %d bids, expected 1", len(resps.Bids))
	}

	// it should be a video bid
	if resps.Bids[0].BidType != openrtb_ext.BidTypeVideo {
		t.Errorf("bid should be type video, got %v", resps.Bids[0].BidType)
	}
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
		User: &openrtb.User{Ext: json.RawMessage(`{"eids": [{
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
	if len(actualAdapterRequests) != 1 {
		t.Errorf("should have 1 request")
	}

	body := getRequestBody(actualAdapterRequests)
	if body == nil {
		t.Errorf("invalid body - expecting valid body")
	}

	userExt := getUserExt(body.User)
	if userExt == nil {
		t.Errorf("invalid user.ext - should have eids")
	}

	if len(userExt.Eids) != 2 {
		t.Errorf("user.ext.eids should have 2 elements")
	}

	if userExt.Eids[0].Source != "adserver.org" {
		t.Errorf("invalid eids -- does not match input")
	}
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
	if err != nil {
		t.Errorf("Payload is invalid - %v", err)
	}
	bid = openrtb.Bid(bidResp.SeatBid[0].Bid[0])

	// get the EXT from it too..
	var bidExt amxBidExt
	err = json.Unmarshal(bid.Ext, &bidExt)
	if err != nil {
		t.Errorf("Invalid bid.ext: %v", err)
	}

	data := interpolateImpressions(bid, bidExt)
	find := strings.Index(data, "example2.com/nurl")
	if find == -1 {
		t.Errorf("String was not found")
	}

	if strings.Index(data, "example.com/pixel.png") == -1 {
		t.Errorf("ext.himp not interpolated into vast: %s", data)
	}
}
