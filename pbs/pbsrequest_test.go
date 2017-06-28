package pbs

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/prebid/prebid-server/cache/dummycache"
)

func TestParseSimpleRequest(t *testing.T) {
	body := []byte(`{
        "tid": "abcd",
        "ad_units": [
            {
                "code": "first",
                "sizes": [{"w": 300, "h": 250}],
                "bids": [
                    {
                        "bidder": "indexExchange"
                    },
                    {
                        "bidder": "appnexus"
                    }
                ]
            },
            {
                "code": "second",
                "sizes": [{"w": 728, "h": 90}],
                "bids": [
                    {
                        "bidder": "indexExchange"
                    },
                    {
                        "bidder": "appnexus"
                    }
                ]
            }
        ]
    }
    `)
	r := httptest.NewRequest("POST", "/auction", bytes.NewBuffer(body))
	r.Header.Add("Referer", "http://nytimes.com/cool.html")
	d, _ := dummycache.New()

	pbs_req, err := ParsePBSRequest(r, d)
	if err != nil {
		t.Fatalf("Parse simple request failed: %v", err)
	}
	if pbs_req.Tid != "abcd" {
		t.Errorf("Parse TID failed")
	}
	if len(pbs_req.AdUnits) != 2 {
		t.Errorf("Parse ad units failed")
	}

	// see if our internal representation is intact
	if len(pbs_req.Bidders) != 3 {
		t.Fatalf("Should have three bidders (2 for index) not %d", len(pbs_req.Bidders))
	}
	if pbs_req.Bidders[0].BidderCode != "indexExchange" {
		t.Errorf("First bidder not index")
	}
	if len(pbs_req.Bidders[0].AdUnits) != 1 {
		t.Errorf("Index bidder should have 1 ad unit")
	}
	if pbs_req.Bidders[1].BidderCode != "appnexus" {
		t.Errorf("Second bidder not appnexus")
	}
	if len(pbs_req.Bidders[1].AdUnits) != 2 {
		t.Errorf("AppNexus bidder should have 2 ad unit")
	}
	if pbs_req.Bidders[2].BidderCode != "indexExchange" {
		t.Errorf("Third bidder not index")
	}
	if len(pbs_req.Bidders[2].AdUnits) != 1 {
		t.Errorf("Index bidder should have 1 ad unit")
	}
	if pbs_req.Bidders[1].AdUnits[0].BidID == "" {
		t.Errorf("UUID should have been generated for empty BidID")
	}
	if pbs_req.Bidders[2].AdUnits[0].BidID == "" {
		t.Errorf("UUID should have been generated for empty BidID")
	}
}

func TestHeaderParsing(t *testing.T) {
	body := []byte(`{
        "tid": "abcd",
        "ad_units": [
            {
                "code": "first",
                "sizes": [{"w": 300, "h": 250}],
                "bidders": [
                {
                    "bidder": "indexExchange",
                    "params": {
                        "id": "417",
                        "siteID": "test-site"
                    }
                }
                ]
            }
        ]
    }
    `)
	r := httptest.NewRequest("POST", "/auction", bytes.NewBuffer(body))
	r.Header.Add("Referer", "http://nytimes.com/cool.html")
	r.Header.Add("User-Agent", "Mozilla/")
	d, _ := dummycache.New()

	d.Config().Set("dummy", dummyConfig)

	pbs_req, err := ParsePBSRequest(r, d)
	if err != nil {
		t.Fatalf("Parse simple request failed")
	}
	if pbs_req.Url != "http://nytimes.com/cool.html" {
		t.Errorf("Failed to pull URL from referrer")
	}
	if pbs_req.Domain != "nytimes.com" {
		t.Errorf("Failed to parse TLD from referrer: %s not nytimes.com", pbs_req.Domain)
	}
	if pbs_req.Device.UA != "Mozilla/" {
		t.Errorf("Failed to pull User-Agent from referrer")
	}
}

var dummyConfig = `
[
							{
									"bidder": "indexExchange",
									"bid_id": "22222222",
									"params": {
											"id": "4",
											"siteID": "186774",
											"timeout": "10000"
									}

							},
							{
									"bidder": "audienceNetwork",
									"bid_id": "22222225",
									"params": {
									}
							},
							{
									"bidder": "pubmatic",
									"bid_id": "22222223",
									"params": {
											"publisherId": "156009",
											"adSlot": "39620189@728x90"
									}
							},
							{
									"bidder": "appnexus",
									"bid_id": "22222224",
									"params": {
											"placementId": "10433394"
									}
							}
					]
		`

func TestParseConfig(t *testing.T) {
	body := []byte(`{
        "tid": "abcd",
        "ad_units": [
            {
                "code": "first",
                "sizes": [{"w": 300, "h": 250}],
                "bids": [
                    {
                        "bidder": "indexExchange"
                    },
                    {
                        "bidder": "appnexus"
                    }
                ]
            },
            {
                "code": "second",
                "sizes": [{"w": 728, "h": 90}],
                "config_id": "abcd"
            }
        ]
    }
    `)
	r := httptest.NewRequest("POST", "/auction", bytes.NewBuffer(body))
	r.Header.Add("Referer", "http://nytimes.com/cool.html")
	d, _ := dummycache.New()

	d.Config().Set("dummy", dummyConfig)

	pbs_req, err := ParsePBSRequest(r, d)
	if err != nil {
		t.Fatalf("Parse simple request failed: %v", err)
	}
	if pbs_req.Tid != "abcd" {
		t.Errorf("Parse TID failed")
	}
	if len(pbs_req.AdUnits) != 2 {
		t.Errorf("Parse ad units failed")
	}

	// see if our internal representation is intact
	if len(pbs_req.Bidders) != 5 {
		t.Fatalf("Should have 5 bidders (2 for index) not %d", len(pbs_req.Bidders))
	}
	if pbs_req.Bidders[0].BidderCode != "indexExchange" {
		t.Errorf("First bidder not index")
	}
	if len(pbs_req.Bidders[0].AdUnits) != 1 {
		t.Errorf("Index bidder should have 1 ad unit")
	}
	if pbs_req.Bidders[1].BidderCode != "appnexus" {
		t.Errorf("Second bidder not appnexus")
	}
	if len(pbs_req.Bidders[1].AdUnits) != 2 {
		t.Errorf("AppNexus bidder should have 2 ad unit")
	}
	if pbs_req.Bidders[2].BidderCode != "indexExchange" {
		t.Errorf("Third bidder not index")
	}
	if len(pbs_req.Bidders[2].AdUnits) != 1 {
		t.Errorf("Index bidder should have 1 ad unit")
	}
}
