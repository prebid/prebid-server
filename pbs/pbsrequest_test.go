package pbs

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/config"
)

const mimeVideoMp4 = "video/mp4"
const mimeVideoFlv = "video/x-flv"

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
                        "bidder": "ix"
                    },
                    {
                        "bidder": "appnexus"
                    }
                ]
            },
            {
                "code": "second",
                "sizes": [{"w": 728, "h": 90}],
                "media_types" :["banner", "video"],
				"video" : {
					"mimes" : ["video/mp4", "video/x-flv"]
				},
                "bids": [
                    {
                        "bidder": "ix"
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
	hcc := config.HostCookie{}

	pbs_req, err := ParsePBSRequest(r, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, d, &hcc)
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
	if len(pbs_req.Bidders) != 2 {
		t.Fatalf("Should have two bidders not %d", len(pbs_req.Bidders))
	}
	if pbs_req.Bidders[0].BidderCode != "ix" {
		t.Errorf("First bidder not index")
	}
	if len(pbs_req.Bidders[0].AdUnits) != 2 {
		t.Errorf("Index bidder should have 2 ad unit")
	}
	if pbs_req.Bidders[1].BidderCode != "appnexus" {
		t.Errorf("Second bidder not appnexus")
	}
	if len(pbs_req.Bidders[1].AdUnits) != 2 {
		t.Errorf("AppNexus bidder should have 2 ad unit")
	}
	if pbs_req.Bidders[1].AdUnits[0].BidID == "" {
		t.Errorf("ID should have been generated for empty BidID")
	}
	if pbs_req.AdUnits[1].MediaTypes[0] != "banner" {
		t.Errorf("Instead of banner MediaType received %s", pbs_req.AdUnits[1].MediaTypes[0])
	}
	if pbs_req.AdUnits[1].MediaTypes[1] != "video" {
		t.Errorf("Instead of video MediaType received %s", pbs_req.AdUnits[1].MediaTypes[0])
	}
	if pbs_req.AdUnits[1].Video.Mimes[0] != mimeVideoMp4 {
		t.Errorf("Instead of video/mp4 mimes received %s", pbs_req.AdUnits[1].Video.Mimes)
	}
	if pbs_req.AdUnits[1].Video.Mimes[1] != mimeVideoFlv {
		t.Errorf("Instead of video/flv mimes received %s", pbs_req.AdUnits[1].Video.Mimes)
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
                    "bidder": "ix",
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
	hcc := config.HostCookie{}

	d.Config().Set("dummy", dummyConfig)

	pbs_req, err := ParsePBSRequest(r, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, d, &hcc)
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
									"bidder": "ix",
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
											"placementId": "1"
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
                        "bidder": "ix"
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
	hcc := config.HostCookie{}

	d.Config().Set("dummy", dummyConfig)

	pbs_req, err := ParsePBSRequest(r, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, d, &hcc)
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
	if len(pbs_req.Bidders) != 4 {
		t.Fatalf("Should have 4 bidders not %d", len(pbs_req.Bidders))
	}
	if pbs_req.Bidders[0].BidderCode != "ix" {
		t.Errorf("First bidder not index")
	}
	if len(pbs_req.Bidders[0].AdUnits) != 2 {
		t.Errorf("Index bidder should have 1 ad unit")
	}
	if pbs_req.Bidders[1].BidderCode != "appnexus" {
		t.Errorf("Second bidder not appnexus")
	}
	if len(pbs_req.Bidders[1].AdUnits) != 2 {
		t.Errorf("AppNexus bidder should have 2 ad unit")
	}
}

func TestParseMobileRequestFirstVersion(t *testing.T) {
	body := []byte(`{
	   "max_key_length":20,
	   "user":{
	      "gender":0,
	      "buyeruid":"test_buyeruid"
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
	hcc := config.HostCookie{}

	pbs_req, err := ParsePBSRequest(r, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, d, &hcc)
	if err != nil {
		t.Fatalf("Parse simple request failed: %v", err)
	}
	if pbs_req.Tid != "abcd" {
		t.Errorf("Parse TID failed")
	}
	if len(pbs_req.AdUnits) != 1 {
		t.Errorf("Parse ad units failed")
	}
	// We are expecting all user fields to be nil. We don't parse user on v0.0.1 of prebid mobile
	if pbs_req.User.BuyerUID != "" {
		t.Errorf("Parse user buyeruid failed %s", pbs_req.User.BuyerUID)
	}
	if pbs_req.User.Gender != "" {
		t.Errorf("Parse user gender failed %s", pbs_req.User.Gender)
	}
	if pbs_req.User.Yob != 0 {
		t.Errorf("Parse user year of birth failed %d", pbs_req.User.Yob)
	}
	if pbs_req.User.ID != "" {
		t.Errorf("Parse user id failed %s", pbs_req.User.ID)
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
	      "ver":"0.0.2"
	   },
	   "sdk":{
	      "version":"0.0.2",
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
	hcc := config.HostCookie{}

	pbs_req, err := ParsePBSRequest(r, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, d, &hcc)
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
	if pbs_req.App.Ver != "0.0.2" {
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
	if pbs_req.SDK.Version != "0.0.2" {
		t.Errorf("Parse sdk version failed")
	}
	if pbs_req.SDK.Source != "prebid-mobile" {
		t.Errorf("Parse sdk source failed")
	}
	if pbs_req.SDK.Platform != "iOS" {
		t.Errorf("Parse sdk platform failed")
	}
	if pbs_req.Device.IP == "" {
		t.Errorf("Parse device ip failed %s", pbs_req.Device.IP)
	}
}

func TestParseMalformedMobileRequest(t *testing.T) {
	body := []byte(`{
	   "max_key_length":20,
	   "user":{
	      "gender":0,
	      "buyeruid":"test_buyeruid"
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
	hcc := config.HostCookie{}

	pbs_req, err := ParsePBSRequest(r, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, d, &hcc)
	if err != nil {
		t.Fatalf("Parse simple request failed: %v", err)
	}
	if pbs_req.Tid != "abcd" {
		t.Errorf("Parse TID failed")
	}
	if len(pbs_req.AdUnits) != 1 {
		t.Errorf("Parse ad units failed")
	}
	// We are expecting all user fields to be nil. Since no SDK version is passed in
	if pbs_req.User.BuyerUID != "" {
		t.Errorf("Parse user buyeruid failed %s", pbs_req.User.BuyerUID)
	}
	if pbs_req.User.Gender != "" {
		t.Errorf("Parse user gender failed %s", pbs_req.User.Gender)
	}
	if pbs_req.User.Yob != 0 {
		t.Errorf("Parse user year of birth failed %d", pbs_req.User.Yob)
	}
	if pbs_req.User.ID != "" {
		t.Errorf("Parse user id failed %s", pbs_req.User.ID)
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

func TestParseRequestWithInstl(t *testing.T) {
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
	         "bids": [
                    {
                        "bidder": "ix"
                    },
                    {
                        "bidder": "appnexus"
                    }
                ],
	         "code":"5d748364ee9c46a2b112892fc3551b6f",
	         "instl": 1
	      }
	   ],
	   "cache_markup":1,
	   "app":{
	      "bundle":"AppNexus.PrebidMobileDemo",
	      "ver":"0.0.2"
	   },
	   "sdk":{
	      "version":"0.0.2",
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
	hcc := config.HostCookie{}

	pbs_req, err := ParsePBSRequest(r, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, d, &hcc)
	if err != nil {
		t.Fatalf("Parse simple request failed: %v", err)
	}
	if len(pbs_req.Bidders) != 2 {
		t.Errorf("Should have 2 bidders. ")
	}
	if pbs_req.Bidders[0].AdUnits[0].Instl != 1 {
		t.Errorf("Parse instl failed.")
	}
	if pbs_req.Bidders[1].AdUnits[0].Instl != 1 {
		t.Errorf("Parse instl failed.")
	}

}

func TestTimeouts(t *testing.T) {
	doTimeoutTest(t, 10, 15, 10, 0)
	doTimeoutTest(t, 10, 0, 10, 0)
	doTimeoutTest(t, 5, 5, 10, 0)
	doTimeoutTest(t, 15, 15, 0, 0)
	doTimeoutTest(t, 15, 0, 20, 15)
}

func doTimeoutTest(t *testing.T, expected int, requested int, max uint64, def uint64) {
	t.Helper()
	cfg := &config.AuctionTimeouts{
		Default: def,
		Max:     max,
	}
	body := fmt.Sprintf(`{
		"tid": "abcd",
		"timeout_millis": %d,
		"app":{
			"bundle":"AppNexus.PrebidMobileDemo",
			"ver":"0.0.2"
		},
		"ad_units": [
				{
						"code": "first",
						"sizes": [{"w": 300, "h": 250}],
						"bids": [
								{
										"bidder": "ix"
								}
						]
				}
		]
}`, requested)
	r := httptest.NewRequest("POST", "/auction", strings.NewReader(body))
	d, _ := dummycache.New()
	parsed, err := ParsePBSRequest(r, cfg, d, &config.HostCookie{})
	if err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
	if parsed.TimeoutMillis != int64(expected) {
		t.Errorf("Expected %dms timeout, got %dms", expected, parsed.TimeoutMillis)
	}
}

func TestParsePBSRequestUsesHostCookie(t *testing.T) {
	body := []byte(`{
        "tid": "abcd",
        "ad_units": [
            {
                "code": "first",
                "sizes": [{"w": 300, "h": 250}],
                "bidders": [
                {
                    "bidder": "bidder1",
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
	r, err := http.NewRequest("POST", "/auction", bytes.NewBuffer(body))
	r.Header.Add("Referer", "http://nytimes.com/cool.html")
	if err != nil {
		t.Fatalf("new request failed")
	}
	r.AddCookie(&http.Cookie{Name: "key", Value: "testcookie"})
	d, _ := dummycache.New()
	hcc := config.HostCookie{
		CookieName: "key",
		Family:     "family",
		OptOutCookie: config.Cookie{
			Name:  "trp_optout",
			Value: "true",
		},
	}

	pbs_req, err2 := ParsePBSRequest(r, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, d, &hcc)
	if err2 != nil {
		t.Fatalf("Parse simple request failed %v", err2)
	}
	if uid, _, _ := pbs_req.Cookie.GetUID("family"); uid != "testcookie" {
		t.Errorf("Failed to leverage host cookie space for user identifier")
	}
}
