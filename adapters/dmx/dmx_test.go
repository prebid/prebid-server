package dmx

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

var (
	bidRequest string
)

func TestFetchParams(t *testing.T) {
	var w, h int = 300, 250

	var width, height int64 = int64(w), int64(h)
	var arrImp []openrtb2.Imp
	var imps = fetchParams(
		dmxExt{Bidder: dmxParams{
			TagId:       "222",
			PublisherId: "5555",
		}},
		openrtb2.Imp{ID: "32"},
		openrtb2.Imp{ID: "32"},
		arrImp,
		&openrtb2.Banner{W: &width, H: &height, Format: []openrtb2.Format{
			{W: 300, H: 250},
		}},
		nil,
		1)
	var imps2 = fetchParams(
		dmxExt{Bidder: dmxParams{
			DmxId:    "222",
			MemberId: "5555",
		}},
		openrtb2.Imp{ID: "32"},
		openrtb2.Imp{ID: "32"},
		arrImp,
		&openrtb2.Banner{W: &width, H: &height, Format: []openrtb2.Format{
			{W: 300, H: 250},
		}},
		nil,
		1)
	if len(imps) == 0 {
		t.Errorf("should increment the length by one")
	}

	if len(imps2) == 0 {
		t.Errorf("should increment the length by one")
	}

}
func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "dmxtest", bidder)
}

func TestMakeRequestsOtherPlacement(t *testing.T) {
	var w, h int = 300, 250

	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{
		Endpoint: "https://dmx.districtm.io/b/v2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb2.BidRequest{
		User: &openrtb2.User{ID: "bscakucbkasucbkasunscancasuin"},
		Imp:  []openrtb2.Imp{imp1},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "10007",
			},
		},
		ID: "1234",
	}

	actualAdapterRequests, err := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if actualAdapterRequests == nil {
		t.Errorf("request should be nil")
	}
	if len(err) != 0 {
		t.Errorf("We should have no error")
	}

}

func TestMakeRequestsInvalid(t *testing.T) {
	var w, h int = 300, 250

	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{
		Endpoint: "https://dmx.districtm.io/b/v2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp1},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "10007",
			},
		},
		ID: "1234",
	}

	actualAdapterRequests, err := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if len(actualAdapterRequests) != 0 {
		t.Errorf("request should be nil")
	}
	if len(err) == 0 {
		t.Errorf("We should have no error")
	}

}

func TestMakeRequestNoSite(t *testing.T) {
	var w, h int = 300, 250

	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{
		Endpoint: "https://dmx.districtm.io/b/v2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp1},
		App: &openrtb2.App{ID: "cansanuabnua", Publisher: &openrtb2.Publisher{ID: "whatever"}},
		ID:  "1234",
	}

	actualAdapterRequests, _ := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if len(actualAdapterRequests) != 1 {
		t.Errorf("openrtb type should be an Array when it's an App")
	}
	var the_body openrtb2.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &the_body); err != nil {
		t.Errorf("failed to read bid request")
	}

	if the_body.App == nil {
		t.Errorf("app property should be populated")
	}

	if the_body.App.Publisher.ID == "" {
		t.Errorf("Missing publisher ID must be in")
	}
}

func TestMakeRequestsApp(t *testing.T) {
	var w, h int = 300, 250

	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{
		Endpoint: "https://dmx.districtm.io/b/v2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp1},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "10007",
			},
		},
		App: &openrtb2.App{ID: "cansanuabnua", Publisher: &openrtb2.Publisher{ID: "whatever"}},
		ID:  "1234",
	}

	actualAdapterRequests, _ := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if len(actualAdapterRequests) != 1 {
		t.Errorf("openrtb type should be an Array when it's an App")
	}
	var the_body openrtb2.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &the_body); err != nil {
		t.Errorf("failed to read bid request")
	}

	if the_body.App == nil {
		t.Errorf("app property should be populated")
	}

}

func TestMakeRequestsNoUser(t *testing.T) {
	var w, h int = 300, 250

	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{
		Endpoint: "https://dmx.districtm.io/b/v2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp1},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "10007",
			},
		},
		ID: "1234",
	}

	actualAdapterRequests, _ := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if actualAdapterRequests != nil {
		t.Errorf("openrtb type should be empty")
	}

}

func TestMakeRequests(t *testing.T) {
	var w, h int = 300, 250

	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{
		Endpoint: "https://dmx.districtm.io/b/v2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}
	imp2 := openrtb2.Imp{
		ID:  "imp2",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}
	imp3 := openrtb2.Imp{
		ID:  "imp3",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp1, imp2, imp3},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "10007",
			},
		},
		User: &openrtb2.User{ID: "districtmID"},
		ID:   "1234",
	}

	actualAdapterRequests, _ := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if len(actualAdapterRequests) != 1 {
		t.Errorf("should have 1 request")
	}
	var the_body openrtb2.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &the_body); err != nil {
		t.Errorf("failed to read bid request")
	}

	if len(the_body.Imp) != 3 {
		t.Errorf("must have 3 bids")
	}

}

func TestMakeBidVideo(t *testing.T) {
	var w, h int = 640, 480

	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{
		Endpoint: "https://dmx.districtm.io/b/v2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Video: &openrtb2.Video{
			W:     width,
			H:     height,
			MIMEs: []string{"video/mp4"},
		}}

	inputRequest := openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp1},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "10007",
			},
		},
		User: &openrtb2.User{ID: "districtmID"},
		ID:   "1234",
	}

	actualAdapterRequests, _ := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if len(actualAdapterRequests) != 1 {
		t.Errorf("should have 1 request")
	}
	var the_body openrtb2.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &the_body); err != nil {
		t.Errorf("failed to read bid request")
	}

	if len(the_body.Imp) != 1 {
		t.Errorf("must have 1 bids")
	}
}

func TestMakeBidsNoContent(t *testing.T) {
	var w, h int = 300, 250

	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{
		Endpoint: "https://dmx.districtm.io/b/v2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp1},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "10007",
			},
		},
		User: &openrtb2.User{ID: "districtmID"},
		ID:   "1234",
	}

	actualAdapterRequests, _ := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	_, err204 := bidder.MakeBids(&inputRequest, actualAdapterRequests[0], &adapters.ResponseData{StatusCode: 204})

	if err204 != nil {
		t.Errorf("Was expecting nil")
	}

	_, err400 := bidder.MakeBids(&inputRequest, actualAdapterRequests[0], &adapters.ResponseData{StatusCode: 400})

	if err400 == nil {
		t.Errorf("Was expecting error")
	}

	_, err500 := bidder.MakeBids(&inputRequest, actualAdapterRequests[0], &adapters.ResponseData{StatusCode: 500})

	if err500 == nil {
		t.Errorf("Was expecting error")
	}

	bidResponse := &adapters.ResponseData{
		StatusCode: 200,
		Body: []byte(`{
  "id": "JdSgvXjee0UZ",
  "seatbid": [
    {
      "bid": [
        {
          "id": "16-40dbf1ef_0gKywr9JnzPAW4bE-1",
          "impid": "imp1",
          "price": 2.3456,
          "adm": "<some html here \/>",
          "nurl": "dmxnotificationurlhere",
          "adomain": [
            "brand.com",
            "advertiser.net"
          ],
          "cid": "12345",
          "crid": "232303",
          "cat": [
            "IAB20-3"
          ],
          "attr": [
            2
          ],
          "w": 300,
          "h": 600,
          "language": "en"
        }
      ],
    "seat": "10001"
    }
  ],
  "cur": "USD"
}`),
	}

	bidResponseNoMatch := &adapters.ResponseData{
		StatusCode: 200,
		Body: []byte(`{
  "id": "JdSgvXjee0UZ",
  "seatbid": [
    {
      "bid": [
        {
          "id": "16-40dbf1ef_0gKywr9JnzPAW4bE-1",
          "impid": "djvnsvns",
          "price": 2.3456,
          "adm": "<some html here \/>",
          "nurl": "dmxnotificationurlhere",
          "adomain": [
            "brand.com",
            "advertiser.net"
          ],
          "cid": "12345",
          "crid": "232303",
          "cat": [
            "IAB20-3"
          ],
          "attr": [
            2
          ],
          "w": 300,
          "h": 600,
          "language": "en"
        }
      ],
    "seat": "10001"
    }
  ],
  "cur": "USD"
}`),
	}

	bids, _ := bidder.MakeBids(&inputRequest, actualAdapterRequests[0], bidResponse)
	if bids == nil {
		t.Errorf("ads not parse")
	}
	bidsNoMatching, _ := bidder.MakeBids(&inputRequest, actualAdapterRequests[0], bidResponseNoMatch)
	if bidsNoMatching == nil {
		t.Errorf("ads not parse")
	}

}
func TestUserExtEmptyObject(t *testing.T) {
	var w, h int = 300, 250

	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{
		Endpoint: "https://dmx.districtm.io/b/v2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp1, imp1, imp1},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "10007",
			},
		},
		User: &openrtb2.User{Ext: json.RawMessage(`{}`)},
		ID:   "1234",
	}

	actualAdapterRequests, _ := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
	if len(actualAdapterRequests) != 0 {
		t.Errorf("should have 0 request")
	}
}
func TestUserEidsOnly(t *testing.T) {
	var w, h int = 300, 250

	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{
		Endpoint: "https://dmx.districtm.io/b/v2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp1, imp1, imp1},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "10007",
			},
		},
		User: &openrtb2.User{Ext: json.RawMessage(`{"eids": [{
                "source": "adserver.org",
                "uids": [{
                    "id": "111111111111",
                    "ext": {
                        "rtiPartner": "TDID"
                    }
                }]
            },{
                "source": "netid.de",
                "uids": [{
                    "id": "11111111"
                }]
            }]
            }`)},
		ID: "1234",
	}

	actualAdapterRequests, _ := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
	if len(actualAdapterRequests) != 1 {
		t.Errorf("should have 1 request")
	}
}

func TestUsersEids(t *testing.T) {
	var w, h int = 300, 250

	var width, height int64 = int64(w), int64(h)

	bidder, buildErr := Builder(openrtb_ext.BidderDmx, config.Adapter{
		Endpoint: "https://dmx.districtm.io/b/v2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"bidder\":{\"dmxid\": \"1007\", \"memberid\": \"123456\", \"seller_id\":\"1008\"}}"),
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp1, imp1, imp1},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "10007",
			},
		},
		User: &openrtb2.User{ID: "districtmID", Ext: json.RawMessage(`{"eids": [{
                "source": "adserver.org",
                "uids": [{
                    "id": "111111111111",
                    "ext": {
                        "rtiPartner": "TDID"
                    }
                }]
            },
			{
                "source": "pubcid.org",
                "uids": [{
                    "id": "11111111"
                }]
            },
			{
                "source": "id5-sync.com",
                "uids": [{
                    "id": "ID5-12345"
                }]
            },            
			{
                "source": "parrable.com",
                "uids": [{
                    "id": "01.1563917337.test-eid"
                }]
            },
			{
                "source": "identityLink",
                "uids": [{
                    "id": "11111111"
                }]
            },
			{
                "source": "criteo",
                "uids": [{
                    "id": "11111111"
                }]
            },
			{
                "source": "britepool.com",
                "uids": [{
                    "id": "11111111"
                }]
            },
			{
                "source": "liveintent.com",
                "uids": [{
                    "id": "11111111"
                }]
            },
			{
                "source": "netid.de",
                "uids": [{
                    "id": "11111111"
                }]
            }]
            }`)},
		ID: "1234",
	}

	actualAdapterRequests, _ := bidder.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
	if len(actualAdapterRequests) != 1 {
		t.Errorf("should have 1 request")
	}
	var the_body openrtb2.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &the_body); err != nil {
		t.Errorf("failed to read bid request")
	}

	if len(the_body.Imp) != 3 {
		t.Errorf("must have 3 bids")
	}
}
func TestVideoImpInsertion(t *testing.T) {
	var bidResp openrtb2.BidResponse
	var bid openrtb2.Bid
	payload := []byte(`{
    "id": "some-request-id",
    "seatbid": [
        {
            "bid": [
                {
                    "id": "video1",
                    "impid": "video1",
                    "price": 5.01,
                    "nurl": "https://demo.arripiblik.com/359585167267151",
                    "adm": "<?xml version=\"1.0\" encoding=\"UTF-8\"?><VAST version=\"3.0\"><Ad id=\"5f3c0f61_1aoDbsYiNHbYYpJ2qPtKgERbGiH\"><Wrapper><AdSystem>BidSwitch</AdSystem><VASTAdTagURI><![CDATA[https://bid.g.doubleclick.net/dbm/vast?dbm_c=AKAmf-Bsi1tzgaMFvkO8rVFA2uRjshf-8MKbfsVGvhoqjxCwhfgWnsfVqpOaKDWTYGa5YSdXijhgb8o2TGCuSC2sqawX0WP-Fw&dbm_d=AKAmf-B7vNPyDI7QTdv8f2N0jQJ9hMssfJqj7g1dhwwGRPxcWD8AbrxPgDmysYMj6IOE719Jb9IfX5eUQ7M9cki4w7q3XEI1L1AsUAZYc-HOzPIRQnOOTzypKOmzyfTjoC-r1KNgDeUPv1Z6g4BK6RH9vmRm0ML6wj4S04oJjZDzTE71QGWMZSfAhQhMDzlnSsj0JI1ruWhr_yGER4Qt61oAHC4aUnHWb7V2c7m0Z80mFoDtasDsEjF-QUJrA9LTg_taatSS1emAT2SpSyM98e_66b4YE6dOJG4hqWHl1QAsiDNQOsXZ_oEAGVdnXgS_3VIhdmHy2svbTTtvlzoGTj7tRCrDnmKdJq5dcDZgsZfCeNDBeQQXAzV1L824G5B7MyHKPHksgakRqbAr6y2-VPUuL0eeFGERNBAzQp6r2i6D3w9a3JrsdWPUw_j-6ph9Qw5T-tx8cbZ5zYH1a1RlAIYXuH15Wg-lofEOLFAud59ASP0El_xZK99fcbqcqjqAbaAkzLoTADGRv6ZYZj4wXirZ6R0PAFX4PtXawiRDt9e1oxnweT5_BKx77DtKe21yxGc2QEsQmxmxzSusDYxiCuZuWh7m1Pzp3_WsRTHKEl8T0KZAwtJDn5GjuaR_wxpztusRBtakyhb25e8xICiJvynehLCiTC3jEF5us6_M-y1RD6i3J6VJ5idGBqvYnNk2K_pwsKc0zRbC1hmSNp35xkccXFz9wK1acUq5dEUnRK--49OsfydcdfOKlxMHAbMgB4LJ3RpweBMNt1eeeKNrFRdFDXIYamkdWbr19QhAMjHghibLg6zfgxpxe2Ee0-yQqXx2Dp4pEc-11NUzoIvZuuYIj65YFPXAh9eYU3k8V6iGJcYpoac5nKdZI6qZipNWQ6ZktnUTJ60XR_wk1MlSgVJaGMlagtzuFGDL0joKYDoU898nea406NrqLUsCznXLxBqCmTXtxFZuCDTJ_kwMsyg1J5l8Jbi7McCUsIAI9jm_JhCnIAFE8x_mAANEgPW2KpVKsZP2OufTizw-8ZLS-djLzB-9EMKkOnT8EB6A--PwXyJkQUTqifgfRsg_W3bnVAX2NwpjSC-NuZ2tPIO83O_2M3EctszXJe0A-7cPmhJ68U6LDNm-SFvan4C5gnvi0eHMOJDtMGbCRCy8hqi0VIlU2IZkaL7D8TwAUDFp8zL0fABsnwLUe-14arGfs_4NIIZoJdQXoBG9uz0e6sLY4RjcjK_s60weaoBYxO8kSL-3TsrD-oMVJrCA4NOjnGxobu4H-kYqDkW4bNjA3yQ0LQVQuJUw6RUBZXTSz_9IleIQSey4060dlwA1O1zESsb3oQGa3S7E25Lof3noX5Gq2SwfT8JEoVUI03ORusmqMrnvgqlNP_kq0MPUlb_zzssb5V3jlLVsu9V8svhTIo5nz6zF-Ydp6pqn68bT-ohQzyLPJLJEm1CBEwOvur2TmmyQ-W7MriJCm_uUYfRJGJCXw44whnFKbp3CZeUmFHWLK4lauabyo3csEE5DJkMClctxMxSRp6uZOyiYlOqi8S2YJBvL_NEppzrN2W1cfKEQyNIFj1WsGx62tPY-TzOgArnEetNS3mUHQi5F1Vz6QUdmuIKQpseDqU_GVhAOVCTeh6MNpjM59SXmMFY4ram7U6WZTIniRjX27m-XVaAmzqKH-s0GlmvpO4Fssb9wQc7jHDcqXcsyqbhDTrlIuezBMxjfykPDi2hs96PRv4N5ADvEuKAPkVAKZ2pbt6lr1pZLnSFjXxIlLYXpULQwObVxcdbt0Sw4m6aHtBTyvJnz6bS7z4wpbW87My7swTWwqtBdhEo_jm4_ZHph0FB9_kw46lrWtAIk_qU6AOoE2lAQ8VrRXAOn9LUVeu5JuLCnuVNvMygzMRMn5KvUjGuxMtD53xTcAXgmbd_5Q0eryr4lMKlDFop2Djei9pyxJR1rTQ4xTfHG3kB-MgdyGZLuFuHtILkPjuVQXhNCuBFMe6dRAtuDvptk1y8VAULdda0YNDLo960PxeTFjbqrRz5wfeUqnu1FV6h9cERcJBGSZlAPH254XNcCwwYyyYr0oD2yQZxmSlr2slQpaW-XdwdGie3vlrmYhfe4-5VrDMHymfKIM-ZUuD3sq6V03Efg14JDMghu9IHO0xbefQuTFBCen1AHBVy4mwJm0Kxp6jEgdNXb7Taa0AF2zwzORu0W0P8lTtWFv3Pg5JWplYorfdJSQGr7hDuxLpyeRiJjzft9liSp22APvHumvJCNgNe0DrWddzW0sRMZOLAFQn1F8cQwLX7mIFEplD3MEQht2HYe4gR8sN44lHQw6CmYV0Ai_tjL1oS17BShey-JkVSPdWVp9r3VfNV1releGX3u-12HtheEvOSsv7qQxIU30Ui10QjnSony-ORz48c4_wbhGmucwUx37Ggf5XhmvVjJUEMDlhYtCj2-rtik2-nskOeSNYKo9hb0DPcahsBEiL4vz7fhgEiivFV0FxiHq0pEukcmOFgE6V7X_upwKli65YEOxrH_GZn2DmjjcUJHD0UNLHOe4D7DV16K7YC9HfodCtzeX0LWZnMCKXv76UiXy_sVITPm4G6Td4AXliC2zwiI_URnQqdCyhRIA40E2qjU31ED56DpkOSo1WiU4fv2mv-n5lyb99lzL2srVV0KFYcvwDOyOnNByXmFw18w6-9snWvYtCDY_AbWtvlSI_60OxCOxipqxGVvA4HXWlFUvOl3OtqRf6apMZIaEpFDcL4N0NGuItEKRb04ZE6lv1zNOxC9WBQHZV_pqK9RQ7YbKrWfcL_c79XQji0Bd3mE6k08AfsjYugcWy-wsjmq467VhLWe4OcSAB8B2Qq_eZl5RPrM6LayPWpuz637TeHNfMwuta9R394K36rAtWRf3Ns_oDJz6g&cid=CAASEuRoF_6hDi9IvsHr8we-jyLVhw]]></VASTAdTagURI><Error><![CDATA[https://gce-sc.bidswitch.net/vast_error/O2lw3vm7UGHsHYVBXTjm9Y-DOn5osUxLug71FGBtENsDQCc-y60Y47eIRTQ6TYpRGzvXKzk3o6ZC1FRWXCtQvUdjR-4JrsHILqJGIQjWxB8dGjZlUsDqwpLZNdcECrS16f-XsuNngtxHNKHFdGQ-tfvCihmYUYAPTATvkDSVXDLrwPLAIbEC11ElAmyUUlFqqYMFHvu71ibAo55IU_pQCfvScb0WcKunBti5lcbnSaOny5VE6Kc2fwHXAzG2Tbu4wyRxenfkeMPAaq5aGvIFL2fWlgz9_Y7tUaUq-k6YZxzlTJ3QbwnmbvL3LejwjkG6BnhR5DuJ-X_EWhKWmna0YsxXA-vnFBmqcH6pyuKQ5C93ZApV8xq_N86vccZ2nVTOI6DLL-8N8cVBKvmp2lL0vAPLU2A5uEaaoX8xuw322lv8ksG1McwGwlFoaWkzjBsJivY2eNeFxXCsFBF6BYiBKLCWy62iecTnQfJykTx3orDsnQez89JPW52DQSsxSb0_YPpUpfclYrBe_FAMle4AOBpZs8ib7hsGXxNOkgJdtwz8_bIIlv-U4Hmoym9ulx0svFV78boErVYrNBf6D43tHl027bfkZgXC0CsVpdmZKUHSpcI3mBQKPC5Qb-tCgVpsLK4xUMvFvJFLXIX1yjj062J4ZH9fqm3kNZdfESq4XIVOlx7aMYoDxxnBrSjthW0KKvruGzicYm7c8Nwp19xjTQwR83Cks2FPzxbAm3jAmP0vaNhS7_xkBpN_nVvxis18qaY1YYt4KQAqtxr90Y9rG0wOcmQQf52sbXzd4DqVs3H4PM4v_ZmqJOSFpvDczzKPasn0mV43TyaG-UOnkX1nFqXjMnty8-gpGPqkO5RyhKfP1vTQxpVDI_VaEiOPdNCUwXmhoxJJophsVX4FNBg_kwF-FL2d59j2Oi6gBPASSZ94dr8CMgRC1YZ_SRLwr90h/]]></Error><Impression><![CDATA[https://gce-sc.bidswitch.net/imp/1.9432/BSWhttps_A_B_Badx.g.doubleclick.net_Bpagead_Badview_Cai_RCuMI3Z7KdXrLWMIK5jAan44DIDI3a871cpZKH-u8K7u__b7-kYEAEgg__3mH2DJ3oqLwKTYD8gBBagDAcgDE5gEAKoE4wFP0EQmjUWxRmz76rmwUIT95PKyz7RN57CiJGB-sRzdZcyC4Y-6HUFDwGRTcvbxghpdmUAckrP4ZCE8BqxDB1S1GfTJ0RedqQ__jF1fZSAbUS__sk29Fo0aVpGDQHonn9DXIHSQyqD6KjtPGBdK2zNBgGcCet519-DXoAADunurUqLNITnWZbV__-W3GHtah6ZJB4grkIP3pZqbvCc7p__No0or0BTzAsyVKwm5CUZ6DfDm-GCy3V6SLjPVNtFoo6E89XbxIlSqsXWb8T__YtxVd9k-4at6Qt1sk8jx0SIvu5rcTbeoZ__8AElq-XqskC4AQDiAXksbqxIJIFBggDEAIYAZIFBggbEAIYAZIFCggiEAMYAUjPrVOSBQYIHRAEGAGSBQYIHRABGAGSBQYIHhABGAGQBgGgBk-AB8afu7cBqAeOzhuoB9XJG6gHk9gbqAe6BqgH8NkbqAfy2RuoB-zVG6gHpr4bqAfs1RvYBwDyBwkQ6J17GOTk3WbSCAcIgGEQARgf8ggXYmlkZGVyLWRpc3RyaWN0bV8xNzM2MzSACgTICwGwE9mS0QjIE8jXmAjQEwDYEwqIFALYFAE_Jsigh_Rw8Q7gjefARc_Jcmd_RChdjYS1wdWItNzM1MDg5NzEzODA5OTk1OBAAGAE_Jpr_R38_A_I_WAUCTION__PRICE_X_Jcid_RCAASEuRoF__6hDi9IvsHr8we-jyLVhw_Jtpd_RAGWhJmvwUOaWQX5xLWpPeUyjI1k5ciEvQjY2Irxt7p0U1kq77g/O2lw3vm7UGHsHYVBXTjm9Y-DOn5osUxLug71FGBtENsDQCc-y60Y47eIRTQ6TYpRGzvXKzk3o6ZC1FRWXCtQvUdjR-4JrsHILqJGIQjWxB8dGjZlUsDqwpLZNdcECrS16f-XsuNngtxHNKHFdGQ-tfvCihmYUYAPTATvkDSVXDLrwPLAIbEC11ElAmyUUlFqqYMFHvu71ibAo55IU_pQCfvScb0WcKunBti5lcbnSaOny5VE6Kc2fwHXAzG2Tbu4wyRxenfkeMPAaq5aGvIFL2fWlgz9_Y7tUaUq-k6YZxzlTJ3QbwnmbvL3LejwjkG6BnhR5DuJ-X_EWhKWmna0YsxXA-vnFBmqcH6pyuKQ5C93ZApV8xq_N86vccZ2nVTOI6DLL-8N8cVBKvmp2lL0vAPLU2A5uEaaoX8xuw322lv8ksG1McwGwlFoaWkzjBsJivY2eNeFxXCsFBF6BYiBKLCWy62iecTnQfJykTx3orDsnQez89JPW52DQSsxSb0_YPpUpfclYrBe_FAMle4AOBpZs8ib7hsGXxNOkgJdtwz8_bIIlv-U4Hmoym9ulx0svFV78boErVYrNBf6D43tHl027bfkZgXC0CsVpdmZKUHSpcI3mBQKPC5Qb-tCgVpsLK4xUMvFvJFLXIX1yjj062J4ZH9fqm3kNZdfESq4XIVOlx7aMYoDxxnBrSjthW0KKvruGzicYm7c8Nwp19xjTQwR83Cks2FPzxbAm3jAmP0vaNhS7_xkBpN_nVvxis18qaY1YYt4KQAqtxr90Y9rG0wOcmQQf52sbXzd4DqVs3H4PM4v_ZmqJOSFpvDczzKPasn0mV43TyaG-UOnkX1nFqXjMnty8-gpGPqkO5RyhKfP1vTQxpVDI_VaEiOPdNCUwXmhoxJJophsVX4FNBg_kwF-FL2d59j2Oi6gBPASSZ94dr8CMgRC1YZ_SRLwr90h/]]></Impression><Impression><![CDATA[https://us-east-sync.bidswitch.net/sync_cors?ssp=districtm&dsp_id=16&imp=1]]></Impression><Creatives></Creatives></Wrapper></Ad></VAST><!-- DMX - seat 10009 - crid 16_215446116 -->",
                    "crid": "76575664756",
                    "dealid": "dmx-deal-hp-24",
                    "w": 640,
                    "h": 480,
                    "ext": {
                        "prebid": {
                            "type": "video"
                        }
                    }
                },
                {
                    "id": "some-impression-id",
                    "impid": "some-impression-id",
                    "price": 5.01,
                    "adm": "<img src='https://via.placeholder.com/300x250.png?text=dmx+2.0+300x250' height='250' width='300'/>",
                    "crid": "1346943998",
                    "dealid": "dmx-deal-hp-24",
                    "w": 300,
                    "h": 250,
                    "ext": {
                        "prebid": {
                            "type": "banner"
                        }
                    }
                },
                {
                    "id": "some-impression-id2",
                    "impid": "some-impression-id2",
                    "price": 5.01,
                    "adm": "<img src='https://via.placeholder.com/728x90.png?text=dmx+2.0+728x90' height='90' width='728'/>",
                    "crid": "1424798162",
                    "dealid": "dmx-deal-hp-24",
                    "w": 728,
                    "h": 90,
                    "ext": {
                        "prebid": {
                            "type": "banner"
                        }
                    }
                }
            ],
            "seat": "dmx"
        }
    ]
}`)

	err := json.Unmarshal(payload, &bidResp)
	if err != nil {
		t.Errorf("Payload is invalid")
	}
	bid = openrtb2.Bid(bidResp.SeatBid[0].Bid[0])
	data := videoImpInsertion(&bid)
	find := strings.Index(data, "demo.arripiblik.com")
	if find == -1 {
		t.Errorf("String was not found")
	}

}
