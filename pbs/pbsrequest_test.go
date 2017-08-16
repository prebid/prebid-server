package pbs

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/prebid/prebid-server/cache/dummycache"
)

func TestParseMediaTypes(t *testing.T) {
	types1 := []string{"Banner"}
	t1 := ParseMediaTypes(types1)
	assert.Equal(t, len(t1), 1)
	assert.Equal(t, t1[0], MEDIA_TYPE_BANNER)

	types2 := []string{"Banner", "Video"}
	t2 := ParseMediaTypes(types2)
	assert.Equal(t, len(t2), 2)
	assert.Equal(t, t2[0], MEDIA_TYPE_BANNER)
	assert.Equal(t, t2[1], MEDIA_TYPE_VIDEO)

	types3 := []string{"Banner", "Vo"}
	t3 := ParseMediaTypes(types3)
	assert.Equal(t, len(t3), 1)
	assert.Equal(t, t3[0], MEDIA_TYPE_BANNER)
}

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
		t.Errorf("ID should have been generated for empty BidID")
	}
	if pbs_req.Bidders[2].AdUnits[0].BidID == "" {
		t.Errorf("ID should have been generated for empty BidID")
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

func TestParseMobileRequest(t *testing.T) {
	body := []byte(`{
	   "max_key_length":20,
	   "user":{
	      "gender":"F",
	      "buyeruid":"test_buyeruid",
	      "yob":2000,
	      "id":"testid"
	   },
	   "prebid_version":"0.21.0-pre",
	   "sort_bids":1,
	   "ad_units":[
	      {
	         "sizes":[
	            {
	               "w":300,
	               "h":250
	            }
	         ],
	         "config_id":"ad5ffb41-3492-40f3-9c25-ade093eb4e5f",
	         "code":"5d748364ee9c46a2b112892fc3551b6f"
	      }
	   ],
	   "cache_markup":1,
	   "app":{
	      "bundle":"AppNexus.PrebidMobileDemo",
	      "ver":"0.0.1"
	   },
	   "sdk":{
	      "version":"0.0.1",
	      "platform":"iOS",
	      "source":"prebid-mobile"
	   },
	   "device":{
	      "ifa":"test_device_ifa",
	      "osv":"9.3.5",
	      "os":"iOS",
	      "make":"Apple",
	      "model":"iPhone6,1"
	   },
	   "tid":"abcd",
	   "account_id":"aecd6ef7-b992-4e99-9bb8-65e2d984e1dd"
	}
    `)
	r := httptest.NewRequest("POST", "/auction", bytes.NewBuffer(body))
	d, _ := dummycache.New()

	pbs_req, err := ParsePBSRequest(r, d)
	if err != nil {
		t.Fatalf("Parse simple request failed: %v", err)
	}
	if pbs_req.Tid != "abcd" {
		t.Errorf("Parse TID failed")
	}
	if len(pbs_req.AdUnits) != 1 {
		t.Errorf("Parse ad units failed")
	}

	if pbs_req.User.BuyerUID != "test_buyeruid" {
		t.Errorf("Parse user buyeruid failed")
	}
	if pbs_req.User.Gender != "F" {
		t.Errorf("Parse user gender failed")
	}
	if pbs_req.User.Yob != 2000 {
		t.Errorf("Parse user year of birth failed")
	}
	if pbs_req.User.ID != "testid" {
		t.Errorf("Parse user id failed")
	}
	if pbs_req.App.Bundle != "AppNexus.PrebidMobileDemo" {
		t.Errorf("Parse app bundle failed")
	}
	if pbs_req.App.Ver != "0.0.1" {
		t.Errorf("Parse app version failed")
	}

	if pbs_req.Device.IFA != "test_device_ifa" {
		t.Errorf("Parse device ifa failed")
	}
	if pbs_req.Device.OSV != "9.3.5" {
		t.Errorf("Parse device osv failed")
	}
	if pbs_req.Device.OS != "iOS" {
		t.Errorf("Parse device os failed")
	}
	if pbs_req.Device.Make != "Apple" {
		t.Errorf("Parse device make failed")
	}
	if pbs_req.Device.Model != "iPhone6,1" {
		t.Errorf("Parse device model failed")
	}
}
