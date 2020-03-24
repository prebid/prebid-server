package dmx

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

var (
	bidRequest string
)

func TestFetchParams(t *testing.T) {
	var w, h int = 300, 250

	var width, height uint64 = uint64(w), uint64(h)
	var arrImp []openrtb.Imp
	var imps = fetchParams(
		dmxExt{Bidder: dmxParams{
			TagId:       "222",
			PublisherId: "5555",
		}},
		openrtb.Imp{ID: "32"},
		openrtb.Imp{ID: "32"},
		arrImp,
		openrtb.Banner{W: &width, H: &height, Format: []openrtb.Format{
			{W: 300, H: 250},
		}},
		1)
	var imps2 = fetchParams(
		dmxExt{Bidder: dmxParams{
			DmxId:    "222",
			MemberId: "5555",
		}},
		openrtb.Imp{ID: "32"},
		openrtb.Imp{ID: "32"},
		arrImp,
		openrtb.Banner{W: &width, H: &height, Format: []openrtb.Format{
			{W: 300, H: 250},
		}},
		1)
	if len(imps) == 0 {
		t.Errorf("should increment the length by one")
	}

	if len(imps2) == 0 {
		t.Errorf("should increment the length by one")
	}

}
func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "dmxtest", new(DmxAdapter))
}

func TestMakeRequestsOtherPlacement(t *testing.T) {
	var w, h int = 300, 250

	var width, height uint64 = uint64(w), uint64(h)

	adapter := NewDmxBidder("https://dmx.districtm.io/b/v2", "10007")
	imp1 := openrtb.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"tagid\": \"1007\", \"placement_id\": \"123456\"}"),
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
				ID: "10007",
			},
		},
		ID: "1234",
	}

	actualAdapterRequests, err := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
	if actualAdapterRequests != nil {
		t.Errorf("request should be nil")
	}
	if len(err) == 0 {
		t.Errorf("We should have at least one Error")
	}

}

func TestMakeRequestsInvalid(t *testing.T) {
	var w, h int = 300, 250

	var width, height uint64 = uint64(w), uint64(h)

	adapter := NewDmxBidder("https://dmx.districtm.io/b/v2", "10007")
	imp1 := openrtb.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"dmxid\": \"1007\", \"memberid\": \"123456\"}"),
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
				ID: "10007",
			},
		},
		ID: "1234",
	}

	actualAdapterRequests, err := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})
	if actualAdapterRequests != nil {
		t.Errorf("request should be nil")
	}
	if len(err) == 0 {
		t.Errorf("We should have at least one Error")
	}

}

func TestMakeRequestsNoImp(t *testing.T) {
	adapter := NewDmxBidder("https://dmx.districtm.io/b/v2", "10007")
	inputRequest := openrtb.BidRequest{
		Imp: []openrtb.Imp{},
		Site: &openrtb.Site{
			Publisher: &openrtb.Publisher{
				ID: "10007",
			},
		},
		App: &openrtb.App{ID: "cansanuabnua", Publisher: &openrtb.Publisher{ID: "whatever"}},
		ID:  "1234",
	}
	actualAdapterRequests, err := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if actualAdapterRequests != nil {
		t.Errorf("request should be nil")
	}
	if len(err) == 0 {
		t.Errorf("We should have at least one Error")
	}
}

func TestMakeRequestsApp(t *testing.T) {
	var w, h int = 300, 250

	var width, height uint64 = uint64(w), uint64(h)

	adapter := NewDmxBidder("https://dmx.districtm.io/b/v2", "10007")
	imp1 := openrtb.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"dmxid\": \"1007\", \"memberid\": \"123456\"}"),
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
				ID: "10007",
			},
		},
		App: &openrtb.App{ID: "cansanuabnua", Publisher: &openrtb.Publisher{ID: "whatever"}},
		ID:  "1234",
	}

	actualAdapterRequests, _ := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if len(actualAdapterRequests) != 1 {
		t.Errorf("openrtb type should be an Array when it's an App")
	}
	var the_body openrtb.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &the_body); err != nil {
		t.Errorf("failed to read bid request")
	}

	if the_body.App == nil {
		t.Errorf("app property should be populated")
	}

}

func TestMakeRequestsNoUser(t *testing.T) {
	var w, h int = 300, 250

	var width, height uint64 = uint64(w), uint64(h)

	adapter := NewDmxBidder("https://dmx.districtm.io/b/v2", "10007")
	imp1 := openrtb.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"dmxid\": \"1007\", \"memberid\": \"123456\"}"),
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
				ID: "10007",
			},
		},
		ID: "1234",
	}

	actualAdapterRequests, _ := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if actualAdapterRequests != nil {
		t.Errorf("openrtb type should be empty")
	}

}

func TestMakeRequests(t *testing.T) {
	//server := httptest.NewServer(http.HandlerFunc(DummyDmxServer))
	var w, h int = 300, 250

	var width, height uint64 = uint64(w), uint64(h)

	adapter := NewDmxBidder("https://dmx.districtm.io/b/v2", "10007")
	imp1 := openrtb.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"dmxid\": \"1007\", \"memberid\": \"123456\"}"),
		Banner: &openrtb.Banner{
			W: &width,
			H: &height,
			Format: []openrtb.Format{
				{W: 300, H: 250},
			},
		}}
	imp2 := openrtb.Imp{
		ID:  "imp2",
		Ext: json.RawMessage("{\"dmxid\": \"1007\", \"memberid\": \"123456\"}"),
		Banner: &openrtb.Banner{
			W: &width,
			H: &height,
			Format: []openrtb.Format{
				{W: 300, H: 250},
			},
		}}
	imp3 := openrtb.Imp{
		ID:  "imp3",
		Ext: json.RawMessage("{\"dmxid\": \"1007\", \"memberid\": \"123456\"}"),
		Banner: &openrtb.Banner{
			W: &width,
			H: &height,
			Format: []openrtb.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb.BidRequest{
		Imp: []openrtb.Imp{imp1, imp2, imp3},
		Site: &openrtb.Site{
			Publisher: &openrtb.Publisher{
				ID: "10007",
			},
		},
		User: &openrtb.User{ID: "districtmID"},
		ID:   "1234",
	}

	actualAdapterRequests, _ := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	if len(actualAdapterRequests) != 1 {
		t.Errorf("should have 1 request")
	}
	var the_body openrtb.BidRequest
	if err := json.Unmarshal(actualAdapterRequests[0].Body, &the_body); err != nil {
		t.Errorf("failed to read bid request")
	}

	if len(the_body.Imp) != 3 {
		t.Errorf("must have 3 bids")
	}

}

func TestMakeBidsNoContent(t *testing.T) {
	var w, h int = 300, 250

	var width, height uint64 = uint64(w), uint64(h)

	adapter := NewDmxBidder("https://dmx.districtm.io/b/v2", "10007")
	imp1 := openrtb.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"dmxid\": \"1007\", \"memberid\": \"123456\"}"),
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
				ID: "10007",
			},
		},
		User: &openrtb.User{ID: "districtmID"},
		ID:   "1234",
	}

	actualAdapterRequests, _ := adapter.MakeRequests(&inputRequest, &adapters.ExtraRequestInfo{})

	_, err204 := adapter.MakeBids(&inputRequest, actualAdapterRequests[0], &adapters.ResponseData{StatusCode: 204})

	if err204 == nil {
		t.Errorf("Was expecting error")
	}

	_, err400 := adapter.MakeBids(&inputRequest, actualAdapterRequests[0], &adapters.ResponseData{StatusCode: 400})

	if err400 == nil {
		t.Errorf("Was expecting error")
	}

	_, err500 := adapter.MakeBids(&inputRequest, actualAdapterRequests[0], &adapters.ResponseData{StatusCode: 500})

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

	bids, _ := adapter.MakeBids(&inputRequest, actualAdapterRequests[0], bidResponse)
	if bids == nil {
		t.Errorf("ads not parse")
	}
	bidsNoMatching, _ := adapter.MakeBids(&inputRequest, actualAdapterRequests[0], bidResponseNoMatch)
	if bidsNoMatching == nil {
		t.Errorf("ads not parse")
	}

}
