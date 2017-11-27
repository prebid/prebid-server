package prebid_cache_client

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fmt"
)

var delay time.Duration
var (
	MaxValueLength = 1024 * 10
	MaxNumValues   = 10
)

type putAnyObject struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

type putAnyRequest struct {
	Puts []putAnyObject `json:"puts"`
}

func DummyPrebidCacheServer(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read the request body.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	var put putAnyRequest

	err = json.Unmarshal(body, &put)
	if err != nil {
		http.Error(w, "Request body "+string(body)+" is not valid JSON.", http.StatusBadRequest)
		return
	}

	if len(put.Puts) > MaxNumValues {
		http.Error(w, fmt.Sprintf("More keys than allowed: %d", MaxNumValues), http.StatusBadRequest)
		return
	}

	resp := response{
		Responses: make([]responseObject, len(put.Puts)),
	}
	for i, p := range put.Puts {
		resp.Responses[i].UUID = fmt.Sprintf("UUID-%d", i+1) // deterministic for testing
		if len(p.Value) > MaxValueLength {
			http.Error(w, fmt.Sprintf("Value is larger than allowed size: %d", MaxValueLength), http.StatusBadRequest)
			return
		}
		if len(p.Value) == 0 {
			http.Error(w, "Missing value.", http.StatusBadRequest)
			return
		}
		if p.Type != "xml" && p.Type != "json" {
			http.Error(w, fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type), http.StatusBadRequest)
			return
		}
	}

	bytes, err := json.Marshal(&resp)
	if err != nil {
		http.Error(w, "Failed to serialize UUIDs into JSON.", http.StatusInternalServerError)
		return
	}
	if delay > 0 {
		<-time.After(delay)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func TestPrebidClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyPrebidCacheServer))
	defer server.Close()

	cobj := make([]*CacheObject, 3)

	// example bids from lifestreet, facebook, and appnexus
	cobj[0] = &CacheObject{
		Value: &BidCache{
			Adm:    "<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\" \"//www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\"><html xmlns=\"//www.w3.org/1999/xhtml\">\n<head> \n<meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\" />\n<!-- Copyright (c) 2016 LifeStreet Corporation -->\n<script>\n    var ad_protocol_scheme = 'http';\n    var hypeAnimationDelayedExecute = false;\n</script>\n<script src=\"http://cdn.lfstmedia.com/~cdn/Ads/ad_shared/js/prefixfree.min.js\"></script>\n<!-- -->\n<script>\nfunction writeViewPort() {\n    var ua = navigator.userAgent;\n    var viewportChanged = false;\n    var scale = 0;\n    if (ua.indexOf(\"Android\") >= 0 && ua.indexOf(\"AppleWebKit\") >= 0) {\n        var webkitVersion = parseFloat(ua.slice(ua.indexOf(\"AppleWebKit\") + 12));\n        if (webkitVersion < 535) {\n            viewportChanged = true;\n            scale = getScaleWithScreenwidth();\n          document.write('<meta name=\"viewport\" content=\"width=320, initial-scale=' + scale + ', minimum-scale=' + scale + ', maximum-scale=' + scale + '\" />');\n        }\n    }\n\n    if (ua.indexOf(\"Firefox\") >= 0) {\n        viewportChanged = true;\n        scale = (getScaleWithScreenwidth() / 2);\n        document.write('<meta name=\"viewport\" content=\"width=320, user-scalable=false, initial-scale=' + scale + '\" />');\n    }\n\n    if (!viewportChanged) {\n        document.write('<meta name=\"viewport\" content=\"width=320, user-scalable=false\" />');\n    }\n\n    if (ua.indexOf(\"IEMobile\") >= 0) {\n        document.write('<meta name=\"MobileOptimized\" content=\"320\" />');\n    }\n\n    document.write('<meta name=\"HandheldFriendly\" content=\"true\"/>');\n}\n\nfunction getScaleWithScreenwidth() {\n    var viewportWidth = 320;\n    var screenWidth = window.screen.width / window.devicePixelRatio;\n    return (screenWidth / viewportWidth);\n}\nwriteViewPort();\n</script>\n\n<!-- -->\n<script src='http://cdn.lfstmedia.com/~cdn/Ads/ad_shared/js/scale.min.js'></script><!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<style>\n/*  */\n    #lsm_mobile_ad.animation_stopped *, .animation_stopped #container * {\n        -webkit-animation: none !important;\n        animation: none !important;\n    }\n    #AnimationFailover { display:none;}\n    .animation_stopped #AnimationFailover { display:block;}\n    body {padding:0; margin:0;-webkit-transform: translateZ(0px);line-height:1.2;}\n    /*  */\n    img{border:0px;}\n    a{text-decoration:none; cursor:pointer; color:#1F768C; z-index: 1;}\n    a:hover, a:hover *{text-decoration:underline;}\n    a.nounderline:hover{text-decoration:none;}\n    .hidden{display:none;}\n    .show{display:block;}\n    \n    #lsm_overlay {left: 0;top: 0;margin: 0;padding: 0;overflow: hidden;}\n    #lsm_overlay_inner {position: absolute;left: 0;top: 0;margin: 0;padding: 0;overflow: hidden;width:320px;height:480px;}\n    /*          */\n        .mp_center {position:static;top:initial;left:initial;margin-left:0px !important;margin-top:0px !important;}        /*  */\n    /*  */\n    #container {position: relative;left: 0;top: 0;margin: 0;padding: 0;width: 320px;height: 480px; background:#FFFFFF; color:#000; font-size:12px; font-family:Arial, Helvetica, sans-serif;z-index:1;text-align:left;}\n    #animation_blocks * {-webkit-user-select: none;-webkit-tap-highlight-color: rgba(0,0,0,0);}\n    #click_overlay{width:320px; height:480px; position:absolute; top: 0px; left: 0px; z-index:9999; display: block;}\n    \n    .text, .logo_extra, .image{position:absolute; float:left;}\n    .image a {display:inline-block;}\n    \n    /*  */\n    \n    /*  */\n    \n    /*  */\n    \n    /*  */\n    \n    /*  */\n    a.hide {\n        visibility:hidden !important;\n    }\n    a.Text1, a.Text2, a.Text3, a.Text4, a.Text5 {\n        visibility:visible !important;\n    }\n    a.hideText1, a.hideText2, a.hideText3, a.hideText4, a.hideText5 {\n        display:none !important;\n    }\n    \n    /*  */\n    \n    /* CSS3 Animations */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n\n    /*  */\n    \n    /*      */\n\n    /*  */\n    \n    /*      */\n\n    /*  */\n    \n    /*      */\n\n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n    /*  */\n    \n    /*      */\n    \n      \n/*  */\n\n/*  */\n</style>\n<!--  -->\n\n<!--  -->\n<!--  -->\n</head>\n\n<body id=\"lsm_mobile_ad\">\n<!-- MoPub RTB cut here BEGIN -->\n<img src=\"http://md-nj.lfstmedia.com/track/1002?__ads=ip8070-qXer7PudnkSfL6VObUy8Aq&__adt=5606652576928142935&__ade=2&__stamp=1506349598367\" width=\"0\" height=\"0\" style=\"position:absolute; visibility:hidden\" />\n<!-- Close Button Underlay\n     -->\n<div id=\"lsm_overlay\">\n<div id=\"lsm_overlay_inner\">\n<!--  -->\n<div id=\"container\">\n<div id=\"animation_blocks\">\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n</div>\n<!-- -->\n    <a id=\"click_overlay\" target=\"_blank\" href=\"http://md-nj.lfstmedia.com/click/cmp6178/5606652576928142935/2?__ads=ip8070-qXer7PudnkSfL6VObUy8Aq&adkey=6f5&_ifa=87ECBA49-908A-428F-9DE7-4B9CED4F486C&_ifamd5=06410baf3501ce3936fbef4c2109cfe6&_ifasha1=1a4ffaf85f3240f9e9f24e7d99c30ec8e38102b5&slot=slot178682&__stamp=1506349598366&ad=crv65774&_cx=$$CX$$&_cy=$$CY$$&_celt=$$ELT-ID$$&redirectURL=\"></a>\n<!--  -->\n<!--  -->\n<!-- Background Elemets -->\n\n<!--  -->\n<!--  -->\n\n<!-- IMAGE -->\n\n<!--  -->\n    <div class=\"image\" style=\"top:0px;left:0px;z-index:1;display:block;width:320px;text-align:left;\">\n        <a id=\"Image1\" href=\"javascript:;\" target=\"_blank\" class=\"Image1_universal\">\n            <img src=\"http://cdn.lfstmedia.com/~cdn/Ads/1b2bd280-d280-1b2b-99d4-6eb6c97ec732.png\" style=\"height:480px;width:320px;\" border=\"0\" />\n        </a>\n    </div>\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n     \n<!-- TEXT -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!--  -->\n<!-- Border\n     -->\n<script type=\"text/javascript\" src=\"https://cdn.lfstmedia.com/~cdn/JS/02/tracking-1.0.js\"></script>\n<script type=\"text/javascript\">\n    var tracking = lsm.Tracking.getInstance();\n    tracking.setConfiguration([\n        {\n            'type': 'open_link',\n            'pixels': []\n        }\n    ]);\n    tracking.addClickEventListener();\n</script>\n</div>\n</div>\n</div>\n<!--\n-->\n<!--  -->\n<!-- -->\n<script type=\"text/javascript\">\n    var scale_ua = navigator.userAgent;\n    if (scale_ua.indexOf(\"Android\") != -1) {\n        document.getElementById(\"lsm_overlay\").style.position = \"static\";\n    }\n    scaleAd(320, 480);\n</script>\n<!-- -->\n\n<!--  -->\n<script>\n</script>\n<!--  -->\n<script>\n    var clickOverlay = document.getElementById(\"click_overlay\");\n    function disableClickOverlay() {\n        clickOverlay.style.zIndex = '0';\n    }\n    function enableClickOverlay() {\n        setTimeout(function(){ clickOverlay.style.zIndex = '9999';}, 500);\n    }  \n</script>\n\n<!--MRC Accredited Traffic Measurement-->\n<!-- Heatmap Pixel -->\n<script type=\"text/javascript\" src=\"http://md-nj.lfstmedia.com/~cdn/JS/02/hm.js\"></script>\n<img height=\"0\" width=\"0\" style=\"border-style:none;\" alt=\"\" src=\"http://md-nj.lfstmedia.com/syspixel?__ads=ip8070-qXer7PudnkSfL6VObUy8Aq&__adt=5606652576928142935&__ade=2&type=tracking&rqc=0w23qR-q7O43weAFbbpzD-gxdTCKySMe-Ef74TByOeGfsOXY0_QKnpwzqvXgrzPZOLZfI__68S9kCKELawQtZcO6kMyvlPM55uCaRZWng_j5btuPaEuXyA&pab=true&__stamp=1506349598367\"/>\n<img height=\"0\" width=\"0\" style=\"border-style:none;\" alt=\"\" src=\"http://md-nj.lfstmedia.com/track/60?__ads=ip8070-qXer7PudnkSfL6VObUy8Aq&__adt=5606652576928142935&__ade=2&__stamp=1506349598367\"/>\n</body>\n</html>",
			NURL:   "http://test.com/syspixel?__ads=ip8070-3PJ4q4QyZxnHE6woGe1sQ3&__adt=4122105428549383603&__ade=2&type=tracking&rqc=0w23qR-q7O7MsGkWlR9wOBm8qL7msKBtSKRJV3Pw0a0tZ47xJTnT2JwzqvXgrzPZOLZfI__68S9kCKELawQtZcO6kMyvlPM55uCaRZWng_j5btuPaEuXyA&pab=true",
			Width:  300,
			Height: 250,
		},
	}
	cobj[1] = &CacheObject{
		Value: &BidCache{
			Adm:    "{\"type\":\"ID\",\"bid_id\":\"8255649814109237089\",\"placement_id\":\"1995257847363113_1997038003851764\",\"resolved_placement_id\":\"1995257847363113_1997038003851764\",\"sdk_version\":\"4.25.0-appnexus.bidding\",\"device_id\":\"87ECBA49-908A-428F-9DE7-4B9CED4F486C\",\"template\":7,\"payload\":\"null\"}",
			NURL:   "https://www.facebook.com/audiencenetwork/nurl/?partner=442648859414574&app=1995257847363113&placement=1997038003851764&auction=d3013e9e-ca55-4a86-9baa-d44e31355e1d&impression=bannerad1&request=7187783259538616534&bid=3832427901228167009&ortb_loss_code=0",
			Width:  300,
			Height: 250,
		},
	}
	cobj[2] = &CacheObject{
		Value: &BidCache{
			Adm:    "<script type=\"application/javascript\" src=\"http://nym1-ib.adnxs.com/ab?e=wqT_3QLVBqBVAwAAAwDWAAUBCN_rpM4FEIziq9qV-8avPRiq8Nq0r-ek-wcqLQkAAAECCOA_EQEHNAAA4D8ZAAAAQOF6hD8hERIAKREJoDCV4t0EOL4HQL4HSAJQy5aGI1it2kRgAGiRQHj14AOAAQCKAQNVU0SSBQbwUpgBrAKgAfoBqAEBsAEAuAECwAEDyAEC0AEC2AEA4AEA8AEAigI7dWYoJ2EnLCAxMzk5NzAwLCAxNTA2MzU4NzUxKTt1ZigncicsIDczNTAxNTE1Nh4A8I2SAu0BIXREUGp6d2llLTVJSEVNdVdoaU1ZQUNDdDJrUXdBRGdBUUFSSXZnZFFsZUxkQkZnQVlOSUdhQUJ3WEhqU0E0QUJYSWdCMGdPUUFRR1lBUUdnQVFHb0FRT3dBUUM1QVNtTGlJTUFBT0Ffd1FFcGk0aURBQURnUDhrQmtzSzlsZXQwMGpfWkFRQUFBAQMkUEFfNEFFQTlRRQEOLEFtQUlBb0FJQXRRSQUQAHYNCIh3QUlBeUFJQTRBSUE2QUlBLUFJQWdBTUVrQU1BbUFNQnFBTwXQaHVnTUpUbGxOTWpveU9USXmaAi0hOWdoQW5naQUgAEUN8ChyZHBFSUFRb0FEbzIwAFjYAugH4ALH0wHyAhEKBkFEVl9JRBIHMSlqBRQIQ1BHBRQYMzU0NjYyNwEUCAVDUAET9BUBCDE0OTkwNzUwgAMBiAMBkAMAmAMUoAMBqgMAwAOsAsgDANIDKAgAEiQ0NzJhYjY4MS03MDUxLTQzMjktOTc5MS1hZTI4YTg4ZWJmNmLYAwDgAwDoAwL4AwCABACSBAkvb3BlbnJ0YjKYBACoBLj7A7IEDAgAEAAYACAAMAA4ALgEAMAEAMgEANIECU5ZTTI6MjkyMtoEAggB4AQA8ATLloYjggUZQXBwTmV4dXMuUHJlYmlkTW9iaWxlRGVtb4gFAZgFAKAF____________AaoFJERDMzVGRjlGLTA0RjUtNDBFQi1CRDJFLTA1MzY5QjVCOUMxNsAFAMkFAAAAAAAA8D_SBQkJAAAAAAAAAADYBQHgBQA.&s=49790274de0e076a2b8b9577c2cce27ff3919239&pp=${AUCTION_PRICE}&\"></script>",
			Width:  300,
			Height: 250,
		},
	}

	InitPrebidCache(server.URL)

	ctx := context.TODO()
	err := Put(ctx, cobj)
	if err != nil {
		t.Fatalf("pbc put failed: %v", err)
	}

	if cobj[0].UUID != "UUID-1" {
		t.Errorf("First object UUID was '%s', should have been 'UUID-1'", cobj[0].UUID)
	}
	if cobj[1].UUID != "UUID-2" {
		t.Errorf("Second object UUID was '%s', should have been 'UUID-2'", cobj[1].UUID)
	}
	if cobj[2].UUID != "UUID-3" {
		t.Errorf("Third object UUID was '%s', should have been 'UUID-3'", cobj[2].UUID)
	}

	delay = 5 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err = Put(ctx, cobj)
	if err == nil {
		t.Fatalf("pbc put succeeded but should have timed out")
	}
}

// Prevents #197
func TestEmptyBids(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("The server should not be called.")
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	InitPrebidCache(server.URL)

	if err := Put(context.Background(), []*CacheObject{}); err != nil {
		t.Errorf("Error on Put: %v", err)
	}
}
